package ui

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/campaign"
	"github.com/ssn688/sim/internal/config"
	"github.com/ssn688/sim/internal/save"
	"github.com/ssn688/sim/internal/sim"
)

func TestReloadAfterSessionDispose(t *testing.T) {
	a := NewApp(config.Settings{}, nil)
	a.StartNewGame()

	// Simulate GPU resources + waterfall history created during play.
	a.waterfallImg = ebiten.NewImage(64, 64)
	a.waterfallPix = make([]byte, 64*64*4)
	a.passivePPI = ebiten.NewImage(64, 64)
	a.spectrumFuzzyImg = ebiten.NewImage(64, 64)
	a.activePlotImg = ebiten.NewImage(64, 64)
	a.wepsMapImg = ebiten.NewImage(64, 64)
	a.ensureTactical()
	a.tactical.bathyImg = ebiten.NewImage(64, 64)
	a.tactical.bathyPix = make([]byte, 64*64*4)
	row := make([]float64, acoustics.BearingWaterfallBins)
	a.bearingWaterfalls.Hull.PushCopy(row, 10)
	a.bearingWaterfalls.Towed.PushCopy(row, 20)

	dir := t.TempDir()
	path := filepath.Join(dir, "s.sav")
	if err := save.Save(path, a.Engine); err != nil {
		t.Fatal(err)
	}

	a.exitToMenu()
	if a.Engine != nil {
		t.Fatal("engine should be nil")
	}

	engine, err := save.LoadClean(path)
	if err != nil {
		t.Fatal(err)
	}
	a.beginGameSession(engine)

	// Warm paths that previously crashed after Dispose / waterfall Reset.
	a.ensureWaterfallImage()
	a.updateSimulationUI()
	for i := 0; i < 5; i++ {
		a.Engine.Update(0.05)
		a.updateSimulationUI()
	}
	view := tacticalMapView{0, 0, 128, 128, 0, 0, a.tactical.zoom}
	_ = a.ensureTacticalBathyImage(view, a.Engine.Scenario.Bathy, false)
	if a.Engine.Scenario.Player == nil {
		t.Fatal("nil player")
	}
}

func TestSecondLoadAfterNewGame(t *testing.T) {
	a := NewApp(config.Settings{}, nil)
	a.StartNewGame()
	a.waterfallImg = ebiten.NewImage(32, 32)
	dir := t.TempDir()
	path := filepath.Join(dir, "s.sav")
	_ = save.Save(path, a.Engine)
	a.exitToMenu()

	eng, err := save.LoadClean(path)
	if err != nil {
		t.Fatal(err)
	}
	a.beginGameSession(eng)
	a.waterfallImg = ebiten.NewImage(32, 32)
	a.exitToMenu()

	eng2, err := save.LoadClean(path)
	if err != nil {
		t.Fatal(err)
	}
	a.beginGameSession(eng2)
	a.Engine.Update(0.1)
	a.updateSimulationUI()
}

func TestLoadReplacesRunningEngineWithoutExit(t *testing.T) {
	// In case UI ever loads over a live session.
	a := NewApp(config.Settings{}, nil)
	a.StartNewGame()
	a.waterfallImg = ebiten.NewImage(32, 32)
	a.passivePPI = ebiten.NewImage(32, 32)
	dir := t.TempDir()
	path := filepath.Join(dir, "s.sav")
	engSave := sim.NewEngine(campaign.DemoRuntime())
	engSave.Campaign = campaign.RuntimeMeta{
		ScenarioID: campaign.DemoScenarioID,
		MissionID:  campaign.DemoMissionTraining,
		Completed:  map[campaign.MissionID]bool{},
		Vars:       map[string]string{},
	}
	_ = save.Save(path, engSave)

	eng, err := save.LoadClean(path)
	if err != nil {
		t.Fatal(err)
	}
	a.beginGameSession(eng)
	a.Engine.Update(0.1)
	a.updateSimulationUI()
}
