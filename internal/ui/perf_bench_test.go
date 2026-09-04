package ui

import (
	"fmt"
	"math"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/config"
	"github.com/bubnov-mikhail/ssn688/internal/sim"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func benchApp() *App {
	a := NewApp(config.Settings{}, nil)
	a.Engine = sim.NewEngine(campaign.DemoRuntime())
	a.ensureTactical()
	return a
}

func TestWaterfallChipCacheReusesLayout(t *testing.T) {
	a := benchApp()
	sonar := &a.Engine.Sonar
	sonar.Contacts = []acoustics.Contact{
		{ID: "C01", SourceEntityID: "e1", BearingDeg: 45, EstimatedRangeYd: 5000},
		{ID: "C02", SourceEntityID: "e2", BearingDeg: 120, EstimatedRangeYd: 6000},
	}
	a.waterfallStamp = 1
	c1 := a.cachedWaterfallContactChips(sonar)
	c2 := a.cachedWaterfallContactChips(sonar)
	if len(c1) == 0 || len(c1) != len(c2) {
		t.Fatalf("chips=%d %d", len(c1), len(c2))
	}
	if &c1[0] != &c2[0] {
		t.Fatal("expected cached slice reuse")
	}
}

func TestAppendAllEntitiesNoAllocWhenWarm(t *testing.T) {
	sc := campaign.DemoRuntime()
	dst := make([]*world.Entity, 0, 16)
	dst = sc.AppendAllEntities(dst)
	allocs := testing.AllocsPerRun(100, func() {
		_ = sc.AppendAllEntities(dst[:0])
	})
	if allocs != 0 {
		t.Fatalf("AppendAllEntities warm path allocs=%v", allocs)
	}
}

func BenchmarkBuildWaterfallContactChips(b *testing.B) {
	a := benchApp()
	sonar := &a.Engine.Sonar
	sonar.Contacts = make([]acoustics.Contact, 12)
	for i := range sonar.Contacts {
		sonar.Contacts[i] = acoustics.Contact{
			ID:             fmt.Sprintf("C%02d", i+1),
			SourceEntityID: fmt.Sprintf("src%d", i),
			BearingDeg:     float64(i * 27),
		}
	}
	dst := make([]contactChip, 0, 12)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dst = a.buildWaterfallContactChips(sonar, dst[:0])
	}
}

func BenchmarkTacticalBathyRaster(b *testing.B) {
	a := benchApp()
	bathy := a.Engine.Scenario.Bathy
	mapX := tacticalPanelX + 8
	mapY := tacticalPanelY + 40
	mapW := tacticalPanelW() - 16
	mapH := tacticalPanelH - 52
	cx, cy := a.tacticalViewCenter()
	view := tacticalMapView{mapX, mapY, mapW, mapH, cx, cy, a.tactical.zoom}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		a.invalidateTacticalBathy()
		_, _ = a.ensureTacticalBathyImage(view, bathy, false)
	}
}

func BenchmarkCachedWaterfallContactChips(b *testing.B) {
	a := benchApp()
	sonar := &a.Engine.Sonar
	sonar.Contacts = []acoustics.Contact{
		{ID: "C01", SourceEntityID: "e1", BearingDeg: 45},
	}
	a.waterfallStamp = 1
	a.cachedWaterfallContactChips(sonar)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = a.cachedWaterfallContactChips(sonar)
	}
}

func periBenchReady(a *App) (*world.Entity, *acoustics.PeriscopeState, world.Weather) {
	player := a.Engine.Scenario.Player
	player.DepthFt = 60
	peri := &a.Engine.Periscope
	peri.Extension = 1
	peri.Order = acoustics.PeriMastRaise
	peri.Zoom = acoustics.PeriZoomLow
	peri.TrainRelDeg = 20
	a.ensurePeriscopeImage()
	ensurePeriShipSprites()
	return player, peri, a.Engine.Scenario.Weather
}

func TestPeriBackgroundCacheSkipsRebuild(t *testing.T) {
	prev := periBGCacheEnabled
	periBGCacheEnabled = true
	t.Cleanup(func() { periBGCacheEnabled = prev })

	a := benchApp()
	player, peri, weather := periBenchReady(a)
	gt := 1.0
	a.renderPeriscopeIRFrame(player, peri, weather, gt)
	key1 := a.periBgCacheKey
	bg0 := a.periBgPix[0]

	// Move only surface ships — background plate must stay cached.
	for _, e := range a.Engine.Scenario.Entities {
		if e != nil && e.Kind == world.KindSurfaceShip {
			e.X += 40
			e.Y += 25
			e.HeadingDeg = math.Mod(e.HeadingDeg+7, 360)
		}
	}
	a.renderPeriscopeIRFrame(player, peri, weather, gt)
	if a.periBgCacheKey != key1 {
		t.Fatalf("bg cache key changed on ship motion: %d → %d", key1, a.periBgCacheKey)
	}
	if a.periBgPix[0] != bg0 {
		t.Fatal("bg plate rewritten while view pose was static")
	}
}

func BenchmarkPeriscopeIRFrame(b *testing.B) {
	type mode struct {
		name  string
		cache bool
		sweep bool // change train each iter → force bg rebuild
	}
	modes := []mode{
		{name: "BGCache_staticView", cache: true, sweep: false},
		{name: "NoCache_staticView", cache: false, sweep: false},
		{name: "BGCache_trainSweep", cache: true, sweep: true},
		{name: "NoCache_trainSweep", cache: false, sweep: true},
	}
	for _, m := range modes {
		b.Run(m.name, func(b *testing.B) {
			prev := periBGCacheEnabled
			periBGCacheEnabled = m.cache
			b.Cleanup(func() { periBGCacheEnabled = prev })

			a := benchApp()
			player, peri, weather := periBenchReady(a)
			gt := 2.0
			a.renderPeriscopeIRFrame(player, peri, weather, gt) // warm

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Ships always move: foreground is live either way.
				for _, e := range a.Engine.Scenario.Entities {
					if e != nil && e.Kind == world.KindSurfaceShip {
						e.X += 3
						e.HeadingDeg = math.Mod(e.HeadingDeg+0.4, 360)
					}
				}
				if m.sweep {
					peri.TrainRelDeg = math.Mod(peri.TrainRelDeg+1.1, 360)
				}
				a.renderPeriscopeIRFrame(player, peri, weather, gt)
			}
		})
	}
}
