package ai

import (
	"math"
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func testCoastBathy() world.Bathymetry {
	const w, h = 24, 12
	depths := make([]float32, w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if i < 8 {
				depths[j*w+i] = -10
			} else {
				depths[j*w+i] = 120
			}
		}
	}
	return world.Bathymetry{
		Width: w, Height: h,
		OriginX: 0, OriginY: 0,
		CellSize: 100,
		Depths:   depths,
	}
}

func TestShoreAvoidance_TurnsFromCoast(t *testing.T) {
	b := testCoastBathy()
	ship := &world.Entity{
		ID: "mv_test", Kind: world.KindSurfaceShip, Side: world.SideNeutral,
		Status: world.StatusActive, SignatureID: "merchant",
		X: 950, Y: 600, HeadingDeg: 90, OrderedHead: 270, // west toward land
		SpeedKts: 10, OrderedSpeed: 10, AIState: "TRANSIT",
	}
	applyShoreAvoidance(ship, &b)
	// Should steer east-ish (open water), not west.
	relWest := math.Abs(normalizeRel(ship.OrderedHead - 270))
	if relWest < 45 {
		t.Fatalf("ordered head %.0f still points at land (was 270)", ship.OrderedHead)
	}
	if courseThreatensShore(&b, ship.X, ship.Y, ship.OrderedHead) {
		t.Fatalf("still heading toward shore: %.0f°", ship.OrderedHead)
	}
}

func TestShoreAvoidance_OpenWaterUntouched(t *testing.T) {
	b := testCoastBathy()
	ship := &world.Entity{
		ID: "mv_open", Kind: world.KindSurfaceShip, Side: world.SideNeutral,
		Status: world.StatusActive,
		X: 1200, Y: 600, HeadingDeg: 0, OrderedHead: 0, // north, parallel to coast
		SpeedKts: 10, OrderedSpeed: 10, AIState: "TRANSIT",
	}
	before := ship.OrderedHead
	applyShoreAvoidance(ship, &b)
	if ship.OrderedHead != before {
		t.Fatalf("open water course changed: %.0f -> %.0f", before, ship.OrderedHead)
	}
}

func TestShoreAvoidance_TooCloseNow(t *testing.T) {
	b := testCoastBathy()
	ship := &world.Entity{
		ID: "mv_close", Kind: world.KindSurfaceShip, Side: world.SideNeutral,
		Status: world.StatusActive,
		X: 820, Y: 600, HeadingDeg: 270, OrderedHead: 270,
		SpeedKts: 8, OrderedSpeed: 8, AIState: "TRANSIT",
	}
	applyShoreAvoidance(ship, &b)
	rel := math.Abs(normalizeRel(ship.OrderedHead - 270))
	if rel < 30 {
		t.Fatalf("expected turn away from west coast, head=%.0f", ship.OrderedHead)
	}
	if courseThreatensShore(&b, ship.X, ship.Y, ship.OrderedHead) {
		t.Fatalf("still ordered toward shore at %.0f°", ship.OrderedHead)
	}
}
