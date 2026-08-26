package ui

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/i18n"
	"github.com/ssn688/sim/internal/render"
)

const (
	cmQuickGapAbove = 10
	cmQuickGapBelow = 10
	cmQuickBtnH     = 40
	cmQuickBtnGap   = 10
)

func (a *App) cmQuickPanelRect() (x, y, w, h int) {
	x = render.ScreenW - ownShipPanelX
	w = ownShipPanelW
	y = ownShipPanelY + ownShipPanelH + cmQuickGapAbove
	_, miniY, _, _ := a.minimapRect()
	h = miniY - y - cmQuickGapBelow
	if h < 80 {
		h = 80
	}
	return x, y, w, h
}

func (a *App) cmQuickButtons(decoyN, jitterN int) []uiButton {
	px, py, pw, ph := a.cmQuickPanelRect()
	pad := 14
	innerW := pw - pad*2
	specs := []struct{ id, label, tip string }{
		{"cm_quick_decoy", fmt.Sprintf("%s  %d", a.L(i18n.UIDecoy), decoyN), "Launch acoustic decoy (ADC) toward nearest threat"},
		{"cm_quick_jitter", fmt.Sprintf("%s  %d", a.L(i18n.UIJitter), jitterN), "Launch broadband jammer toward nearest threat"},
	}
	btnW := (innerW - cmQuickBtnGap) / 2
	if btnW < 80 {
		btnW = 80
	}
	y := py + ph - pad - cmQuickBtnH
	x0 := px + pad
	out := make([]uiButton, len(specs))
	for i, s := range specs {
		out[i] = uiButton{
			ID: s.id, Label: s.label, Tooltip: s.tip,
			X: x0 + i*(btnW+cmQuickBtnGap), Y: y, W: btnW, H: cmQuickBtnH,
		}
	}
	return out
}

func (a *App) updateCMQuickPanel() {
	if a.Engine == nil || a.Engine.Scenario == nil || a.Engine.Scenario.Player == nil {
		return
	}
	if a.Mode != ModeGame && a.Mode != ModePaused {
		return
	}
	// Full WEPS already has CM controls; keep quick panel everywhere else for panic dumps.
	if a.CurrentScreen == ScreenFireControl {
		return
	}
	player := a.Engine.Scenario.Player
	decoyN := a.Engine.CM.DecoyLeft(player.ID)
	jitterN := a.Engine.CM.JitterLeft(player.ID)
	buttons := a.cmQuickButtons(decoyN, jitterN)
	mx, my := ebiten.CursorPosition()
	a.updateButtonTooltips(buttons, mx, my)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for _, b := range buttons {
			if !b.contains(mx, my) {
				continue
			}
			if b.ID == "cm_quick_decoy" && decoyN <= 0 {
				a.Status(i18n.StatusDecoyEmpty)
				break
			}
			if b.ID == "cm_quick_jitter" && jitterN <= 0 {
				a.Status(i18n.StatusJitterEmpty)
				break
			}
			a.uiPressedID = b.ID
			a.uiPressedAt = time.Now()
			switch b.ID {
			case "cm_quick_decoy":
				_, msg := a.Engine.LaunchPlayerDecoy()
				a.StatusMessage = msg
			case "cm_quick_jitter":
				_, msg := a.Engine.LaunchPlayerJitter()
				a.StatusMessage = msg
			}
			break
		}
	}
	if a.uiPressedID == "cm_quick_decoy" || a.uiPressedID == "cm_quick_jitter" {
		if time.Since(a.uiPressedAt) > 120*time.Millisecond {
			a.uiPressedID = ""
		}
	}
}

func (a *App) drawCMQuickPanel(screen *ebiten.Image) {
	if a.Engine == nil || a.Engine.Scenario == nil || a.Engine.Scenario.Player == nil {
		return
	}
	if a.Mode != ModeGame && a.Mode != ModePaused {
		return
	}
	if a.CurrentScreen == ScreenFireControl {
		return
	}
	// On full TACTICAL the minimap is gone — still keep CM reachable under OWN SHIP.
	player := a.Engine.Scenario.Player
	decoyN := a.Engine.CM.DecoyLeft(player.ID)
	jitterN := a.Engine.CM.JitterLeft(player.ID)

	px, py, pw, ph := a.cmQuickPanelRect()
	render.DrawConsolePanel(screen, px, py, pw, ph)
	render.DrawText(screen, a.L(i18n.UICountermeas), px+12, py+20, render.ColorPlateLabel, true)
	render.DrawText(screen, a.L(i18n.UICMSubtitle), px+12, py+40, render.ColorPhosphorDim, true)

	render.DrawText(screen, fmt.Sprintf("%s  %d", a.L(i18n.UIADCLeft), decoyN), px+12, py+68, render.ColorPhosphor, true)
	render.DrawText(screen, fmt.Sprintf("%s  %d", a.L(i18n.UIJitterLeft), jitterN), px+12, py+88, render.ColorPhosphor, true)

	if decoyN <= 0 && jitterN <= 0 {
		render.DrawText(screen, a.L(i18n.UIMagazineEmpty), px+12, py+112, render.ColorWarn, true)
	}

	mx, my := ebiten.CursorPosition()
	buttons := a.cmQuickButtons(decoyN, jitterN)
	for _, b := range buttons {
		hover := b.contains(mx, my)
		pressed := a.uiPressedID == b.ID
		disabled := (b.ID == "cm_quick_decoy" && decoyN <= 0) || (b.ID == "cm_quick_jitter" && jitterN <= 0)
		if disabled {
			render.DrawBevelButtonDisabled(screen, b.X, b.Y, b.W, b.H, b.Label)
		} else {
			render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
		}
	}

	if a.uiTooltip != "" {
		for _, b := range buttons {
			if b.contains(mx, my) {
				render.DrawTooltip(screen, mx, my, a.uiTooltip)
				break
			}
		}
	}
}
