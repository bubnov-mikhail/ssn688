package ui

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/bubnov-mikhail/ssn688/internal/audio"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/config"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/save"
	"github.com/bubnov-mikhail/ssn688/internal/sim"
)

const (
	gameHeaderH  = 40
	headerBtnH   = 28
	headerBtnGap = 8
	headerBtnPad = 12
)

func disposeImage(img **ebiten.Image) {
	if img == nil || *img == nil {
		return
	}
	(*img).Dispose()
	*img = nil
}

// releaseSessionCaches frees GPU textures and large session buffers.
// Engine is left for the caller to replace or nil.
func (a *App) releaseSessionCaches() {
	render.ClearBlitCache()
	a.disposeWaterfallImages()
	disposeImage(&a.passivePPI)
	a.passivePPIPixels = nil
	disposeImage(&a.spectrumFuzzyImg)
	a.spectrumFuzzyPix = nil
	disposeImage(&a.activePlotImg)
	a.activePlotPix = nil
	a.activePlotGridPix = nil
	disposeImage(&a.wepsMapImg)
	disposeImage(&a.tactical.bathyImg)
	disposeImage(&a.tactical.minimapBathyImg)
	a.tactical.bathyPix = nil
	a.tactical.minimapBathyPix = nil
	a.tactical.bakedBathyRGBA = nil
	a.tactical.bakedBathy = nil
	a.tactical = tacticalState{}
	a.disposePeriscopeImage()
	a.periShipScratch = nil
	a.periDepth = nil

	a.bearingWaterfalls.Reset()
	a.lastWaterfallSample = 0
	a.waterfallPendingScroll = false
	a.waterfallFullRebuild = true
	a.waterfallScratch = nil
	a.waterfallRng = nil
	a.passivePPIStamp = -1
	a.passivePPIPending = true
	a.ppiEnergies = nil
	a.ppiSmoothed = nil
	a.ppiFloorN = nil
	a.ppiGrainN = nil
	a.ppiSens = nil
	a.ppiLUT = nil
	a.ppiLUTSize = 0
	a.spectrumFuzzyStamp = -1
	a.spectrumFuzzyKey = 0
	a.spectrumCacheBins = nil
	a.spectrumCacheAt = -1
	a.spectrumFuzzyLevels = nil
	a.waterfallChipCache = nil
	a.waterfallChipCacheKey = 0
	a.sonarBtnScratch = nil
	a.enemyPingHeardAt = map[string]float64{}
	a.lastPingPlayed = 0
	a.selectedContactID = ""
	a.selectedPlotMarkerID = ""
	a.pendingPlotMarker = false
	a.referenceProfileIdx = 0
	a.librarySelectedID = ""
	a.libraryCatalogScroll = 0
	a.libraryDetailScroll = 0
	a.contactTableScroll = contactTableScrollState{}
	a.mastCommScroll = 0
	a.layerSurveyWasActive = false
	a.lastUpdateWall = time.Time{}
	a.activeEchoFlashes = nil
	a.activeEchoFlashSeq = 0
	a.activeRangeScaleYd = 12000
	a.activePlotGridScaleYd = 0
	a.activePlotGridDirty = true
	a.activeSliderDrag = ""
	a.compassDrag = false
	a.wepsMapZoom = 0.05
	a.uiHoverID = ""
	a.uiTooltip = ""
	a.uiPressedID = ""
	a.navHoverIdx = -1
	a.navTooltip = ""
	a.hitVignetteAt = time.Time{}
	a.hitShakeAt = time.Time{}
	a.dcTabAlert = false
	a.mastTabAlert = false
	disposeImage(&a.hitShakeBuf)
}

// exitToMenu returns to the main menu and drops the running scenario from memory.
func (a *App) exitToMenu() {
	a.endScenarioUI()
	a.releaseSessionCaches()
	a.Engine = nil
	a.Mode = ModeMenu
	a.CurrentScreen = ScreenPassive
	a.StatusMessage = ""
	if a.Audio != nil {
		a.Audio.StopAll()
	}
}

func (a *App) beginGameSession(engine *sim.Engine) {
	a.endScenarioUI()
	a.releaseSessionCaches()
	a.Engine = engine
	a.Mode = ModeGame
	a.CurrentScreen = ScreenPassive
	a.tactical = tacticalState{zoom: 0.035, smoothedPos: map[string]smoothedContactPos{}, fitPending: true}
	a.reportedTorpedoIDs = map[string]bool{}
	a.ownTorpedoIDs = map[string]bool{}
	a.torpedoThreatActive = map[string]bool{}
	a.syncOwnTorpedoIDs()
	a.StatusMessage = ""
}

func (a *App) StartNewGame() {
	a.SelectedScenarioID = campaign.DemoScenarioID
	a.LoadoutMix = 0.25
	a.startSelectedMission()
}

func (a *App) loadSelectedSave() {
	if len(a.LoadFiles) == 0 {
		a.Status(i18n.StatusNoSaves)
		return
	}
	if a.LoadIndex < 0 || a.LoadIndex >= len(a.LoadFiles) {
		a.Status(i18n.StatusSelectSave)
		return
	}
	engine, err := save.LoadClean(a.LoadFiles[a.LoadIndex])
	if err != nil {
		a.Statusf(i18n.StatusLoadFailed, err.Error())
		return
	}
	a.beginGameSession(engine)
}

func (a *App) quickSave() {
	if a.Engine == nil {
		return
	}
	dir, err := config.SavesDir()
	if err != nil {
		a.Statusf(i18n.StatusSaveFailed, err.Error())
		return
	}
	name := fmt.Sprintf("quicksave_%03d.sav", int(a.Engine.Clock.GameTime)%1000)
	path := filepath.Join(dir, name)
	if err := save.Save(path, a.Engine); err != nil {
		a.Statusf(i18n.StatusSaveFailed, err.Error())
		return
	}
	a.Statusf(i18n.StatusGameSaved, name)
	if a.Audio != nil {
		a.Audio.PlayClip(audio.ClipCaptSaveComplete, "")
	}
}

func (a *App) headerButtonRects() (saveX, endX, exitX, y, wSave, wEnd, wExit, h int) {
	h = headerBtnH
	wSave = render.ButtonWidth(a.L(i18n.UISave), 20)
	wEnd = render.ButtonWidth(a.L(i18n.UIEndMission), 12)
	wExit = render.ButtonWidth(a.L(i18n.UIExit), 20)
	y = (gameHeaderH - h) / 2
	exitX = render.ScreenW - headerBtnPad - wExit
	endX = exitX - headerBtnGap - wEnd
	saveX = endX - headerBtnGap - wSave
	return saveX, endX, exitX, y, wSave, wEnd, wExit, h
}

func (a *App) headerHit(mx, my, x, y, w, h int) bool {
	return mx >= x && mx < x+w && my >= y && my < y+h
}

// handleHeaderButtons processes SAVE/END MISSION/EXIT. Returns true if a click was consumed.
func (a *App) handleHeaderButtons() bool {
	if a.Engine == nil {
		return false
	}
	saveX, endX, exitX, y, wSave, wEnd, wExit, h := a.headerButtonRects()
	mx, my := ebiten.CursorPosition()
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	if a.headerHit(mx, my, saveX, y, wSave, h) {
		a.uiPressedID = "hdr_save"
		a.uiPressedAt = time.Now()
		a.quickSave()
		return true
	}
	if a.headerHit(mx, my, endX, y, wEnd, h) {
		if !a.missionEndEligible() {
			return true
		}
		a.uiPressedID = "hdr_end"
		a.uiPressedAt = time.Now()
		a.showConfirm(confirmEndMission, a.L(i18n.UIConfirmEndTitle), a.L(i18n.UIConfirmEndBody))
		return true
	}
	if a.headerHit(mx, my, exitX, y, wExit, h) {
		a.uiPressedID = "hdr_exit"
		a.uiPressedAt = time.Now()
		a.showConfirm(confirmExitMenu, a.L(i18n.UIConfirmExitTitle), a.L(i18n.UIConfirmExitBody))
		return true
	}
	return false
}

func (a *App) drawHeaderButtons(screen *ebiten.Image) {
	saveX, endX, exitX, y, wSave, wEnd, wExit, h := a.headerButtonRects()
	mx, my := ebiten.CursorPosition()
	saveHover := a.headerHit(mx, my, saveX, y, wSave, h)
	endHover := a.missionEndEligible() && a.headerHit(mx, my, endX, y, wEnd, h)
	exitHover := a.headerHit(mx, my, exitX, y, wExit, h)
	savePressed := a.uiPressedID == "hdr_save" && time.Since(a.uiPressedAt) < 120*time.Millisecond
	endPressed := a.uiPressedID == "hdr_end" && time.Since(a.uiPressedAt) < 120*time.Millisecond
	exitPressed := a.uiPressedID == "hdr_exit" && time.Since(a.uiPressedAt) < 120*time.Millisecond
	render.DrawBevelButton(screen, saveX, y, wSave, h, a.L(i18n.UISave), saveHover, savePressed)
	if a.missionEndEligible() {
		render.DrawBevelButton(screen, endX, y, wEnd, h, a.L(i18n.UIEndMission), endHover, endPressed)
	} else {
		render.DrawBevelButtonDisabled(screen, endX, y, wEnd, h, a.L(i18n.UIEndMission))
	}
	render.DrawBevelButton(screen, exitX, y, wExit, h, a.L(i18n.UIExit), exitHover, exitPressed)
}
