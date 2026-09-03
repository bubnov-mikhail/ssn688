package simreplay

import (
	"os"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestRecordKiloStaysNavigable(t *testing.T) {
	path := "../../scenarios_generated/taiwan_formosa_watch.json"
	if _, err := os.Stat(path); err != nil {
		t.Skip(err)
	}
	rep, err := RecordMission(RecordOptions{
		ScenarioPath: path,
		MissionID:    "tw_attribution",
		Seed:         1,
		MaxSec:       5400,
		SampleSec:    10,
		PlayerIdle:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	sc, _ := campaign.ParseScenarioJSON(data, path)
	m := campaign.FindMission(&sc, "tw_attribution")
	bathy := campaign.TheaterChart(&sc, m.TheaterID)
	for _, fr := range rep.Frames {
		for _, u := range fr.Units {
			if u.ID != "rf_kilo_quiet" {
				continue
			}
			if !bathy.NavigableFor(u.X, u.Y, world.KindSubmarine, 160) {
				t.Fatalf("kilo on land/shallow at t=%.0f pos=%.0f,%.0f depth_ft=160", fr.Time, u.X, u.Y)
			}
		}
	}
}
