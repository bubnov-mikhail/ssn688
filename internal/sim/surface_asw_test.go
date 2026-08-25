package sim

import (
	"testing"

	"github.com/ssn688/sim/internal/campaign"
	"github.com/ssn688/sim/internal/world"
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
