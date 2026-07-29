package ui

import (
	"fmt"
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/config"
	"github.com/ssn688/sim/internal/sim"
	"github.com/ssn688/sim/internal/world"
)

func benchApp() *App {
	a := NewApp(config.Settings{}, nil)
	a.Engine = sim.NewEngine(world.NewTrainingScenario())
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
	sc := world.NewTrainingScenario()
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
	mapW := tacticalPanelW - 16
	mapH := tacticalPanelH - 52
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		a.invalidateTacticalBathy()
		_ = a.ensureTacticalBathyImage(mapX, mapY, mapW, mapH, bathy)
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
