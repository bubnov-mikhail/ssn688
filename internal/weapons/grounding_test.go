package weapons

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

// landStripBathy: water on the west edge, land toward +X.
func landStripBathy() *world.Bathymetry {
	return &world.Bathymetry{
		Width: 2, Height: 2,
		OriginX: 0, OriginY: 0,
		CellSize: 100,
		Depths: []float32{
			400, -20,
			400, -20,
		},
	}
}

func TestTorpedoDetonatesOnShore(t *testing.T) {
	bathy := landStripBathy()
	fish := &Torpedo{
		ID: "MK48-shore", Side: world.SidePlayer,
		X: 10, Y: 50, DepthFt: 80, HeadingDeg: 90, // due east into land
		SpeedKts: 55, CruiseKts: 55, RunDepthFt: 80,
		Armed: true, Alive: true, Mode: ModeWire, Age: 5,
		LastPingTime: -1,
	}
	var det *Detonation
	for i := 0; i < 200 && fish.Alive; i++ {
		det = fish.Advance(0.1, float64(i)*0.1, nil, nil, bathy)
	}
	if fish.Alive {
		t.Fatalf("fish swam through land to (%.0f,%.0f) bottom=%.0f", fish.X, fish.Y, bathy.DepthAtFt(fish.X, fish.Y))
	}
	if det == nil || !det.Grounded {
		t.Fatalf("expected grounded detonation, got %+v", det)
	}
	if bathy.DepthAtFt(det.X, det.Y) > 0 && fish.DepthFt < bathy.DepthAtFt(det.X, det.Y) {
		t.Fatalf("impact not on land/bottom: pos=(%.0f,%.0f) bottom=%.0f depth=%.0f", det.X, det.Y, bathy.DepthAtFt(det.X, det.Y), fish.DepthFt)
	}
}

func TestTorpedoDetonatesOnSeafloor(t *testing.T) {
	bathy := &world.Bathymetry{
		Width: 2, Height: 2,
		OriginX: 0, OriginY: 0,
		CellSize: 200,
		Depths: []float32{
			60, 60,
			60, 60,
		},
	}
	fish := &Torpedo{
		ID: "MK48-bottom", Side: world.SidePlayer,
		X: 50, Y: 50, DepthFt: 200, HeadingDeg: 0,
		SpeedKts: 28, CruiseKts: 28, RunDepthFt: 200,
		Armed: true, Alive: true, Mode: ModeWire, Age: 5,
		LastPingTime: -1,
	}
	det := fish.Advance(0.1, 1, nil, nil, bathy)
	if fish.Alive || det == nil || !det.Grounded {
		t.Fatalf("expected seafloor grounding: alive=%v det=%+v bottom=%.0f", fish.Alive, det, bathy.DepthAtFt(fish.X, fish.Y))
	}
}

func TestTorpedoNilBathyNoGround(t *testing.T) {
	fish := &Torpedo{
		ID: "MK48-open", Side: world.SidePlayer,
		X: 0, Y: 0, DepthFt: 200, HeadingDeg: 0,
		SpeedKts: 55, CruiseKts: 55, RunDepthFt: 200,
		Armed: true, Alive: true, Mode: ModeWire, Age: 5,
		LastPingTime: -1,
	}
	det := fish.Advance(0.1, 1, nil, nil, nil)
	if det != nil || !fish.Alive {
		t.Fatalf("nil bathy should not ground: alive=%v det=%+v", fish.Alive, det)
	}
}

func TestExerciseTorpedoSignalsInsteadOfDetonating(t *testing.T) {
	target := &world.Entity{
		ID: "enemy", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 40, Y: 0, DepthFt: 200,
	}
	fish := &Torpedo{
		ID: "MK48X-1", Side: world.SidePlayer, TargetID: target.ID,
		X: 0, Y: 0, DepthFt: 200, HeadingDeg: 90, OrderedHead: 90,
		SpeedKts: 55, CruiseKts: 55, RunDepthFt: 200,
		Armed: true, Alive: true, Mode: ModeSearch, Age: 5,
		LastPingTime: -1, OrdnanceType: OrdnanceMk48Exercise,
		TerminalMode: TerminalSignal, ClearDistYd: 500,
	}
	det := fish.Advance(0.1, 1, []*world.Entity{target}, nil, nil)
	if fish.Alive || det == nil {
		t.Fatalf("exercise fish should terminate with signal, alive=%v det=%+v", fish.Alive, det)
	}
	if !det.SignalOnly || det.Hit != nil {
		t.Fatalf("expected signal-only terminal effect, got %+v", det)
	}
}

func TestExerciseTorpedoSignalsOnOwnship(t *testing.T) {
	own := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 40, Y: 0, DepthFt: 200,
	}
	fish := &Torpedo{
		ID: "MK48X-2", ParentSubID: "player", Side: world.SidePlayer,
		X: 0, Y: 0, DepthFt: 200, HeadingDeg: 90, OrderedHead: 90,
		SpeedKts: 55, CruiseKts: 55, RunDepthFt: 200,
		Armed: true, Alive: true, Mode: ModeWire, Age: 5,
		LastPingTime: -1, OrdnanceType: OrdnanceMk48Exercise,
		TerminalMode: TerminalSignal, ClearDistYd: TubeClearYd + 10,
	}
	det := fish.Advance(0.1, 1, []*world.Entity{own}, nil, nil)
	if fish.Alive || det == nil {
		t.Fatalf("exercise fish should terminate on ownship, alive=%v det=%+v", fish.Alive, det)
	}
	if !det.SignalOnly {
		t.Fatalf("expected signal-only, got %+v", det)
	}
}

func TestWarshotDetonatesOnOwnship(t *testing.T) {
	own := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 40, Y: 0, DepthFt: 200,
	}
	fish := &Torpedo{
		ID: "MK48-9", ParentSubID: "player", Side: world.SidePlayer,
		X: 0, Y: 0, DepthFt: 200, HeadingDeg: 90, OrderedHead: 90,
		SpeedKts: 55, CruiseKts: 55, RunDepthFt: 200,
		Armed: true, Alive: true, Mode: ModeSearch, Age: 5,
		LastPingTime: -1, OrdnanceType: OrdnanceMk48,
		TerminalMode: TerminalExplode, ClearDistYd: 500,
	}
	det := fish.Advance(0.1, 1, []*world.Entity{own}, nil, nil)
	if fish.Alive || det == nil || det.SignalOnly || det.Hit != own {
		t.Fatalf("warshot should detonate on ownship, alive=%v det=%+v", fish.Alive, det)
	}
}

func TestEnemyTorpedoFusesOnAnyPlatform(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 40, Y: 0, DepthFt: 200,
	}
	ally := &world.Entity{
		ID: "ally_688", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 40, Y: 0, DepthFt: 200,
	}
	enemy := &world.Entity{
		ID: "enemy_other", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 40, Y: 0, DepthFt: 200,
	}
	civ := &world.Entity{
		ID: "civ_tanker", Kind: world.KindSurfaceShip, Side: world.SideNeutral,
		Status: world.StatusActive, X: 40, Y: 0, DepthFt: 0,
	}
	mkEnemyFish := func() *Torpedo {
		return &Torpedo{
			ID: "ETORP-1", ParentSubID: "enemy_foxtrot", TargetID: "player", Side: world.SideEnemy,
			X: 0, Y: 0, DepthFt: 200, HeadingDeg: 90, OrderedHead: 90,
			SpeedKts: 55, CruiseKts: 55, RunDepthFt: 200,
			Armed: true, Alive: true, Mode: ModeSearch, Age: 5,
			LastPingTime: -1, OrdnanceType: OrdnanceMk48,
			TerminalMode: TerminalExplode, ClearDistYd: 500,
		}
	}

	for _, tgt := range []*world.Entity{player, ally, enemy, civ} {
		fish := mkEnemyFish()
		// Match depth for sub proximity; surface fish uses under-keel rule.
		if tgt.Kind == world.KindSurfaceShip {
			fish.DepthFt = 50
			fish.RunDepthFt = 50
		}
		det := fish.Advance(0.1, 1, []*world.Entity{tgt}, nil, nil)
		if fish.Alive || det == nil || det.Hit != tgt {
			t.Fatalf("enemy fish should hit %s, alive=%v det=%+v", tgt.ID, fish.Alive, det)
		}
	}
}

func TestEnemySeekerPrefersFriendlyOverNeutral(t *testing.T) {
	fish := &Torpedo{
		ID: "ETORP-2", ParentSubID: "enemy_foxtrot", TargetID: "player", Side: world.SideEnemy,
		X: 0, Y: 0, DepthFt: 200, HeadingDeg: 90, OrderedHead: 90,
		SpeedKts: 40, CruiseKts: 40, RunDepthFt: 200,
		Armed: true, Alive: true, Mode: ModeSearch, Age: 5,
		SeekerOn: true, LastPingTime: -1, ClearDistYd: 500,
		OrdnanceType: OrdnanceMk48, TerminalMode: TerminalExplode,
	}
	// Both well inside cone/range; friendly should win despite being slightly farther.
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 900, Y: 0, DepthFt: 200,
	}
	civ := &world.Entity{
		ID: "civ_tanker", Kind: world.KindSurfaceShip, Side: world.SideNeutral,
		Status: world.StatusActive, X: 700, Y: 0, DepthFt: 0,
	}
	best := fish.acquireInCone([]*world.Entity{civ, player}, nil, 10)
	if best == nil || best.ID != "player" {
		t.Fatalf("expected prefer player over nearer neutral, got %v", best)
	}
}
