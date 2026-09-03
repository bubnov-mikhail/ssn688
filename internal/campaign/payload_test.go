package campaign

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestApplyUnitPayloads(t *testing.T) {
	n := 3
	m := &MissionDef{
		Units: []UnitSpec{{
			ID: "enemy_kilo", Kind: world.KindSubmarine, Side: world.SideEnemy,
			Payload: &UnitPayload{Torpedoes: &n},
		}},
	}
	fc := weapons.NewFireControl()
	ApplyUnitPayloads(&fc, m, nil)
	if fc.EnemyMagazine["enemy_kilo"] != 3 {
		t.Fatalf("torpedoes=%d", fc.EnemyMagazine["enemy_kilo"])
	}
	h := 8
	m.Units = append(m.Units, UnitSpec{
		ID: "ally_688", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Payload: &UnitPayload{Torpedoes: &n, Harpoons: &h},
	})
	ApplyUnitPayloads(&fc, m, nil)
	if fc.AllyHarpoonMag["ally_688"] != 8 {
		t.Fatalf("harpoons=%d", fc.AllyHarpoonMag["ally_688"])
	}
	c := 4
	m.Units = append(m.Units, UnitSpec{
		ID: "rf_kilo", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Payload: &UnitPayload{CruiseMissiles: &c},
	})
	ApplyUnitPayloads(&fc, m, nil)
	if fc.EnemyASCMMag["rf_kilo"] != 4 {
		t.Fatalf("cruise=%d", fc.EnemyASCMMag["rf_kilo"])
	}
	ex := 2
	m.Units = append(m.Units, UnitSpec{
		ID: "ex_hulk_a", Kind: world.KindSurfaceShip, Side: world.SideEnemy, ExerciseTarget: true,
		Payload: &UnitPayload{ExerciseTorpedoes: &ex, ShipTubes: intPtr(99)},
	})
	ApplyUnitPayloads(&fc, m, nil)
	if fc.EnemyExerciseTube["ex_hulk_a"] != 2 {
		t.Fatalf("exercise=%d", fc.EnemyExerciseTube["ex_hulk_a"])
	}
	if fc.EnemyShipTube["ex_hulk_a"] != 0 {
		t.Fatalf("ship tubes=%d", fc.EnemyShipTube["ex_hulk_a"])
	}
	if fc.EnemyRastrub["ex_hulk_a"] != 0 {
		t.Fatalf("rastrub=%d", fc.EnemyRastrub["ex_hulk_a"])
	}
	zero := 0
	m.Units = append(m.Units, UnitSpec{
		ID: "plan_sw_patrol_a", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Payload: &UnitPayload{ASWRockets: &zero, ShipTubes: &zero, RBU: &zero},
	})
	ApplyUnitPayloads(&fc, m, nil)
	if fc.EnemyRastrub["plan_sw_patrol_a"] != 0 {
		t.Fatalf("plan rastrub=%d", fc.EnemyRastrub["plan_sw_patrol_a"])
	}
}

func intPtr(n int) *int { return &n }
