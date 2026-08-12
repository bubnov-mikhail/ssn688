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
	if len(sc.Routes) < 3 {
		t.Fatalf("want transit routes, got %d", len(sc.Routes))
	}
	minX, minY, maxX, maxY := sc.Bathy.BoundsYards()
	for _, r := range sc.Routes {
		if r == nil || !r.PingPong || r.UniqueCount() < 20 {
			t.Fatalf("route %+v want dense PingPong (≥20 WP)", r)
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
	dRoutes := MinDistToRoutesYd(sc.Player.X, sc.Player.Y, sc.Routes)
	if dRoutes < 500 || dRoutes > 3000 {
		t.Fatalf("player route clearance=%.0f want ≤3000 (and off the lane)", dRoutes)
	}
	// SW-ish: closer to SW corner than to NE.
	sw := hypot2(sc.Player.X-(minX+1800), sc.Player.Y-(minY+1800))
	ne := hypot2(sc.Player.X-(maxX-1800), sc.Player.Y-(maxY-1800))
	if sw > ne {
		t.Fatalf("player not near SW corner: sw=%.0f ne=%.0f", sw, ne)
	}
	for _, e := range sc.Entities {
		if e == nil || e.RouteID == "" {
			t.Fatalf("%v missing route", e)
		}
	}
}

func hypot2(dx, dy float64) float64 {
	return dx*dx + dy*dy
}
