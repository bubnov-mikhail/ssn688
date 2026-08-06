package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/world"
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
	once sync.Once
	btns []uiButton
}

func ownShipPanelRect() (x, y, w, h int) {
	return render.ScreenW - ownShipPanelX, ownShipPanelY, ownShipPanelW, ownShipPanelH
}

func ownShipButtons() []uiButton {
	cachedOwnShipButtons.once.Do(func() {
		px, py, pw, _ := ownShipPanelRect()
		pad := 12
		innerX := px + pad
		innerW := pw - pad*2

		// Three equal control bands under the title.
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

		btns := make([]uiButton, 0, 12)
		btns = append(btns, mkRow(band0+28, []struct{ id, label, tip string }{
			{"own_eot_up", "▲", "Ring up — next ahead bell"},
			{"own_eot_stop", "STOP", "All stop"},
			{"own_eot_down", "▼", "Ring down — toward astern"},
		})...)
		btns = append(btns, mkRow(band1+28, []struct{ id, label, tip string }{
			{"hdg_port10", "◄◄", "Come left 10 degrees"},
			{"hdg_port", "◄", "Come left 5 degrees"},
			{"hdg_stbd", "►", "Come right 5 degrees"},
			{"hdg_stbd10", "►►", "Come right 10 degrees"},
		})...)
		btns = append(btns, mkRow(band2+28, []struct{ id, label, tip string }{
			{"dep_shallow", "▲", "Rise 20 feet"},
			{"dep_deep", "▼", "Dive 20 feet"},
			{"dep_periscope", "PD", "Periscope depth"},
			{"dep_hold", "HOLD", "Hold present depth"},
		})...)
		cachedOwnShipButtons.btns = btns
	})
	return cachedOwnShipButtons.btns
}

func (a *App) ownShipPanelVisible() bool {
	return a.Mode == ModeGame || a.Mode == ModePaused
}

func (a *App) updateOwnShipPanel(player *world.Entity) {
	if a.CurrentScreen == ScreenManeuver || player == nil || !a.ownShipPanelVisible() {
		return
	}
	buttons := ownShipButtons()
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
				a.stepEOT(player, -1)
			case "own_eot_down":
				a.stepEOT(player, +1)
			case "own_eot_stop":
				a.ringEOT(player, 0)
			default:
				a.maneuverButtonAction(b.ID, player)
			}
			break
		}
	}
	if a.uiPressedID != "" && time.Since(a.uiPressedAt) > 120*time.Millisecond {
		// Don't clear IDs owned by other screens mid-frame; only own-ship presses.
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
	render.DrawText(screen, "OWN SHIP", px+12, py+18, render.ColorPlateLabel, true)

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

	actSpd := fmt.Sprintf("ACT %.1f", p.SpeedKts)
	ordSpd := fmt.Sprintf("ORD %.0f", p.OrderedSpeed)
	if p.SpeedKts < -0.05 {
		actSpd = fmt.Sprintf("ACT %.1f AST", -p.SpeedKts)
	}
	if p.OrderedSpeed < -0.05 {
		ordSpd = fmt.Sprintf("ORD %.0f AST", -p.OrderedSpeed)
	}
	drawBand(band0, "SPEED", actSpd, ordSpd)
	drawBand(band1, "COURSE", fmt.Sprintf("ACT %.0f°", p.HeadingDeg), fmt.Sprintf("ORD %.0f°", p.OrderedHead))
	drawBand(band2, "DEPTH", fmt.Sprintf("ACT %.0f", p.DepthFt), fmt.Sprintf("ORD %.0f", p.OrderedDepth))

	mx, my := ebiten.CursorPosition()
	for _, b := range ownShipButtons() {
		hover := b.contains(mx, my)
		pressed := a.uiPressedID == b.ID
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	}

	if a.uiTooltip != "" {
		// Only show if hovering an own-ship control (avoid stealing other screens' tips).
		for _, b := range ownShipButtons() {
			if b.contains(mx, my) {
				render.DrawTooltip(screen, mx, my, a.uiTooltip)
				break
			}
		}
	}
}
