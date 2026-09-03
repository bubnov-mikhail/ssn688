package ai

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestSelectEnemyQuarryPrefersDetectedAlly(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 50000, DepthFt: 300, SpeedKts: 0,
	}
	ally := &world.Entity{
		ID: "ally_spruance", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "spruance",
		X: 0, Y: 6000, SpeedKts: 14, Damage: world.NewFullHealth(),
	}
	foe := &world.Entity{
		ID: "grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "grisha",
		X: 0, Y: 0, SpeedKts: 14, Defcon: world.DefconWeaponsFree,
		Damage: world.NewFullHealth(), CrewSkill: 80,
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	ents := []*world.Entity{foe, ally}
	q := SelectEnemyQuarry(foe, ents, player, model, 10)
	if q == nil || q.ID != "ally_spruance" {
		t.Fatalf("expected ally quarry, got %v", q)
	}
}

func TestEnemyAIProsecutesAllySurface(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 50000, DepthFt: 300,
	}
	ally := &world.Entity{
		ID: "ally_spruance", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "spruance",
		X: 0, Y: 5500, SpeedKts: 14, Defcon: world.DefconWeaponsFree,
		Damage: world.NewFullHealth(),
	}
	grisha := &world.Entity{
		ID: "enemy_grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "grisha",
		X: 0, Y: 0, SpeedKts: 12, Defcon: world.DefconWeaponsFree,
		Damage: world.NewFullHealth(), CrewSkill: 90,
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	for step := 0; step < 120; step++ {
		gt := 10 + float64(step)*0.1
		UpdateEnemyAI([]*world.Entity{grisha, ally}, player, gt, 0.1, model, nil, EvadeContext{}, nil, nil)
		switch grisha.AIState {
		case "RBU", "RADAR_TRACK", "SHIP_TUBE", "CLOSING", "INTERCEPT", "TRACKING", "RASTRUB":
			return
		}
	}
	t.Fatalf("enemy should prosecute ally surface, got state=%s prosecuting=%v",
		grisha.AIState, grisha.AIProsecuting)
}
