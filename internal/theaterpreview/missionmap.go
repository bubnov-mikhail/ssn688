package theaterpreview

import (
	"fmt"
	"os"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// MissionMap is bathy + mission routes for the sim player (no PNG).
type MissionMap struct {
	Bathy  *world.Bathymetry
	Routes []*world.Route
}

// LoadMissionMap reads scenario JSON and builds runtime routes for a mission.
func LoadMissionMap(scenarioPath, missionID string) (*MissionMap, error) {
	data, err := os.ReadFile(scenarioPath)
	if err != nil {
		return nil, err
	}
	scDef, err := campaign.ParseScenarioJSON(data, scenarioPath)
	if err != nil {
		return nil, err
	}
	m := campaign.FindMission(&scDef, campaign.MissionID(missionID))
	if m == nil {
		return nil, fmt.Errorf("mission %q not found", missionID)
	}
	bathy := campaign.TheaterChart(&scDef, m.TheaterID)
	if bathy == nil || !bathy.Valid() {
		return nil, fmt.Errorf("mission %q: invalid bathy for theater %q", missionID, m.TheaterID)
	}
	routes, _ := campaign.RuntimeRoutes(m.Routes)
	return &MissionMap{Bathy: bathy, Routes: routes}, nil
}
