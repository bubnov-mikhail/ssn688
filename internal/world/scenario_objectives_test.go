package world

import (
	"strings"
	"testing"
)

func TestTrainingObjectivesIdentifyAndDestroy(t *testing.T) {
	sc := NewTrainingScenario()
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

	// Sink sub without ID — primary done.
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

	// Sink Grisha without ID — still open.
	for _, e := range sc.Entities {
		if e.ID == "enemy_grisha" {
			e.Status = StatusSunk
		}
	}
	sc.CheckObjectives()
	if findObj(sc, "obj_grisha").Complete {
		t.Fatal("grisha kill without ID should stay open")
	}

	sc.NoteIdentified("enemy_grisha")
	sc.CheckObjectives()
	if !findObj(sc, "obj_grisha").Complete {
		t.Fatal("grisha should complete after ID+kill")
	}

	sc.NoteIdentified("civ_tanker")
	sc.CheckObjectives()
	if !findObj(sc, "obj_tanker").Complete {
		t.Fatal("tanker should complete on ID only")
	}
	if !sc.MissionComplete() {
		t.Fatal("all tasks met")
	}
}

func TestMissionStatusReportShowsIDAndPriority(t *testing.T) {
	sc := NewTrainingScenario()
	sc.NoteIdentified("civ_tanker")
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

func TestFollowOnTaskingMentionsIdentify(t *testing.T) {
	sc := NewTrainingScenario()
	if len(sc.CommSchedule) == 0 || sc.CommSchedule[0].AtSec != 20 {
		t.Fatal("expected 20s follow-on")
	}
	txt := sc.CommSchedule[0].Text
	for _, needle := range []string{"PRIMARY", "SECONDARY", "IDENTIFY", "800", "80 PCT", "TANKER"} {
		if !strings.Contains(txt, needle) {
			t.Fatalf("tasking missing %q:\n%s", needle, txt)
		}
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
