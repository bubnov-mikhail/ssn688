package campaign

import (
	"math/rand"
	"time"

	"github.com/ssn688/sim/internal/world"
)

// TheaterChart returns the shared bathymetry for a theater. Missions store only
// TheaterID so several missions can share one depth grid.
func TheaterChart(sc *ScenarioDef, id TheaterID) *world.Bathymetry {
	if sc == nil {
		return nil
	}
	for i := range sc.Theaters {
		t := &sc.Theaters[i]
		if t.ID == id && t.Chart != nil && t.Chart.Valid() {
			return t.Chart
		}
	}
	return nil
}

// ResolveMissionBathy returns the theater chart for a scenario mission (save/reload).
func ResolveMissionBathy(scenarioID ScenarioID, missionID MissionID) *world.Bathymetry {
	sc := ScenarioByIDCompatible(scenarioID)
	if sc == nil {
		return nil
	}
	var theaterID TheaterID
	if m := FindMission(sc, missionID); m != nil {
		theaterID = m.TheaterID
	} else if len(sc.Missions) > 0 {
		theaterID = sc.Missions[0].TheaterID
	} else if len(sc.Theaters) > 0 {
		theaterID = sc.Theaters[0].ID
	}
	return TheaterChart(sc, theaterID)
}

// RuntimeObjectives copies static templates into live mission tasks.
func RuntimeObjectives(templates []ObjectiveTemplate) []world.Objective {
	out := make([]world.Objective, len(templates))
	for i, t := range templates {
		out[i] = world.Objective{
			ID:           t.ID,
			Description:  t.Description,
			TargetID:     t.TargetID,
			Primary:      t.Primary,
			NeedIdentify: t.NeedIdentify,
			NeedDestroy:  t.NeedDestroy,
		}
	}
	return out
}

// Instantiate builds a live world.Scenario from campaign data (theater, routes, units, COMM).
func Instantiate(scDef *ScenarioDef, m *MissionDef, _ BuildContext) *world.Scenario {
	if scDef == nil || m == nil {
		return nil
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	bathy := TheaterChart(scDef, m.TheaterID)
	if bathy == nil || !bathy.Valid() {
		return nil
	}

	routes := make([]*world.Route, 0, len(m.Routes))
	clearance := make([]*world.Route, 0, len(m.Routes))
	for _, spec := range m.Routes {
		r := buildRoute(spec)
		if r == nil {
			continue
		}
		routes = append(routes, r)
		if spec.PlayerClearance {
			clearance = append(clearance, r)
		}
	}

	player := spawnEntity(rng, m.Player)
	if player == nil {
		return nil
	}
	placePlayer(rng, player, bathy, clearance, m.Player)
	world.ClampSubToBottom(player, bathy)

	ents := make([]*world.Entity, 0, len(m.Units))
	for _, spec := range m.Units {
		e := spawnEntity(rng, spec)
		if e == nil {
			continue
		}
		placeUnit(rng, e, player, bathy, routes, spec)
		world.ClampSubToBottom(e, bathy)
		if spec.Combatant {
			world.InitCombatantDamage(e)
		}
		ents = append(ents, e)
	}

	if m.Player.Combatant {
		world.InitCombatantDamage(player)
	}

	return &world.Scenario{
		Name:         m.Title,
		Description:  m.Description,
		Player:       player,
		Entities:     ents,
		Bathy:        bathy,
		Weather:      world.RandomWeather(rng),
		Routes:       routes,
		Objectives:   RuntimeObjectives(m.Objectives),
		CommBriefing: m.CommBriefing,
		CommSchedule: append([]world.CommScheduledMessage(nil), m.CommSchedule...),
	}
}

func buildRoute(spec RouteSpec) *world.Route {
	if spec.ID == "" || len(spec.Waypoints) < 2 {
		return nil
	}
	wps := append([]world.Waypoint(nil), spec.Waypoints...)
	r := &world.Route{ID: spec.ID, Waypoints: wps}
	switch spec.Mode {
	case RoutePingPong:
		r.PingPong = true
	case RouteLoop:
		r.Looped = true
		// Ensure a closing point: if last ≠ first, leave UniqueCount = n and wrap via Looped.
		// If author already duplicated first at end, UniqueCount drops the duplicate.
	case RouteOpen, "":
		// one-way
	default:
		return nil
	}
	return r
}

func spawnEntity(rng *rand.Rand, spec UnitSpec) *world.Entity {
	if spec.ID == "" {
		return nil
	}
	depth := spec.DepthFt
	if spec.DepthJitter > 0 {
		depth += rng.Float64() * spec.DepthJitter
	}
	e := &world.Entity{
		ID:           spec.ID,
		Name:         spec.Name,
		Kind:         spec.Kind,
		Side:         spec.Side,
		Status:       world.StatusActive,
		SignatureID:  spec.SignatureID,
		DepthFt:      depth,
		HeadingDeg:   spec.HeadingDeg,
		SpeedKts:     spec.SpeedKts,
		OrderedSpeed: spec.SpeedKts,
		OrderedDepth: depth,
		OrderedHead:  spec.HeadingDeg,
		LengthFt:     spec.LengthFt,
		AIState:      spec.AIState,
		Defcon:       spec.Defcon,
	}
	if spec.CrewSkill > 0 || spec.CrewJitter > 0 {
		e.CrewSkill = world.RandomCrewSkill(spec.CrewSkill, spec.CrewJitter, rng.Float64())
	}
	return e
}

func placePlayer(rng *rand.Rand, player *world.Entity, bathy *world.Bathymetry, routes []*world.Route, spec UnitSpec) {
	corner := spec.Corner
	if corner == "" {
		corner = "SW"
	}
	if world.PlaceNearChartCorner(player, bathy, corner, routes, spec.MinRouteYd, spec.MaxRouteYd) {
		return
	}
	origin := &world.Entity{}
	world.PlaceAwayFrom(rng, player, origin, 0, 4000, nil, bathy)
}

func placeUnit(rng *rand.Rand, e, player *world.Entity, bathy *world.Bathymetry, routes []*world.Route, spec UnitSpec) {
	r := world.FindRoute(routes, spec.RouteID)
	if spec.Spawn == SpawnOnRoute && r != nil && world.PlaceOnRouteFraction(e, r, spec.RouteFrac, bathy) {
		return
	}
	if spec.FallbackCorner != "" {
		if world.PlaceNearChartCorner(e, bathy, spec.FallbackCorner, nil, 0, 0) {
			if r != nil {
				world.AssignRoute(e, r)
			}
			return
		}
	}
	minR, maxR := spec.FallbackMinYd, spec.FallbackMaxYd
	if maxR <= 0 {
		minR, maxR = 2500, 9000
	}
	world.PlaceAwayFrom(rng, e, player, minR, maxR, nil, bathy)
	if r != nil {
		world.AssignRoute(e, r)
	}
}
