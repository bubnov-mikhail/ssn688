package sim

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// ASCM must not fire on omniscient truth while the boat is still quiet PATROL
// with no acoustic solution on the surface quarry.
func TestEnemyASCMRequiresSurfaceSense(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 80000, DepthFt: 300,
		Damage: world.NewFullHealth(),
	}
	ally := &world.Entity{
		ID: "ally_spruance", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "spruance",
		X: 0, Y: 25000, SpeedKts: 14, DepthFt: 0,
		Damage: world.NewFullHealth(),
	}
	yasen := &world.Entity{
		ID: "rf_yasen", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "yasen_m",
		X: 0, Y: 0, DepthFt: 60, SpeedKts: 6, OrderedDepth: 60,
		Defcon: world.DefconWeaponsFree, AIState: "PATROL",
		Damage: world.NewFullHealth(), CrewSkill: 95,
	}
	sc := &world.Scenario{
		Player:   player,
		Entities: []*world.Entity{yasen, ally},
	}
	eng := NewEngine(sc)
	eng.FireControl.EnemyASCMMag = map[string]int{yasen.ID: 16}
	eng.Clock.GameTime = 100

	if q := eng.enemyASCMQuarry(yasen, player, eng.Clock.GameTime); q != nil {
		t.Fatalf("expected no sensed quarry at long range, got %s", q.ID)
	}
	eng.tryEnemyASCMShots(player, eng.Clock.GameTime)
	if eng.FireControl.EnemyASCMLeft(yasen.ID) != 16 {
		t.Fatalf("omniscient ASCM fire while green PATROL, mag=%d", eng.FireControl.EnemyASCMLeft(yasen.ID))
	}
	if len(eng.FireControl.ActiveHarpoons) != 0 {
		t.Fatal("unexpected harpoon in flight")
	}

	// Classified track on the hull unlocks release even without live SNR this tick.
	yasen.Track = world.AITrack{
		Valid: true, X: ally.X, Y: ally.Y, DepthFt: 0,
		ClassConf: 0.7, HoldSec: 8, UpdatedAt: eng.Clock.GameTime,
	}
	if q := eng.enemyASCMQuarry(yasen, player, eng.Clock.GameTime); q == nil || q.ID != ally.ID {
		t.Fatalf("expected track-matched quarry, got %v", q)
	}
	eng.tryEnemyASCMShots(player, eng.Clock.GameTime)
	if eng.FireControl.EnemyASCMLeft(yasen.ID) >= 16 {
		t.Fatal("expected ASCM after classified surface track")
	}
	if yasen.AIState != "FIRING" {
		t.Fatalf("launch should paint FIRING, got %s", yasen.AIState)
	}
}
