package campaign

import (
	"github.com/ssn688/sim/internal/world"
)

// AllScenarios returns registered campaigns in menu order.
func AllScenarios() []ScenarioDef {
	return []ScenarioDef{DemoScenario()}
}

// ScenarioByID finds a campaign definition.
func ScenarioByID(id ScenarioID) *ScenarioDef {
	for _, sc := range AllScenarios() {
		if sc.ID == id {
			s := sc
			return &s
		}
	}
	return nil
}

// MissionByID finds a mission within a scenario.
func MissionByID(scenarioID ScenarioID, missionID MissionID) *MissionDef {
	sc := ScenarioByID(scenarioID)
	if sc == nil {
		return nil
	}
	for i := range sc.Missions {
		if sc.Missions[i].ID == missionID {
			return &sc.Missions[i]
		}
	}
	return nil
}

// BuildMission constructs runtime scenario state for a mission.
func BuildMission(scenarioID ScenarioID, missionID MissionID, ctx BuildContext) *world.Scenario {
	m := MissionByID(scenarioID, missionID)
	if m == nil || m.Build == nil {
		return nil
	}
	return m.Build(ctx)
}
