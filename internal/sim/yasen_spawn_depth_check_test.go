package sim

import (
	"math"
	"os"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestYasenSpawnsOnFirstWaypoint(t *testing.T) {
	data, err := os.ReadFile("../../scenarios_generated/taiwan_formosa_watch.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := campaign.ParseScenarioJSON(data, "x")
	if err != nil {
		t.Fatal(err)
	}
	m := campaign.FindMission(&sc, "tw_combined_asw")
	rt := campaign.Instantiate(&sc, m, campaign.BuildContext{Vars: map[string]string{}})
	var y *world.Entity
	for _, e := range rt.Entities {
		if e != nil && e.ID == "rf_yasen" {
			y = e
			break
		}
	}
	var route *world.Route
	for _, rr := range rt.Routes {
		if rr != nil && rr.ID == "route_yasen" {
			route = rr
			break
		}
	}
	if y == nil || route == nil {
		t.Fatal("missing yasen/route")
	}
	wp0 := route.Waypoints[0]
	bot := rt.Bathy.DepthAtFt(wp0.X, wp0.Y)
	t.Logf("wp0=(%.0f,%.0f) bottom=%.1f yasen depth=%.0f ordered=%.0f pos=(%.0f,%.0f) dist=%.0f",
		wp0.X, wp0.Y, bot, y.DepthFt, y.OrderedDepth, y.X, y.Y, math.Hypot(y.X-wp0.X, y.Y-wp0.Y))
	if math.Hypot(y.X-wp0.X, y.Y-wp0.Y) > 50 {
		t.Fatalf("expected spawn on wp0, dist=%.0f", math.Hypot(y.X-wp0.X, y.Y-wp0.Y))
	}
}
