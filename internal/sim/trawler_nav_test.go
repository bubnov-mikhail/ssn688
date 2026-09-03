package sim

import (
	"os"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// Trawler must progress past the 3rd waypoint without orbiting (COLREGS/shore resume loops).
func TestTrawlerPassesThirdWaypoint(t *testing.T) {
	data, err := os.ReadFile("../../scenarios_generated/taiwan_formosa_watch.json")
	if err != nil {
		t.Skip("scenarios_generated/taiwan_formosa_watch.json not present")
	}
	scDef, err := campaign.ParseScenarioJSON(data, "taiwan_formosa_watch.json")
	if err != nil {
		t.Fatal(err)
	}
	m := campaign.FindMission(&scDef, "tw_attribution")
	if m == nil {
		t.Fatal("tw_attribution missing")
	}
	rt := campaign.Instantiate(&scDef, m, campaign.BuildContext{Vars: map[string]string{}})
	if rt == nil {
		t.Fatal("instantiate failed")
	}
	eng := NewEngine(rt)
	campaign.ApplyUnitPayloads(&eng.FireControl, m, map[string]string{})

	var trawler *world.Entity
	for _, e := range eng.Scenario.Entities {
		if e != nil && e.ID == "civ_trawler" {
			trawler = e
			break
		}
	}
	if trawler == nil {
		t.Fatal("civ_trawler missing")
	}
	if player := eng.Scenario.Player; player != nil {
		player.OrderedSpeed = 0
		player.SpeedKts = 0
	}

	dt := 1.0 / TickRate
	maxWP := trawler.RouteWP
	for eng.Clock.GameTime < 60*60 {
		eng.Update(dt)
		if trawler.RouteWP > maxWP {
			maxWP = trawler.RouteWP
		}
	}
	if maxWP < 3 {
		t.Fatalf("trawler max route wp=%d want >=3 after 60min (final wp=%d pos %.0f,%.0f)",
			maxWP, trawler.RouteWP, trawler.X, trawler.Y)
	}
}
