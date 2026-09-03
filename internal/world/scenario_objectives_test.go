package world

import (
	"strings"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/i18n"
)

func testMissionScenario() *Scenario {
	fox := &Entity{ID: "enemy_foxtrot", Status: StatusActive}
	grisha := &Entity{ID: "enemy_grisha", Status: StatusActive}
	tanker := &Entity{ID: "civ_tanker", Status: StatusActive}
	player := &Entity{ID: "player", Status: StatusActive}
	return &Scenario{
		Player:   player,
		Entities: []*Entity{fox, grisha, tanker},
		Objectives: []Objective{
			{ID: "obj_foxtrot", Primary: true, NeedDestroy: true, TargetID: "enemy_foxtrot", Description: i18n.T("sink diesel", "sink diesel")},
			{ID: "obj_grisha", NeedIdentify: true, NeedDestroy: true, TargetID: "enemy_grisha", Description: i18n.T("ID and sink surface", "ID and sink surface")},
			{ID: "obj_tanker", NeedIdentify: true, TargetID: "civ_tanker", Description: i18n.T("ID tanker", "ID tanker")},
		},
	}
}

func TestTrainingObjectivesIdentifyAndDestroy(t *testing.T) {
	sc := testMissionScenario()
	if len(sc.Objectives) != 3 {
		t.Fatalf("want 3 objectives, got %d", len(sc.Objectives))
	}
	fox := findObj(sc, "obj_foxtrot")
	grisha := findObj(sc, "obj_grisha")
	tanker := findObj(sc, "obj_tanker")
	if fox == nil || !fox.Primary || !fox.NeedDestroy || fox.NeedIdentify {
		t.Fatalf("foxtrot obj %+v", fox)
	}
	if grisha == nil || !grisha.NeedIdentify || !grisha.NeedDestroy {
		t.Fatalf("grisha obj %+v", grisha)
	}
	if tanker == nil || !tanker.NeedIdentify || tanker.NeedDestroy {
		t.Fatalf("tanker obj %+v", tanker)
	}

	sc.CheckObjectives()
	if sc.MissionComplete() {
		t.Fatal("fresh scenario should not be complete")
	}

	for _, e := range sc.Entities {
		if e.ID == "enemy_foxtrot" {
			e.Status = StatusSunk
		}
	}
	sc.CheckObjectives()
	if !findObj(sc, "obj_foxtrot").Complete {
		t.Fatal("foxtrot should complete on sink")
	}
	if findObj(sc, "obj_grisha").Complete {
		t.Fatal("grisha must not complete without ID+kill")
	}

	for _, e := range sc.Entities {
		if e.ID == "enemy_grisha" {
			e.Status = StatusSunk
		}
	}
	sc.CheckObjectives()
	if !findObj(sc, "obj_grisha").Complete {
		t.Fatal("grisha should complete on sink (wreck confirms ID)")
	}

	sc.NoteIdentified("civ_tanker", 0)
	sc.CheckObjectives()
	if !findObj(sc, "obj_tanker").Complete {
		t.Fatal("tanker should complete on ID only")
	}
	if !sc.MissionComplete() {
		t.Fatal("all tasks met")
	}
}

func TestMissionStatusReportShowsIDAndPriority(t *testing.T) {
	sc := testMissionScenario()
	sc.NoteIdentified("civ_tanker", 0)
	rep := sc.MissionStatusReport()
	if !strings.Contains(rep, "PRI") || !strings.Contains(rep, "SEC") {
		t.Fatalf("missing PRI/SEC:\n%s", rep)
	}
	if !strings.Contains(rep, "ID:YES") {
		t.Fatalf("tanker ID not reported:\n%s", rep)
	}
	if !strings.Contains(rep, "ID:NO") {
		t.Fatalf("open ID task missing:\n%s", rep)
	}
	if !strings.Contains(rep, "KILL:NO") {
		t.Fatalf("kill status missing:\n%s", rep)
	}
}

func TestMissionStatusReportOmitsHiddenObjectives(t *testing.T) {
	sc := testMissionScenario()
	sink := findObj(sc, "obj_tanker")
	if sink == nil {
		t.Fatal("missing tanker obj")
	}
	sink.Description = i18n.T("SECRET SINK TANKER", "SECRET SINK TANKER")
	sink.NeedDestroy = true
	sink.NeedIdentify = false
	sink.Hidden = true
	rep := sc.MissionStatusReport()
	if strings.Contains(rep, "SECRET SINK TANKER") {
		t.Fatalf("hidden objective leaked into REPORT:\n%s", rep)
	}
	sc.RevealObjective("obj_tanker")
	rep = sc.MissionStatusReport()
	if !strings.Contains(rep, "SECRET SINK TANKER") {
		t.Fatalf("revealed objective missing from REPORT:\n%s", rep)
	}
}

func findObj(sc *Scenario, id string) *Objective {
	for i := range sc.Objectives {
		if sc.Objectives[i].ID == id {
			return &sc.Objectives[i]
		}
	}
	return nil
}
