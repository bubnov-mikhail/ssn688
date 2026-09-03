package sim

import (
	"os"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func kiloASCMEngAt(t *testing.T, targetSec float64) (*Engine, *world.Entity, *world.Entity) {
	t.Helper()
	path := "../../scenarios_generated/taiwan_formosa_watch.json"
	data, _ := os.ReadFile(path)
	sc, _ := campaign.ParseScenarioJSON(data, path)
	m := campaign.FindMission(&sc, "tw_attribution")
	ctx := campaign.BuildContext{Vars: map[string]string{}}
	rt := campaign.Instantiate(&sc, m, ctx)
	eng := NewEngine(rt)
	campaign.ApplyUnitPayloads(&eng.FireControl, m, ctx.Vars)
	player := eng.Scenario.Player
	player.OrderedSpeed = 0
	player.SpeedKts = 0
	var kilo *world.Entity
	for _, e := range eng.Scenario.Entities {
		if e != nil && e.ID == "rf_kilo_quiet" {
			kilo = e
		}
	}
	dt := 1.0 / TickRate
	for eng.Clock.GameTime < targetSec {
		eng.Update(dt)
	}
	return eng, kilo, player
}

func TestKiloASCMFiresBy1800(t *testing.T) {
	eng, kilo, _ := kiloASCMEngAt(t, 1800)
	if kilo == nil {
		t.Fatal("no kilo")
	}
	left := eng.FireControl.EnemyASCMLeft(kilo.ID)
	if left >= 4 {
		t.Fatalf("mag still full=%d", left)
	}
}
