package campaign

import (
	"strings"
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestCounterstrokeBranches(t *testing.T) {
	ReloadScenarios()
	scDef := ScenarioByID(DemoScenarioID)
	if scDef == nil {
		t.Fatal("missing demo")
	}

	t.Run("grisha penalty and tanker id pending", func(t *testing.T) {
		rt := BuildMission(DemoScenarioID, DemoMissionCounterstroke, BuildContext{Vars: map[string]string{}})
		if rt == nil {
			t.Fatal("nil runtime")
		}
		ids := map[string]bool{}
		for _, e := range rt.Entities {
			ids[e.ID] = true
			if e.ID == "enemy_grisha" && (e.CrewSkill < 75 || e.CrewSkill > 95) {
				t.Fatalf("veteran grisha crew %.1f", e.CrewSkill)
			}
			if e.ID == "civ_tanker" && !e.AllyIgnore {
				t.Fatal("tanker should be ally-ignore")
			}
			if e.ID == "ally_688" {
				t.Fatal("bremerton should be absent")
			}
		}
		if !ids["enemy_kilo"] || !ids["enemy_udaloy"] || !ids["enemy_grisha"] {
			t.Fatalf("units %+v", ids)
		}
		obj := map[string]world.Objective{}
		for _, o := range rt.Objectives {
			obj[o.ID] = o
		}
		if _, ok := obj["obj_tanker_id"]; !ok {
			t.Fatal("want tanker ID objective")
		}
		if o, ok := obj["obj_tanker_sink_hidden"]; !ok || !o.Hidden {
			t.Fatal("want hidden sink")
		}
		if _, ok := obj["obj_tanker_sink_known"]; ok {
			t.Fatal("known sink should be absent")
		}
		txt := ""
		for _, m := range rt.CommSchedule {
			txt += m.Text
		}
		if !strings.Contains(strings.ToLower(txt), "identify") || !strings.Contains(strings.ToLower(txt), "tanker") {
			t.Fatalf("tasking missing tanker ID:\n%s", txt)
		}
		if strings.Contains(txt, "Sink tanker MT Horizon — now assessed") || strings.Contains(txt, "now assessed hostile support") {
			t.Fatal("should not order sink before ID")
		}
	})

	t.Run("grisha gone tanker known", func(t *testing.T) {
		rt := BuildMission(DemoScenarioID, DemoMissionCounterstroke, BuildContext{Vars: map[string]string{
			"grisha_neutralized": "true",
			"tanker_identified":  "true",
		}})
		if rt == nil {
			t.Fatal("nil runtime")
		}
		for _, e := range rt.Entities {
			if e.ID == "enemy_grisha" {
				t.Fatal("grisha should be absent")
			}
		}
		obj := map[string]world.Objective{}
		for _, o := range rt.Objectives {
			obj[o.ID] = o
			if o.Hidden {
				t.Fatalf("unexpected hidden %s", o.ID)
			}
		}
		if _, ok := obj["obj_tanker_sink_known"]; !ok {
			t.Fatal("want known sink primary")
		}
		if _, ok := obj["obj_tanker_id"]; ok {
			t.Fatal("ID task should be absent")
		}
		txt := ""
		for _, m := range rt.CommSchedule {
			txt += m.Text
		}
		if !strings.Contains(strings.ToLower(txt), "sink") || !strings.Contains(txt, "MT Horizon") {
			t.Fatalf("want sink order:\n%s", txt)
		}
		if !strings.Contains(txt, "°") || !strings.Contains(strings.ToLower(txt), "course") {
			t.Fatalf("want lat/lon + course in COMM:\n%s", txt)
		}
	})
}

func TestResolveMissionOutputsByObjective(t *testing.T) {
	ReloadScenarios()
	sc := ScenarioByID(DemoScenarioID)
	out := ResolveMissionOutputs(sc, DemoMissionTraining, true, []ObjectiveOutcome{
		{ID: "obj_foxtrot", Complete: true},
		{ID: "obj_grisha", Complete: false},
		{ID: "obj_tanker", Complete: true},
	})
	if out["foxtrot_neutralized"] != "true" {
		t.Fatalf("foxtrot var %+v", out)
	}
	if out["grisha_neutralized"] == "true" {
		t.Fatal("grisha should not be set")
	}
	if out["tanker_identified"] != "true" {
		t.Fatalf("tanker var %+v", out)
	}
}
