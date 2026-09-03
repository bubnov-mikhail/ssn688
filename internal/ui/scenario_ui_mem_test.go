package ui

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/config"
	"github.com/bubnov-mikhail/ssn688/internal/render"
)

func scenarioListBenchApp(t *testing.T) *App {
	t.Helper()
	if err := render.InitFonts(); err != nil {
		t.Fatal(err)
	}
	campaign.ReloadScenarios()
	a := NewApp(config.DefaultSettings(), nil)
	a.Mode = ModeScenarioList
	defs := campaign.AllScenarios()
	if len(defs) == 0 {
		t.Fatal("no scenarios loaded")
	}
	a.ScenarioListIndex = 0
	for i, d := range defs {
		if d.ID == "taiwan_formosa_watch" {
			a.ScenarioListIndex = i
			break
		}
	}
	a.ensureScenarioSelection()
	return a
}

func TestScenarioListDrawHeapGrowth(t *testing.T) {
	t.Skip("requires ebiten game loop; use tools/profile_scenario_ui or ./ssn688 -pprof")
}
