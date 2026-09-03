package sim

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestDispatchMissionEventFireWeapon(t *testing.T) {
	shooter := &world.Entity{ID: "ally_spruance", Kind: world.KindSurfaceShip, Side: world.SidePlayer, Status: world.StatusActive, X: 15000, Y: -12200}
	target := &world.Entity{ID: "ex_hulk_a", Kind: world.KindSurfaceShip, Side: world.SideNeutral, Status: world.StatusActive, X: 15000, Y: -13300}
	e := NewEngine(&world.Scenario{
		Player:   &world.Entity{ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer, Status: world.StatusActive, X: 15880, Y: -13334, DepthFt: 120},
		Entities: []*world.Entity{shooter, target},
		MissionEvents: []world.MissionEvent{{
			ID: "ex_fish_warn", WhenType: "time", WhenAtSec: 180,
			Actions: []world.MissionEventAction{{
				Type: "fire_weapon", ShooterID: "ally_spruance", TargetID: "ex_hulk_a", Weapon: "exercise_torpedo",
			}},
		}},
		FiredEventIDs: map[string]bool{},
	})
	e.dispatchMissionEvents(179)
	if len(e.FireControl.ActiveTorpedoes) != 0 {
		t.Fatalf("expected no torpedo before t=180, got %d", len(e.FireControl.ActiveTorpedoes))
	}
	e.dispatchMissionEvents(180)
	if len(e.FireControl.ActiveTorpedoes) != 1 {
		t.Fatalf("expected 1 torpedo at t=180, got %d", len(e.FireControl.ActiveTorpedoes))
	}
	torp := e.FireControl.ActiveTorpedoes[0]
	if torp.ParentSubID != "ally_spruance" || torp.TargetID != "ex_hulk_a" {
		t.Fatalf("unexpected torpedo parent/target: %s -> %s", torp.ParentSubID, torp.TargetID)
	}
	if e.Scenario.FiredEventIDs["ex_fish_warn"] != true {
		t.Fatal("event should be marked fired")
	}
}

func TestDispatchMissionDestroyUnit(t *testing.T) {
	civ := &world.Entity{
		ID: "civ_dawn", Name: "MV Pacific Dawn", Kind: world.KindSurfaceShip, Side: world.SideNeutral,
		Status: world.StatusActive, SignatureID: "merchant", X: 9315, Y: -1226, SpeedKts: 10,
	}
	shadow := &world.Entity{
		ID: "rf_shadow", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: -3995, Y: -2234, Defcon: world.DefconAware,
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: -295, Y: 816, DepthFt: 60,
	}
	e := NewEngine(&world.Scenario{
		Player:   player,
		Entities: []*world.Entity{civ, shadow},
		MissionEvents: []world.MissionEvent{{
			ID: "dawn_sinking", WhenType: "time", WhenAtSec: 828,
			Actions: []world.MissionEventAction{{
				Type: "destroy_unit", UnitID: "civ_dawn", AttributedTo: "plan_watch",
			}},
		}},
		FiredEventIDs: map[string]bool{},
	})
	e.dispatchMissionEvents(827)
	if civ.Status != world.StatusActive {
		t.Fatalf("civ should still be active before event, status=%v", civ.Status)
	}
	e.dispatchMissionEvents(828)
	if civ.Status != world.StatusSinking {
		t.Fatalf("civ should be sinking, status=%v", civ.Status)
	}
	if len(e.Sonar.BioTransients) == 0 {
		t.Fatal("expected blast transient on waterfall")
	}
	found := false
	for _, tr := range e.Sonar.BioTransients {
		if tr.Kind == "blast" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected blast-kind passive transient")
	}
	if shadow.Defcon >= world.DefconWeaponsFree {
		t.Fatalf("neutral blast must not raise enemy to weapons free: %d", shadow.Defcon)
	}
}

func TestDispatchMissionEventPlotMarker(t *testing.T) {
	label := i18n.T("RENDEZVOUS", "РАНДЕВУ")
	e := NewEngine(&world.Scenario{
		MissionEvents: []world.MissionEvent{{
			ID: "intel_rendezvous_marker", WhenType: "time", WhenAtSec: 120,
			Actions: []world.MissionEventAction{{
				Type: "plot_marker", ID: "rendezvous", X: 11900, Y: -10400, Name: label,
			}},
		}},
		FiredEventIDs: map[string]bool{},
	})
	e.dispatchMissionEvents(119)
	if len(e.PlotMarkers) != 0 {
		t.Fatalf("expected no marker before t=120, got %d", len(e.PlotMarkers))
	}
	e.dispatchMissionEvents(120)
	if len(e.PlotMarkers) != 1 {
		t.Fatalf("expected 1 marker at t=120, got %d", len(e.PlotMarkers))
	}
	m := e.PlotMarkers[0]
	if m.ID != "rendezvous" || m.X != 11900 || m.Y != -10400 {
		t.Fatalf("unexpected marker: %+v", m)
	}
	if m.DisplayLabel("en") != "RENDEZVOUS" {
		t.Fatalf("label en: %q", m.DisplayLabel("en"))
	}
}

func TestDispatchMissionEventCommDeduped(t *testing.T) {
	text := i18n.T("warn", "warn")
	e := NewEngine(&world.Scenario{
		CommSchedule: []world.CommScheduledMessage{{ID: "ex_fish_warn", AtSec: 180, Text: text}},
		MissionEvents: []world.MissionEvent{{
			ID: "ex_fish_warn", WhenType: "time", WhenAtSec: 180,
			Actions: []world.MissionEventAction{{Type: "comm_schedule", ID: "ex_fish_warn", Text: text}},
		}},
		FiredEventIDs: map[string]bool{},
	})
	e.dispatchMissionEvents(180)
	if len(e.Scenario.CommSchedule) != 1 {
		t.Fatalf("comm should not duplicate, got %d messages", len(e.Scenario.CommSchedule))
	}
}

func TestDispatchMissionEventTaskingAttrRendezvousMarker(t *testing.T) {
	label := i18n.T("RENDEZVOUS", "РАНДЕВУ")
	text := i18n.T("task", "task")
	e := NewEngine(&world.Scenario{
		MissionEvents: []world.MissionEvent{{
			ID: "tasking_attr", WhenType: "time", WhenAtSec: 25,
			Actions: []world.MissionEventAction{
				{Type: "comm_schedule", ID: "tasking_attr", Text: text},
				{Type: "plot_marker", ID: "strait_rendezvous", X: 7500, Y: 500, Name: label},
			},
		}},
		FiredEventIDs: map[string]bool{},
	})
	e.dispatchMissionEvents(24)
	if len(e.PlotMarkers) != 0 {
		t.Fatalf("expected no marker before t=25, got %d", len(e.PlotMarkers))
	}
	e.dispatchMissionEvents(25)
	if len(e.PlotMarkers) != 1 || e.PlotMarkers[0].ID != "strait_rendezvous" {
		t.Fatalf("expected strait_rendezvous at t=25, got %+v", e.PlotMarkers)
	}
}

func TestDispatchMissionEventEnemyProsecutesAllies(t *testing.T) {
	ally := &world.Entity{
		ID: "ally_spruance", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, X: 9000, Y: -1000,
	}
	hunter := &world.Entity{
		ID: "plan_grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 9200, Y: -800, AIProsecuting: true,
	}
	bremerton := &world.Entity{
		ID: "ally_688", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 11000, Y: -5000, AIState: "PATROL", RouteID: "route_ally_688",
	}
	e := NewEngine(&world.Scenario{
		Player:   &world.Entity{ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer, Status: world.StatusActive},
		Entities: []*world.Entity{ally, hunter, bremerton},
		MissionEvents: []world.MissionEvent{
			{
				ID: "enemy_contact_comm", WhenType: "enemy_prosecutes_allies",
				Actions: []world.MissionEventAction{{
					Type: "ally_sub_assist", X: 7500, Y: 500,
				}},
			},
			{
				ID: "enemy_contact_group_marker", WhenType: "enemy_prosecutes_allies",
				Actions: []world.MissionEventAction{{
					Type: "plot_marker", ID: "enemy_group", X: 9200, Y: -600, Name: i18n.T("HOSTILE GROUP", "ГРУППА"),
				}},
			},
		},
		FiredEventIDs: map[string]bool{},
	})
	e.dispatchMissionEvents(0)
	if !e.Scenario.FiredEventIDs["enemy_contact_comm"] {
		t.Fatal("contact comm event should fire")
	}
	if len(e.PlotMarkers) != 1 {
		t.Fatalf("expected 1 marker, got %d", len(e.PlotMarkers))
	}
	if e.PlotMarkers[0].ID != "enemy_group" {
		t.Fatalf("expected enemy_group marker, got %s", e.PlotMarkers[0].ID)
	}
	if bremerton.AIState != "INTERCEPT" {
		t.Fatalf("Bremerton should intercept, state=%s", bremerton.AIState)
	}
	if bremerton.RouteID != "" {
		t.Fatalf("Bremerton route should clear, got %q", bremerton.RouteID)
	}
}

func TestDispatchMissionEventShadowCueTiming(t *testing.T) {
	earlyRU := "не запрашивайте"
	lateRU := "Можете запросить завершение миссии"
	makeEng := func() *Engine {
		sc := &world.Scenario{
			Objectives: []world.Objective{{
				ID: "obj_rf_shadow", TargetID: "rf_shadow", NeedIdentify: true, Primary: true,
			}},
			MissionEvents: []world.MissionEvent{
				{
					ID: "provocation", WhenType: "time", WhenAtSec: 900,
					Actions: []world.MissionEventAction{{Type: "comm_schedule", ID: "provocation", Text: i18n.T("prov", "prov")}},
				},
				{
					ID: "shadow_cue_early", WhenType: "objective_identified", ObjectiveID: "obj_rf_shadow",
					UnlessEventID: "provocation",
					Actions: []world.MissionEventAction{{Type: "comm_schedule", ID: "shadow_cue", Text: i18n.T("early", earlyRU)}},
				},
				{
					ID: "shadow_cue_late", WhenType: "objective_identified", ObjectiveID: "obj_rf_shadow",
					RequireEventID: "provocation",
					Actions: []world.MissionEventAction{{Type: "comm_schedule", ID: "shadow_cue", Text: i18n.T("late", lateRU)}},
				},
			},
			FiredEventIDs: map[string]bool{},
			FiredEventAt:  map[string]float64{},
		}
		return NewEngine(sc)
	}

	// ID before provocation → early text.
	e := makeEng()
	e.Scenario.NoteIdentified("rf_shadow", 500)
	e.dispatchMissionEvents(500)
	if !e.Scenario.FiredEventIDs["shadow_cue_early"] || e.Scenario.FiredEventIDs["shadow_cue_late"] {
		t.Fatalf("early=%v late=%v", e.Scenario.FiredEventIDs["shadow_cue_early"], e.Scenario.FiredEventIDs["shadow_cue_late"])
	}
	if len(e.Scenario.CommSchedule) != 1 || e.Scenario.CommSchedule[0].Text.GetText("ru") != earlyRU {
		t.Fatalf("comm=%q", e.Scenario.CommSchedule[0].Text.GetText("ru"))
	}

	// Provocation first, then ID → late text.
	e = makeEng()
	e.dispatchMissionEvents(900)
	e.Scenario.NoteIdentified("rf_shadow", 950)
	e.dispatchMissionEvents(950)
	if e.Scenario.FiredEventIDs["shadow_cue_early"] || !e.Scenario.FiredEventIDs["shadow_cue_late"] {
		t.Fatalf("early=%v late=%v", e.Scenario.FiredEventIDs["shadow_cue_early"], e.Scenario.FiredEventIDs["shadow_cue_late"])
	}
	if len(e.Scenario.CommSchedule) != 2 || e.Scenario.CommSchedule[1].Text.GetText("ru") != lateRU {
		t.Fatalf("comm=%q", e.Scenario.CommSchedule[1].Text.GetText("ru"))
	}
}
