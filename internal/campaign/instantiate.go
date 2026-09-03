package campaign

import (
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/world"
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

	routes, clearance := RuntimeRoutes(m.Routes)

	player := spawnEntity(rng, m.Player)
	if player == nil {
		return nil
	}
	placePlayer(rng, player, bathy, routes, clearance, m.Player)
	world.ClampSubToBottom(player, bathy)

	ents := make([]*world.Entity, 0, len(m.Units))
	placed := []*world.Entity{player}
	for _, spec := range m.Units {
		if !specMatchesVars(spec.RequireVar, spec.UnlessVar, vars) {
			continue
		}
		e := spawnEntity(rng, spec)
		if e == nil {
			continue
		}
		placeUnit(rng, e, placed, bathy, routes, spec)
		world.ClampSubToBottom(e, bathy)
		if spec.Combatant {
			world.InitCombatantDamage(e)
		}
		ents = append(ents, e)
		placed = append(placed, e)
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
		EndAfterEventID: m.EndAfterEvent,
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
		we := world.MissionEvent{
			ID: ev.ID, WhenType: ev.When.Type, WhenAtSec: ev.When.AtSec,
			ObjectiveID: ev.When.ObjectiveID, UnitID: ev.When.UnitID,
			RequireEventID: ev.When.RequireEvent, UnlessEventID: ev.When.UnlessEvent,
		}
		for _, act := range ev.Actions {
			we.Actions = append(we.Actions, world.MissionEventAction{
				Type: act.Type, ID: act.ID, Text: act.Text.TT(), AtSec: act.AtSec,
				UnitID: act.UnitID, ShooterID: act.ShooterID, AttributedTo: act.AttributedTo,
				TargetID: act.TargetID, Weapon: act.Weapon,
				Defcon: act.Defcon, AIState: act.AIState,
				Var: act.Var, Value: act.Value, ObjectiveID: act.ObjectiveID,
				X: act.X, Y: act.Y, Name: act.Name.TT(),
			})
		}
		out = append(out, we)
	}
	return out
}

// RuntimeRoutes builds world routes from mission specs. Second return is the
// subset marked player_clearance (for spawn placement). Used by Instantiate and
// save reload — route geometry is not persisted in .sav files.
func RuntimeRoutes(specs []RouteSpec) (all, clearance []*world.Route) {
	all = make([]*world.Route, 0, len(specs))
	clearance = make([]*world.Route, 0)
	for _, spec := range specs {
		r := buildRoute(spec)
		if r == nil {
			continue
		}
		all = append(all, r)
		if spec.PlayerClearance {
			clearance = append(clearance, r)
		}
	}
	return all, clearance
}

// RuntimeComm builds briefing + schedule (with event merges and placeholders)
// for the given entities. Used by Instantiate and save reload.
func RuntimeComm(m *MissionDef, player *world.Entity, ents []*world.Entity, vars map[string]string) (briefing i18n.TranslatedText, schedule []world.CommScheduledMessage) {
	if m == nil {
		return nil, nil
	}
	events := FilterEvents(m.Events, vars)
	schedule = ApplyCommEvents(m.CommSchedule, events)
	byID := entityMap(append([]*world.Entity{player}, ents...))
	start := m.StartTimeSec
	schedule = expandCommPlaceholders(schedule, byID, start)
	briefing = ExpandPlaceholdersTT(m.CommBriefing.TT(), byID, start, 0)
	return briefing, schedule
}

// AppendFiredEventComm re-applies comm_schedule actions from already-fired
// mission events so late traffic stays on the runtime schedule after load.
func AppendFiredEventComm(schedule []world.CommScheduledMessage, events []world.MissionEvent, fired map[string]bool, player *world.Entity, ents []*world.Entity, startSec float64) []world.CommScheduledMessage {
	if len(events) == 0 || len(fired) == 0 {
		return schedule
	}
	byID := entityMap(append([]*world.Entity{player}, ents...))
	have := map[string]bool{}
	for _, m := range schedule {
		if m.ID != "" {
			have[m.ID] = true
		}
	}
	for _, ev := range events {
		if ev.ID == "" || !fired[ev.ID] {
			continue
		}
		for _, act := range ev.Actions {
			if act.Type != "comm_schedule" {
				continue
			}
			id := act.ID
			if id == "" {
				id = ev.ID
			}
			if id == "" || have[id] {
				continue
			}
			at := act.AtSec
			schedule = append(schedule, world.CommScheduledMessage{
				ID: id, AtSec: at, Text: ExpandPlaceholdersTT(act.Text, byID, startSec, at),
			})
			have[id] = true
		}
	}
	return schedule
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
		AllyIgnore:     spec.AllyIgnore,
		ExerciseTarget: spec.ExerciseTarget,
	}
	if spec.CrewSkill > 0 || spec.CrewJitter > 0 {
		e.CrewSkill = world.RandomCrewSkill(spec.CrewSkill, spec.CrewJitter, rng.Float64())
	}
	return e
}

func placePlayer(rng *rand.Rand, player *world.Entity, bathy *world.Bathymetry, allRoutes, clearance []*world.Route, spec UnitSpec) {
	switch spec.Spawn {
	case SpawnOnRoute:
		var route *world.Route
		for _, r := range allRoutes {
			if r != nil && r.ID == spec.RouteID {
				route = r
				break
			}
		}
		if route != nil && world.PlaceOnRouteFraction(player, route, spec.RouteFrac, bathy) {
			return
		}
		if spec.FallbackCorner != "" {
			if world.PlaceNearChartCorner(player, bathy, spec.FallbackCorner, clearance, spec.FallbackMinYd, spec.FallbackMaxYd, 0) {
				return
			}
		}
	case SpawnChartCorner, "":
		corner := spec.Corner
		if corner == "" {
			corner = "SW"
		}
		if world.PlaceNearChartCorner(player, bathy, corner, clearance, spec.MinRouteYd, spec.MaxRouteYd, spec.CornerInsetYd) {
			return
		}
	}
	placeOnWater(rng, player, spec.MinRouteYd, spec.MaxRouteYd, nil, bathy)
}

func placeUnit(rng *rand.Rand, e *world.Entity, placed []*world.Entity, bathy *world.Bathymetry, routes []*world.Route, spec UnitSpec) {
	switch spec.Spawn {
	case SpawnChartCorner:
		corner := spec.Corner
		if corner == "" {
			corner = "SW"
		}
		if world.PlaceNearChartCorner(e, bathy, corner, routes, spec.MinRouteYd, spec.MaxRouteYd, spec.CornerInsetYd) {
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
			if world.PlaceNearChartCorner(e, bathy, spec.FallbackCorner, routes, minYd, maxYd, 0) {
				return
			}
		}
	}
	placeOnWater(rng, e, 2000, 12000, placed, bathy)
}

func placeOnWater(rng *rand.Rand, e *world.Entity, minR, maxR float64, placed []*world.Entity, bathy *world.Bathymetry) {
	ref := &world.Entity{}
	if len(placed) > 0 && placed[0] != nil {
		ref = placed[0]
	}
	world.PlaceAwayFrom(rng, e, ref, minR, maxR, placed, bathy)
}
