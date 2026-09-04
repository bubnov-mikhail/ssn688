package ui

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/audio"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/config"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/layout"
	"github.com/bubnov-mikhail/ssn688/internal/platform"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/sim"
	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

type AppMode int

const (
	ModeMenu AppMode = iota
	ModeGame
	ModeSettings
	ModeLoad
	ModePaused
	ModeScenarioList
	ModeScenarioBrief
)

type Screen int

const (
	ScreenPassive Screen = iota
	ScreenActive
	ScreenSpectrum
	ScreenLibrary
	ScreenFireControl
	ScreenManeuver
	ScreenTactical
	ScreenDamage
	ScreenMast // extensible masts: radio, ESM, radar, periscope
)

type App struct {
	Mode                   AppMode
	Settings               config.Settings
	Engine                 *sim.Engine
	Audio                  *audio.Manager
	CurrentScreen          Screen
	MenuIndex              int
	LoadFiles              []string
	LoadIndex              int
	StatusMessage          string
	bearingWaterfalls      BearingWaterfallBank
	lastWaterfallSample    float64
	waterfallScratch       []float64
	waterfallAltCounter    int
	waterfallImg           *ebiten.Image
	waterfallPix           []byte
	waterfallRng           *rand.Rand
	waterfallStamp         float64
	waterfallArray         acoustics.PassiveArrayKind
	waterfallPendingScroll bool
	waterfallFullRebuild   bool
	passivePPI             *ebiten.Image
	passivePPIPixels       []byte
	passivePPIStamp        float64
	passivePPIArray        acoustics.PassiveArrayKind
	passivePPIPending      bool
	ppiEnergies            []float64
	ppiSmoothed            []float64
	ppiFloorN              []float64
	ppiGrainN              []float64
	ppiSens                []float64
	ppiLUT                 []ppiPixelLUT
	ppiLUTSize             int
	spectrumFuzzyImg       *ebiten.Image
	spectrumFuzzyPix       []byte
	spectrumFuzzyStamp     int64
	spectrumFuzzyKey       float64
	spectrumCacheBins      []float64
	spectrumCacheAt        float64
	spectrumCacheBearing   float64
	spectrumFuzzyLevels    []float64
	waterfallChipCache     []contactChip
	waterfallChipCacheKey  uint64
	sonarBtnScratch        []sonarUIButton
	enemyPingHeardAt       map[string]float64
	blastHeardAt           float64 // LastBlastAt already played as explosion SFX
	lastPingPlayed         float64
	uiHoverID              string
	uiHoverSince           time.Time
	uiTooltip              string
	uiPressedID            string
	uiPressedAt            time.Time
	navHoverIdx            int
	navHoverSince          time.Time
	navTooltip             string
	lastUpdateWall         time.Time
	activeSliderDrag       string
	activeEchoFlashes      []activeEchoFlash
	activeEchoFlashSeq     uint64
	activeRangeScaleYd     float64
	activePlotImg          *ebiten.Image
	activePlotPix          []byte
	activePlotGridPix      []byte
	activePlotGridScaleYd  float64
	activePlotGridDirty    bool
	compassDrag            bool
	selectedContactID      string
	selectedPlotMarkerID   string
	pendingPlotMarker      bool            // M pressed while already on PLOT
	reportedTorpedoIDs     map[string]bool // hostile fish already announced by WEPS
	ownTorpedoIDs          map[string]bool // player Mk48 IDs (alive or spent — suppress hostile alert)
	torpedoThreatActive    map[string]bool // torpedoes currently assessed as threatening ownship
	referenceProfileIdx    int
	contactTableScroll     contactTableScrollState
	mastCommScroll         int
	periImg                *ebiten.Image
	periPix                []byte // composited frame (bg + ships/FX)
	periBgPix              []byte // cached sky/sea/land plate
	periBgCacheKey         uint64
	periMarkerHits         []contactChip
	periShipScratch        []periShipDraw
	periLandHit            []float64
	periLandElev           []float64
	periLandOK             []bool
	periLandHitTmp         []float64
	periLandElevTmp        []float64
	periLandOKTmp          []bool
	periDepth              []float32 // per-pixel closest ship range (0 = empty)
	librarySelectedID      string
	libraryCatalogScroll   int
	libraryDetailScroll    int
	layerSurveyWasActive   bool
	tactical               tacticalState
	wepsMapZoom            float64
	wepsMapImg             *ebiten.Image
	wepsOrdnanceMenuTube   int // 0 = closed; 1–4 open dropdown

	// Ownship hit feedback (wall-clock; cheap overlays, no steady-state cost).
	hitVignetteAt time.Time
	hitShakeAt    time.Time
	hitShakeBuf   *ebiten.Image
	dcTabAlert    bool // blink DC nav until player opens Damage Control
	mastTabAlert  bool // blink MAST nav while due COMM traffic is undelivered

	SelectedScenarioID      campaign.ScenarioID
	ScenarioListIndex       int
	scenarioBackstoryScroll int
	scenarioBriefDescScroll int
	scenarioBriefMDKey      string
	scenarioBriefMDLines    []render.MDLine
	scenarioListMDKey       string
	scenarioListMDLines     []render.MDLine
	briefMapCacheKey        string
	briefProgressID         campaign.ScenarioID
	briefProgress           campaign.Progress
	briefProgressOK         bool
	scenarioListDetailKey   string
	scenarioListDetailImg   *ebiten.Image
	scenarioBriefDetailKey  string
	scenarioBriefDetailImg   *ebiten.Image
	scenarioScreenActive     bool
	scenarioUIDirty          bool
	scenarioUIHoverKey       string
	briefDebrief            bool
	briefMissionID          campaign.MissionID
	LoadoutMix              float64
	LoadoutTubes            campaign.TubeLoadout
	loadoutDragging         bool
	loadoutOrdnanceMenuTube int
	confirm                 confirmDialog
}

func NewApp(settings config.Settings, audioMgr *audio.Manager) *App {
	settings.Language = i18n.NormalizeLang(settings.Language)
	i18n.SetLang(settings.Language)
	return &App{
		Mode:                ModeMenu,
		Settings:            settings,
		Audio:               audioMgr,
		CurrentScreen:       ScreenPassive,
		MenuIndex:           0,
		enemyPingHeardAt:    map[string]float64{},
		reportedTorpedoIDs:  map[string]bool{},
		ownTorpedoIDs:       map[string]bool{},
		torpedoThreatActive: map[string]bool{},
		activeRangeScaleYd:  12000,
		wepsMapZoom:         0.05,
		LoadoutMix:          0.25,
	}
}

func (a *App) Update() error {
	switch a.Mode {
	case ModeMenu:
		return a.updateMenu()
	case ModeSettings:
		a.updateSettings()
	case ModeLoad:
		a.updateLoad()
	case ModeScenarioList:
		return a.updateScenarioList()
	case ModeScenarioBrief:
		return a.updateScenarioBrief()
	case ModeGame, ModePaused:
		if a.confirmActive() {
			a.updateConfirmDialog()
			return nil
		}
		a.updateGame()
	}
	return nil
}

func (a *App) refreshLoadList() {
	dir, err := config.SavesDir()
	if err != nil {
		a.LoadFiles = nil
		return
	}
	entries, _ := os.ReadDir(dir)
	var files []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sav" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if meta, ok := campaign.ReadSaveCampaignMeta(path); ok {
			sc := campaign.ScenarioByID(meta.ScenarioID)
			if sc == nil || !sc.Compatible {
				continue
			}
		}
		files = append(files, path)
	}
	sort.Strings(files)
	a.LoadFiles = files
}

func (a *App) updateGame() {
	if a.Engine == nil {
		return
	}
	realDT := 1.0 / 60.0
	nowWall := time.Now()
	if !a.lastUpdateWall.IsZero() {
		realDT = nowWall.Sub(a.lastUpdateWall).Seconds()
		if realDT < 0 {
			realDT = 0
		}
		// Cap spike length so a tab-switch doesn't dump minutes of sim at once,
		// but keep enough headroom that hitchy Draws don't starve GameTime at 1x.
		if realDT > 0.25 {
			realDT = 0.25
		}
	}
	a.lastUpdateWall = nowWall

	if a.handleHeaderButtons() {
		return
	}

	a.updateNavBar()

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if a.Mode == ModePaused {
			a.Mode = ModeGame
		} else {
			a.Mode = ModePaused
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		a.Engine.Clock.TogglePause()
		a.Audio.PlayClip(audio.ClipCaptHoldSimulation, "")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
		a.cycleSpeedUp()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
		a.cycleSpeedDown()
	}

	// Screen selection (F-keys to avoid conflict with fire control tubes).
	// M opens PLOT from other screens; on PLOT, M places a chart marker.
	screenKeys := []ebiten.Key{ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF4, ebiten.KeyF5, ebiten.KeyF6, ebiten.KeyM, ebiten.KeyF7, ebiten.KeyF8}
	for i, k := range screenKeys {
		if inpututil.IsKeyJustPressed(k) {
			switch k {
			case ebiten.KeyM:
				if a.CurrentScreen != ScreenTactical {
					a.CurrentScreen = ScreenTactical
				} else {
					a.pendingPlotMarker = true
				}
			case ebiten.KeyF7:
				a.CurrentScreen = ScreenDamage
			case ebiten.KeyF8:
				a.CurrentScreen = ScreenMast
			default:
				a.CurrentScreen = Screen(i)
			}
		}
	}
	a.clearDCTabAlertIfOnDamage()

	a.tryDebugPeriAccidentHit()
	a.handleScreenInput()

	if a.Mode != ModePaused {
		a.Engine.Update(realDT)
		a.updateSimulationUI()
	}

	if a.Engine != nil {
		env := &a.Engine.Acoustics.Env
		active := env.LayerSurveyActive(a.Engine.Clock.GameTime)
		if a.layerSurveyWasActive && !active && env.LayerSurveyKnown {
			a.Status(i18n.StatusBTCastComplete)
			a.Audio.PlayClip(audio.ClipSonarLayerSurveyComplete, a.L(i18n.VoiceLayerSurveyComplete))
		}
		a.layerSurveyWasActive = active
	}

	if a.CurrentScreen != ScreenManeuver && a.CurrentScreen != ScreenPassive && a.CurrentScreen != ScreenSpectrum && a.CurrentScreen != ScreenTactical && a.CurrentScreen != ScreenDamage {
		a.uiTooltip = ""
		a.uiHoverID = ""
	}
	if a.navHoverIdx < 0 {
		a.navTooltip = ""
	}
	// Own-ship quick panel (replaces OBJECTIVES) on every screen except full HELM.
	if a.Engine != nil && a.Engine.Scenario != nil {
		a.updateOwnShipPanel(a.Engine.Scenario.Player)
	}
	a.updateCMQuickPanel()

	for _, ev := range a.Engine.PopEvents() {
		a.StatusMessage = ev
		if isWeaponImpactEvent(ev) {
			sub := ""
			if strings.HasPrefix(ev, "Torpedo struck bottom") {
				sub = i18n.LocalizeRuntimeMessage("Torpedo impact.", a.Lang())
			}
			a.Audio.PlayClip(audio.ClipWepsImpactConfirmed, sub)
		}
		if strings.HasPrefix(ev, "AUTO-RETRACT") {
			switch {
			case strings.Contains(ev, "towed array") && strings.Contains(ev, "masts"):
				a.Audio.PlayClip(audio.ClipSonarRetractTowed, a.L(i18n.StatusAutoRetractBoth))
			case strings.Contains(ev, "towed array"):
				a.Audio.PlayClip(audio.ClipSonarRetractTowed, a.L(i18n.StatusAutoRetractTowed))
			default:
				a.Audio.PlayClip(audio.ClipDiveMakeDepth, a.L(i18n.StatusAutoLowerMasts))
			}
		} else if strings.Contains(ev, "RBU barrage") {
			a.Audio.PlayClip(audio.ClipWepsImpactConfirmed, a.L(i18n.StatusRBUBarrage))
		} else if strings.HasPrefix(ev, "Contact ") && strings.Contains(ev, " identified:") {
			a.Audio.PlayClip(audio.ClipSonarContactClassified, i18n.LocalizeRuntimeMessage(ev, a.Lang()))
		}
		a.playOwnshipCasualtyVoice(ev)
		if isOwnshipDamageFXEvent(ev) {
			a.triggerOwnshipHitFX()
		}
		if ev == "Torpedo launch detected (hostile)" {
			a.Audio.PlayClip(audio.ClipWepsTorpedoInWater, a.L(i18n.VoiceTorpedoInWater))
			a.markLatestHostileTorpedoReported()
		}
	}
	a.pollHostileTorpedoAlerts()
	a.pollTorpedoCollisionAlerts()
	if a.Engine.ESM.ChirpPending {
		a.Engine.ESM.ChirpPending = false
		if a.CurrentScreen == ScreenMast && a.Audio != nil {
			a.Audio.PlayESMHit()
		}
	}
	if a.Engine.COMM.TrafficWaitingNotify {
		a.Engine.COMM.TrafficWaitingNotify = false
		a.StatusMessage = a.L(i18n.UIStatusHFTrafficPending)
		if a.Audio != nil {
			a.Audio.PlayClip(audio.ClipCaptCommTrafficWaiting, a.L(i18n.VoiceCommTrafficWaiting))
		}
	}
	a.mastTabAlert = a.Engine.COMM.TrafficWaiting
	if a.Engine.COMM.NotifyPending {
		a.Engine.COMM.NotifyPending = false
		a.StatusMessage = a.L(i18n.UIStatusFlashTraffic)
		if a.Audio != nil {
			a.Audio.PlayClip(audio.ClipCaptCommMessage, a.L(i18n.VoiceCommMessage))
		}
		// Keep newest traffic visible in the scroll pane.
		a.mastCommScroll = 1 << 20
	}

	if a.Engine.Scenario.MissionComplete() {
		a.StatusMessage = a.L(i18n.UIStatusMissionComplete)
	}
	if a.Engine.Scenario.MissionFailed() {
		msg := a.Engine.Scenario.FailReason
		if msg == "" {
			msg = a.L(i18n.UIStatusMissionFailedBare)
		}
		a.StatusMessage = a.L(i18n.UIStatusMissionFailed) + msg
	}

	// Active ping sound (same loud echolocator sample as enemy pings).
	if a.Engine.Sonar.LastPingTime > a.lastPingPlayed {
		a.Audio.PlayEnemyPing()
		a.lastPingPlayed = a.Engine.Sonar.LastPingTime
	}
	a.updateEnemyPingAudio()
	a.updateBlastExplosionAudio()
	a.updateContactPropellerAudio()

	if inpututil.IsKeyJustPressed(ebiten.KeyS) && ebiten.IsKeyPressed(ebiten.KeyControl) {
		a.quickSave()
	}
}

// updateContactPropellerAudio loops hydrophone listen FX while the player is
// on PASSIVE. Ambient sea noise always plays on that screen; contact propeller
// / bow wash / torpedo run layer on top when a moving contact is selected.
// Contact track loudness tracks sonar SNR (waterfall brightness).
// On HELM, a dedicated ownship propulsion loop (sub+combatant mix) plays with
// gain ramping 0→50% from 0.1→8 kn; stretch peaks at HelmPropellerRefSpeedKts.
func (a *App) updateContactPropellerAudio() {
	if a.Audio == nil {
		return
	}
	ambient := 0.0
	gCombatant := 0.0
	gFishing := 0.0
	gMerchant := 0.0
	gTanker := 0.0
	subProp := 0.0
	bowGain := 0.0
	torpRun := 0.0
	helmProp := 0.0
	propSpeed := 1.0
	bowSpeed := 1.0
	torpSpeed := 1.0
	helmSpeed := 1.0
	live := a.Mode == ModeGame && a.Engine != nil && !a.Engine.Clock.Paused && a.Engine.Scenario != nil
	onPassive := live && a.CurrentScreen == ScreenPassive
	onHelm := live && a.CurrentScreen == ScreenManeuver
	if onPassive {
		ambient = 0.32 // under contact tracks; present with no selection
		c := a.selectedContact(&a.Engine.Sonar)
		if c != nil && c.SourceEntityID != "" {
			var ent *world.Entity
			for _, e := range a.Engine.AcousticEmitters() {
				if e != nil && e.ID == c.SourceEntityID {
					ent = e
					break
				}
			}
			if ent != nil && ent.InWater() {
				gain := listenGainFromContactSNR(c.SNR)
				switch {
				case ent.Kind == world.KindTorpedo || c.Kind == world.KindTorpedo:
					if ent.SpeedKts >= 0.5 {
						torpRun = gain
						torpSpeed = audio.TorpedoListenSpeed(ent.SpeedKts)
					}
				case ent.Kind == world.KindSurfaceShip:
					if ent.SpeedKts >= audio.PropellerMinSpeedKts {
						propSpeed = audio.PropellerListenSpeed(ent.SpeedKts, ent.SignatureID)
						bowSpeed = propSpeed
						switch ent.SignatureID {
						case "fishing":
							gFishing = gain
						case "merchant":
							gMerchant = gain
						case "tanker":
							gTanker = gain
						default:
							gCombatant = gain
						}
						if beam := world.SurfaceHullBeamRel(ent); beam > 0 {
							// Bow wash under propeller (~18–32% of prop gain by hull beam).
							bowGain = gain * (0.18 + 0.14*beam)
						}
					}
				case ent.Kind == world.KindSubmarine:
					if ent.SpeedKts >= audio.PropellerMinSpeedKts {
						subProp = gain
						propSpeed = audio.PropellerListenSpeed(ent.SpeedKts, ent.SignatureID)
					}
				}
			}
		}
	}
	if onHelm {
		if p := a.Engine.Scenario.Player; p != nil {
			spd := math.Abs(p.SpeedKts)
			helmProp = audio.HelmPropellerGain(spd)
			if helmProp > 0 {
				helmSpeed = audio.HelmPropellerListenSpeed(spd)
			}
		}
	}
	a.Audio.SetLoopingFX(audio.FXPassiveAmbient, ambient, 1)
	a.Audio.SetLoopingFX(audio.FXPropellerHydrophone, gCombatant, propSpeed)
	a.Audio.SetLoopingFX(audio.FXPropellerFishing, gFishing, propSpeed)
	a.Audio.SetLoopingFX(audio.FXPropellerMerchant, gMerchant, propSpeed)
	a.Audio.SetLoopingFX(audio.FXPropellerTanker, gTanker, propSpeed)
	a.Audio.SetLoopingFX(audio.FXPropellerSubmarine, subProp, propSpeed)
	a.Audio.SetLoopingFX(audio.FXPropellerHelm, helmProp, helmSpeed)
	a.Audio.SetLoopingFX(audio.FXBowWash, bowGain, bowSpeed)
	a.Audio.SetLoopingFX(audio.FXTorpedoRun, torpRun, torpSpeed)
}

// listenGainFromContactSNR maps track PeakSNR to phone loudness so faint
// waterfall traces whisper and hot red lines dominate.
func listenGainFromContactSNR(snr float64) float64 {
	clarity := acoustics.SpectrumClarity01(snr)
	// Floor keeps a barely-held contact audible when selected; ceiling at 1.
	g := 0.08 + 0.92*clarity
	if g > 1 {
		return 1
	}
	return g
}

// updateSimulationUI runs lightweight background UI state that must stay warm
// while the player is on another screen (smoothed positions, waterfall history, etc.).
func (a *App) updateSimulationUI() {
	a.ensureTactical()
	a.updateSmoothedContactPositions()
	a.updateBearingWaterfall()
}

func (a *App) cycleSpeedUp() {
	scales := []float64{0.5, 1, 2, 4, 8}
	cur := a.Engine.Clock.TimeScale
	for i, s := range scales {
		if math.Abs(s-cur) < 0.01 && i < len(scales)-1 {
			a.Engine.Clock.TimeScale = scales[i+1]
			a.Engine.Clock.Paused = false
			a.Audio.PlayClip(navSpeedClip(scales[i+1]), "")
			return
		}
	}
}

func (a *App) updateEnemyPingAudio() {
	if a.Engine == nil || a.Audio == nil || a.Engine.Scenario == nil || a.Engine.Scenario.Player == nil {
		return
	}
	player := a.Engine.Scenario.Player
	now := a.Engine.Clock.GameTime
	for _, ent := range a.Engine.Scenario.Entities {
		if ent == nil || !ent.Alive() || ent.LastPingTime <= 0 || now-ent.LastPingTime > 12 {
			continue
		}
		heardAt, ok := a.enemyPingHeardAt[ent.ID]
		if ok && heardAt >= ent.LastPingTime {
			continue
		}
		delay := ent.RangeYardsTo(player) / acoustics.SoundSpeedYdPerSec
		if now < ent.LastPingTime+delay {
			continue
		}
		a.Audio.PlayEnemyPing()
		a.enemyPingHeardAt[ent.ID] = ent.LastPingTime
	}
}

// updateBlastExplosionAudio plays the underwater blast one-shot on any screen
// when the acoustic wave arrives (LastBlastAt already includes travel delay).
func (a *App) updateBlastExplosionAudio() {
	if a.Engine == nil || a.Audio == nil || a.Mode != ModeGame {
		return
	}
	if a.Engine.Clock.Paused {
		return
	}
	sonar := &a.Engine.Sonar
	arrive := sonar.LastBlastAt
	if arrive <= 0 || arrive <= a.blastHeardAt {
		return
	}
	now := a.Engine.Clock.GameTime
	if now < arrive {
		return
	}
	player := a.Engine.Scenario.Player
	gain := 0.85
	if player != nil {
		dist := math.Hypot(player.X-sonar.LastBlastX, player.Y-sonar.LastBlastY)
		// Same hear bubble as blast transient (~12 kyd).
		if dist > 12000 {
			a.blastHeardAt = arrive
			return
		}
		gain = 0.35 + 0.65*(1-dist/12000)
		if gain > 1 {
			gain = 1
		}
	}
	a.Audio.PlayUnderwaterExplosion(gain)
	a.blastHeardAt = arrive
}

func (a *App) cycleSpeedDown() {
	scales := []float64{0.5, 1, 2, 4, 8}
	cur := a.Engine.Clock.TimeScale
	for i, s := range scales {
		if math.Abs(s-cur) < 0.01 && i > 0 {
			a.Engine.Clock.TimeScale = scales[i-1]
			a.Engine.Clock.Paused = false
			a.Audio.PlayClip(navSpeedClip(scales[i-1]), "")
			return
		}
	}
}

func navSpeedClip(scale float64) audio.ClipID {
	switch scale {
	case 0.5:
		return audio.ClipNavSpeedHalf
	case 1:
		return audio.ClipNavSpeedNormal
	case 2:
		return audio.ClipNavSpeedDouble
	case 4:
		return audio.ClipNavSpeedQuad
	case 8:
		return audio.ClipNavSpeedEight
	default:
		return audio.ClipNavSpeedNormal
	}
}

// tryDebugPeriAccidentHit: Debug-only undocumented hotkey H — onboard explosion on
// the peri-locked surface ship (no DEFCON escalate). Steals H from sonar/WEPS when armed.
func (a *App) tryDebugPeriAccidentHit() {
	if a.Mode != ModeGame && a.Mode != ModePaused {
		return
	}
	if !a.Settings.Debug || a.Engine == nil {
		return
	}
	if !inpututil.IsKeyJustPressed(ebiten.KeyH) {
		return
	}
	peri := &a.Engine.Periscope
	if !peri.Locked() || peri.LockEntityID == "" {
		return
	}
	if a.Engine.DebugAccidentHit(peri.LockEntityID) {
		a.StatusMessage = "DEBUG: onboard explosion (accident)"
	}
}

// debugPeriHitStealsH is true when Debug peri-lock would claim H this frame.
func (a *App) debugPeriHitStealsH() bool {
	return a.Settings.Debug && a.Engine != nil && a.Engine.Periscope.Locked()
}

func (a *App) handleScreenInput() {
	player := a.Engine.Scenario.Player
	fc := &a.Engine.FireControl
	sonar := &a.Engine.Sonar

	switch a.CurrentScreen {
	case ScreenPassive:
		a.updateSonarUIButtons(a.cachedPassiveArrayButtons(), sonar)
		a.updateSonarUIButtons(a.cachedPassiveBandButtons(), sonar)
		a.updateSonarUIButtons(a.cachedPassiveTowedButtons(), sonar)
		a.handleSonarArrayKeys(sonar, true)
		a.updatePassiveScreen(sonar)
		if inpututil.IsKeyJustPressed(ebiten.KeyP) {
			sonar.PassiveEnabled = !sonar.PassiveEnabled
			if sonar.PassiveEnabled {
				a.Audio.PlayClip(audio.ClipSonarPassiveOn, "")
			} else {
				a.Audio.PlayClip(audio.ClipSonarPassiveOff, "")
			}
		}
	case ScreenActive:
		a.updateActiveScreen(sonar)
	case ScreenSpectrum:
		a.updateSonarUIButtons(a.cachedSpectrumArrayButtons(), sonar)
		a.handleSonarArrayKeys(sonar, false)
		a.updateSpectrumScreen(sonar)
		if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
			a.selectedContactID = ""
			sonar.SpectrumBearing -= 1
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
			a.selectedContactID = ""
			sonar.SpectrumBearing += 1
		}
	case ScreenLibrary:
		a.updateLibraryInput()
	case ScreenFireControl:
		a.handleFireControl(fc, player)
	case ScreenManeuver:
		a.updateManeuverUI(player)
	case ScreenTactical:
		a.updateTacticalUI()
	case ScreenDamage:
		a.updateDamageUI()
	case ScreenMast:
		a.updateMastUI()
	}
}

func (a *App) Draw(screen *ebiten.Image) {
	switch a.Mode {
	case ModeMenu:
		a.drawMenu(screen)
	case ModeSettings:
		a.drawSettings(screen)
	case ModeLoad:
		a.drawLoad(screen)
	case ModeScenarioList:
		a.drawScenarioList(screen)
	case ModeScenarioBrief:
		a.drawScenarioBrief(screen)
	case ModeGame, ModePaused:
		screen.Fill(render.ColorBG)
		a.drawGameWithHitFX(screen)
	}
}

func (a *App) drawGame(screen *ebiten.Image) {
	render.DrawConsoleBackdrop(screen)

	switch a.CurrentScreen {
	case ScreenPassive:
		a.drawPassive(screen)
	case ScreenActive:
		a.drawActive(screen)
	case ScreenSpectrum:
		a.drawSpectrum(screen)
	case ScreenLibrary:
		a.drawLibrary(screen)
	case ScreenFireControl:
		a.drawFireControl(screen)
	case ScreenManeuver:
		a.drawManeuver(screen)
	case ScreenTactical:
		a.drawTactical(screen)
	case ScreenDamage:
		a.drawDamage(screen)
	case ScreenMast:
		a.drawMast(screen)
	default:
		render.DrawConsoleBackdrop(screen)
	}

	a.drawGameHeader(screen)
	a.drawOwnShipPanel(screen)
	a.drawCMQuickPanel(screen)

	if a.StatusMessage != "" {
		render.DrawText(screen, a.displayStatus(), 20, render.ScreenH-navBarH-18, render.ColorWarn, false)
	}
	if sub, ok := a.Audio.Subtitle(); ok {
		render.FillRect(screen, 200, render.ScreenH-navBarH-50, render.ScreenW-400, 36, render.ColorPanelDark)
		render.DrawText(screen, sub, 220, render.ScreenH-navBarH-28, render.ColorHighlight, false)
	}

	if a.Mode == ModePaused {
		render.DrawTextLarge(screen, a.L(i18n.UIPaused), 720, 420, render.ColorDanger)
	}
	if a.confirmActive() {
		a.drawConfirmDialog(screen)
	}
	if a.Engine != nil && a.Engine.Scenario != nil && a.Engine.Scenario.Player != nil {
		if a.Engine.Scenario.Player.Status == world.StatusSinking {
			render.DrawTextLarge(screen, a.L(i18n.UIOwnshipSinking), 520, 460, render.ColorDanger)
		} else if a.Engine.Scenario.Player.Status == world.StatusSunk {
			render.DrawTextLarge(screen, a.L(i18n.UIOwnshipLost), 650, 460, render.ColorDanger)
		}
	}

	a.drawTacticalMinimap(screen)
	a.drawNavBar(screen)
}

func (a *App) drawGameHeader(screen *ebiten.Image) {
	render.FillRect(screen, 0, 0, render.ScreenW, gameHeaderH, render.ColorPanelBezel)
	render.FillRect(screen, 0, 2, render.ScreenW, gameHeaderH-2, render.ColorPanelDark)
	render.DrawText(screen, fmt.Sprintf("SSN-688  |  %s %s  |  %s %s  |  %s %s",
		a.L(i18n.UIHeaderTime), a.missionClockLabel(),
		a.L(i18n.UIHeaderSpeed), a.Engine.Clock.SpeedLabel(),
		a.L(i18n.UIHeaderScreen), a.screenName(a.CurrentScreen)), 20, 28, render.ColorText, false)
	a.drawHeaderButtons(screen)
}

func (a *App) missionClockLabel() string {
	if a.Engine == nil || a.Engine.Scenario == nil {
		return formatElapsedClock(0)
	}
	return world.FormatMissionClock(a.Engine.Scenario.StartTimeSec, a.Engine.Clock.GameTime)
}

func (a *App) screenName(s Screen) string {
	switch s {
	case ScreenPassive:
		return a.L(i18n.UINavPassive)
	case ScreenActive:
		return a.L(i18n.UINavActive)
	case ScreenSpectrum:
		return a.L(i18n.UINavSpectrum)
	case ScreenLibrary:
		return a.L(i18n.UINavLibrary)
	case ScreenFireControl:
		return a.L(i18n.UINavWeps)
	case ScreenManeuver:
		return a.L(i18n.UINavHelm)
	case ScreenTactical:
		return a.L(i18n.UINavPlot)
	case ScreenDamage:
		return a.L(i18n.UINavDC)
	case ScreenMast:
		return a.L(i18n.UINavMast)
	default:
		return "?"
	}
}

func formatElapsedClock(sec float64) string {
	return world.FormatMissionClock(0, sec)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

var errQuit = fmt.Errorf("quit")

func IsQuit(err error) bool {
	return err == errQuit
}

// Draw helpers for screens in screens.go - forward declarations used from draw_screens.go

func (a *App) drawPassive(screen *ebiten.Image) {
	sonar := &a.Engine.Sonar
	player := a.Engine.Scenario.Player
	render.DrawPassiveConsole(screen)

	a.drawArraySelector(screen, sonar, layout.PassiveArrayLabelX, layout.PassiveArrayLabelY, a.cachedPassiveArrayButtons())
	a.drawListenBandSelector(screen, sonar)
	a.drawTowedControls(screen, sonar)
	a.drawPassiveContactTable(screen, sonar)

	a.drawBearingWaterfall(screen, sonar)
	a.drawWaterfallContactChips(screen, sonar)

	// Section legends — light gray engraved labels, no nameplates.
	titleX := layout.PassiveTitleLabelX
	titleY := layout.PassiveTitleLabelY + 20
	render.DrawScreenTitle(screen, a.L(i18n.UITitleWaterfall), titleX, titleY)

	sel := selectedContactLabel(sonar, a.selectedContactID)
	passiveExtra := ""
	if a.Engine.Clock.GameTime < sonar.SonarDeafUntil {
		passiveExtra = fmt.Sprintf("  |  %s %.0fs", a.L(i18n.UISonarBlind), sonar.SonarDeafUntil-a.Engine.Clock.GameTime)
	}
	render.DrawText(screen, fmt.Sprintf("%s: %d  |  %s: %s  |  %s: %s  |  %s: %s%s",
		a.L(i18n.UIContacts), len(sonar.Contacts),
		a.L(i18n.UIArray), a.sonarArrayLabel(sonar),
		a.L(i18n.UISelected), sel,
		a.L(i18n.UILayer), i18n.LocalizeLayerName(a.Engine.Acoustics.Env.LayerNameKnown(player.DepthFt), a.Lang()),
		passiveExtra),
		layout.PassiveStatusTextX, layout.PassiveStatusTextY, render.ColorDim, false)

	selfNoise := acoustics.PassiveSelfNoiseSeverity(sonar.PassiveArray, player.SpeedKts, player.DepthFt, sonar.TowedCablePct)
	noiseY := layout.PassiveSelfNoiseMonitorY + 22
	render.DrawText(screen, a.L(i18n.UISelfNoise), layout.PassiveSelfNoiseLabelX, noiseY, render.ColorPlateLabel, true)
	label := a.L(i18n.UIQuiet)
	if selfNoise > 0.75 {
		label = a.L(i18n.UIDeafening)
	} else if selfNoise > 0.45 {
		label = a.L(i18n.UIFlowNoise)
	} else if selfNoise > 0.15 {
		label = a.L(i18n.UIRising)
	}
	render.DrawText(screen, label, layout.PassiveSelfNoiseStatusX, noiseY, render.ColorAmber, true)
	barX := layout.PassiveSelfNoiseBarX
	barW := layout.PassiveSelfNoiseBarW
	if barW < 80 {
		barW = 80
	}
	render.FillRect(screen, barX, noiseY-12, barW, 14, render.ColorPanelInset)
	if fill := int(float64(barW-4) * selfNoise); fill > 0 {
		clr := render.ColorPhosphor
		if selfNoise > 0.45 {
			clr = render.ColorWarn
		}
		if selfNoise > 0.75 {
			clr = render.ColorDanger
		}
		render.FillRect(screen, barX+2, noiseY-10, fill, 10, clr)
	}

	mx, my := ebiten.CursorPosition()
	if a.uiTooltip != "" {
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
	if !platform.Mobile() {
		render.DrawText(screen, "[P] passive  [B] array  [N] listen band  [U/Y/H] towed  click contact -> spectrum", layout.PassiveHintLabelX, layout.PassiveHintLabelY+12, render.ColorPlateLabel, true)
	}
}

func (a *App) drawActive(screen *ebiten.Image) {
	sonar := &a.Engine.Sonar
	player := a.Engine.Scenario.Player
	render.DrawConsolePanel(screen, activePanelX, activePanelY, activePanelW(), 700)
	render.DrawConsolePanel(screen, activeSideX(), activeSideY, activeSideW, 700)
	render.DrawMonitor(screen, activePlotX, activePlotY, activePlotW(), activePlotH)
	render.DrawScreenTitle(screen, a.L(i18n.UITitleActive), layout.PassiveTitleLabelX, layout.PassiveTitleLabelY+20)
	status := a.L(i18n.UIStandby)
	statusClr := render.ColorPlateLabel
	if sonar.ActiveEnabled {
		status = a.L(i18n.UITransmitReady)
		statusClr = render.ColorActive
	}
	echoYd := a.activeEchoReachYd(sonar)
	echoLabel := "—"
	if echoYd > 0 {
		echoLabel = fmt.Sprintf("%.1f kyd", echoYd/1000)
	}
	render.DrawText(screen, fmt.Sprintf("%s  |  SPD %.1f kts  |  ECHO REACH %s", status, player.SpeedKts, echoLabel), layout.PassiveTitleLabelX, layout.PassiveTitleLabelY+40, statusClr, true)

	a.drawActiveRangeDisplay(screen, sonar)
	a.drawActiveControls(screen, sonar)
	a.drawActiveContactTable(screen, sonar)

	if a.activeSonarDamaged() {
		msg := a.L(i18n.UIActiveDamaged)
		render.DrawText(screen, msg, activePlotX+180, activePlotY+activePlotH/2, render.ColorWarn, false)
	}

	mx, my := ebiten.CursorPosition()
	if a.uiTooltip != "" {
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
	if !platform.Mobile() {
		render.DrawText(screen, "[A] toggle  [F] ping  click plot / table → select contact", 40, 720, render.ColorDim, true)
	}
}

func isWeaponImpactEvent(ev string) bool {
	switch {
	case strings.HasPrefix(ev, "Target destroyed:"):
		return true
	case strings.HasPrefix(ev, "Torpedo struck bottom"):
		return true
	case ev == "Underwater explosion":
		return true
	default:
		return false
	}
}

func (a *App) markOwnTorpedo(id string) {
	if a == nil || id == "" {
		return
	}
	if a.ownTorpedoIDs == nil {
		a.ownTorpedoIDs = map[string]bool{}
	}
	a.ownTorpedoIDs[id] = true
	if a.reportedTorpedoIDs == nil {
		a.reportedTorpedoIDs = map[string]bool{}
	}
	// Own fish blast can linger on sonar after detonation — never announce as hostile.
	a.reportedTorpedoIDs[id] = true
}

func (a *App) syncOwnTorpedoIDs() {
	if a == nil || a.Engine == nil {
		return
	}
	if a.ownTorpedoIDs == nil {
		a.ownTorpedoIDs = map[string]bool{}
	}
	fc := &a.Engine.FireControl
	for _, t := range fc.ActiveTorpedoes {
		if t != nil && t.Side == world.SidePlayer {
			a.markOwnTorpedo(t.ID)
		}
	}
	for i := range fc.Tubes {
		if id := fc.Tubes[i].TorpedoID; id != "" {
			a.markOwnTorpedo(id)
		}
	}
}

func (a *App) playOwnshipCasualtyVoice(ev string) {
	if a == nil || a.Audio == nil {
		return
	}
	switch {
	case strings.HasPrefix(ev, "PLAYER SUBMARINE FATAL DAMAGE"),
		strings.HasPrefix(ev, "PLAYER SUBMARINE LOST"):
		a.Audio.PlayClip(audio.ClipCaptOwnshipLost, a.L(i18n.VoiceOwnshipLost))
	case strings.HasPrefix(ev, "OWN SHIP CRITICAL"):
		a.Audio.PlayClip(audio.ClipCaptCriticalDamage, a.L(i18n.VoiceCriticalDamage))
	case strings.HasPrefix(ev, "OWN SHIP HIT"):
		a.Audio.PlayClip(audio.ClipCaptOwnshipHit, a.L(i18n.VoiceOwnshipHit))
	}
}

func (a *App) pollHostileTorpedoAlerts() {
	if a.Engine == nil {
		return
	}
	if a.reportedTorpedoIDs == nil {
		a.reportedTorpedoIDs = map[string]bool{}
	}
	for i := range a.Engine.Sonar.Contacts {
		c := &a.Engine.Sonar.Contacts[i]
		if c.ConfirmedClass == "" || c.Kind != world.KindTorpedo {
			continue
		}
		id := c.SourceEntityID
		if id == "" {
			id = c.ID
		}
		if a.reportedTorpedoIDs[id] || a.isOwnTorpedoContact(c) {
			continue
		}
		a.reportedTorpedoIDs[id] = true
		a.Statusf(i18n.StatusTorpedoInWaterContact, c.ID)
		a.Audio.PlayClip(audio.ClipWepsTorpedoInWater, a.L(i18n.VoiceTorpedoInWater))
	}
}

// markLatestHostileTorpedoReported suppresses a second "torpedo in the water"
// voice when the same fish is later classified on sonar.
func (a *App) markLatestHostileTorpedoReported() {
	if a.Engine == nil {
		return
	}
	if a.reportedTorpedoIDs == nil {
		a.reportedTorpedoIDs = map[string]bool{}
	}
	var latest *weapons.Torpedo
	for _, t := range a.Engine.FireControl.ActiveTorpedoes {
		if t == nil || !t.Alive || t.Side != world.SideEnemy {
			continue
		}
		if latest == nil || t.Age <= latest.Age {
			latest = t
		}
	}
	if latest != nil {
		a.reportedTorpedoIDs[latest.ID] = true
	}
}

func (a *App) pollTorpedoCollisionAlerts() {
	if a == nil || a.Engine == nil || a.Engine.Scenario == nil || a.Engine.Scenario.Player == nil {
		return
	}
	if a.torpedoThreatActive == nil {
		a.torpedoThreatActive = map[string]bool{}
	}
	player := a.Engine.Scenario.Player
	now := a.Engine.Clock.GameTime
	current := make(map[string]bool)

	for i := range a.Engine.Sonar.Contacts {
		c := &a.Engine.Sonar.Contacts[i]
		if c.Kind != world.KindTorpedo || !acoustics.ContactTMAAccurate(c) {
			continue
		}
		wx, wy := a.contactPlotWorld(player, c, now)
		if !torpedoThreatensOwnship(player.X, player.Y, player.HeadingDeg, player.SpeedKts, wx, wy, c.TMACourseDeg, c.TMASpeedKts) {
			continue
		}
		id := c.SourceEntityID
		if id == "" {
			id = c.ID
		}
		current[id] = true
		if !a.torpedoThreatActive[id] {
			a.torpedoThreatActive[id] = true
			a.Statusf(i18n.StatusIncomingTorpedoContact, c.ID)
			a.Audio.PlayClip(audio.ClipWepsTorpedoHeadingOwnship, a.L(i18n.VoiceIncomingTorpedo))
		}
	}

	for _, t := range a.Engine.FireControl.ActiveTorpedoes {
		if t == nil || !t.Alive || t.Side != world.SidePlayer {
			continue
		}
		if !torpedoThreatensOwnship(player.X, player.Y, player.HeadingDeg, player.SpeedKts, t.X, t.Y, t.HeadingDeg, t.SpeedKts) {
			continue
		}
		id := t.ID
		current[id] = true
		if !a.torpedoThreatActive[id] {
			a.torpedoThreatActive[id] = true
			a.Statusf(i18n.StatusIncomingTorpedoOwnFish, t.ID)
			a.Audio.PlayClip(audio.ClipWepsTorpedoHeadingOwnship, a.L(i18n.VoiceIncomingTorpedo))
		}
	}

	for id := range a.torpedoThreatActive {
		if !current[id] {
			delete(a.torpedoThreatActive, id)
		}
	}
}

func torpedoThreatensOwnship(px, py, pHeadDeg, pSpeedKts, tx, ty, tHeadDeg, tSpeedKts float64) bool {
	const (
		lookaheadSec = 14 * 60.0
		missYd       = 260.0
	)
	return world.CollisionThreat2D(px, py, pHeadDeg, pSpeedKts, tx, ty, tHeadDeg, tSpeedKts, lookaheadSec, missYd)
}

func (a *App) torpedoThreatMarkerActive(id string) bool {
	if a == nil || a.Engine == nil || id == "" || !a.torpedoThreatActive[id] {
		return false
	}
	return math.Mod(a.Engine.Clock.GameTime, 1.0) < 0.2
}
