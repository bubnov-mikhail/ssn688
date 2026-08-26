package sim

import (
	"github.com/ssn688/sim/internal/ai"
	"github.com/ssn688/sim/internal/campaign"
	"github.com/ssn688/sim/internal/world"
)

func (e *Engine) dispatchMissionEvents(gameTime float64) {
	if e == nil || e.Scenario == nil || len(e.Scenario.MissionEvents) == 0 {
		return
	}
	if e.Scenario.FiredEventIDs == nil {
		e.Scenario.FiredEventIDs = map[string]bool{}
	}
	for _, ev := range e.Scenario.MissionEvents {
		if ev.ID == "" || e.Scenario.FiredEventIDs[ev.ID] {
			continue
		}
		if !e.missionEventReady(ev) {
			continue
		}
		e.Scenario.FiredEventIDs[ev.ID] = true
		e.applyMissionEventActions(ev, gameTime)
	}
}

func (e *Engine) missionEventReady(ev world.MissionEvent) bool {
	switch ev.WhenType {
	case "objective_identified":
		if ev.ObjectiveID == "" || e.Scenario == nil {
			return false
		}
		for _, o := range e.Scenario.Objectives {
			if o.ID == ev.ObjectiveID && o.Identified {
				return true
			}
		}
		return false
	case "objective_complete":
		if ev.ObjectiveID == "" || e.Scenario == nil {
			return false
		}
		for _, o := range e.Scenario.Objectives {
			if o.ID == ev.ObjectiveID && o.Complete {
				return true
			}
		}
		return false
	case "unit_destroyed":
		if ev.UnitID == "" || e.Scenario == nil {
			return false
		}
		for _, ent := range e.Scenario.AllEntities() {
			if ent != nil && ent.ID == ev.UnitID && ent.Status != world.StatusActive {
				return true
			}
		}
		return false
	default:
		// time / var_* handled at Instantiate into CommSchedule
		return false
	}
}

func (e *Engine) applyMissionEventActions(ev world.MissionEvent, gameTime float64) {
	for _, act := range ev.Actions {
		switch act.Type {
		case "reveal_objective":
			id := act.ObjectiveID
			if id == "" {
				id = ev.ObjectiveID
			}
			e.Scenario.RevealObjective(id)
		case "comm_schedule":
			id := act.ID
			if id == "" {
				id = ev.ID
			}
			at := act.AtSec
			if at <= 0 {
				at = gameTime
			}
			byID := map[string]*world.Entity{}
			for _, ent := range e.Scenario.AllEntities() {
				if ent != nil && ent.ID != "" {
					byID[ent.ID] = ent
				}
			}
			e.Scenario.CommSchedule = append(e.Scenario.CommSchedule, world.CommScheduledMessage{
				ID: id, AtSec: at, Text: campaign.ExpandPlaceholders(act.Text, byID, e.Scenario.StartTimeSec, at),
			})
		case "set_defcon":
			for _, ent := range e.Scenario.Entities {
				if ent != nil && ent.ID == act.UnitID {
					ent.RaiseDefcon(act.Defcon)
				}
			}
		case "set_ai_state":
			for _, ent := range e.Scenario.Entities {
				if ent != nil && ent.ID == act.UnitID {
					ent.AIState = act.AIState
				}
			}
		}
	}
}

// noteEnemySurfaceWeaponFired triggers allied surface ASSIST once when a hostile
// surface combatant employs ASW weapons (RBU / tubes / Rastrub).
func (e *Engine) noteEnemySurfaceWeaponFired() {
	if e == nil || e.Scenario == nil {
		return
	}
	e.Scenario.FiredEventIDs = ensureFired(e.Scenario.FiredEventIDs)
	if e.Scenario.FiredEventIDs["ally_surface_assist"] {
		return
	}
	e.Scenario.FiredEventIDs["ally_surface_assist"] = true
	ai.TriggerAllySurfaceAssist(e.Scenario.Entities, e.Scenario.Player)
}

func ensureFired(m map[string]bool) map[string]bool {
	if m == nil {
		return map[string]bool{}
	}
	return m
}
