package campaign

import (
	"math"
	"strings"
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestDemoRuntimeInlineChart(t *testing.T) {
	ReloadScenarios()
	scDef := ScenarioByID(DemoScenarioID)
	if scDef == nil || len(scDef.Theaters) == 0 || scDef.Theaters[0].Chart == nil || !scDef.Theaters[0].Chart.Valid() {
		t.Fatal("demo scenario missing inline theater bathy")
	}
	sc := DemoRuntime()
	if sc == nil || sc.Bathy == nil || !sc.Bathy.Valid() {
		t.Fatal("demo runtime missing bathymetry")
	}
	if sc.Bathy != scDef.Theaters[0].Chart {
		t.Fatal("demo should use inline theater chart from JSON")
	}
	for _, e := range sc.AllEntities() {
		if !sc.Bathy.NavigableFor(e.X, e.Y, e.Kind, e.DepthFt) {
			t.Fatalf("%s spawned in invalid water at (%.0f,%.0f)", e.ID, e.X, e.Y)
		}
	}
}

func TestDemoDiagonalPlacement(t *testing.T) {
	ReloadScenarios()
	for i := 0; i < 8; i++ {
		sc := DemoRuntime()
		if sc.Player == nil {
			t.Fatal("missing player")
		}
		if d := world.MinDistToRoutesYd(sc.Player.X, sc.Player.Y, sc.Routes); d < 500 || d > 3000 {
			t.Fatalf("attempt %d: player–route=%.0f want ≤3000 yd", i, d)
		}
		var fox, grisha, allyDD, allySS *world.Entity
		civs := 0
		for _, e := range sc.Entities {
			if e == nil {
				continue
			}
			switch e.ID {
			case "enemy_foxtrot":
				fox = e
			case "enemy_grisha":
				grisha = e
			case "ally_spruance":
				allyDD = e
			case "ally_688":
				allySS = e
			case "civ_merchant", "civ_tanker", "civ_trawler":
				civs++
			}
			if e.Side != world.SidePlayer && e.RouteID == "" {
				t.Fatalf("%s has no route", e.ID)
			}
			if (e.ID == "ally_spruance" || e.ID == "ally_688") && e.RouteID == "" {
				t.Fatalf("%s has no route", e.ID)
			}
		}
		if fox == nil || grisha == nil || civs != 3 {
			t.Fatalf("units missing fox=%v grisha=%v civs=%d", fox != nil, grisha != nil, civs)
		}
		if allyDD == nil || allySS == nil {
			t.Fatal("missing ally Spruance / ally 688")
		}
		if fox.Defcon != world.DefconPassive || grisha.Defcon != world.DefconPassive {
			t.Fatal("hostiles should start passive")
		}
		if allyDD.Defcon != world.DefconHostile || allySS.Defcon != world.DefconHostile {
			t.Fatalf("allies want DEFCON 2, got dd=%d ss=%d", allyDD.Defcon, allySS.Defcon)
		}
		minX, minY, maxX, maxY := sc.Bathy.BoundsYards()
		seX, seY := maxX-1800, minY+1800
		for _, a := range []*world.Entity{allyDD, allySS} {
			if a.RouteID != "route_ally_edge" {
				t.Fatalf("%s route=%q want route_ally_edge", a.ID, a.RouteID)
			}
			if a.Side != world.SidePlayer {
				t.Fatalf("%s side %v", a.ID, a.Side)
			}
			dSE := math.Hypot(a.X-seX, a.Y-seY)
			dSW := math.Hypot(a.X-(minX+1800), a.Y-(minY+1800))
			if dSE > dSW {
				t.Fatalf("%s not near SE: se=%.0f sw=%.0f", a.ID, dSE, dSW)
			}
			if dSE > 12000 {
				t.Fatalf("%s too far from SE corner: %.0f yd", a.ID, dSE)
			}
		}
		_ = maxY
		if fox.CrewSkill < 20 || fox.CrewSkill > 40 {
			t.Fatalf("foxtrot crew skill %.1f want 30±10", fox.CrewSkill)
		}
		if grisha.CrewSkill < 40 || grisha.CrewSkill > 80 {
			t.Fatalf("grisha crew skill %.1f want 60±20", grisha.CrewSkill)
		}
	}
}

func TestDemoDiagonalRoutes(t *testing.T) {
	ReloadScenarios()
	sc := DemoRuntime()
	if len(sc.Routes) < 4 {
		t.Fatalf("want transit + ally routes, got %d", len(sc.Routes))
	}
	minX, minY, maxX, maxY := sc.Bathy.BoundsYards()
	var allyRoute *world.Route
	transit := make([]*world.Route, 0, len(sc.Routes))
	for _, r := range sc.Routes {
		if r == nil || !r.PingPong {
			t.Fatalf("route %+v want PingPong", r)
		}
		if r.ID == "route_ally_edge" {
			allyRoute = r
			continue
		}
		transit = append(transit, r)
		if r.UniqueCount() < 20 {
			t.Fatalf("%s want dense PingPong (≥20 WP), got %d", r.ID, r.UniqueCount())
		}
		first, last := r.Waypoints[0], r.Waypoints[r.UniqueCount()-1]
		if first.Y < last.Y-2000 {
			t.Fatalf("%s not NW→SE oriented: first=(%.0f,%.0f) last=(%.0f,%.0f)", r.ID, first.X, first.Y, last.X, last.Y)
		}
		if first.X > last.X+2000 {
			t.Fatalf("%s not NW→SE oriented X", r.ID)
		}
		for i := 0; i < r.UniqueCount(); i++ {
			wp := r.Waypoints[i]
			if !sc.Bathy.NavigableFor(wp.X, wp.Y, world.KindSurfaceShip, 0) {
				t.Fatalf("%s wp%d not navigable", r.ID, i)
			}
			if d := sc.Bathy.DistanceToShoreYd(wp.X, wp.Y); d < 900 {
				t.Fatalf("%s wp%d shore=%.0f", r.ID, i, d)
			}
		}
	}
	if allyRoute == nil || allyRoute.UniqueCount() < 4 {
		t.Fatal("missing ally edge patrol route")
	}
	af, al := allyRoute.Waypoints[0], allyRoute.Waypoints[allyRoute.UniqueCount()-1]
	if af.X < al.X-1000 || af.Y > al.Y+1000 {
		t.Fatalf("ally edge not SE→NW: first=(%.0f,%.0f) last=(%.0f,%.0f)", af.X, af.Y, al.X, al.Y)
	}
	dRoutes := world.MinDistToRoutesYd(sc.Player.X, sc.Player.Y, transit)
	if dRoutes < 500 || dRoutes > 3000 {
		t.Fatalf("player transit clearance=%.0f want ≤3000 (and off the lane)", dRoutes)
	}
	sw := hypot2(sc.Player.X-(minX+1800), sc.Player.Y-(minY+1800))
	ne := hypot2(sc.Player.X-(maxX-1800), sc.Player.Y-(maxY-1800))
	if sw > ne {
		t.Fatalf("player not near SW corner: sw=%.0f ne=%.0f", sw, ne)
	}
	_ = maxY
	for _, e := range sc.Entities {
		if e == nil {
			continue
		}
		if e.RouteID == "" {
			t.Fatalf("%v missing route", e)
		}
	}
}

func TestDemoFollowOnTasking(t *testing.T) {
	ReloadScenarios()
	sc := DemoRuntime()
	if sc == nil || sc.CommBriefing.GetText("en") == "" {
		t.Fatal("missing briefing")
	}
	if len(sc.CommSchedule) == 0 || sc.CommSchedule[0].AtSec != 20 {
		t.Fatal("expected 20s follow-on")
	}
	txt := sc.CommSchedule[0].Text.GetText("en")
	for _, needle := range []string{"Primary", "Secondary", "identify", "3000", "80%", "tanker"} {
		if !strings.Contains(strings.ToLower(txt), strings.ToLower(needle)) {
			t.Fatalf("tasking missing %q:\n%s", needle, txt)
		}
	}
}

func hypot2(dx, dy float64) float64 {
	return dx*dx + dy*dy
}
