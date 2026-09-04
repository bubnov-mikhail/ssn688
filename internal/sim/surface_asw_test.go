package sim

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestEnemyGrishaRBUAtOverlapRange(t *testing.T) {
	sc := campaign.DemoRuntime()
	e := NewEngine(sc)
	player := sc.Player
	if player == nil {
		t.Fatal("no player")
	}
	var grisha *world.Entity
	for _, ent := range sc.Entities {
		if ent != nil && ent.ID == "enemy_grisha" {
			grisha = ent
			break
		}
	}
	if grisha == nil {
		t.Fatal("no grisha")
	}
	grisha.Defcon = world.DefconWeaponsFree
	grisha.AIState = "RBU"
	grisha.X, grisha.Y = 0, 0
	player.X, player.Y = 0, 1400
	player.DepthFt = 60
	player.OrderedDepth = 60
	grisha.Track = world.AITrack{
		Valid: true, ClassConf: 0.25, HoldSec: 10,
		X: player.X, Y: player.Y, DepthFt: player.DepthFt,
	}

	gameTime := 44.0 // int(gameTime*10)%44 == 0
	before := len(e.FireControl.ActiveRBU)
	e.tryEnemySurfaceWeapons(player, gameTime)
	if len(e.FireControl.ActiveRBU) <= before {
		t.Fatal("expected RBU salvo at 1400 yd / periscope depth, not blocked by ship tubes")
	}
	if len(e.FireControl.ActiveTorpedoes) > 0 {
		t.Fatal("shallow target at overlap range should get RBU, not tubes")
	}
}

func TestEnemyGrishaTubesVsSurfaceQuarry(t *testing.T) {
	sc := campaign.DemoRuntime()
	e := NewEngine(sc)
	player := sc.Player
	var grisha, ally *world.Entity
	for _, ent := range sc.Entities {
		if ent == nil {
			continue
		}
		switch ent.ID {
		case "enemy_grisha":
			grisha = ent
		case "ally_spruance":
			ally = ent
		}
	}
	if grisha == nil || ally == nil {
		t.Fatal("need grisha + ally")
	}
	grisha.Defcon = world.DefconWeaponsFree
	grisha.AIState = "SHIP_TUBE"
	grisha.CrewSkill = 90
	grisha.X, grisha.Y = 0, 0
	ally.X, ally.Y = 0, 1400
	ally.DepthFt = 0
	player.X, player.Y = 0, 50000 // far — quarry should be ally
	player.DepthFt = 400
	grisha.Track = world.AITrack{
		Valid: true, ClassConf: 0.9, HoldSec: 30,
		X: ally.X, Y: ally.Y, DepthFt: 0,
	}

	gameTime := 2.2 // int(gameTime*10)%22 == 0 (veteran tube tick)
	beforeT := len(e.FireControl.ActiveTorpedoes)
	beforeR := len(e.FireControl.ActiveRBU)
	e.tryEnemySurfaceWeapons(player, gameTime)
	if len(e.FireControl.ActiveRBU) > beforeR {
		t.Fatal("surface quarry must not get RBU")
	}
	if len(e.FireControl.ActiveTorpedoes) <= beforeT {
		t.Fatal("expected ship-tube fish vs surface quarry")
	}
}

func TestEnemyGrishaRBUDryFallsThroughToTubes(t *testing.T) {
	sc := campaign.DemoRuntime()
	e := NewEngine(sc)
	player := sc.Player
	var grisha *world.Entity
	for _, ent := range sc.Entities {
		if ent != nil && ent.ID == "enemy_grisha" {
			grisha = ent
			break
		}
	}
	if grisha == nil {
		t.Fatal("no grisha")
	}
	grisha.Defcon = world.DefconWeaponsFree
	grisha.AIState = "RBU"
	grisha.CrewSkill = 90
	grisha.X, grisha.Y = 0, 0
	player.X, player.Y = 0, 1400
	player.DepthFt = 60
	grisha.Track = world.AITrack{
		Valid: true, ClassConf: 0.9, HoldSec: 30,
		X: player.X, Y: player.Y, DepthFt: 60,
	}
	e.FireControl.EnemyRBU = map[string]int{grisha.ID: 0}

	gameTime := 0.0 // %44==0 and %22==0 for veteran tube tick
	before := len(e.FireControl.ActiveTorpedoes)
	e.tryEnemySurfaceWeapons(player, gameTime)
	if len(e.FireControl.ActiveTorpedoes) <= before {
		t.Fatal("empty RBU magazine should fall through to ship tubes")
	}
	_ = weapons.RBUMagazineDefault
}
