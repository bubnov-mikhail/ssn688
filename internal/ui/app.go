package ui

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/config"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/save"
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
)

type App struct {
	Mode           AppMode
	Settings       config.Settings
	Engine         *sim.Engine
	Audio          *audio.Manager
	CurrentScreen  Screen
	MenuIndex      int
	LoadFiles      []string
	LoadIndex      int
	StatusMessage         string
	bearingWaterfalls     BearingWaterfallBank
	lastWaterfallSample   float64
	lastPingPlayed        float64
	debugMapZoom   float64
	uiHoverID      string
	uiHoverSince   time.Time
	uiTooltip      string
	uiPressedID    string
	uiPressedAt    time.Time
	navHoverIdx    int
	navHoverSince  time.Time
	navTooltip          string
	compassDrag         bool
	selectedContactID   string
	referenceProfileIdx int
	layerSurveyWasActive bool
}

func NewApp(settings config.Settings, audioMgr *audio.Manager) *App {
	return &App{
		Mode:          ModeMenu,
		Settings:      settings,
		Audio:         audioMgr,
		CurrentScreen: ScreenPassive,
		MenuIndex:     0,
		debugMapZoom:  1.0,
	}
}

func (a *App) StartNewGame() {
	a.Engine = sim.NewEngine(world.NewTrainingScenario())
	a.Mode = ModeGame
	a.CurrentScreen = ScreenPassive
	a.StatusMessage = "Mission: destroy hostile surface unit and submarine."
	a.bearingWaterfalls.Reset()
	a.lastWaterfallSample = 0
	a.selectedContactID = ""
	a.referenceProfileIdx = 0
	a.layerSurveyWasActive = false
	a.Audio.PlayClip(audio.ClipCaptMissionBrief, "")
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

func (a *App) updateLoad() {
	if len(a.LoadFiles) == 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			a.Mode = ModeMenu
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		a.LoadIndex = (a.LoadIndex + 1) % len(a.LoadFiles)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		a.LoadIndex = (a.LoadIndex + len(a.LoadFiles) - 1) % len(a.LoadFiles)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		engine, err := save.LoadClean(a.LoadFiles[a.LoadIndex])
		if err == nil {
			a.Engine = engine
			a.Mode = ModeGame
		} else {
			a.StatusMessage = "Load failed: " + err.Error()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.Mode = ModeMenu
	}
}

func (a *App) updateGame() {
	if a.Engine == nil {
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

	// Screen selection (F-keys to avoid conflict with fire control tubes)
	screenKeys := []ebiten.Key{ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF4, ebiten.KeyF5, ebiten.KeyF6, ebiten.KeyM}
	for i, k := range screenKeys {
		if inpututil.IsKeyJustPressed(k) {
			if k == ebiten.KeyM {
				a.CurrentScreen = ScreenTactical
			} else {
				a.CurrentScreen = Screen(i)
			}
		}
	}

	a.handleScreenInput()

	if a.Mode != ModePaused {
		a.Engine.Update(1.0 / 60.0)
	}

	if a.Engine != nil {
		env := &a.Engine.Acoustics.Env
		active := env.LayerSurveyActive(a.Engine.Clock.GameTime)
		if a.layerSurveyWasActive && !active && env.LayerSurveyKnown {
			a.StatusMessage = "BT cast complete — thermocline boundaries plotted."
			a.Audio.PlayClip(audio.ClipSonarPassiveOn, "Layer survey complete.")
		}
		a.layerSurveyWasActive = active
	}

	if a.CurrentScreen != ScreenManeuver && a.CurrentScreen != ScreenPassive && a.CurrentScreen != ScreenSpectrum {
		a.uiTooltip = ""
		a.uiHoverID = ""
	}
	if a.navHoverIdx < 0 {
		a.navTooltip = ""
	}

	for _, ev := range a.Engine.PopEvents() {
		a.StatusMessage = ev
		a.Audio.PlayClip(audio.ClipWepsImpactConfirmed, "")
	}

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

	// Active ping sound
	if a.Engine.Sonar.LastPingTime > a.lastPingPlayed {
		a.Audio.PlayPing()
		a.lastPingPlayed = a.Engine.Sonar.LastPingTime
	}

	// Bearing waterfall sampling (bearing vs time, newest at top).
	a.updateBearingWaterfall()

	if inpututil.IsKeyJustPressed(ebiten.KeyS) && ebiten.IsKeyPressed(ebiten.KeyControl) {
		a.quickSave()
	}
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
		arrayBtns := passiveArrayButtons(960, 90)
		towedBtns := towedControlButtons(960, 216)
		allBtns := append(arrayBtns, towedBtns...)
		a.updateSonarUIButtons(allBtns, sonar)
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
		if inpututil.IsKeyJustPressed(ebiten.KeyA) {
			sonar.ActiveEnabled = !sonar.ActiveEnabled
			a.Audio.PlayClip(audio.ClipSonarActiveStandby, "")
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF) {
			sonar.LastPingTime = 0
			a.Audio.PlayClip(audio.ClipSonarActivePing, "")
		}
	case ScreenSpectrum:
		a.updateSonarUIButtons(passiveArrayButtons(40, 128), sonar)
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
	case ScreenFireControl:
		a.handleFireControl(fc, player)
	case ScreenManeuver:
		a.updateManeuverUI(player)
	}
}

func (a *App) handleFireControl(fc *weapons.FireControl, player *world.Entity) {
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit1) {
		fc.SelectTube(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit2) {
		fc.SelectTube(2)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit3) {
		fc.SelectTube(3)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit4) {
		fc.SelectTube(4)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyO) {
		if fc.OpenOuterDoor() {
			a.Audio.PlayClip(audio.TubeClip("outer_door_open", fc.SelectedTube),
				fmt.Sprintf("Outer door open, tube %d.", fc.SelectedTube))
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		fc.CloseOuterDoor()
		a.Audio.PlayClip(audio.ClipWepsOuterDoorClosed, "")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		fc.GyroAngleDeg += 5
		if fc.GyroAngleDeg > 180 {
			fc.GyroAngleDeg -= 360
		}
		a.Audio.PlayClip(audio.ClipWepsGyroSet, fmt.Sprintf("Gyro angle %d degrees.", int(fc.GyroAngleDeg)))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		fc.RunDepthFt += 50
		a.Audio.PlayClip(audio.ClipWepsRunDepthSet, fmt.Sprintf("Torpedo run depth %d feet.", int(fc.RunDepthFt)))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && !ebiten.IsKeyPressed(ebiten.KeyControl) {
		if fc.SpeedSetting == "HIGH" {
			fc.SpeedSetting = "LOW"
			a.Audio.PlayClip(audio.ClipWepsSpeedLow, "Torpedo speed LOW.")
		} else {
			fc.SpeedSetting = "HIGH"
			a.Audio.PlayClip(audio.ClipWepsSpeedHigh, "Torpedo speed HIGH.")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		fc.SeekerEnabled = !fc.SeekerEnabled
		if fc.SeekerEnabled {
			a.Audio.PlayClip(audio.ClipWepsSeekerOn, "")
		} else {
			a.Audio.PlayClip(audio.ClipWepsSeekerOff, "")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if torp := fc.Shoot(player); torp != nil {
			a.Audio.PlayTorpedoLaunch()
			a.Audio.PlayClip(audio.TubeClip("torpedo_away", fc.SelectedTube),
				fmt.Sprintf("Torpedo away, tube %d.", fc.SelectedTube))
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) && len(fc.ActiveTorpedoes) > 0 {
		fc.CutWire(fc.ActiveTorpedoes[len(fc.ActiveTorpedoes)-1])
		a.Audio.PlayClip(audio.ClipWepsWireCut, "")
	}
}

func (a *App) quickSave() {
	dir, err := config.SavesDir()
	if err != nil {
		return
	}
	path := filepath.Join(dir, fmt.Sprintf("quicksave_%03d.sav", int(a.Engine.Clock.GameTime)%1000))
	_ = save.Save(path, a.Engine)
	a.StatusMessage = "Game saved."
	a.Audio.PlayClip(audio.ClipCaptSaveComplete, "")
}

func (a *App) Draw(screen *ebiten.Image) {
	switch a.Mode {
	case ModeMenu:
		a.drawMenu(screen)
	case ModeSettings:
		screen.Fill(render.ColorBG)
		a.drawSettings(screen)
	case ModeLoad:
		screen.Fill(render.ColorBG)
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

func (a *App) drawLoad(screen *ebiten.Image) {
	render.DrawTextLarge(screen, "LOAD GAME", 680, 80, render.ColorText)
	if len(a.LoadFiles) == 0 {
		render.DrawText(screen, "No save files found.", 500, 200, render.ColorWarn, false)
	} else {
		for i, f := range a.LoadFiles {
			clr := render.ColorDim
			if i == a.LoadIndex {
				clr = render.ColorHighlight
			}
			render.DrawText(screen, filepath.Base(f), 400, 180+i*40, clr, false)
		}
	}
	render.DrawText(screen, "ENTER to load, ESC to cancel", 500, 700, render.ColorDim, false)
}

func (a *App) drawGame(screen *ebiten.Image) {
	render.FillRect(screen, 0, 0, render.ScreenW, 40, render.ColorPanel)
	render.DrawText(screen, fmt.Sprintf("SSN-688  |  TIME %s  |  SPEED %s  |  SCREEN %s",
		formatTime(a.Engine.Clock.GameTime), a.Engine.Clock.SpeedLabel(), screenName(a.CurrentScreen)), 20, 28, render.ColorText, false)

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
	}

	// Objectives sidebar
	render.FillRect(screen, render.ScreenW-300, 50, 290, 200, render.ColorPanel)
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
		render.FillRect(screen, 200, render.ScreenH-navBarH-50, render.ScreenW-400, 36, render.ColorPanel)
		render.DrawText(screen, sub, 220, render.ScreenH-navBarH-28, render.ColorHighlight, false)
	}

	if a.Mode == ModePaused {
		render.DrawTextLarge(screen, "PAUSED", 720, 420, render.ColorDanger)
	}

	a.drawDebugMap(screen)
	a.drawNavBar(screen)
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
	render.FillRect(screen, 20, 50, 900, 700, render.ColorPanel)
	render.DrawTextLarge(screen, "PASSIVE SONAR", 40, 90, render.ColorSonar)
	sel := selectedContactLabel(sonar, a.selectedContactID)
	render.DrawText(screen, fmt.Sprintf("PASSIVE: %v  |  ARRAY: %s  |  Contacts: %d  |  Selected: %s  |  Layer: %s",
		sonar.PassiveEnabled, a.sonarArrayLabel(sonar), len(sonar.Contacts), sel,
		a.Engine.Acoustics.Env.LayerNameKnown(player.DepthFt)), 40, 120, render.ColorDim, false)
	cav := acoustics.CavitationSeverity(player.DepthFt, player.SpeedKts)
	if cav > 0.2 {
		render.DrawText(screen, fmt.Sprintf("CAVITATION WARNING: %.0f%%", cav*100), 40, 145, render.ColorDanger, true)
	}

	a.drawPassiveBearingPlot(screen, player, sonar)

	render.FillRect(screen, 940, 50, 340, 700, render.ColorPanel)
	a.drawArraySelector(screen, sonar, 960, 80)
	a.drawTowedControls(screen, sonar, 960, 160)
	a.drawWaterfallContactChips(screen, sonar)
	a.drawBearingWaterfall(screen, player, sonar)
	mx, my := ebiten.CursorPosition()
	if a.uiTooltip != "" {
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
	render.DrawText(screen, "[P] passive  [B] array  [U/Y/H] towed  click contact → spectrum", 40, 720, render.ColorDim, true)
}

func (a *App) drawActive(screen *ebiten.Image) {
	sonar := &a.Engine.Sonar
	render.FillRect(screen, 20, 50, 1100, 700, render.ColorPanel)
	render.DrawTextLarge(screen, "ACTIVE SONAR", 40, 90, render.ColorActive)
	render.DrawText(screen, fmt.Sprintf("ACTIVE: %v  POWER: %.0f%%  PING INT: %.0fs", sonar.ActiveEnabled, sonar.ActivePower*100, sonar.PingInterval), 40, 120, render.ColorDim, false)

	cx, cy := 500.0, 400.0
	for _, c := range sonar.Contacts {
		if c.DetectedBy != "active" && c.Confidence < 0.5 {
			continue
		}
		rad := c.BearingDeg * math.Pi / 180
		rng := math.Min(300, c.EstimatedRangeYd/50)
		x := cx + math.Sin(rad)*rng
		y := cy - math.Cos(rad)*rng
		render.DrawLine(screen, cx, cy, x, y, render.ColorActive)
		render.FillRect(screen, int(x)-5, int(y)-5, 10, 10, render.ColorActive)
		render.DrawText(screen, fmt.Sprintf("%s R%.1f kyd", contactLongLabel(&c), c.EstimatedRangeYd/1000), int(x)+8, int(y), render.ColorText, true)
	}
	render.DrawText(screen, "[A] toggle  [F] fire ping", 40, 720, render.ColorDim, true)
}

func (a *App) drawLibrary(screen *ebiten.Image) {
	render.FillRect(screen, 20, 50, 1100, 700, render.ColorPanel)
	render.DrawTextLarge(screen, "SIGNATURE LIBRARY", 40, 90, render.ColorText)
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
	for i, c := range a.Engine.Sonar.Contacts {
		label := contactClassLabel(&c)
		if c.ConfirmedClass != "" {
			label = c.ConfirmedClass + " (confirmed)"
		}
		render.DrawText(screen, fmt.Sprintf("%s: %s (%.0f%%) brg %.0f  %s  %d bands  %.0fs",
			c.ID, label, c.Confidence*100, c.BearingDeg, contactTypeLabel(&c), c.BandsAbove, c.ListenTime),
			600, 160+i*25, render.ColorSonar, true)
	}
}

func kindName(k world.EntityKind) string {
	switch k {
	case world.KindSubmarine:
		return "SUB"
	case world.KindSurfaceShip:
		return "SURFACE"
	default:
		return "UNK"
	}
}

func (a *App) drawFireControl(screen *ebiten.Image) {
	fc := &a.Engine.FireControl
	render.FillRect(screen, 20, 50, 1100, 700, render.ColorPanel)
	render.DrawTextLarge(screen, "FIRE CONTROL - MK48 ADCAP", 40, 90, render.ColorWarn)

	for i, t := range fc.Tubes {
		state := tubeStateName(t.State)
		clr := render.ColorDim
		if i+1 == fc.SelectedTube {
			clr = render.ColorHighlight
		}
		render.DrawText(screen, fmt.Sprintf("TUBE %d: %s %s", t.Number, t.TorpedoType, state), 40, 140+i*35, clr, false)
	}

	render.DrawText(screen, fmt.Sprintf("GYRO ANGLE: %d deg", int(fc.GyroAngleDeg)), 500, 140, render.ColorText, false)
	render.DrawText(screen, fmt.Sprintf("RUN DEPTH: %d ft", int(fc.RunDepthFt)), 500, 180, render.ColorText, false)
	render.DrawText(screen, fmt.Sprintf("SPEED: %s", fc.SpeedSetting), 500, 220, render.ColorText, false)
	render.DrawText(screen, fmt.Sprintf("SEEKER: %v", fc.SeekerEnabled), 500, 260, render.ColorText, false)

	render.DrawText(screen, "Active torpedoes:", 40, 320, render.ColorWarn, false)
	for i, t := range fc.ActiveTorpedoes {
		render.DrawText(screen, fmt.Sprintf("%s hdg %.0f spd %.0f seeker %v wire %v", t.ID, t.HeadingDeg, t.SpeedKts, t.SeekerOn, !t.WireCut), 40, 350+i*25, render.ColorSonar, true)
	}

	render.DrawText(screen, "[1-4] tube [O] open [C] close [G] gyro [D] depth [S] speed [H] seeker [ENTER] shoot [W] wire cut", 40, 680, render.ColorDim, true)
}

func tubeStateName(s weapons.TubeState) string {
	switch s {
	case weapons.TubeEmpty:
		return "EMPTY"
	case weapons.TubeLoaded:
		return "LOADED"
	case weapons.TubeDoorOpen:
		return "DOOR OPEN"
	case weapons.TubeFired:
		return "FIRED"
	default:
		return "?"
	}
}

func (a *App) drawTactical(screen *ebiten.Image) {
	render.FillRect(screen, 20, 50, 1100, 700, render.ColorPanel)
	render.DrawTextLarge(screen, "TACTICAL PLOT", 40, 90, render.ColorText)

	ox, oy := 570.0, 400.0
	scale := 0.03

	player := a.Engine.Scenario.Player
	render.FillRect(screen, int(ox+player.X*scale)-6, int(oy-player.Y*scale)-6, 12, 12, render.ColorHighlight)
	for _, e := range a.Engine.Scenario.Entities {
		clr := render.ColorDanger
		if e.Kind == world.KindSubmarine {
			clr = render.ColorWarn
		}
		if e.Alive() {
			x := ox + e.X*scale
			y := oy - e.Y*scale
			render.FillRect(screen, int(x)-5, int(y)-5, 10, 10, clr)
			render.DrawText(screen, e.Name+" "+e.AIState, int(x)+8, int(y), render.ColorDim, true)
		}
	}
	for _, t := range a.Engine.FireControl.ActiveTorpedoes {
		if t.Alive {
			x := ox + t.X*scale
			y := oy - t.Y*scale
			render.FillRect(screen, int(x)-3, int(y)-3, 6, 6, render.ColorActive)
		}
	}
}
