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
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/config"
	"github.com/ssn688/sim/internal/layout"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/sim"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

type AppMode int

const (
	ModeMenu AppMode = iota
	ModeGame
	ModeSettings
	ModeLoad
	ModePaused
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
	lastPingPlayed         float64
	debugMapZoom           float64
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
	pendingPlotMarker      bool // M pressed while already on PLOT
	reportedTorpedoIDs     map[string]bool // hostile fish already announced by WEPS
	torpedoThreatActive    map[string]bool // torpedoes currently assessed as threatening ownship
	referenceProfileIdx    int
	contactTableScroll     contactTableScrollState
	layerSurveyWasActive   bool
	tactical               tacticalState
	wepsMapZoom            float64
	wepsMapImg             *ebiten.Image
}

func NewApp(settings config.Settings, audioMgr *audio.Manager) *App {
	return &App{
		Mode:                ModeMenu,
		Settings:            settings,
		Audio:               audioMgr,
		CurrentScreen:       ScreenPassive,
		MenuIndex:           0,
		debugMapZoom:        1.0,
		enemyPingHeardAt:    map[string]float64{},
		reportedTorpedoIDs:  map[string]bool{},
		torpedoThreatActive: map[string]bool{},
		activeRangeScaleYd:  12000,
		wepsMapZoom:         0.05,
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
	case ModeGame, ModePaused:
		a.updateGame()
	}
	return nil
}

func (a *App) updateSettings() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		_ = config.Save(a.Settings)
		ebiten.SetFullscreen(a.Settings.Fullscreen)
		a.Audio.SetVolumes(a.Settings.MasterVolume, a.Settings.VoiceVolume, a.Settings.EffectsVolume)
		a.Mode = ModeMenu
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		a.Settings.Fullscreen = !a.Settings.Fullscreen
		ebiten.SetFullscreen(a.Settings.Fullscreen)
	}
	if ebiten.IsKeyPressed(ebiten.KeyBracketLeft) {
		a.Settings.MasterVolume = clamp(a.Settings.MasterVolume-0.02, 0, 1)
	}
	if ebiten.IsKeyPressed(ebiten.KeyBracketRight) {
		a.Settings.MasterVolume = clamp(a.Settings.MasterVolume+0.02, 0, 1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		a.Settings.Debug = !a.Settings.Debug
	}
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
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sav" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
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
		if realDT > 0.10 {
			realDT = 0.10
		}
	}
	a.lastUpdateWall = nowWall

	if a.handleHeaderButtons() {
		return
	}

	a.updateDebugMapInput()
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
	screenKeys := []ebiten.Key{ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF4, ebiten.KeyF5, ebiten.KeyF6, ebiten.KeyM, ebiten.KeyF7}
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
			default:
				a.CurrentScreen = Screen(i)
			}
		}
	}

	a.handleScreenInput()

	if a.Mode != ModePaused {
		a.Engine.Update(realDT)
		a.updateSimulationUI()
	}

	if a.Engine != nil {
		env := &a.Engine.Acoustics.Env
		active := env.LayerSurveyActive(a.Engine.Clock.GameTime)
		if a.layerSurveyWasActive && !active && env.LayerSurveyKnown {
			a.StatusMessage = "BT cast complete — thermocline boundaries plotted."
			a.Audio.PlayClip(audio.ClipSonarLayerSurveyComplete, "Layer survey complete.")
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

	for _, ev := range a.Engine.PopEvents() {
		a.StatusMessage = ev
		if isWeaponImpactEvent(ev) {
			a.Audio.PlayClip(audio.ClipWepsImpactConfirmed, "")
		}
		if ev == "Torpedo launch detected (hostile)" {
			a.Audio.PlayClip(audio.ClipWepsTorpedoInWater, "Torpedo in the water.")
			a.markLatestHostileTorpedoReported()
		}
	}
	a.pollHostileTorpedoAlerts()
	a.pollTorpedoCollisionAlerts()

	if a.Engine.Scenario.MissionComplete() {
		a.StatusMessage = "MISSION COMPLETE - All targets destroyed."
	}
	if a.Engine.Scenario.MissionFailed() {
		msg := a.Engine.Scenario.FailReason
		if msg == "" {
			msg = "MISSION FAILED"
		}
		a.StatusMessage = "MISSION FAILED — " + msg
	}

	// Active ping sound (same loud echolocator sample as enemy pings).
	if a.Engine.Sonar.LastPingTime > a.lastPingPlayed {
		a.Audio.PlayEnemyPing()
		a.lastPingPlayed = a.Engine.Sonar.LastPingTime
	}
	a.updateEnemyPingAudio()

	if inpututil.IsKeyJustPressed(ebiten.KeyS) && ebiten.IsKeyPressed(ebiten.KeyControl) {
		a.quickSave()
	}
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

func (a *App) handleScreenInput() {
	player := a.Engine.Scenario.Player
	fc := &a.Engine.FireControl
	sonar := &a.Engine.Sonar

	switch a.CurrentScreen {
	case ScreenPassive:
		a.updateSonarUIButtons(cachedPassiveArrayButtons(), sonar)
		a.updateSonarUIButtons(cachedPassiveBandButtons(), sonar)
		a.updateSonarUIButtons(cachedPassiveTowedButtons(), sonar)
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
		a.updateSonarUIButtons(cachedSpectrumArrayButtons(), sonar)
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
	}
}

func (a *App) Draw(screen *ebiten.Image) {
	switch a.Mode {
	case ModeMenu:
		a.drawMenu(screen)
	case ModeSettings:
		screen.Fill(render.ColorBG)
		a.drawSettings(screen)
	case ModeLoad:
		a.drawLoad(screen)
	case ModeGame, ModePaused:
		screen.Fill(render.ColorBG)
		a.drawGame(screen)
	}
}

func (a *App) drawSettings(screen *ebiten.Image) {
	render.DrawTextLarge(screen, "SETTINGS", 700, 80, render.ColorText)
	render.DrawText(screen, fmt.Sprintf("Fullscreen (F): %v", a.Settings.Fullscreen), 500, 200, render.ColorText, false)
	render.DrawText(screen, fmt.Sprintf("Master Volume ([ ]): %.0f%%", a.Settings.MasterVolume*100), 500, 260, render.ColorText, false)
	render.DrawText(screen, fmt.Sprintf("Voice Volume: %.0f%%", a.Settings.VoiceVolume*100), 500, 300, render.ColorText, false)
	render.DrawText(screen, fmt.Sprintf("Effects Volume: %.0f%%", a.Settings.EffectsVolume*100), 500, 340, render.ColorText, false)
	render.DrawText(screen, fmt.Sprintf("Debug minimap (D): %v", a.Settings.Debug), 500, 400, render.ColorText, false)
	render.DrawText(screen, "Green=calm  Yellow=search  Red=attack  Gray=destroyed", 500, 430, render.ColorDim, true)
	render.DrawText(screen, "ENTER or ESC to save and return", 500, 500, render.ColorDim, false)
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
	default:
		render.DrawConsoleBackdrop(screen)
	}

	a.drawGameHeader(screen)

	render.DrawConsolePanel(screen, render.ScreenW-300, 50, 290, 200)
	render.DrawText(screen, "OBJECTIVES", render.ScreenW-280, 75, render.ColorWarn, false)
	for i, o := range a.Engine.Scenario.Objectives {
		mark := "[ ]"
		clr := render.ColorDim
		if o.Complete {
			mark = "[X]"
			clr = render.ColorSonar
		}
		render.DrawText(screen, mark+" "+o.Description, render.ScreenW-280, 100+i*30, clr, true)
	}

	if a.StatusMessage != "" {
		render.DrawText(screen, a.StatusMessage, 20, render.ScreenH-navBarH-18, render.ColorWarn, false)
	}
	if sub, ok := a.Audio.Subtitle(); ok {
		render.FillRect(screen, 200, render.ScreenH-navBarH-50, render.ScreenW-400, 36, render.ColorPanelDark)
		render.DrawText(screen, sub, 220, render.ScreenH-navBarH-28, render.ColorHighlight, false)
	}

	if a.Mode == ModePaused {
		render.DrawTextLarge(screen, "PAUSED", 720, 420, render.ColorDanger)
	}
	if a.Engine.Scenario.Player.Status == world.StatusSinking {
		render.DrawTextLarge(screen, "OWN SHIP HIT - SINKING", 520, 460, render.ColorDanger)
	} else if a.Engine.Scenario.Player.Status == world.StatusSunk {
		render.DrawTextLarge(screen, "OWN SHIP LOST", 650, 460, render.ColorDanger)
	}

	a.drawDebugMap(screen)
	a.drawNavBar(screen)
}

func (a *App) drawGameHeader(screen *ebiten.Image) {
	render.FillRect(screen, 0, 0, render.ScreenW, gameHeaderH, render.ColorPanelBezel)
	render.FillRect(screen, 0, 2, render.ScreenW, gameHeaderH-2, render.ColorPanelDark)
	render.DrawText(screen, fmt.Sprintf("SSN-688  |  TIME %s  |  SPEED %s  |  SCREEN %s",
		formatTime(a.Engine.Clock.GameTime), a.Engine.Clock.SpeedLabel(), screenName(a.CurrentScreen)), 20, 28, render.ColorText, false)
	a.drawHeaderButtons(screen)
}

func screenName(s Screen) string {
	names := []string{"PASSIVE", "ACTIVE", "SPECTRUM", "LIBRARY", "WEPS", "MANEUVER", "TACTICAL"}
	if int(s) < len(names) {
		return names[s]
	}
	return "UNKNOWN"
}

func formatTime(sec float64) string {
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	s := int(sec) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
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

	a.drawArraySelector(screen, sonar, layout.PassiveArrayLabelX, layout.PassiveArrayLabelY, cachedPassiveArrayButtons())
	a.drawListenBandSelector(screen, sonar)
	a.drawTowedControls(screen, sonar)
	a.drawPassiveContactTable(screen, sonar)

	a.drawBearingWaterfall(screen, sonar)
	a.drawWaterfallContactChips(screen, sonar)

	// Section legends — light gray engraved labels, no nameplates.
	titleX := layout.PassiveTitleLabelX
	titleY := layout.PassiveTitleLabelY + 12
	render.DrawText(screen, "BEARING WATERFALL", titleX, titleY, render.ColorPlateLabel, true)

	sel := selectedContactLabel(sonar, a.selectedContactID)
	passiveExtra := ""
	if a.Engine.Clock.GameTime < sonar.SonarDeafUntil {
		passiveExtra = fmt.Sprintf("  |  SONAR BLIND %.0fs", sonar.SonarDeafUntil-a.Engine.Clock.GameTime)
	}
	render.DrawText(screen, fmt.Sprintf("CONTACTS: %d  |  ARRAY: %s  |  SELECTED: %s  |  LAYER: %s%s",
		len(sonar.Contacts), a.sonarArrayLabel(sonar), sel,
		a.Engine.Acoustics.Env.LayerNameKnown(player.DepthFt), passiveExtra),
		layout.PassiveStatusTextX, layout.PassiveStatusTextY, render.ColorDim, false)

	selfNoise := acoustics.PassiveSelfNoiseSeverity(sonar.PassiveArray, player.SpeedKts, player.DepthFt, sonar.TowedCablePct)
	noiseY := layout.PassiveSelfNoiseMonitorY + 22
	render.DrawText(screen, "SELF-NOISE", layout.PassiveSelfNoiseLabelX, noiseY, render.ColorPlateLabel, true)
	label := "QUIET"
	if selfNoise > 0.75 {
		label = "DEAFENING"
	} else if selfNoise > 0.45 {
		label = "FLOW NOISE"
	} else if selfNoise > 0.15 {
		label = "RISING"
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
	render.DrawText(screen, "[P] passive  [B] array  [N] listen band  [U/Y/H] towed  click contact -> spectrum", layout.PassiveHintLabelX, layout.PassiveHintLabelY+12, render.ColorPlateLabel, true)
}

func (a *App) drawActive(screen *ebiten.Image) {
	sonar := &a.Engine.Sonar
	player := a.Engine.Scenario.Player
	render.DrawConsolePanel(screen, activePanelX, activePanelY, activePanelW, 700)
	render.DrawConsolePanel(screen, activeSideX, activeSideY, activeSideW, 700)
	render.DrawMonitor(screen, activePlotX, activePlotY, activePlotW, activePlotH)
	render.DrawText(screen, "ACTIVE SONAR", 40, 90, render.ColorPlateLabel, true)
	status := "STANDBY"
	statusClr := render.ColorDim
	if sonar.ActiveEnabled {
		status = "TRANSMIT READY"
		statusClr = render.ColorActive
	}
	echoYd := a.activeEchoReachYd(sonar)
	echoLabel := "—"
	if echoYd > 0 {
		echoLabel = fmt.Sprintf("%.1f kyd", echoYd/1000)
	}
	render.DrawText(screen, fmt.Sprintf("%s  |  SPD %.1f kts  |  ECHO REACH %s", status, player.SpeedKts, echoLabel), 40, 120, statusClr, false)

	a.drawActiveRangeDisplay(screen, sonar)
	a.drawActiveControls(screen, sonar)
	a.drawActiveContactTable(screen, sonar)

	if a.activeSonarDamaged() {
		msg := "ACTIVE SONAR DAMAGED — NO TRANSMIT"
		render.DrawText(screen, msg, activePlotX+180, activePlotY+activePlotH/2, render.ColorWarn, false)
	}

	mx, my := ebiten.CursorPosition()
	if a.uiTooltip != "" {
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
	render.DrawText(screen, "[A] toggle  [F] ping  click plot / table → select contact", 40, 720, render.ColorDim, true)
}

func (a *App) updateLibraryInput() {
	if a.Engine == nil {
		return
	}
	mx, my := ebiten.CursorPosition()
	scrollContactTableWheel(mx, my, 600, 142, 520, 18*25, len(a.Engine.Sonar.Contacts), 18, &a.contactTableScroll.library)
}

func (a *App) drawLibrary(screen *ebiten.Image) {
	render.DrawConsolePanel(screen, 20, 50, 1100, 700)
	render.DrawText(screen, "SIGNATURE LIBRARY", 40, 90, render.ColorPlateLabel, true)
	y := 130
	for _, p := range world.SignatureLibrary {
		render.DrawText(screen, fmt.Sprintf("%s - %s (%s)", p.Class, p.Name, kindName(p.Kind)), 40, y, render.ColorText, false)
		for _, b := range p.Bands {
			render.DrawText(screen, fmt.Sprintf("  %.0f-%.0f Hz  %.0f dB", b.LowHz, b.HighHz, b.LevelDB), 60, y+20, render.ColorDim, true)
			y += 20
		}
		render.DrawText(screen, fmt.Sprintf("  Blade rate: %.1f Hz  Cavitation onset: %.0f dB", p.BladeRateHz, p.CavitationDB), 60, y+20, render.ColorDim, true)
		y += 45
	}

	render.DrawText(screen, "CLASSIFIED CONTACTS", 600, 130, render.ColorWarn, false)
	const (
		listX       = 600
		listY       = 160
		listW       = 520
		rowH        = 25
		visibleRows = 18
	)
	a.contactTableScroll.library = clampContactTableScroll(a.contactTableScroll.library, len(a.Engine.Sonar.Contacts), visibleRows)
	start, end := contactTableWindow(len(a.Engine.Sonar.Contacts), a.contactTableScroll.library, visibleRows)
	for row, i := 0, start; i < end; i, row = i+1, row+1 {
		c := a.Engine.Sonar.Contacts[i]
		label := contactClassLabel(&c)
		if c.ConfirmedClass != "" {
			label = c.ConfirmedClass + " (confirmed)"
		}
		render.DrawText(screen, fmt.Sprintf("%s: %s (%.0f%%) brg %s  %d bands  %.0fs",
			c.ID, label, c.Confidence*100, contactBearingLabel(&c), c.BandsAbove, c.ListenTime),
			listX, listY+row*rowH, render.ColorSonar, true)
	}
	drawContactTableScrollbar(screen, listX+listW+4, listY-12, visibleRows*rowH, len(a.Engine.Sonar.Contacts), visibleRows, a.contactTableScroll.library)
}

func kindName(k world.EntityKind) string {
	switch k {
	case world.KindSubmarine:
		return "SUB"
	case world.KindSurfaceShip:
		return "SURFACE"
	case world.KindTorpedo:
		return "TORPEDO"
	case world.KindCountermeasure:
		return "CM"
	default:
		return "UNK"
	}
}

func isWeaponImpactEvent(ev string) bool {
	switch {
	case strings.HasPrefix(ev, "Target destroyed:"):
		return true
	case strings.HasPrefix(ev, "PLAYER SUBMARINE HIT"):
		return true
	case ev == "Underwater explosion":
		return true
	default:
		return false
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
		a.StatusMessage = fmt.Sprintf("Torpedo in the water — contact %s.", c.ID)
		a.Audio.PlayClip(audio.ClipWepsTorpedoInWater, "Torpedo in the water.")
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
			a.StatusMessage = fmt.Sprintf("Incomming torpedo! Contact %s crossing ownship track.", c.ID)
			a.Audio.PlayClip(audio.ClipWepsTorpedoHeadingOwnship, "Incomming torpedo!")
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
			a.StatusMessage = fmt.Sprintf("Incomming torpedo! Own fish %s crossing ownship track.", t.ID)
			a.Audio.PlayClip(audio.ClipWepsTorpedoHeadingOwnship, "Incomming torpedo!")
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
