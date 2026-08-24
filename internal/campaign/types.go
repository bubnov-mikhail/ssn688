package campaign

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/ssn688/sim/internal/world"
)

// ScenarioID identifies a multi-mission campaign.
type ScenarioID string

// MissionID identifies one mission within a scenario.
type MissionID string

const DemoScenarioID ScenarioID = "demo_catalina"

// ObjectiveTemplate describes a mission task before runtime state exists.
type ObjectiveTemplate struct {
	ID           string
	Description  string
	TargetID     string
	Primary      bool
	NeedIdentify bool
	NeedDestroy  bool
	Hidden       bool // secondary/hidden tasks not shown until revealed
}

// OutputRule records a campaign variable when a mission ends.
type OutputRule struct {
	Key                 string
	Value               string
	WhenPrimaryComplete bool
}

// BuildContext carries cross-mission state into scenario construction.
type BuildContext struct {
	Vars   map[string]string
	Inputs map[string]string
}

// MissionDef is the static definition of one mission.
type MissionDef struct {
	ID          MissionID
	Title       string
	Description string
	Build       func(ctx BuildContext) *world.Scenario
	Objectives  []ObjectiveTemplate
	Outputs     []OutputRule
}

// ScenarioDef is a campaign: linked missions with narrative framing.
type ScenarioDef struct {
	ID                ScenarioID
	Title             string
	Backstory         string
	CoverFile         string // under assets/scenarios/
	PostscriptSuccess string
	PostscriptFailure string
	Missions          []MissionDef
}

// Progress tracks player advancement through a scenario (persisted in saves).
type Progress struct {
	ScenarioID        ScenarioID
	CompletedMissions map[MissionID]bool
	Vars              map[string]string
	LoadoutMix        float64
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
}

func MissionHash(m MissionDef) string {
	sum := sha256.Sum256([]byte(string(m.ID) + "|" + m.Title))
	return hex.EncodeToString(sum[:6])
}

func (p *Progress) Clone() Progress {
	out := Progress{
		ScenarioID: p.ScenarioID,
		LoadoutMix: p.LoadoutMix,
	}
	out.CompletedMissions = copyMissionMap(p.CompletedMissions)
	out.Vars = copyStringMap(p.Vars)
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
		ScenarioID: r.ScenarioID,
		LoadoutMix: r.LoadoutMix,
		Vars:       copyStringMap(r.Vars),
	}
	p.CompletedMissions = copyMissionMap(r.Completed)
	return p
}

func (p *Progress) CurrentMission(def *ScenarioDef) *MissionDef {
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

func (p *Progress) CurrentMissionIndex(def *ScenarioDef) int {
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

func (p *Progress) ScenarioComplete(def *ScenarioDef) bool {
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
