package weapons

import (
	"math"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestExerciseTargetCannotLaunchRastrub(t *testing.T) {
	fc := NewFireControl()
	ship := &world.Entity{
		ID: "ex_hulk_a", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "exercise_hulk", ExerciseTarget: true,
		X: 0, Y: 0,
	}
	tgt := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 5000, DepthFt: 200,
	}
	if fc.LaunchRastrub(ship, tgt, 10) != nil {
		t.Fatal("exercise hulk must not launch Rastrub")
	}
	if fc.rastrubAmmo(ship) != 0 {
		t.Fatalf("rastrub ammo=%d", fc.rastrubAmmo(ship))
	}
}

func TestLaunchRastrubSpawnsUMGT1AfterFlight(t *testing.T) {
	fc := NewFireControl()
	ship := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, HeadingDeg: 0,
	}
	tgt := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 5000, DepthFt: 200, HeadingDeg: 90, SpeedKts: 6,
	}
	flight := fc.LaunchRastrub(ship, tgt, 10)
	if flight == nil {
		t.Fatal("expected Rastrub launch")
	}
	if fc.EnemyRastrub[ship.ID] != RastrubMagazineDefault-1 {
		t.Fatalf("rastrub mag %d", fc.EnemyRastrub[ship.ID])
	}
	if spawned := fc.AdvanceRastrub(10 + flight.FlightSec*0.5); len(spawned) != 0 {
		t.Fatal("too early splash")
	}
	spawned := fc.AdvanceRastrub(10 + flight.FlightSec + 0.1)
	if len(spawned) != 1 {
		t.Fatalf("expected 1 UMGT-1, got %d", len(spawned))
	}
	fish := spawned[0]
	if fish.Class != ClassUMGT1 || fish.Mode != ModeSearch || !fish.SeekerOn || !fish.WireCut {
		t.Fatalf("umgt1 state class=%d mode=%d seek=%v wire=%v", fish.Class, fish.Mode, fish.SeekerOn, fish.WireCut)
	}
	if fish.ClearDistYd < UMGT1TubeClearYd {
		t.Fatal("Rastrub splash should be tube-clear")
	}
}

func TestLaunchShipTubeRangeAndClass(t *testing.T) {
	fc := NewFireControl()
	ship := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0,
	}
	far := &world.Entity{
		ID: "p", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 5000, DepthFt: 150,
	}
	if fc.LaunchShipTube(ship, far) != nil {
		t.Fatal("ship tube should reject Rastrub-range target")
	}
	near := &world.Entity{
		ID: "p2", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 1500, DepthFt: 150,
	}
	fish := fc.LaunchShipTube(ship, near)
	if fish == nil {
		t.Fatal("expected ship-tube UMGT-1")
	}
	if fish.Class != ClassUMGT1 || fish.ClearDistYd != 0 {
		t.Fatalf("class=%d clear=%.0f", fish.Class, fish.ClearDistYd)
	}
	if fc.EnemyShipTube[ship.ID] != ShipTubeMagazineDefault-1 {
		t.Fatalf("ship tube mag %d", fc.EnemyShipTube[ship.ID])
	}
}

func TestUMGT1ShortSeekAndAge(t *testing.T) {
	r, _ := seekAcquireLimitsFor(ClassUMGT1, 100, 100, nil)
	if r > UMGT1SeekRangeYd+1 {
		t.Fatalf("umgt1 seek %.0f", r)
	}
	rh, _ := seekAcquireLimitsFor(ClassHeavy, 100, 100, nil)
	if rh <= r {
		t.Fatalf("heavy seek should exceed umgt1: %.0f vs %.0f", rh, r)
	}
}

// Closing hull stays inside UMGT proximity of its own fresh fish — fuse must ignore ParentSubID
// and own-side surface hulls for lightweight ASW fish.
func TestShipTubeFishNeverKillsLauncher(t *testing.T) {
	fc := NewFireControl()
	ship := &world.Entity{
		ID: "plan_grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "grisha",
		X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 18, DepthFt: 0, // at/under exit speed gate
	}
	tgt := &world.Entity{
		ID: "ally", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 1200, DepthFt: 0,
	}
	fish := fc.LaunchShipTube(ship, tgt)
	if fish == nil {
		t.Fatal("expected ship-tube fish")
	}
	// Sprint after launch (CLOSING doctrine) — still must not fuse on own hull.
	ship.SpeedKts = 28
	targets := []*world.Entity{ship, tgt}
	const dt = 0.1
	for step := 0; step < 80; step++ {
		ship.Y += ship.SpeedKts * world.KnotsToYPS * dt
		det := fish.Advance(dt, float64(step)*dt, targets, nil, nil)
		if det != nil && det.Hit != nil && det.Hit.ID == ship.ID {
			t.Fatalf("own fish fused on launcher at age=%.1f dist=%.0f",
				fish.Age, math.Hypot(ship.X-fish.X, ship.Y-fish.Y))
		}
		if det != nil || !fish.Alive {
			return
		}
	}
}

func TestShipTubeRejectedWhileSprinting(t *testing.T) {
	fc := NewFireControl()
	ship := &world.Entity{
		ID: "plan_grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "grisha",
		X: 0, Y: 0, SpeedKts: 28,
	}
	tgt := &world.Entity{
		ID: "ally", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 1200,
	}
	if fc.LaunchShipTube(ship, tgt) != nil {
		t.Fatal("ship tube must not launch while faster than UMGT exit speed")
	}
}

func TestUMGT1FuseSkipsOwnSideSurface(t *testing.T) {
	fish := &Torpedo{
		ID: "SET40-1", ParentSubID: "other", Side: world.SideEnemy, Class: ClassUMGT1,
		Alive: true, Armed: true, Mode: ModeSearch, Age: 5, ClearDistYd: 100,
		X: 0, Y: 0, DepthFt: 40, SpeedKts: 40,
	}
	friend := &world.Entity{
		ID: "rf_udaloy", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 30, Y: 0, DepthFt: 0,
	}
	if fish.validFuseTarget(friend) {
		t.Fatal("lightweight ASW fish must not fuse on own-side surface")
	}
}
