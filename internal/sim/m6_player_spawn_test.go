package sim

import (
	"math"
	"os"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
)

func TestBreakPressurePlayerSpawnBetweenIslands(t *testing.T) {
	data, err := os.ReadFile("../../scenarios_generated/taiwan_formosa_watch.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := campaign.ParseScenarioJSON(data, "x")
	if err != nil {
		t.Fatal(err)
	}
	m := campaign.FindMission(&sc, "tw_break_pressure")
	rt := campaign.Instantiate(&sc, m, campaign.BuildContext{})
	p := rt.Player
	if p == nil {
		t.Fatal("no player")
	}
	wantX, wantY := 7800.0, -9000.0
	dist := math.Hypot(p.X-wantX, p.Y-wantY)
	t.Logf("player at (%.0f,%.0f) depth=%.0f dist_from_wp0=%.0f", p.X, p.Y, p.DepthFt, dist)
	if dist > 50 {
		t.Fatalf("expected spawn near (7800,-9000), got (%.0f,%.0f)", p.X, p.Y)
	}
}
