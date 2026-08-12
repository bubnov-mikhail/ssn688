package ui

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/config"
	"github.com/ssn688/sim/internal/render"
)

const (
	settingsPanelX    = 380
	settingsPanelY    = 88
	settingsPanelW    = 840
	settingsPanelH    = 580
	settingsRowH      = 52
	settingsLabelX    = settingsPanelX + 28
	settingsControlX  = settingsPanelX + 360
	settingsBtnW      = 44
	settingsBtnH      = 32
	settingsVolTrackW = 220
	settingsVolStep   = 0.05
)

func (a *App) settingsRowY(row int) int {
	return settingsPanelY + 72 + row*settingsRowH
}

func (a *App) settingsToggleRects(row int) (offX, onX, y, w, h int) {
	w = settingsBtnW
	h = settingsBtnH
	y = a.settingsRowY(row) + 6
	offX = settingsControlX
	onX = offX + w + 8
	return
}

func (a *App) settingsVolRects(row int) (decX, trackX, incX, y, trackW, btnW, btnH int) {
	btnW = settingsBtnW
	btnH = settingsBtnH
	y = a.settingsRowY(row) + 6
	decX = settingsControlX
	trackX = decX + btnW + 8
	incX = trackX + settingsVolTrackW + 8
	trackW = settingsVolTrackW
	return
}

func (a *App) settingsBackRect() (x, y, w, h int) {
	w = 200
	h = 48
	x = settingsPanelX + (settingsPanelW-w)/2
	y = settingsPanelY + settingsPanelH - 72
	return
}

func (a *App) settingsHitToggle(row int, on bool) bool {
	offX, onX, y, w, h := a.settingsToggleRects(row)
	mx, my := ebiten.CursorPosition()
	if on {
		return mx >= onX && mx < onX+w && my >= y && my < y+h
	}
	return mx >= offX && mx < offX+w && my >= y && my < y+h
}

func (a *App) settingsHitVolDec(row int) bool {
	decX, _, _, y, _, w, h := a.settingsVolRects(row)
	mx, my := ebiten.CursorPosition()
	return mx >= decX && mx < decX+w && my >= y && my < y+h
}

func (a *App) settingsHitVolInc(row int) bool {
	_, _, incX, y, _, w, h := a.settingsVolRects(row)
	mx, my := ebiten.CursorPosition()
	return mx >= incX && mx < incX+w && my >= y && my < y+h
}

func (a *App) settingsHitVolTrack(row int) bool {
	_, trackX, _, y, trackW, _, h := a.settingsVolRects(row)
	mx, my := ebiten.CursorPosition()
	return mx >= trackX && mx < trackX+trackW && my >= y && my < y+h
}

func (a *App) settingsVolFromTrack(row int, mx int) float64 {
	_, trackX, _, _, trackW, _, _ := a.settingsVolRects(row)
	f := float64(mx-trackX) / float64(trackW)
	return clamp(f, 0, 1)
}

func (a *App) settingsVolPtr(row int) *float64 {
	switch row {
	case 1:
		return &a.Settings.MasterVolume
	case 2:
		return &a.Settings.VoiceVolume
	case 3:
		return &a.Settings.EffectsVolume
	default:
		return nil
	}
}

func (a *App) settingsApplyLive() {
	ebiten.SetFullscreen(a.Settings.Fullscreen)
	a.applySettingsAudio()
}

func (a *App) applySettingsAudio() {
	if a == nil || a.Audio == nil {
		return
	}
	a.Audio.SetVolumes(a.Settings.MasterVolume, a.Settings.VoiceVolume, a.Settings.EffectsVolume)
}

func (a *App) settingsSaveAndBack() {
	_ = config.Save(a.Settings)
	a.settingsApplyLive()
	a.Mode = ModeMenu
}

func (a *App) updateSettings() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.settingsSaveAndBack()
		return
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	mx, my := ebiten.CursorPosition()

	if bx, by, bw, bh := a.settingsBackRect(); mx >= bx && mx < bx+bw && my >= by && my < by+bh {
		a.uiPressedID = "settings_back"
		a.uiPressedAt = time.Now()
		a.settingsSaveAndBack()
		return
	}

	if a.settingsHitToggle(0, false) {
		a.uiPressedID = "set_fs_off"
		a.uiPressedAt = time.Now()
		a.Settings.Fullscreen = false
		a.settingsApplyLive()
		return
	}
	if a.settingsHitToggle(0, true) {
		a.uiPressedID = "set_fs_on"
		a.uiPressedAt = time.Now()
		a.Settings.Fullscreen = true
		a.settingsApplyLive()
		return
	}

	if a.settingsHitToggle(4, false) {
		a.uiPressedID = "set_dbg_off"
		a.uiPressedAt = time.Now()
		a.Settings.Debug = false
		return
	}
	if a.settingsHitToggle(4, true) {
		a.uiPressedID = "set_dbg_on"
		a.uiPressedAt = time.Now()
		a.Settings.Debug = true
		return
	}

	for row := 1; row <= 3; row++ {
		p := a.settingsVolPtr(row)
		if p == nil {
			continue
		}
		if a.settingsHitVolDec(row) {
			a.uiPressedID = fmt.Sprintf("set_vol_dec_%d", row)
			a.uiPressedAt = time.Now()
			*p = clamp(*p-settingsVolStep, 0, 1)
			a.applySettingsAudio()
			return
		}
		if a.settingsHitVolInc(row) {
			a.uiPressedID = fmt.Sprintf("set_vol_inc_%d", row)
			a.uiPressedAt = time.Now()
			*p = clamp(*p+settingsVolStep, 0, 1)
			a.applySettingsAudio()
			return
		}
		if a.settingsHitVolTrack(row) {
			a.uiPressedID = fmt.Sprintf("set_vol_track_%d", row)
			a.uiPressedAt = time.Now()
			*p = a.settingsVolFromTrack(row, mx)
			a.applySettingsAudio()
			return
		}
	}
}

func (a *App) drawSettingsToggleRow(screen *ebiten.Image, row int, label string, on bool, pressedOff, pressedOn string) {
	y := a.settingsRowY(row) + 20
	render.DrawText(screen, label, settingsLabelX, y, render.ColorPlateLabel, true)
	offX, onX, by, w, h := a.settingsToggleRects(row)
	offHover := a.settingsHitToggle(row, false)
	onHover := a.settingsHitToggle(row, true)
	if !on {
		render.FillRect(screen, offX-2, by-2, w+4, h+4, render.ColorAmber)
	}
	if on {
		render.FillRect(screen, onX-2, by-2, w+4, h+4, render.ColorAmber)
	}
	offPressed := a.uiPressedID == pressedOff && time.Since(a.uiPressedAt) < 120*time.Millisecond
	onPressed := a.uiPressedID == pressedOn && time.Since(a.uiPressedAt) < 120*time.Millisecond
	render.DrawBevelButton(screen, offX, by, w, h, "OFF", offHover, offPressed)
	render.DrawBevelButton(screen, onX, by, w, h, "ON", onHover, onPressed)
}

func (a *App) drawSettingsVolumeRow(screen *ebiten.Image, row int, label string, vol float64) {
	y := a.settingsRowY(row) + 20
	render.DrawText(screen, label, settingsLabelX, y, render.ColorPlateLabel, true)
	decX, trackX, incX, by, trackW, w, h := a.settingsVolRects(row)
	decHover := a.settingsHitVolDec(row)
	incHover := a.settingsHitVolInc(row)
	trackHover := a.settingsHitVolTrack(row)
	decID := fmt.Sprintf("set_vol_dec_%d", row)
	incID := fmt.Sprintf("set_vol_inc_%d", row)
	render.DrawBevelButton(screen, decX, by, w, h, "−", decHover, a.uiPressedID == decID && time.Since(a.uiPressedAt) < 120*time.Millisecond)
	render.DrawBevelButton(screen, incX, by, w, h, "+", incHover, a.uiPressedID == incID && time.Since(a.uiPressedAt) < 120*time.Millisecond)

	trackClr := render.ColorPanelInset
	if trackHover {
		trackClr = render.ColorPanelMid
	}
	render.FillRect(screen, trackX, by, trackW, h, trackClr)
	if fillW := int(float64(trackW-4) * vol); fillW > 0 {
		render.FillRect(screen, trackX+2, by+2, fillW, h-4, render.ColorPhosphorDim)
	}
	render.DrawText(screen, fmt.Sprintf("%.0f%%", vol*100), trackX+trackW+56, y, render.ColorPhosphor, true)
}

func (a *App) settingsHoverTooltip(mx, my int) string {
	if a.settingsHitToggle(0, false) || a.settingsHitToggle(0, true) {
		return "Borderless full-screen display"
	}
	if a.settingsHitToggle(4, false) || a.settingsHitToggle(4, true) {
		return "Show true unit positions, headings, and AI routes on PLOT / minimap (DEBUG)"
	}
	for row := 1; row <= 3; row++ {
		if a.settingsHitVolDec(row) || a.settingsHitVolInc(row) || a.settingsHitVolTrack(row) {
			switch row {
			case 1:
				return "Overall game volume"
			case 2:
				return "Officer voice lines"
			case 3:
				return "Sonar pings and weapon effects"
			}
		}
	}
	if bx, by, bw, bh := a.settingsBackRect(); mx >= bx && mx < bx+bw && my >= by && my < by+bh {
		return "Save preferences and return to menu"
	}
	return ""
}

func (a *App) drawSettings(screen *ebiten.Image) {
	render.DrawMenuBackground(screen)

	title := "SETTINGS"
	titleW := len(title) * 14
	render.DrawTextLarge(screen, title, (render.ScreenW-titleW)/2, settingsPanelY-28, render.ColorText)

	render.DrawConsolePanel(screen, settingsPanelX, settingsPanelY, settingsPanelW, settingsPanelH)
	border := color.RGBA{78, 78, 84, 255}
	render.FillRect(screen, settingsPanelX, settingsPanelY, settingsPanelW, 1, border)
	render.FillRect(screen, settingsPanelX, settingsPanelY+settingsPanelH-1, settingsPanelW, 1, border)
	render.FillRect(screen, settingsPanelX, settingsPanelY, 1, settingsPanelH, border)
	render.FillRect(screen, settingsPanelX+settingsPanelW-1, settingsPanelY, 1, settingsPanelH, border)
	render.DrawText(screen, "PREFERENCES", settingsPanelX+24, settingsPanelY+28, render.ColorPlateLabel, true)

	a.drawSettingsToggleRow(screen, 0, "FULLSCREEN", a.Settings.Fullscreen, "set_fs_off", "set_fs_on")
	a.drawSettingsVolumeRow(screen, 1, "MASTER VOLUME", a.Settings.MasterVolume)
	a.drawSettingsVolumeRow(screen, 2, "VOICE VOLUME", a.Settings.VoiceVolume)
	a.drawSettingsVolumeRow(screen, 3, "EFFECTS VOLUME", a.Settings.EffectsVolume)
	a.drawSettingsToggleRow(screen, 4, "DEBUG PLOT", a.Settings.Debug, "set_dbg_off", "set_dbg_on")

	mx, my := ebiten.CursorPosition()
	bx, by, bw, bh := a.settingsBackRect()
	backHover := mx >= bx && mx < bx+bw && my >= by && my < by+bh
	backPressed := a.uiPressedID == "settings_back" && time.Since(a.uiPressedAt) < 120*time.Millisecond
	render.DrawBevelButton(screen, bx, by, bw, bh, "SAVE & BACK", backHover, backPressed)

	if tip := a.settingsHoverTooltip(mx, my); tip != "" {
		render.DrawTooltip(screen, mx, my, tip)
	}

	hint := "CLICK CONTROLS TO ADJUST  ·  ESC OR SAVE & BACK"
	hintW := len(hint) * 7
	render.DrawText(screen, hint, (render.ScreenW-hintW)/2, settingsPanelY+settingsPanelH+24, render.ColorPhosphorDim, true)
}
