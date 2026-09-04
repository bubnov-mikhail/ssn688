package sim

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func yasenASCMPair(t *testing.T, allyYd float64) (*Engine, *world.Entity, *world.Entity) {
	t.Helper()
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 90000, DepthFt: 300,
		Damage: world.NewFullHealth(),
	}
	ally := &world.Entity{
		ID: "ally_spruance", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "spruance",
		X: 0, Y: allyYd, SpeedKts: 14, DepthFt: 0,
		Damage: world.NewFullHealth(),
	}
	yasen := &world.Entity{
		ID: "rf_yasen", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "yasen_m",
		X: 0, Y: 0, DepthFt: 60, SpeedKts: 6, OrderedDepth: 60, OrderedSpeed: 6,
		Defcon: world.DefconWeaponsFree, AIState: "PATROL", CrewSkill: 95,
		Damage: world.NewFullHealth(),
	}
	eng := NewEngine(&world.Scenario{
		Player:   player,
		Entities: []*world.Entity{yasen, ally},
	})
	eng.FireControl.EnemyASCMMag = map[string]int{yasen.ID: 16}
	return eng, yasen, ally
}

func TestYasenASCMFiresWhenSurfaceSensed(t *testing.T) {
	eng, yasen, _ := yasenASCMPair(t, 7000)
	player := eng.Scenario.Player
	dt := 1.0 / TickRate
	for eng.Clock.GameTime < 180 {
		eng.Update(dt)
		if eng.FireControl.EnemyASCMLeft(yasen.ID) < 16 {
			if yasen.AIState == "PATROL" || yasen.AIState == "" {
				// FIRING may last one tick; allow SHADOW/ATTACK/DATUM too.
				t.Fatalf("launch while still calm PATROL")
			}
			return
		}
	}
	t.Fatalf("no ASCM with surface in acoustic reach (depth=%.0f state=%s)",
		yasen.DepthFt, yasen.AIState)
	_ = player
}

func TestYasenASCMCooldownSpacing(t *testing.T) {
	eng, yasen, _ := yasenASCMPair(t, 7000)
	player := eng.Scenario.Player
	var launchAt []float64
	prev := 16
	dt := 1.0 / TickRate
	for eng.Clock.GameTime < 600 {
		eng.Update(dt)
		left := eng.FireControl.EnemyASCMLeft(yasen.ID)
		if left < prev {
			launchAt = append(launchAt, eng.Clock.GameTime)
			prev = left
		}
	}
	t.Logf("launches at %v (n=%d)", launchAt, len(launchAt))
	if len(launchAt) == 0 {
		t.Fatal("expected ASCM launches")
	}
	for i := 1; i < len(launchAt); i++ {
		gap := launchAt[i] - launchAt[i-1]
		if gap < weapons.EnemyASCMCooldownSec-0.2 {
			t.Fatalf("launch %d gap %.1fs < cooldown %.0fs", i, gap, weapons.EnemyASCMCooldownSec)
		}
	}
	_ = player
}
