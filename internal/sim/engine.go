package sim

import (
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/ai"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

const TickRate = 10.0

// Engine runs the simulation at fixed timestep.
type Engine struct {
	Clock       Clock
	Scenario    *world.Scenario
	Acoustics   acoustics.Model
	Sonar       acoustics.SonarState
	FireControl weapons.FireControl
	Accum       float64
	Events      []string
}

func NewEngine(scenario *world.Scenario) *Engine {
	return &Engine{
		Clock:       NewClock(),
		Scenario:    scenario,
		Acoustics:   acoustics.NewModel(acoustics.DefaultEnvironment()),
		Sonar:       acoustics.NewSonarState(),
		FireControl: weapons.NewFireControl(),
	}
}

func (e *Engine) Update(realDT float64) {
	if e.Clock.Paused {
		return
	}
	e.Accum += realDT * e.Clock.TimeScale
	step := 1.0 / TickRate
	for e.Accum >= step {
		e.tick(step)
		e.Accum -= step
	}
}

func (e *Engine) tick(dt float64) {
	e.Clock.Advance(dt)
	t := e.Clock.GameTime
	player := e.Scenario.Player

	player.Advance(dt)
	for _, ent := range e.Scenario.Entities {
		if ent.Alive() {
			ent.Advance(dt)
		}
	}

	ai.UpdateAllAI(e.Scenario.Entities, player, t, e.Acoustics)

	emitters := e.Scenario.AllEntities()
	e.Sonar.UpdateTowed(dt)
	acoustics.UpdatePassive(e.Acoustics, player, emitters, &e.Sonar, t)
	acoustics.FireActivePing(e.Acoustics, player, emitters, &e.Sonar, t)

	alive := e.FireControl.ActiveTorpedoes[:0]
	for _, torp := range e.FireControl.ActiveTorpedoes {
		if !torp.Alive {
			continue
		}
		targets := e.Scenario.AllEntities()
		if hit := torp.Advance(dt, targets); hit != nil {
			e.Events = append(e.Events, "Target destroyed: "+hit.Name)
		}
		if torp.Alive {
			alive = append(alive, torp)
		}
	}
	e.FireControl.ActiveTorpedoes = alive

	e.Scenario.CheckObjectives()
	e.Acoustics.Env.UpdateLayerSurvey(t)

	if player.DepthFt > 1200 {
		player.Status = world.StatusSunk
	}
}

func (e *Engine) PopEvents() []string {
	ev := e.Events
	e.Events = nil
	return ev
}

func (e *Engine) EnemyEmitters() []*world.Entity {
	var out []*world.Entity
	for _, ent := range e.Scenario.Entities {
		if ent.Alive() {
			out = append(out, ent)
		}
	}
	return out
}
