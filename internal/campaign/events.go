package campaign

import "github.com/ssn688/sim/internal/world"

// EventWhen triggers an event (extensible for sim-time dispatch).
type EventWhen struct {
	Type string `json:"type"` // time, objective_complete, objective_identified, unit_destroyed, var_eq

	AtSec       float64 `json:"at_sec,omitempty"`
	ObjectiveID string  `json:"objective_id,omitempty"`
	UnitID      string  `json:"unit_id,omitempty"`
	Var         string  `json:"var,omitempty"`
	Value       string  `json:"value,omitempty"`
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
