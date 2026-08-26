package campaign

import (
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/ssn688/sim/internal/i18n"
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

func specMatchesVars(requireVar, unlessVar string, vars map[string]string) bool {
	if requireVar != "" && !VarTruthy(vars, requireVar) {
		return false
	}
	if unlessVar != "" && VarTruthy(vars, unlessVar) {
		return false
	}
	return true
}

// RuntimeObjectives copies static templates into live mission tasks (vars filtered).
func RuntimeObjectives(templates []ObjectiveTemplate, vars map[string]string) []world.Objective {
	out := make([]world.Objective, 0, len(templates))
	for _, t := range templates {
		if !specMatchesVars(t.RequireVar, t.UnlessVar, vars) {
			continue
		}
		out = append(out, world.Objective{
			ID:           t.ID,
			Description:  t.Description.TT(),
			TargetID:     t.TargetID,
			Primary:      t.Primary,
			NeedIdentify: t.NeedIdentify,
			NeedDestroy:  t.NeedDestroy,
			Hidden:       t.Hidden,
		})
	}
	return out
}

// Instantiate builds a live world.Scenario from campaign data (theater, routes, units, COMM).
func Instantiate(scDef *ScenarioDef, m *MissionDef, ctx BuildContext) *world.Scenario {
	if scDef == nil || m == nil {
		return nil
	}
	vars := ctx.Vars
	if vars == nil {
		vars = map[string]string{}
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
		if !specMatchesVars(spec.RequireVar, spec.UnlessVar, vars) {
			continue
		}
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

	events := FilterEvents(m.Events, vars)
	schedule := ApplyCommEvents(m.CommSchedule, events)
	byID := entityMap(append([]*world.Entity{player}, ents...))
	schedule = expandCommPlaceholders(schedule, byID, m.StartTimeSec)
	briefing := ExpandPlaceholdersTT(m.CommBriefing.TT(), byID, m.StartTimeSec, 0)
	lang := i18n.CurrentLang()

	return &world.Scenario{
		Name:          m.Title.GetText(lang),
		Description:   m.Description.GetText(lang),
		Player:        player,
		Entities:      ents,
		Bathy:         bathy,
		Weather:       world.RandomWeather(rng),
		Routes:        routes,
		Objectives:    RuntimeObjectives(m.Objectives, vars),
		CommBriefing:  briefing,
		CommSchedule:  schedule,
		StartTimeSec:  m.StartTimeSec,
		MissionEvents: ToWorldEvents(events),
	}
}

var unitPlaceholderRe = regexp.MustCompile(`\{\{unit\.([a-zA-Z0-9_]+)\.([a-zA-Z0-9_]+)\}\}`)

func entityMap(ents []*world.Entity) map[string]*world.Entity {
	byID := map[string]*world.Entity{}
	for _, e := range ents {
		if e != nil && e.ID != "" {
			byID[e.ID] = e
		}
	}
	return byID
}

func expandCommPlaceholders(schedule []world.CommScheduledMessage, byID map[string]*world.Entity, startSec float64) []world.CommScheduledMessage {
	out := make([]world.CommScheduledMessage, len(schedule))
	for i, m := range schedule {
		out[i] = m
		// mission_time uses transmit AtSec so late mast pickup still shows HQ send clock.
		out[i].Text = ExpandPlaceholdersTT(m.Text, byID, startSec, m.AtSec)
	}
	return out
}

// ExpandPlaceholdersTT expands placeholders in every language of a TranslatedText map.
func ExpandPlaceholdersTT(text i18n.TranslatedText, byID map[string]*world.Entity, startSec, elapsedSec float64) i18n.TranslatedText {
	if text == nil {
		return nil
	}
	out := make(i18n.TranslatedText, len(text))
	for lang, s := range text {
		out[lang] = ExpandPlaceholders(s, byID, startSec, elapsedSec)
	}
	return out
}

// ExpandPlaceholders replaces {{mission_time}} and {{unit.<id>.<field>}} tokens.
func ExpandPlaceholders(text string, byID map[string]*world.Entity, startSec, elapsedSec float64) string {
	if text == "" {
		return text
	}
	if strings.Contains(text, "{{mission_time}}") {
		text = strings.ReplaceAll(text, "{{mission_time}}", world.FormatMissionClock(startSec, elapsedSec))
	}
	return ExpandUnitPlaceholders(text, byID)
}

// ExpandUnitPlaceholders replaces {{unit.<id>.<field>}} tokens.
// Fields: pos/latlon (PLOT nav format), x, y (yards), course/heading (deg),
// speed (kts), depth (ft), name, id.
func ExpandUnitPlaceholders(text string, byID map[string]*world.Entity) string {
	if text == "" || !strings.Contains(text, "{{unit.") {
		return text
	}
	return unitPlaceholderRe.ReplaceAllStringFunc(text, func(tok string) string {
		m := unitPlaceholderRe.FindStringSubmatch(tok)
		if len(m) != 3 {
			return tok
		}
		id, field := m[1], strings.ToLower(m[2])
		e := byID[id]
		if e == nil {
			return tok
		}
		switch field {
		case "pos", "latlon", "position":
			lat, lon := world.WorldToLatLon(e.X, e.Y)
			return world.FormatNavLatLon(lat, lon)
		case "x":
			return fmt.Sprintf("%d", int(math.Round(e.X/100)*100))
		case "y":
			return fmt.Sprintf("%d", int(math.Round(e.Y/100)*100))
		case "course", "heading":
			return fmt.Sprintf("%d", int(math.Round(e.HeadingDeg/5)*5))
		case "speed":
			return fmt.Sprintf("%.0f", e.SpeedKts)
		case "depth":
			return fmt.Sprintf("%.0f", e.DepthFt)
		case "name":
			return e.Name
		case "id":
			return e.ID
		default:
			return tok
		}
	})
}

// ToWorldEvents copies campaign events into world runtime hooks.
func ToWorldEvents(events []EventDef) []world.MissionEvent {
	out := make([]world.MissionEvent, 0, len(events))
	for _, ev := range events {
		we := world.MissionEvent{ID: ev.ID, WhenType: ev.When.Type, ObjectiveID: ev.When.ObjectiveID, UnitID: ev.When.UnitID}
		for _, act := range ev.Actions {
			we.Actions = append(we.Actions, world.MissionEventAction{
				Type: act.Type, ID: act.ID, Text: act.Text.TT(), AtSec: act.AtSec,
				UnitID: act.UnitID, Defcon: act.Defcon, AIState: act.AIState,
				Var: act.Var, Value: act.Value, ObjectiveID: act.ObjectiveID,
			})
		}
		out = append(out, we)
	}
	return out
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
		Name:         spec.Name.GetText(i18n.CurrentLang()),
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
		AllyIgnore:   spec.AllyIgnore,
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
	placeOnWater(rng, player, spec.MinRouteYd, spec.MaxRouteYd, nil, bathy)
}

func placeUnit(rng *rand.Rand, e *world.Entity, player *world.Entity, bathy *world.Bathymetry, routes []*world.Route, spec UnitSpec) {
	switch spec.Spawn {
	case SpawnChartCorner:
		corner := spec.Corner
		if corner == "" {
			corner = "SW"
		}
		if world.PlaceNearChartCorner(e, bathy, corner, routes, spec.MinRouteYd, spec.MaxRouteYd) {
			return
		}
	case SpawnOnRoute, "":
		var route *world.Route
		for _, r := range routes {
			if r != nil && r.ID == spec.RouteID {
				route = r
				break
			}
		}
		if route != nil && world.PlaceOnRouteFraction(e, route, spec.RouteFrac, bathy) {
			return
		}
		if spec.FallbackCorner != "" {
			minYd, maxYd := spec.FallbackMinYd, spec.FallbackMaxYd
			if world.PlaceNearChartCorner(e, bathy, spec.FallbackCorner, routes, minYd, maxYd) {
				return
			}
		}
	}
	placeOnWater(rng, e, 2000, 12000, []*world.Entity{player}, bathy)
}

func placeOnWater(rng *rand.Rand, e *world.Entity, minR, maxR float64, others []*world.Entity, bathy *world.Bathymetry) {
	world.PlaceAwayFrom(rng, e, &world.Entity{}, minR, maxR, others, bathy)
}
