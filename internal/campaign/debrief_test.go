package campaign

import (
	"strings"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestComposeMissionDebriefVariants(t *testing.T) {
	sc := ScenarioByID(DemoScenarioID)
	if sc == nil || len(sc.Missions) == 0 {
		t.Fatal("demo scenario not loaded")
	}
	m := sc.Missions[0]
	lead := "Hostile diesel submarine is confirmed sunk"
	grishaOK := "positively identified and sunk"
	grishaFail := "was not both identified and sunk"
	tankerOK := "MT Horizon was located and positively identified"
	tankerFail := "Tanker search incomplete"

	cases := []struct {
		name     string
		outcomes []ObjectiveOutcome
		want     []string
		not      []string
	}{
		{
			name: "grisha and tanker complete",
			outcomes: []ObjectiveOutcome{
				{ID: "obj_grisha", Identified: true, Destroyed: true, Complete: true},
				{ID: "obj_tanker", Identified: true, Complete: true},
			},
			want: []string{lead, grishaOK, tankerOK},
			not:  []string{grishaFail, tankerFail},
		},
		{
			name: "grisha complete tanker missed",
			outcomes: []ObjectiveOutcome{
				{ID: "obj_grisha", Identified: true, Destroyed: true, Complete: true},
				{ID: "obj_tanker", Complete: false},
			},
			want: []string{lead, grishaOK, tankerFail},
			not:  []string{grishaFail, tankerOK},
		},
		{
			name: "grisha missed tanker complete",
			outcomes: []ObjectiveOutcome{
				{ID: "obj_grisha", Identified: true, Destroyed: false, Complete: false},
				{ID: "obj_tanker", Identified: true, Complete: true},
			},
			want: []string{lead, grishaFail, tankerOK},
			not:  []string{grishaOK, tankerFail},
		},
		{
			name: "both secondary missed",
			outcomes: []ObjectiveOutcome{
				{ID: "obj_grisha", Complete: false},
				{ID: "obj_tanker", Complete: false},
			},
			want: []string{lead, grishaFail, tankerFail},
			not:  []string{grishaOK, tankerOK},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComposeMissionDebrief(m, tc.outcomes, i18n.LangEN)
			for _, s := range tc.want {
				if !strings.Contains(got, s) {
					t.Fatalf("missing %q in:\n%s", s, got)
				}
			}
			for _, s := range tc.not {
				if strings.Contains(got, s) {
					t.Fatalf("unexpected %q in:\n%s", s, got)
				}
			}
		})
	}
}

func TestSnapshotObjectiveOutcomes(t *testing.T) {
	sc := DemoRuntime()
	for _, e := range sc.Entities {
		if e.ID == "enemy_foxtrot" || e.ID == "enemy_grisha" {
			e.Status = world.StatusSunk
		}
	}
	sc.NoteIdentified("enemy_grisha", 0)
	sc.NoteIdentified("civ_tanker", 0)
	got := SnapshotObjectiveOutcomes(sc)
	byID := map[string]ObjectiveOutcome{}
	for _, o := range got {
		byID[o.ID] = o
	}
	if !byID["obj_foxtrot"].Destroyed || !byID["obj_foxtrot"].Complete {
		t.Fatalf("foxtrot: %+v", byID["obj_foxtrot"])
	}
	if !byID["obj_grisha"].Identified || !byID["obj_grisha"].Destroyed || !byID["obj_grisha"].Complete {
		t.Fatalf("grisha: %+v", byID["obj_grisha"])
	}
	if !byID["obj_tanker"].Identified || !byID["obj_tanker"].Complete {
		t.Fatalf("tanker: %+v", byID["obj_tanker"])
	}
}

func TestNextMissionDemoCounterstroke(t *testing.T) {
	sc := ScenarioByID(DemoScenarioID)
	if sc == nil {
		t.Fatal("demo scenario not loaded")
	}
	next := NextMission(sc, DemoMissionTraining)
	if next == nil || next.ID != DemoMissionCounterstroke {
		t.Fatalf("want counterstroke after training, got %#v", next)
	}
	if NextMission(sc, DemoMissionCounterstroke) != nil {
		t.Fatal("counterstroke should be last")
	}
}
