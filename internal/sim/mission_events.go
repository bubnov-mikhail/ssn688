package sim

import (
	"github.com/bubnov-mikhail/ssn688/internal/ai"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
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
		if !e.missionEventReady(ev, gameTime) {
			continue
		}
		e.Scenario.FiredEventIDs[ev.ID] = true
		if e.Scenario.FiredEventAt == nil {
			e.Scenario.FiredEventAt = map[string]float64{}
		}
		e.Scenario.FiredEventAt[ev.ID] = gameTime
		e.applyMissionEventActions(ev, gameTime)
	}
}

func (e *Engine) missionEventReady(ev world.MissionEvent, gameTime float64) bool {
	switch ev.WhenType {
	case "time", "":
		if gameTime < ev.WhenAtSec {
			return false
		}
		return e.missionEventTimingGate(ev, gameTime)
	case "objective_identified":
		if ev.ObjectiveID == "" || e.Scenario == nil {
			return false
		}
		for _, o := range e.Scenario.Objectives {
			if o.ID == ev.ObjectiveID && o.Identified {
				return e.missionEventTimingGate(ev, o.IdentifiedAtSec)
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
	case "enemy_prosecutes_allies":
		return e.enemyProsecutesAllies()
	default:
		return false
	}
}

// missionEventTimingGate enforces require_event / unless_event relative to refTime.
// For objective_identified triggers refTime is IdentifiedAtSec; for time triggers it is gameTime.
func (e *Engine) missionEventTimingGate(ev world.MissionEvent, refTime float64) bool {
	if e == nil || e.Scenario == nil {
		return false
	}
	fired := e.Scenario.FiredEventIDs
	at := e.Scenario.FiredEventAt
	if ev.RequireEventID != "" {
		if fired == nil || !fired[ev.RequireEventID] {
			return false
		}
		if t, ok := at[ev.RequireEventID]; ok && refTime > 0 && refTime < t {
			return false
		}
	}
	if ev.UnlessEventID != "" {
		if fired != nil && fired[ev.UnlessEventID] {
			if t, ok := at[ev.UnlessEventID]; ok && (refTime <= 0 || refTime >= t) {
				return false
			}
		}
	}
	return true
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
			if commMessageScheduled(e.Scenario.CommSchedule, id) {
				break
			}
			at := act.AtSec
			if at <= 0 {
				at = gameTime
			}
			byID := entityByIDMap(e.Scenario)
			e.Scenario.CommSchedule = append(e.Scenario.CommSchedule, world.CommScheduledMessage{
				ID: id, AtSec: at, Text: campaign.ExpandPlaceholdersTT(act.Text, byID, e.Scenario.StartTimeSec, at),
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
		case "fire_weapon":
			e.applyMissionFireWeapon(act, gameTime)
		case "destroy_unit":
			e.applyMissionDestroyUnit(act, gameTime)
		case "plot_marker":
			id := act.ID
			if id == "" {
				id = ev.ID
			}
			x, y := act.X, act.Y
			if id == "enemy_group" {
				if cx, cy, ok := e.enemyGroupCentroid(); ok {
					x, y = cx, cy
				}
			}
			e.AddPlotMarkerAt(id, x, y, act.Name)
		case "ally_sub_assist":
			ai.TriggerAllySubAssist(e.Scenario.Entities, e.Scenario.Player, act.X, act.Y)
		}
	}
}

func (e *Engine) applyMissionFireWeapon(act world.MissionEventAction, gameTime float64) {
	if e == nil || e.Scenario == nil {
		return
	}
	shooterID := act.ShooterID
	if shooterID == "" {
		shooterID = act.UnitID
	}
	targetID := act.TargetID
	if shooterID == "" || targetID == "" || act.Weapon == "" {
		return
	}
	byID := entityByIDMap(e.Scenario)
	shooter := byID[shooterID]
	target := byID[targetID]
	if shooter == nil || target == nil {
		return
	}
	torp, err := e.FireControl.FireScenarioWeapon(shooter, target, act.Weapon, gameTime)
	if err != nil {
		return
	}
	if torp != nil {
		e.EmitTubeTransient(shooter, gameTime, false)
	}
}

func (e *Engine) applyMissionDestroyUnit(act world.MissionEventAction, gameTime float64) {
	if e == nil || e.Scenario == nil {
		return
	}
	targetID := act.UnitID
	if targetID == "" {
		targetID = act.TargetID
	}
	if targetID == "" {
		return
	}
	attributed := act.AttributedTo
	if attributed == "" {
		attributed = act.ShooterID
	}
	target := entityByIDMap(e.Scenario)[targetID]
	if target == nil || !target.Alive() {
		return
	}
	depthFt := target.DepthFt
	if target.Kind == world.KindSurfaceShip {
		depthFt = 0
	}
	det := &weapons.Detonation{
		X: target.X, Y: target.Y, DepthFt: depthFt,
		Hit: target, ShooterID: attributed,
	}
	e.handleDetonation(det, gameTime)
	if target.Alive() {
		target.EnsureDamage()
		target.Damage.Eff[world.SysHull] = 0
		e.beginSinking(target, gameTime)
		e.FireControl.OnPlatformLost(target.ID)
		e.Events = append(e.Events, "Target destroyed: "+target.Name)
	}
}

func entityByIDMap(sc *world.Scenario) map[string]*world.Entity {
	byID := map[string]*world.Entity{}
	if sc == nil {
		return byID
	}
	for _, ent := range sc.AllEntities() {
		if ent != nil && ent.ID != "" {
			byID[ent.ID] = ent
		}
	}
	return byID
}

func commMessageScheduled(schedule []world.CommScheduledMessage, id string) bool {
	for _, m := range schedule {
		if m.ID == id {
			return true
		}
	}
	return false
}

// enemyProsecutesAllies is true when a hostile combatant is prosecuting the player or an ally.
func (e *Engine) enemyProsecutesAllies() bool {
	if e == nil || e.Scenario == nil {
		return false
	}
	player := e.Scenario.Player
	for _, hunter := range e.Scenario.Entities {
		if hunter == nil || !hunter.Alive() || hunter.Side != world.SideEnemy || !hunter.AIProsecuting {
			continue
		}
		for _, q := range e.Scenario.AllEntities() {
			if !world.IsEnemyQuarryTarget(q, player) {
				continue
			}
			if hunter.RangeYardsTo(q) < 35000 {
				return true
			}
		}
	}
	return false
}

// enemyGroupCentroid averages positions of prosecuting enemy combatants (for plot markers).
func (e *Engine) enemyGroupCentroid() (x, y float64, ok bool) {
	if e == nil || e.Scenario == nil {
		return 0, 0, false
	}
	var n int
	for _, ent := range e.Scenario.Entities {
		if ent == nil || !ent.Alive() || ent.Side != world.SideEnemy || !ent.AIProsecuting {
			continue
		}
		if ent.Kind != world.KindSubmarine && ent.Kind != world.KindSurfaceShip {
			continue
		}
		x += ent.X
		y += ent.Y
		n++
	}
	if n == 0 {
		return 0, 0, false
	}
	return x / float64(n), y / float64(n), true
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
