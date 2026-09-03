package sim

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestExerciseTargetFiresSignalTorpedoOnly(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 0, DepthFt: 60,
	}
	hulk := &world.Entity{
		ID: "ex_hulk_a", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "exercise_hulk", ExerciseTarget: true,
		X: 2000, Y: 0, Defcon: world.DefconWeaponsFree, AIState: "SHIP_TUBE",
	}
	hulk.Track.Valid = true
	hulk.Track.X, hulk.Track.Y = player.X, player.Y
	hulk.Track.ClassConf = 0.9
	hulk.Track.HoldSec = 5

	sc := &world.Scenario{
		Player:   player,
		Entities: []*world.Entity{player, hulk},
	}
	eng := NewEngine(sc)
	eng.FireControl.EnemyExerciseTube[hulk.ID] = 2
	eng.FireControl.EnemyShipTube[hulk.ID] = 6 // must not be used

	eng.tryEnemySurfaceWeapons(player, 5.5)
	if len(eng.FireControl.ActiveTorpedoes) != 1 {
		t.Fatalf("expected one exercise fish, got %d", len(eng.FireControl.ActiveTorpedoes))
	}
	torp := eng.FireControl.ActiveTorpedoes[0]
	if torp.OrdnanceType != weapons.OrdnanceMk48Exercise || torp.TerminalMode != weapons.TerminalSignal {
		t.Fatalf("expected exercise signal fish, got ord=%s term=%d", torp.OrdnanceType, torp.TerminalMode)
	}
	if torp.EscalatesDefcon() {
		t.Fatal("exercise fish must not escalate DEFCON")
	}
	if eng.FireControl.ExerciseTubeAmmo(hulk) != 1 {
		t.Fatalf("exercise mag=%d want 1", eng.FireControl.ExerciseTubeAmmo(hulk))
	}
	if eng.FireControl.LaunchShipTube(hulk, player) != nil {
		t.Fatal("combat LW must be blocked for exercise hulk")
	}
}
