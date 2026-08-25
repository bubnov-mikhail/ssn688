package world

import (
	"testing"
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
