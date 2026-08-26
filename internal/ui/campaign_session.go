package ui

import (
	"path/filepath"

	"github.com/ssn688/sim/internal/campaign"
	"github.com/ssn688/sim/internal/config"
	"github.com/ssn688/sim/internal/i18n"
	"github.com/ssn688/sim/internal/save"
	"github.com/ssn688/sim/internal/sim"
)

func (a *App) continueScenario() {
	sc := a.selectedScenarioDef()
	if sc == nil || !sc.Compatible {
		a.Status(i18n.StatusScenarioIncompat)
		return
	}
	path, err := campaign.LatestSaveForScenario(sc.ID)
	if err != nil {
		a.Status(i18n.StatusNoSaveForScenario)
		return
	}
	engine, err := save.LoadClean(path)
	if err != nil {
		a.Statusf(i18n.StatusLoadFailed, err.Error())
		return
	}
	if engine.Campaign.BetweenMissions {
		a.briefDebrief = false
		a.briefMissionID = ""
		a.Mode = ModeScenarioBrief
		a.initScenarioBrief()
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
	a.briefDebrief = false
	a.briefMissionID = ""
	a.Mode = ModeScenarioBrief
	a.initScenarioBrief()
	a.Status(i18n.StatusProgressCleared)
}

func (a *App) startSelectedMission() {
	if a.briefDebrief || !a.selectedScenarioPlayable() {
		return
	}
	sc := a.selectedScenarioDef()
	if sc == nil {
		return
	}
	prog := a.scenarioProgress(sc.ID)
	m := a.briefDisplayedMission(sc)
	if m == nil || prog.CompletedMissions[m.ID] {
		m = prog.CurrentMission(sc)
	}
	if m == nil || prog.CompletedMissions[m.ID] || prog.ScenarioComplete(sc) {
		a.Status(i18n.StatusScenarioComplete)
		return
	}
	ctx := campaign.BuildContextFromProgress(prog)
	runtime := campaign.BuildMission(sc.ID, m.ID, ctx)
	if runtime == nil {
		a.Status(i18n.StatusBuildMissionFail)
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
	campaign.ApplyUnitPayloads(&engine.FireControl, m, prog.Vars)
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
	meta.DebriefPending = true
	meta.DebriefMission = missionID
	meta.DebriefOutcomes = campaign.SnapshotObjectiveOutcomes(a.Engine.Scenario)
	meta.BetweenMissions = true
	meta.ReportEligible = false
	campaign.MergeVars(meta.Vars, campaign.ResolveMissionOutputs(scDef, missionID, primaryOK, meta.DebriefOutcomes))

	a.Engine.Campaign = meta
	a.saveCampaignAutosave(meta, a.Engine, meta.ToProgress().ScenarioComplete(scDef))
	a.releaseSessionCaches()
	a.Engine = nil
	a.Mode = ModeScenarioBrief
	a.SelectedScenarioID = scDef.ID
	a.briefDebrief = true
	a.briefMissionID = missionID
	a.initScenarioBrief()
	a.StatusMessage = ""
}

func (a *App) acknowledgeDebriefAndSelectNext() {
	sc := a.selectedScenarioDef()
	if sc == nil {
		return
	}
	next := campaign.NextMission(sc, a.briefMissionID)
	if next == nil {
		return
	}
	path, err := campaign.LatestSaveForScenario(sc.ID)
	if err == nil {
		if engine, loadErr := save.LoadClean(path); loadErr == nil {
			engine.Campaign.DebriefPending = false
			engine.Campaign.BetweenMissions = true
			_ = save.Save(path, engine)
		}
	}
	a.briefDebrief = false
	a.briefMissionID = next.ID
	a.scenarioBriefDescScroll = 0
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
