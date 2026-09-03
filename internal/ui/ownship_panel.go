package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const (
	ownShipPanelX = 300 // inset from right: ScreenW - 300 (same slot as old OBJECTIVES)
	ownShipPanelW = 290
	ownShipPanelY = 50
	ownShipPanelH = 200

	ownShipBtnH   = 26
	ownShipBtnGap = 4
)

var cachedOwnShipButtons struct {
	mu   sync.Mutex
	lang string
	btns []uiButton
}

func ownShipPanelRect() (x, y, w, h int) {
	return render.ScreenW - ownShipPanelX, ownShipPanelY, ownShipPanelW, ownShipPanelH
}

func (a *App) ownShipButtons() []uiButton {
	lang := a.Lang()
	cachedOwnShipButtons.mu.Lock()
	defer cachedOwnShipButtons.mu.Unlock()
	if cachedOwnShipButtons.lang == lang && cachedOwnShipButtons.btns != nil {
		return cachedOwnShipButtons.btns
	}

	px, py, pw, _ := ownShipPanelRect()
	pad := 12
	innerX := px + pad
	innerW := pw - pad*2

	band0 := py + 28
	bandH := 54
	band1 := band0 + bandH
	band2 := band1 + bandH

	mkRow := func(y int, specs []struct{ id, label, tip string }) []uiButton {
		out := make([]uiButton, len(specs))
		total := 0
		for i, s := range specs {
			out[i] = uiButton{ID: s.id, Label: s.label, Tooltip: s.tip, Y: y, H: ownShipBtnH}
			out[i].W = render.ButtonWidth(s.label, 10)
			total += out[i].W
		}
		total += ownShipBtnGap * (len(specs) - 1)
		x := innerX + (innerW-total)/2
		for i := range out {
			out[i].X = x
			x += out[i].W + ownShipBtnGap
		}
		return out
	}

	L := func(t i18n.TranslatedText) string { return t.GetText(lang) }
	btns := make([]uiButton, 0, 12)
	btns = append(btns, mkRow(band0+28, []struct{ id, label, tip string }{
		{"own_eot_up", "▲", L(i18n.UITipEOTUp)},
		{"own_eot_stop", L(i18n.UIStop), L(i18n.UITipEOTStop)},
		{"own_eot_down", "▼", L(i18n.UITipEOTDown)},
	})...)
	btns = append(btns, mkRow(band1+28, []struct{ id, label, tip string }{
		{"hdg_port10", "◄◄", L(i18n.UITipPort10)},
		{"hdg_port", "◄", L(i18n.UITipPort5)},
		{"hdg_stbd", "►", L(i18n.UITipStbd5)},
		{"hdg_stbd10", "►►", L(i18n.UITipStbd10)},
	})...)
	btns = append(btns, mkRow(band2+28, []struct{ id, label, tip string }{
		{"dep_shallow", "▲", L(i18n.UITipShallow)},
		{"dep_deep", "▼", L(i18n.UITipDeep)},
		{"dep_periscope", L(i18n.UIPD), L(i18n.UITipPD)},
		{"dep_hold", L(i18n.UIHold), L(i18n.UITipHoldDep)},
	})...)
	cachedOwnShipButtons.lang = lang
	cachedOwnShipButtons.btns = btns
	return btns
}

func (a *App) ownShipPanelVisible() bool {
	return a.Mode == ModeGame || a.Mode == ModePaused
}

func (a *App) updateOwnShipPanel(player *world.Entity) {
	if a.CurrentScreen == ScreenManeuver || player == nil || !a.ownShipPanelVisible() {
		return
	}
	buttons := a.ownShipButtons()
	mx, my := ebiten.CursorPosition()
	a.updateButtonTooltips(buttons, mx, my)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for _, b := range buttons {
			if !b.contains(mx, my) {
				continue
			}
			a.uiPressedID = b.ID
			a.uiPressedAt = time.Now()
			switch b.ID {
			case "own_eot_up":
				a.nudgeOrderedSpeed(player, +1)
			case "own_eot_down":
				a.nudgeOrderedSpeed(player, -1)
			case "own_eot_stop":
				a.ringEOT(player, 0)
			default:
				a.maneuverButtonAction(b.ID, player)
			}
			break
		}
	}
	if a.uiPressedID != "" && time.Since(a.uiPressedAt) > 120*time.Millisecond {
		switch a.uiPressedID {
		case "own_eot_up", "own_eot_down", "own_eot_stop",
			"hdg_port10", "hdg_port", "hdg_stbd", "hdg_stbd10",
			"dep_shallow", "dep_deep", "dep_periscope", "dep_hold":
			a.uiPressedID = ""
		}
	}
}

func (a *App) drawOwnShipPanel(screen *ebiten.Image) {
	if a.CurrentScreen == ScreenManeuver || a.Engine == nil || a.Engine.Scenario == nil {
		return
	}
	p := a.Engine.Scenario.Player
	if p == nil {
		return
	}

	px, py, pw, ph := ownShipPanelRect()
	render.DrawConsolePanel(screen, px, py, pw, ph)
	render.DrawText(screen, a.L(i18n.UIOwnShip), px+12, py+18, render.ColorPlateLabel, true)

	pad := 12
	innerX := px + pad
	band0 := py + 28
	bandH := 54
	band1 := band0 + bandH
	band2 := band1 + bandH

	drawBand := func(y int, title, act, ord string) {
		render.DrawText(screen, title, innerX, y+10, render.ColorPlateLabel, true)
		render.DrawText(screen, act, innerX, y+22, render.ColorPhosphor, true)
		render.DrawText(screen, ord, innerX+130, y+22, render.ColorAmber, true)
	}

	actLbl := a.L(i18n.UIAct)
	ordLbl := a.L(i18n.UIOrd)
	astLbl := a.L(i18n.UIAST)
	actSpd := fmt.Sprintf("%s %.1f", actLbl, p.SpeedKts)
	ordSpd := fmt.Sprintf("%s %.0f", ordLbl, p.OrderedSpeed)
	if p.SpeedKts < -0.05 {
		actSpd = fmt.Sprintf("%s %.1f %s", actLbl, -p.SpeedKts, astLbl)
	}
	if p.OrderedSpeed < -0.05 {
		ordSpd = fmt.Sprintf("%s %.0f %s", ordLbl, -p.OrderedSpeed, astLbl)
	}
	drawBand(band0, a.L(i18n.UISpeed), actSpd, ordSpd)
	drawBand(band1, a.L(i18n.UICourse), fmt.Sprintf("%s %.0f°", actLbl, p.HeadingDeg), fmt.Sprintf("%s %.0f°", ordLbl, p.OrderedHead))
	drawBand(band2, a.L(i18n.UIDepth), fmt.Sprintf("%s %.0f", actLbl, p.DepthFt), fmt.Sprintf("%s %.0f", ordLbl, p.OrderedDepth))

	mx, my := ebiten.CursorPosition()
	// Latch STOP only when ordered speed is exactly all-stop; intermediate
	// speeds (from ±1 nudges) leave the telegraph row unlit.
	speedLatched := matchingEOTID(p.OrderedSpeed, p) == "eot_stop"
	for _, b := range a.ownShipButtons() {
		hover := b.contains(mx, my)
		pressed := a.uiPressedID == b.ID || (b.ID == "own_eot_stop" && speedLatched)
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	}

	if a.uiTooltip != "" {
		for _, b := range a.ownShipButtons() {
			if b.contains(mx, my) {
				render.DrawTooltip(screen, mx, my, a.uiTooltip)
				break
			}
		}
	}
}
