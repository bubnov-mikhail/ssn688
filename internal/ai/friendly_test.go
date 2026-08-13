package ai

import (
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

func TestAllyAIDoesNotTargetOwnship(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "los_angeles",
		X: 0, Y: 0, DepthFt: 200, SpeedKts: 5,
		Damage: world.NewFullHealth(),
	}
	ally := &world.Entity{
		ID: "ally_dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy, // wrong on purpose then fix
		Status: world.StatusActive, SignatureID: "spruance",
		X: 0, Y: 2000, SpeedKts: 12, Defcon: world.DefconWeaponsFree,
		CrewSkill: 80, Damage: world.NewFullHealth(), LastPingTime: -100,
	}
	ally.Side = world.SidePlayer
	ally.Track = world.AITrack{
		Valid: true, ClassConf: 0.9, HoldSec: 40,
		X: player.X, Y: player.Y, DepthFt: player.DepthFt,
	}
	// No hostiles — ally must patrol / not weaponize ownship.
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	UpdateFriendlyAI([]*world.Entity{ally}, player, 50, 0.1, model, nil, EvadeContext{}, nil, nil)
	if ally.AIState == "RASTRUB" || ally.AIState == "SHIP_TUBE" || ally.AIState == "RBU" {
		t.Fatalf("ally weaponized without hostiles: %s", ally.AIState)
	}
	if world.IsOwnship(ally, player) || !world.IsAllyAI(ally, player) {
		t.Fatal("ally identity helpers broken")
	}
}

func TestAllySelectsHostileNotFriendly(t *testing.T) {
	ally := &world.Entity{
		ID: "ally_ss", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "los_angeles",
		X: 500, Y: 500, DepthFt: 200, CrewSkill: 90,
		Damage: world.NewFullHealth(), Defcon: world.DefconHostile,
	}
	foe := &world.Entity{
		ID: "enemy_ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "foxtrot",
		X: 0, Y: 2500, DepthFt: 180, SpeedKts: 6,
		Damage: world.NewFullHealth(),
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	q := selectHostileQuarry(ally, []*world.Entity{ally, foe}, model, 10)
	if q != foe {
		t.Fatalf("expected hostile quarry, got %#v", q)
	}
	_ = weapons.ShipTubeMinRangeYd
}

func TestSpruanceWeaponsFreeClosesOnSub(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 0, DepthFt: 200,
	}
	spruance := &world.Entity{
		ID: "ally_spruance", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "spruance",
		X: 0, Y: 0, SpeedKts: 12, Defcon: world.DefconWeaponsFree,
		CrewSkill: 80, Damage: world.NewFullHealth(), LastPingTime: -100,
	}
	foxtrot := &world.Entity{
		ID: "enemy_foxtrot", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "foxtrot",
		X: 0, Y: 6000, DepthFt: 180, SpeedKts: 5,
		Damage: world.NewFullHealth(),
	}
	grisha := &world.Entity{
		ID: "enemy_grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "grisha",
		X: 4000, Y: 0, SpeedKts: 14,
		Damage: world.NewFullHealth(),
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	ents := []*world.Entity{spruance, foxtrot, grisha}
	UpdateFriendlyAI(ents, player, 20, 0.1, model, nil, EvadeContext{Ownship: player}, nil, nil)
	if spruance.AIState != "CLOSING" && spruance.AIState != "PINGING" &&
		spruance.AIState != "RASTRUB" && spruance.AIState != "TRACKING" &&
		spruance.AIState != "DATUM" {
		t.Fatalf("weapons-free Spruance should hunt, got %s", spruance.AIState)
	}
	if spruance.AIState == "PATROL" {
		t.Fatal("Spruance must not idle on PATROL without a route")
	}
	// Prefer Foxtrot over equidistant-ish Grisha via ASW bias.
	q := selectHostileQuarry(spruance, ents, model, 20)
	if q != foxtrot {
		t.Fatalf("ASROC ship should prefer sub quarry, got %v", q)
	}
	// Heading toward foxtrot (~000°) while closing / hunting.
	if spruance.OrderedHead > 35 && spruance.OrderedHead < 325 {
		t.Fatalf("ordered head %.0f should aim near foxtrot (north)", spruance.OrderedHead)
	}
}

func TestSpruanceProsecutesNearbySubWithActive(t *testing.T) {
	spruance := &world.Entity{
		ID: "ally_spruance", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "spruance",
		X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 14,
		Defcon: world.DefconWeaponsFree, CrewSkill: 90,
		Damage: world.NewFullHealth(), LastPingTime: -100,
	}
	foxtrot := &world.Entity{
		ID: "enemy_foxtrot", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "foxtrot",
		X: 0, Y: 3500, DepthFt: 160, SpeedKts: 6,
		Damage: world.NewFullHealth(),
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	gotWeapon := false
	for step := 0; step < 400; step++ {
		gt := float64(step) * 0.1
		updateSurfaceAI(spruance, foxtrot, gt, 0.1, model, nil, EvadeContext{}, nil)
		switch spruance.AIState {
		case "RASTRUB", "SHIP_TUBE":
			gotWeapon = true
		}
		if gotWeapon && TrackClassified(spruance) {
			return
		}
	}
	t.Fatalf("expected classified ASW weapon state vs nearby sub, state=%s conf=%.2f prosecuting=%v",
		spruance.AIState, spruance.Track.ClassConf, spruance.AIProsecuting)
}

func TestFriendlyDefconRisesOnNearbyHostile(t *testing.T) {
	player := &world.Entity{ID: "player", Side: world.SidePlayer, Status: world.StatusActive}
	ally := &world.Entity{
		ID: "ally_dd", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "spruance",
		X: 0, Y: 0, Defcon: world.DefconPassive,
		Damage: world.NewFullHealth(),
	}
	foe := &world.Entity{
		ID: "enemy", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 4000, DepthFt: 200, SpeedKts: 8,
		Damage: world.NewFullHealth(),
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	UpdateFriendlyDefcon([]*world.Entity{ally, foe}, player, model, nil, 20)
	if ally.Defcon < world.DefconHostile {
		t.Fatalf("expected DEFCON hostile from proximity, got %d", ally.Defcon)
	}
}
