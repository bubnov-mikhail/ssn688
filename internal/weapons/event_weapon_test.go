package weapons

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestLaunchExerciseShipTubeRange(t *testing.T) {
	fc := NewFireControl()
	ship := &world.Entity{ID: "ship", Kind: world.KindSurfaceShip, Side: world.SidePlayer, Status: world.StatusActive, X: 0, Y: 0}
	near := &world.Entity{ID: "tgt", Kind: world.KindSurfaceShip, Status: world.StatusActive, X: 0, Y: 500}
	far := &world.Entity{ID: "far", Kind: world.KindSurfaceShip, Status: world.StatusActive, X: 0, Y: 2000}
	if fc.LaunchExerciseShipTube(ship, near) != nil {
		t.Fatal("too close should fail")
	}
	torp := fc.LaunchExerciseShipTube(ship, far)
	if torp == nil {
		t.Fatal("in-range exercise launch failed")
	}
	if torp.TerminalMode != TerminalSignal {
		t.Fatalf("expected exercise terminal signal, got %d", torp.TerminalMode)
	}
	if torp.EscalatesDefcon() {
		t.Fatal("exercise fish must not escalate DEFCON")
	}
}

func TestExerciseHulkMagazineLimited(t *testing.T) {
	fc := NewFireControl()
	hulk := &world.Entity{
		ID: "ex_hulk_a", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "exercise_hulk", ExerciseTarget: true,
		X: 0, Y: 0,
	}
	tgt := &world.Entity{ID: "sub", Kind: world.KindSubmarine, Side: world.SidePlayer, Status: world.StatusActive, X: 0, Y: 2000, DepthFt: 60}
	fc.EnemyExerciseTube[hulk.ID] = 2
	for i := 0; i < 2; i++ {
		if fc.LaunchExerciseShipTube(hulk, tgt) == nil {
			t.Fatalf("launch %d failed", i+1)
		}
	}
	if fc.LaunchExerciseShipTube(hulk, tgt) != nil {
		t.Fatal("third exercise launch should be dry")
	}
	if fc.LaunchShipTube(hulk, tgt) != nil {
		t.Fatal("combat LW blocked")
	}
	if fc.ExerciseTubeAmmo(hulk) != 0 {
		t.Fatalf("mag=%d", fc.ExerciseTubeAmmo(hulk))
	}
}

func TestFireScenarioWeaponExercise(t *testing.T) {
	fc := NewFireControl()
	ship := &world.Entity{ID: "ship", Kind: world.KindSurfaceShip, Side: world.SidePlayer, Status: world.StatusActive, X: 15000, Y: -12200}
	target := &world.Entity{ID: "hulk", Kind: world.KindSurfaceShip, Status: world.StatusActive, X: 15000, Y: -13300}
	torp, err := fc.FireScenarioWeapon(ship, target, EventWeaponExerciseTorpedo, 180)
	if err != nil || torp == nil {
		t.Fatalf("FireScenarioWeapon: %v torp=%v", err, torp)
	}
}
