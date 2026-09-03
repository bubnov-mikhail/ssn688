package campaign

import (
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

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

// MissionBriefMap returns the regional brief map image for a mission, if inlined.
// After WarmMissionBriefMap, bytes are nil but the cache key remains.
func MissionBriefMap(m *MissionDef) ([]byte, string) {
	if m == nil {
		return nil, ""
	}
	key := m.BriefMapCacheKey
	if key == "" && len(m.BriefMapData) == 0 {
		return nil, ""
	}
	if key == "" {
		key = "brief_map:" + string(m.ID)
	}
	return m.BriefMapData, key
}

// WarmMissionBriefMap decodes the brief map once and drops inlined PNG bytes.
func WarmMissionBriefMap(m *MissionDef) string {
	data, key := MissionBriefMap(m)
	if key == "" {
		return ""
	}
	if len(data) > 0 {
		render.EnsureScenarioCoverImage(key, data)
		m.BriefMapData = nil
	}
	return key
}

// WarmScenarioCover decodes cover art into the GPU cache if not already loaded.
func WarmScenarioCover(sc *ScenarioDef, m *MissionDef) string {
	data, key := MissionCover(sc, m)
	if len(data) == 0 || key == "" {
		if m != nil && m.CoverCacheKey != "" {
			return m.CoverCacheKey
		}
		if sc != nil && sc.CoverCacheKey != "" {
			return sc.CoverCacheKey
		}
		return ""
	}
	render.EnsureScenarioCoverImage(key, data)
	return key
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
