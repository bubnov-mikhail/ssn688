package ai

import (
	"math"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func testCoastBathy() world.Bathymetry {
	// Wide chart so shoreLookAheadYd does not always hit the playable edge.
	const w, h = 80, 60
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

func TestSubTerrainAvoidance(t *testing.T) {
	b := testCoastBathy()
	for j := range b.Depths {
		if b.Depths[j] > 0 {
			b.Depths[j] = 260
		}
	}
	sub := &world.Entity{
		Kind: world.KindSubmarine, DepthFt: 160, OrderedDepth: 160,
		X: 1500, Y: 600, HeadingDeg: 270, OrderedHead: 270,
		AIState: "PATROL",
	}
	applyShoreAvoidance(sub, &b)
	relWest := math.Abs(normalizeRel(sub.OrderedHead - 270))
	if relWest < 20 {
		t.Fatalf("sub still ordered toward land at %.0f°", sub.OrderedHead)
	}
}

func TestShoreAvoidance_TurnsFromCoast(t *testing.T) {
	b := testCoastBathy()
	ship := &world.Entity{
		ID: "mv_test", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
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
		ID: "mv_open", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
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
		ID: "mv_close", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
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

func openOceanBathy() world.Bathymetry {
	const w, h = 40, 40
	depths := make([]float32, w*h)
	for i := range depths {
		depths[i] = 2000
	}
	return world.Bathymetry{
		Width: w, Height: h,
		OriginX: 0, OriginY: 0,
		CellSize: 100,
		Depths:   depths,
	}
}

func TestShoreAvoidance_ChartEdge(t *testing.T) {
	b := openOceanBathy()
	// Combatant near east edge, ordered off-chart.
	ship := &world.Entity{
		ID: "ddg_edge", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive,
		X: 3600, Y: 2000, HeadingDeg: 90, OrderedHead: 90,
		SpeedKts: 12, OrderedSpeed: 12, AIState: "PATROL",
	}
	applyShoreAvoidance(ship, &b)
	relEast := math.Abs(normalizeRel(ship.OrderedHead - 90))
	if relEast < 45 {
		t.Fatalf("ship still ordered off-chart at %.0f°", ship.OrderedHead)
	}
	if courseThreatensShore(&b, ship.X, ship.Y, ship.OrderedHead) {
		t.Fatalf("still heading toward chart edge: %.0f°", ship.OrderedHead)
	}
}

func TestSubTerrainAvoidance_ChartEdge(t *testing.T) {
	b := openOceanBathy()
	sub := &world.Entity{
		Kind: world.KindSubmarine, Side: world.SideEnemy,
		DepthFt: 200, OrderedDepth: 200,
		X: 3600, Y: 2000, HeadingDeg: 90, OrderedHead: 90,
		AIState: "PATROL",
	}
	applyShoreAvoidance(sub, &b)
	relEast := math.Abs(normalizeRel(sub.OrderedHead - 90))
	if relEast < 20 {
		t.Fatalf("sub still ordered off-chart at %.0f°", sub.OrderedHead)
	}
}
