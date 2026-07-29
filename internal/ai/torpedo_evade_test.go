package ai

import (
	"testing"

	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

func TestMostThreateningTorpedoAimed(t *testing.T) {
	ship := &world.Entity{
		ID: "enemy_dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 2000, HeadingDeg: 0, SpeedKts: 14,
	}
	// Fish north of ship, heading south toward it.
	fish := &weapons.Torpedo{
		ID: "MK48-1", Side: world.SidePlayer, Alive: true,
		X: 0, Y: 3500, HeadingDeg: 180, SpeedKts: 55, Mode: weapons.ModeSearch,
		LastPingTime: 1,
	}
	miss := &weapons.Torpedo{
		ID: "MK48-2", Side: world.SidePlayer, Alive: true,
		X: 4000, Y: 2000, HeadingDeg: 90, SpeedKts: 55, // going away east
	}
	got := mostThreateningTorpedo(ship, []*weapons.Torpedo{miss, fish})
	if got == nil || got.ID != "MK48-1" {
		t.Fatalf("expected aimed fish, got %#v", got)
	}
}

func TestTryEvadeTorpedoOrdersFlankComb(t *testing.T) {
	ship := &world.Entity{
		ID: "enemy_dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, HeadingDeg: 0, OrderedSpeed: 12,
	}
	fish := &weapons.Torpedo{
		ID: "MK48-1", Side: world.SidePlayer, Alive: true,
		X: 0, Y: 1500, HeadingDeg: 180, SpeedKts: 50, Mode: weapons.ModeWire,
	}
	if !tryEvadeTorpedo(ship, []*weapons.Torpedo{fish}) {
		t.Fatal("expected evade")
	}
	if ship.AIState != "TORPEDO_EVADE" {
		t.Fatalf("state=%s", ship.AIState)
	}
	if ship.OrderedSpeed < 25 {
		t.Fatalf("expected flank speed, got %.0f", ship.OrderedSpeed)
	}
	// Comb should be roughly E or W (90 or 270), not continue north into fish.
	h := ship.OrderedHead
	if h > 20 && h < 160 {
		// ok-ish stbd comb ~90
	} else if h > 200 && h < 340 {
		// port comb ~270
	} else {
		t.Fatalf("unexpected comb heading %.0f", h)
	}
}

func TestSubEvadeChangesDepth(t *testing.T) {
	sub := &world.Entity{
		ID: "enemy_ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, DepthFt: 200, HeadingDeg: 90,
	}
	fish := &weapons.Torpedo{
		ID: "MK48-1", Side: world.SidePlayer, Alive: true,
		X: 800, Y: 0, HeadingDeg: 270, DepthFt: 180, SpeedKts: 48,
		Mode: weapons.ModeSearch, TargetID: sub.ID, LastPingTime: 10,
	}
	if !tryEvadeTorpedo(sub, []*weapons.Torpedo{fish}) {
		t.Fatal("expected evade")
	}
	if sub.OrderedDepth <= sub.DepthFt {
		t.Fatalf("expected deeper ordered depth away from shallow fish, got %.0f", sub.OrderedDepth)
	}
	if sub.OrderedSpeed < 18 {
		t.Fatalf("expected high speed, got %.0f", sub.OrderedSpeed)
	}
}
