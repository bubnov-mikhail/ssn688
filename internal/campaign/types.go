package campaign

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/ssn688/sim/internal/i18n"
	"github.com/ssn688/sim/internal/version"
	"github.com/ssn688/sim/internal/world"
)

// ScenarioID identifies a multi-mission campaign.
type ScenarioID string

// MissionID identifies one mission within a scenario.
type MissionID string

const DemoScenarioID ScenarioID = "demo_catalina"
const DemoMissionTraining MissionID = "catalina_training"
const DemoMissionCounterstroke MissionID = "catalina_counterstroke"

// ObjectiveTemplate describes a mission task before runtime state exists.
type ObjectiveTemplate struct {
	ID           string
	Description  LocText
	TargetID     string
	Primary      bool
	NeedIdentify bool
	NeedDestroy  bool
	Hidden       bool   // secondary/hidden tasks not shown until revealed
	RequireVar   string // include only when campaign var is "true"
	UnlessVar    string // skip when campaign var is "true"
}

// OutputRule records a campaign variable when a mission ends.
type OutputRule struct {
	Key                 string
	Value               string
	WhenPrimaryComplete bool
	WhenObjectiveID     string // set when this objective is Complete at mission end
}

// BuildContext carries cross-mission state into scenario construction.
type BuildContext struct {
	Vars   map[string]string
	Inputs map[string]string
}

// DebriefLine appends a result paragraph based on one objective's outcome.
type DebriefLine struct {
	ObjectiveID string
	OnSuccess   LocText
	OnFail      LocText
}

// ObjectiveOutcome is a persisted snapshot of one task at mission end.
type ObjectiveOutcome struct {
	ID         string
	Identified bool
	Destroyed  bool
	Complete   bool
}

// TheaterID names a shared chart (bathymetry / locale) for one or more missions.
type TheaterID string

const TheaterCatalina TheaterID = "catalina"

// TheaterDef is a location used by missions. Chart is shared by pointer so
// several missions on the same map do not duplicate the depth grid.
type TheaterDef struct {
	ID TheaterID
	// Chart is the bathymetry grid loaded from scenario JSON (data_b64). Required.
	Chart *world.Bathymetry
}

// RouteMode selects how an AI unit travels the waypoint list.
type RouteMode string

const (
	RouteOpen     RouteMode = "open"     // one-way; stop at last waypoint
	RoutePingPong RouteMode = "pingpong" // reverse at either end
	RouteLoop     RouteMode = "loop"     // closed cycle; wrap last → first
)

// RouteSpec is a named lane on the mission theater (explicit waypoints).
type RouteSpec struct {
	ID              string
	Mode            RouteMode
	Waypoints       []world.Waypoint
	PlayerClearance bool // include in ownship corner placement distance checks
}

// SpawnMode places a unit at mission start.
type SpawnMode string

const (
	SpawnChartCorner SpawnMode = "corner" // PlaceNearChartCorner
	SpawnOnRoute     SpawnMode = "route"  // PlaceOnRouteFraction
)

// UnitSpec is a spawnable platform (ownship or traffic/combatant).
type UnitSpec struct {
	ID             string
	Name           LocText
	Kind           world.EntityKind
	Side           world.Side
	SignatureID    string
	LengthFt       float64
	SpeedKts       float64
	DepthFt        float64
	DepthJitter    float64 // added as rng.Float64()*DepthJitter
	HeadingDeg     float64
	AIState        string
	Defcon         int
	CrewSkill      float64
	CrewJitter     float64
	Combatant      bool
	Spawn          SpawnMode
	Corner         string  // SW/SE/… for SpawnChartCorner
	MinRouteYd     float64 // player: stay this far from transit lanes
	MaxRouteYd     float64
	RouteID        string
	RouteFrac      float64
	FallbackCorner string
	FallbackMinYd  float64
	FallbackMaxYd  float64
	RequireVar     string // spawn only when campaign var is "true"
	UnlessVar      string // skip when campaign var is "true"
	AllyIgnore     bool   // allied AI must not attack this unit
	Payload        *UnitPayload
}

// MissionDef is the static definition of one mission.
type MissionDef struct {
	ID            MissionID
	Title         LocText
	Description   LocText
	CoverFile     string // optional; falls back to scenario cover
	CoverData     []byte
	CoverCacheKey string
	TheaterID     TheaterID
	Routes        []RouteSpec
	Player        UnitSpec
	Units         []UnitSpec
	CommBriefing  LocText
	CommSchedule  []world.CommScheduledMessage
	Events        []EventDef
	StartTimeSec  float64 // seconds from midnight (from start_time HH:MM)
	// Build overrides data-driven instantiate. Nil = Instantiate from fields above.
	Build        func(ctx BuildContext) *world.Scenario
	Objectives   []ObjectiveTemplate
	Outputs      []OutputRule
	DebriefLead  LocText
	DebriefLines []DebriefLine
}

// ScenarioDef is a campaign: linked missions with narrative framing.
type ScenarioDef struct {
	ID                 ScenarioID
	Title              LocText
	Backstory          LocText
	CoverFile          string // legacy bundled path; prefer CoverData
	CoverData          []byte
	CoverCacheKey      string
	PostscriptSuccess  LocText
	PostscriptFailure  LocText
	Theaters           []TheaterDef
	Missions           []MissionDef
	Events             []EventDef

	FormatVersion      SemVer
	Version            SemVer
	MinGameVersion     SemVer
	Compatible         bool
	IncompatibleReason string
	SourcePath         string
}

// ApplyCompatibility marks whether this scenario can run on the current game.
func (sc *ScenarioDef) ApplyCompatibility() {
	sc.Compatible = true
	sc.IncompatibleReason = ""
	if sc.FormatVersion.Major != CurrentScenarioFormatMajor() {
		sc.Compatible = false
		sc.IncompatibleReason = fmt.Sprintf(
			"scenario format %s is incompatible with game format %d.x",
			sc.FormatVersion, CurrentScenarioFormatMajor(),
		)
		return
	}
	game := CurrentGameVersion()
	if !game.AtLeast(sc.MinGameVersion) {
		sc.Compatible = false
		sc.IncompatibleReason = fmt.Sprintf(
			"requires game version %s or newer (running %s)",
			sc.MinGameVersion, game,
		)
	}
}

// CurrentGameVersion returns the running game semver (from root VERSION via internal/version).
var CurrentGameVersion = func() SemVer { return ParseSemVer(version.String()) }

// CurrentScenarioFormatMajor is the JSON schema major version the game accepts.
func CurrentScenarioFormatMajor() int { return version.ScenarioFormatMajor }

// Progress tracks player advancement through a scenario (persisted in saves).
type Progress struct {
	ScenarioID        ScenarioID
	CompletedMissions map[MissionID]bool
	Vars              map[string]string
	LoadoutMix        float64
	DebriefPending    bool
	DebriefMission    MissionID
	DebriefOutcomes   []ObjectiveOutcome
}

// RuntimeMeta is live campaign state carried on sim.Engine.
type RuntimeMeta struct {
	ScenarioID      ScenarioID
	MissionID       MissionID
	MissionHash     string
	LoadoutMix      float64
	ReportEligible  bool
	BetweenMissions bool
	Completed       map[MissionID]bool
	Vars            map[string]string
	DebriefPending  bool
	DebriefMission  MissionID
	DebriefOutcomes []ObjectiveOutcome
}

func MissionHash(m MissionDef) string {
	sum := sha256.Sum256([]byte(string(m.ID) + "|" + m.Title.GetText(i18n.LangEN)))
	return hex.EncodeToString(sum[:6])
}

func (p *Progress) Clone() Progress {
	out := Progress{
		ScenarioID:     p.ScenarioID,
		LoadoutMix:     p.LoadoutMix,
		DebriefPending: p.DebriefPending,
		DebriefMission: p.DebriefMission,
	}
	out.CompletedMissions = copyMissionMap(p.CompletedMissions)
	out.Vars = copyStringMap(p.Vars)
	out.DebriefOutcomes = copyOutcomes(p.DebriefOutcomes)
	return out
}

func copyOutcomes(in []ObjectiveOutcome) []ObjectiveOutcome {
	if len(in) == 0 {
		return nil
	}
	out := make([]ObjectiveOutcome, len(in))
	copy(out, in)
	return out
}

func copyMissionMap(m map[MissionID]bool) map[MissionID]bool {
	if len(m) == 0 {
		return map[MissionID]bool{}
	}
	out := make(map[MissionID]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func copyStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (r *RuntimeMeta) ToProgress() Progress {
	p := Progress{
		ScenarioID:      r.ScenarioID,
		LoadoutMix:      r.LoadoutMix,
		Vars:            copyStringMap(r.Vars),
		DebriefPending:  r.DebriefPending,
		DebriefMission:  r.DebriefMission,
		DebriefOutcomes: copyOutcomes(r.DebriefOutcomes),
	}
	p.CompletedMissions = copyMissionMap(r.Completed)
	return p
}

func (p Progress) CurrentMission(def *ScenarioDef) *MissionDef {
	if def == nil {
		return nil
	}
	for i := range def.Missions {
		if !p.CompletedMissions[def.Missions[i].ID] {
			return &def.Missions[i]
		}
	}
	if len(def.Missions) > 0 {
		return &def.Missions[len(def.Missions)-1]
	}
	return nil
}

func (p Progress) CurrentMissionIndex(def *ScenarioDef) int {
	if def == nil {
		return 0
	}
	for i := range def.Missions {
		if !p.CompletedMissions[def.Missions[i].ID] {
			return i
		}
	}
	if len(def.Missions) == 0 {
		return 0
	}
	return len(def.Missions) - 1
}

func (p Progress) ScenarioComplete(def *ScenarioDef) bool {
	if def == nil || len(def.Missions) == 0 {
		return false
	}
	for _, m := range def.Missions {
		if !p.CompletedMissions[m.ID] {
			return false
		}
	}
	return true
}

func BuildContextFromProgress(p Progress) BuildContext {
	return BuildContext{
		Vars:   copyStringMap(p.Vars),
		Inputs: copyStringMap(p.Vars),
	}
}
