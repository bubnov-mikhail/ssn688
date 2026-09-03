package sim

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/ai"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestAllyTrackIdentificationCompletesObjective(t *testing.T) {
	grisha := &world.Entity{
		ID: "plan_grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 5000, Y: -5000,
	}
	ally := &world.Entity{
		ID: "ally_spruance", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, CrewSkill: 75,
		Track: world.AITrack{
			Valid: true, X: 5010, Y: -4990,
			ClassConf: 0.7, HoldSec: 30,
		},
	}
	sc := &world.Scenario{
		Player:   &world.Entity{ID: "player", Side: world.SidePlayer, Status: world.StatusActive},
		Entities: []*world.Entity{grisha, ally},
		Objectives: []world.Objective{
			{
				ID: "obj_grisha", TargetID: "plan_grisha",
				NeedIdentify: true, NeedDestroy: true,
				Description:  i18n.T("sink grisha", "sink grisha"),
			},
		},
	}
	eng := NewEngine(sc)
	eng.syncObjectiveProgress()
	if !sc.Objectives[0].Identified {
		t.Fatal("ally classified track should identify objective target")
	}
	if id := ai.TrackedHostileID(ally, sc.Entities, sc.Player); id != "plan_grisha" {
		t.Fatalf("tracked id=%q", id)
	}
}
