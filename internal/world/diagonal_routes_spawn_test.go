package world_test

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestPlaceOnRouteFraction_SubMarchesAlongRoute(t *testing.T) {
	const w, h = 40, 30
	depths := make([]float32, w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			depths[j*w+i] = 80
		}
	}
	for j := 0; j < 8; j++ {
		for i := 10; i < 30; i++ {
			depths[j*w+i] = 450
		}
	}
	bathy := world.Bathymetry{
		Width: w, Height: h,
		OriginX: -10000, OriginY: -10000,
		CellSize: 500,
		Depths:   depths,
	}
	route := &world.Route{
		ID: "test",
		Waypoints: []world.Waypoint{
			{X: 0, Y: -2000}, // shallow north on chart
			{X: 0, Y: -9000}, // deep south
		},
	}
	sub := &world.Entity{
		Kind: world.KindSubmarine, DepthFt: 280,
	}
	if !world.PlaceOnRouteFraction(sub, route, 0, &bathy) {
		t.Fatal("expected placement along route")
	}
	if sub.Y > -5000 {
		t.Fatalf("sub should march south to deep water, got (%.0f,%.0f)", sub.X, sub.Y)
	}
}
