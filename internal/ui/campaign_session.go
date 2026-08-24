package ui

import (
	"path/filepath"

	"github.com/ssn688/sim/internal/campaign"
	"github.com/ssn688/sim/internal/config"
	"github.com/ssn688/sim/internal/save"
	"github.com/ssn688/sim/internal/sim"
)

func (a *App) continueScenario() {
	sc := a.selectedScenarioDef()
	if sc == nil {
		a.StatusMessage = "No scenario selected."
		return
	}
	path, err := campaign.LatestSaveForScenario(sc.ID)
	if err != nil {
		a.StatusMessage = "No save found for this scenario."
		return
	}
	engine, err := save.LoadClean(path)
	if err != nil {
		a.StatusMessage = "Load failed: " + err.Error()
		return
	}
	if engine.Clock.GameTime > 0.01 {
		engine.Campaign.BetweenMissions = false
	}
	a.beginGameSession(engine)
}

func (a *App) restartScenarioConfirmed() {
	sc := a.selectedScenarioDef()
	if sc == nil {
		return
	}
	_ = campaign.DeleteScenarioSaves(sc.ID)
	a.resetScenarioLoadout()
	a.Mode = ModeScenarioBrief
	a.initScenarioBrief()
	a.StatusMessage = "Scenario progress cleared."
}

func (a *App) startSelectedMission() {
	sc := a.selectedScenarioDef()
	if sc == nil {
		return
	}
	prog := a.scenarioProgress(sc.ID)
	m := prog.CurrentMission(sc)
	if m == nil {
		a.StatusMessage = "Scenario already complete."
		return
	}
	ctx := campaign.BuildContextFromProgress(prog)
	runtime := campaign.BuildMission(sc.ID, m.ID, ctx)
	if runtime == nil {
		a.StatusMessage = "Failed to build mission."
		return
	}
	engine := sim.NewEngine(runtime)
	engine.Campaign = campaign.RuntimeMeta{
		ScenarioID:      sc.ID,
		MissionID:       m.ID,
		MissionHash:     campaign.MissionHash(*m),
		LoadoutMix:      a.LoadoutMix,
		BetweenMissions: true,
		Completed:       prog.CompletedMissions,
		Vars:            prog.Vars,
	}
	engine.Campaign.BetweenMissions = false
	a.ensureLoadoutTubes()
	campaign.ApplyTubeLoadout(&engine.FireControl, a.LoadoutTubes, a.LoadoutMix)
	a.beginGameSession(engine)
}

func (a *App) exitToMenuConfirmed() {
	a.exitToMenu()
}

func (a *App) endMissionConfirmed() {
	if a.Engine == nil {
		return
	}
	scDef := campaign.ScenarioByID(a.Engine.Campaign.ScenarioID)
	if scDef == nil {
		a.exitToMenu()
		return
	}
	missionID := a.Engine.Campaign.MissionID
	primaryOK := a.Engine.Scenario.PrimaryObjectivesComplete()
	meta := a.Engine.Campaign
	if meta.Completed == nil {
		meta.Completed = map[campaign.MissionID]bool{}
	}
	if meta.Vars == nil {
		meta.Vars = map[string]string{}
	}
	meta.Completed[missionID] = true
	campaign.MergeVars(meta.Vars, campaign.ResolveMissionOutputs(scDef, missionID, primaryOK))

	next := campaign.Progress{
		ScenarioID:        meta.ScenarioID,
		CompletedMissions: meta.Completed,
		Vars:              meta.Vars,
		LoadoutMix:        meta.LoadoutMix,
	}
	nextMission := next.CurrentMission(scDef)

	if nextMission == nil {
		a.saveCampaignAutosave(meta, nil, true)
		a.releaseSessionCaches()
		a.Engine = nil
		a.Mode = ModeScenarioList
		a.SelectedScenarioID = scDef.ID
		a.StatusMessage = "Scenario complete — progress saved."
		return
	}

	ctx := campaign.BuildContextFromProgress(next)
	runtime := campaign.BuildMission(scDef.ID, nextMission.ID, ctx)
	if runtime == nil {
		a.StatusMessage = "Failed to prepare next mission."
		return
	}
	engine := sim.NewEngine(runtime)
	meta.MissionID = nextMission.ID
	meta.MissionHash = campaign.MissionHash(*nextMission)
	meta.BetweenMissions = true
	meta.ReportEligible = false
	engine.Campaign = meta
	campaign.ApplyPlayerLoadout(&engine.FireControl, meta.LoadoutMix)
	a.saveCampaignAutosave(meta, engine, false)
	a.releaseSessionCaches()
	a.Engine = nil
	a.Mode = ModeScenarioBrief
	a.SelectedScenarioID = scDef.ID
	a.initScenarioBrief()
	a.StatusMessage = "Mission complete — autosaved for next mission."
}

func (a *App) saveCampaignAutosave(prev campaign.RuntimeMeta, nextEngine *sim.Engine, scenarioDone bool) {
	dir, err := config.SavesDir()
	if err != nil {
		return
	}
	path := filepath.Join(dir, campaign.AutosaveName(prev.ScenarioID))
	if nextEngine != nil {
		nextEngine.Campaign = prev
		nextEngine.Campaign.BetweenMissions = true
		nextEngine.Campaign.ReportEligible = false
		_ = save.Save(path, nextEngine)
		return
	}
	if scDef := campaign.ScenarioByID(prev.ScenarioID); scDef != nil && len(scDef.Missions) > 0 {
		last := scDef.Missions[len(scDef.Missions)-1]
		ctx := campaign.BuildContextFromProgress(prev.ToProgress())
		runtime := campaign.BuildMission(prev.ScenarioID, last.ID, ctx)
		if runtime == nil {
			return
		}
		engine := sim.NewEngine(runtime)
		engine.Campaign = prev
		engine.Campaign.MissionID = last.ID
		engine.Campaign.MissionHash = campaign.MissionHash(last)
		engine.Campaign.BetweenMissions = true
		engine.Campaign.ReportEligible = false
		if scenarioDone {
			for _, m := range scDef.Missions {
				engine.Campaign.Completed[m.ID] = true
			}
		}
		_ = save.Save(path, engine)
	}
}

func (a *App) missionEndEligible() bool {
	if a.Engine == nil || a.Engine.Scenario == nil {
		return false
	}
	return a.Engine.Campaign.ReportEligible && a.Engine.Scenario.PrimaryObjectivesComplete()
}
