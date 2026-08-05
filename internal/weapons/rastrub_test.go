package weapons

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

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
