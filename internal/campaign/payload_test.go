package campaign

import (
	"testing"

	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
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
}
