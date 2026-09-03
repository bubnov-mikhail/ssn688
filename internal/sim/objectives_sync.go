package sim

import (
	"github.com/bubnov-mikhail/ssn688/internal/ai"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func (e *Engine) syncObjectiveProgress() {
	if e.Scenario == nil {
		return
	}
	player := e.Scenario.Player
	gameTime := e.Clock.GameTime
	for _, ent := range e.Scenario.Entities {
		if ent == nil || !world.IsAllyAI(ent, player) || !ent.Alive() {
			continue
		}
		if id := ai.TrackedHostileID(ent, e.Scenario.Entities, player); id != "" {
			e.Scenario.NoteIdentified(id, gameTime)
		}
	}
}
