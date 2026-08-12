package ai

import (
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

// UpdateAllAI drives enemy combatants and neutral shipping.
func UpdateAllAI(entities []*world.Entity, player *world.Entity, gameTime float64, model acoustics.Model, torps []*weapons.Torpedo, cm *weapons.CountermeasureSystem, weather world.Weather, esm *acoustics.ESMState, comm *acoustics.COMMState, peri *acoustics.PeriscopeState, bathy *world.Bathymetry, routes []*world.Route) {
	ctx := EvadeContext{CM: cm, Env: model.Env, GameTime: gameTime, Weather: weather, ESM: esm, COMM: comm, Peri: peri}
	UpdateEnemyAI(entities, player, gameTime, model, torps, ctx, bathy, routes)
	UpdateCivilianAI(entities, player, gameTime, bathy, routes)
}

// UpdateCivilianAI steers neutrals along assigned cruise routes with COLREGS avoidance.
func UpdateCivilianAI(entities []*world.Entity, player *world.Entity, gameTime float64, bathy *world.Bathymetry, routes []*world.Route) {
	all := trafficUniverse(entities, player)

	for _, e := range entities {
		if !e.Alive() || e.Side != world.SideNeutral {
			continue
		}
		updateCivilian(e, all, routes)
		applyShoreAvoidance(e, bathy)
	}
}

func trafficUniverse(entities []*world.Entity, player *world.Entity) []*world.Entity {
	all := make([]*world.Entity, 0, len(entities)+1)
	all = append(all, player)
	all = append(all, entities...)
	return all
}

func updateCivilian(ship *world.Entity, all []*world.Entity, routes []*world.Route) {
	ship.OrderedDepth = 0
	ship.DepthFt = 0
	if !followAssignedRoute(ship, routes, "CRUISE", routeCruiseSpeed(ship)) {
		ship.OrderedSpeed = routeCruiseSpeed(ship)
		ship.AIState = "CRUISE"
	}
	applyColregsTraffic(ship, all)
}

func cruiseSpeed(e *world.Entity) float64 {
	return routeCruiseSpeed(e)
}

func hashID(id string) int {
	h := 0
	for _, c := range id {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}
