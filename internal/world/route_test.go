package world

import (
	"testing"

	"github.com/ssn688/sim/assets"
)

func TestRouteResumePrefersNearAndEnd(t *testing.T) {
	r := &Route{
		ID: "t",
		Waypoints: []Waypoint{
			{X: 0, Y: 0},
			{X: 1000, Y: 0},
			{X: 2000, Y: 0},
			{X: 3000, Y: 0},
			{X: 0, Y: 0},
		},
		Looped: true,
	}
	idx := r.ResumeWaypointIndex(1500, 50)
	if idx != 2 {
		t.Fatalf("resume idx=%d want 2 (near + toward end)", idx)
	}
	idx = r.ResumeWaypointIndex(2900, 0)
	if idx != 3 {
		t.Fatalf("near end: idx=%d want 3", idx)
	}
}

func TestRouteAdvanceAndVisit(t *testing.T) {
	r := &Route{
		ID:        "loop",
		Waypoints: []Waypoint{{X: 0, Y: 0}, {X: 1000, Y: 0}, {X: 0, Y: 0}},
		Looped:    true,
	}
	if r.UniqueCount() != 2 {
		t.Fatalf("unique=%d", r.UniqueCount())
	}
	if r.AdvanceIndex(0) != 1 {
		t.Fatal("advance 0→1")
	}
	if r.AdvanceIndex(1) != 0 {
		t.Fatal("loop wrap")
	}
}

func TestRoutePingPongAdvance(t *testing.T) {
	r := &Route{
		ID: "pp",
		Waypoints: []Waypoint{
			{X: 0, Y: 0},
			{X: 1000, Y: 0},
			{X: 2000, Y: 0},
		},
		PingPong: true,
	}
	idx, dir := r.Advance(2, 1)
	if idx != 1 || dir != -1 {
		t.Fatalf("end reverse got %d/%d want 1/-1", idx, dir)
	}
}

func TestDiagonalRoutesTrainingScenario(t *testing.T) {
	b, err := LoadBathymetry(assets.BathyChart)
	if err != nil {
		t.Fatal(err)
	}
	SetDefaultBathymetry(b)
	sc := NewTrainingScenario()
	if len(sc.Routes) < 4 {
		t.Fatalf("want transit + ally routes, got %d", len(sc.Routes))
	}
	minX, minY, maxX, maxY := sc.Bathy.BoundsYards()
	var allyRoute *Route
	transit := make([]*Route, 0, len(sc.Routes))
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
		// Roughly NW → SE: first more north/west-ish than last.
		if first.Y < last.Y-2000 {
			t.Fatalf("%s not NW→SE oriented: first=(%.0f,%.0f) last=(%.0f,%.0f)", r.ID, first.X, first.Y, last.X, last.Y)
		}
		if first.X > last.X+2000 {
			t.Fatalf("%s not NW→SE oriented X", r.ID)
		}
		for i := 0; i < r.UniqueCount(); i++ {
			wp := r.Waypoints[i]
			if !sc.Bathy.NavigableFor(wp.X, wp.Y, KindSurfaceShip, 0) {
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
	// SE → … → NW: start lower-right, end upper-left.
	if af.X < al.X-1000 || af.Y > al.Y+1000 {
		t.Fatalf("ally edge not SE→NW: first=(%.0f,%.0f) last=(%.0f,%.0f)", af.X, af.Y, al.X, al.Y)
	}
	dRoutes := MinDistToRoutesYd(sc.Player.X, sc.Player.Y, transit)
	if dRoutes < 500 || dRoutes > 3000 {
		t.Fatalf("player transit clearance=%.0f want ≤3000 (and off the lane)", dRoutes)
	}
	// SW-ish: closer to SW corner than to NE.
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

func hypot2(dx, dy float64) float64 {
	return dx*dx + dy*dy
}
