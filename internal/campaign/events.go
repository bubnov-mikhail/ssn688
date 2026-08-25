package campaign

import "github.com/ssn688/sim/internal/world"

// EventWhen triggers an event (extensible for sim-time dispatch).
type EventWhen struct {
	Type string `json:"type"` // time, objective_complete, objective_identified, unit_destroyed, var_eq, var_unset

	AtSec       float64 `json:"at_sec,omitempty"`
	ObjectiveID string  `json:"objective_id,omitempty"`
	UnitID      string  `json:"unit_id,omitempty"`
	Var         string  `json:"var,omitempty"`
	Value       string  `json:"value,omitempty"`
	RequireVar  string  `json:"require_var,omitempty"`
	UnlessVar   string  `json:"unless_var,omitempty"`
}

// VarTruthy reports campaign vars used as booleans ("true").
func VarTruthy(vars map[string]string, key string) bool {
	if key == "" || vars == nil {
		return false
	}
	v := vars[key]
	return v == "true" || v == "1" || v == "yes"
}

// EventMatchesVars filters build-time / schedule events by campaign vars.
func EventMatchesVars(ev EventDef, vars map[string]string) bool {
	if ev.When.RequireVar != "" && !VarTruthy(vars, ev.When.RequireVar) {
		return false
	}
	if ev.When.UnlessVar != "" && VarTruthy(vars, ev.When.UnlessVar) {
		return false
	}
	switch ev.When.Type {
	case "var_eq":
		want := ev.When.Value
		if want == "" {
			want = "true"
		}
		return vars[ev.When.Var] == want
	case "var_unset":
		return !VarTruthy(vars, ev.When.Var)
	default:
		return true
	}
}

// FilterEvents keeps events that match campaign vars.
func FilterEvents(events []EventDef, vars map[string]string) []EventDef {
	out := make([]EventDef, 0, len(events))
	for _, ev := range events {
		if EventMatchesVars(ev, vars) {
			out = append(out, ev)
		}
	}
	return out
}

// EventAction is one side-effect when an event fires.
type EventAction struct {
	Type string `json:"type"` // comm_schedule, set_defcon, set_ai_state, set_var, reveal_objective

	ID          string  `json:"id,omitempty"`
	Text        string  `json:"text,omitempty"`
	AtSec       float64 `json:"at_sec,omitempty"`
	UnitID      string  `json:"unit_id,omitempty"`
	Defcon      int     `json:"defcon,omitempty"`
	AIState     string  `json:"ai_state,omitempty"`
	Var         string  `json:"var,omitempty"`
	Value       string  `json:"value,omitempty"`
	ObjectiveID string  `json:"objective_id,omitempty"`
}

// EventDef is a declarative when/then rule (stored on mission; runtime dispatch TBD).
type EventDef struct {
	ID      string        `json:"id"`
	When    EventWhen     `json:"when"`
	Actions []EventAction `json:"actions"`
}

// ApplyCommEvents merges time-based comm_schedule actions into CommSchedule.
func ApplyCommEvents(schedule []world.CommScheduledMessage, events []EventDef) []world.CommScheduledMessage {
	out := append([]world.CommScheduledMessage(nil), schedule...)
	seen := map[string]bool{}
	for _, m := range out {
		seen[m.ID] = true
	}
	for _, ev := range events {
		if ev.When.Type != "time" && ev.When.Type != "" {
			continue
		}
		at := ev.When.AtSec
		for _, act := range ev.Actions {
			if act.Type != "comm_schedule" {
				continue
			}
			id := act.ID
			if id == "" {
				id = ev.ID
			}
			if seen[id] {
				continue
			}
			t := at
			if act.AtSec > 0 {
				t = act.AtSec
			}
			out = append(out, world.CommScheduledMessage{ID: id, AtSec: t, Text: act.Text})
			seen[id] = true
		}
	}
	return out
}
