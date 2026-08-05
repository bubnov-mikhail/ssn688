package ai

import (
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

func TestSurfaceAIHoldsRastrubStandoff(t *testing.T) {
	ship := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 16,
		Damage: world.NewFullHealth(), LastPingTime: -100, Defcon: world.DefconHostile,
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 5500, DepthFt: 200, SpeedKts: 6,
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	updateSurfaceAI(ship, player, 100, model, nil, EvadeContext{})
	if ship.AIState != "RASTRUB" {
		t.Fatalf("expected RASTRUB, got %s", ship.AIState)
	}
	if ship.OrderedSpeed > 19 {
		t.Fatalf("standoff speed too high: %.1f", ship.OrderedSpeed)
	}
}

func TestSurfaceAIShipTubeBand(t *testing.T) {
	ship := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0,
		Damage: world.NewFullHealth(), LastPingTime: -100, Defcon: world.DefconHostile,
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 1600, DepthFt: 180,
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	updateSurfaceAI(ship, player, 50, model, nil, EvadeContext{})
	if ship.AIState != "SHIP_TUBE" {
		t.Fatalf("expected SHIP_TUBE, got %s", ship.AIState)
	}
	_ = weapons.ShipTubeMinRangeYd
}
