package campaign

import "github.com/ssn688/sim/internal/world"

// NextMission returns the mission after id, or nil if id is last / unknown.
func NextMission(sc *ScenarioDef, id MissionID) *MissionDef {
	if sc == nil {
		return nil
	}
	for i := range sc.Missions {
		if sc.Missions[i].ID == id {
			if i+1 < len(sc.Missions) {
				return &sc.Missions[i+1]
			}
			return nil
		}
	}
	return nil
}

func FindMission(sc *ScenarioDef, id MissionID) *MissionDef {
	if sc == nil {
		return nil
	}
	for i := range sc.Missions {
		if sc.Missions[i].ID == id {
			return &sc.Missions[i]
		}
	}
	return nil
}

func MissionCover(sc *ScenarioDef, m *MissionDef) ([]byte, string) {
	if m != nil && len(m.CoverData) > 0 {
		key := m.CoverCacheKey
		if key == "" {
			key = "mission:" + string(m.ID)
		}
		return m.CoverData, key
	}
	if sc != nil && len(sc.CoverData) > 0 {
		key := sc.CoverCacheKey
		if key == "" {
			key = "scenario:" + string(sc.ID)
		}
		return sc.CoverData, key
	}
	return nil, ""
}

// MissionCoverFile is legacy fallback for embedded JPG paths.
func MissionCoverFile(sc *ScenarioDef, m *MissionDef) string {
	if m != nil && m.CoverFile != "" {
		return m.CoverFile
	}
	if sc != nil {
		return sc.CoverFile
	}
	return ""
}

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

func BuildMission(scenarioID ScenarioID, missionID MissionID, ctx BuildContext) *world.Scenario {
	sc := ScenarioByIDCompatible(scenarioID)
	m := MissionByID(scenarioID, missionID)
	if sc == nil || m == nil {
		return nil
	}
	if m.Build != nil {
		return m.Build(ctx)
	}
	return Instantiate(sc, m, ctx)
}

func DemoRuntime() *world.Scenario {
	return BuildMission(DemoScenarioID, DemoMissionTraining, BuildContext{})
}
