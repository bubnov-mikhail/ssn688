package ui

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/campaign"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/weapons"
)

const (
	scenarioLoadoutH       = 228
	scenarioLoadoutRowH    = 28
	scenarioLoadoutTube0   = 52
	scenarioOrdnanceMenuH  = 22
	scenarioLoadoutOrdBtnW = 88 // fits "Harpoon ▼" with padding
)

func scenarioLoadoutY() int {
	return scenarioPanelY + scenarioPanelH - 16 - 40 - 12 - scenarioLoadoutH
}

func (a *App) scenarioLoadoutRect() (x, y, w, h int) {
	x = scenarioDetailX()
	y = scenarioLoadoutY()
	w = scenarioDetailW
	h = scenarioLoadoutH
	return x, y, w, h
}

func (a *App) resetScenarioLoadout() {
	a.LoadoutMix = 0.25
	a.LoadoutTubes = campaign.DefaultTubeLoadout()
	a.loadoutOrdnanceMenuTube = 0
}

func (a *App) ensureLoadoutTubes() {
	empty := true
	for i := range a.LoadoutTubes {
		if a.LoadoutTubes[i] != "" {
			empty = false
			break
		}
	}
	if empty {
		a.LoadoutTubes = campaign.DefaultTubeLoadout()
	}
}

func (a *App) scenarioLoadoutSliderTrack() (x, y, w, h int) {
	lx, ly, lw, _ := a.scenarioLoadoutRect()
	pad := scenarioDetailPad
	x = lx + pad + 56
	y = ly + scenarioLoadoutH - 38
	w = lw - 2*pad - 112
	h = 14
	return x, y, w, h
}

func (a *App) scenarioLoadoutTubeY(tube int) int {
	_, ly, _, _ := a.scenarioLoadoutRect()
	return ly + scenarioLoadoutTube0 + (tube-1)*scenarioLoadoutRowH
}

func (a *App) scenarioLoadoutOrdnanceBtn(tube, rowY int) sonarUIButton {
	label := "Mk48 ▼"
	if tube >= 1 && tube <= 4 {
		ord := weapons.NormalizeOrdnance(a.LoadoutTubes[tube-1])
		if ord != "" {
			label = ord + " ▼"
		}
	}
	const btnH = 24
	padX := scenarioDetailX() + scenarioDetailPad
	return sonarUIButton{
		ID:    fmt.Sprintf("loadout_ord_%d", tube),
		Label: label,
		X:     padX + 82,
		Y:     rowY + 4 - btnH/2,
		W:     scenarioLoadoutOrdBtnW,
		H:     btnH,
	}
}

func (a *App) scenarioLoadoutOrdnanceMenu(tube, rowY int) []sonarUIButton {
	if a.loadoutOrdnanceMenuTube != tube {
		return nil
	}
	ordBtn := a.scenarioLoadoutOrdnanceBtn(tube, rowY)
	menuY := ordBtn.Y + ordBtn.H + 2
	var btns []sonarUIButton
	for i, ord := range weapons.AllTubeOrdnance() {
		btns = append(btns, sonarUIButton{
			ID:    fmt.Sprintf("loadout_ord_%d_pick_%s", tube, ord),
			Label: ord,
			X:     ordBtn.X,
			Y:     menuY + i*scenarioOrdnanceMenuH,
			W:     ordBtn.W,
			H:     scenarioOrdnanceMenuH - 1,
		})
	}
	return btns
}

func (a *App) handleScenarioLoadoutInput(mx, my int) bool {
	a.ensureLoadoutTubes()
	lx, ly, lw, lh := a.scenarioLoadoutRect()
	if mx < lx || mx >= lx+lw || my < ly || my >= ly+lh {
		return false
	}

	sx, sy, sw, sh := a.scenarioLoadoutSliderTrack()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && hitRect(mx, my, sx, sy-4, sw, sh+8) {
		a.loadoutDragging = true
	}
	if a.loadoutDragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			t := float64(mx-sx) / float64(sw)
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			a.LoadoutMix = t
		} else {
			a.loadoutDragging = false
		}
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return lx <= mx && mx < lx+lw && my >= ly && my < ly+lh
	}

	for tube := 1; tube <= 4; tube++ {
		rowY := a.scenarioLoadoutTubeY(tube)
		for _, b := range a.scenarioLoadoutOrdnanceMenu(tube, rowY) {
			if b.contains(mx, my) {
				a.LoadoutTubes[tube-1] = weapons.NormalizeOrdnance(b.Label)
				a.loadoutOrdnanceMenuTube = 0
				a.uiPressedID = b.ID
				a.uiPressedAt = time.Now()
				return true
			}
		}
		ordBtn := a.scenarioLoadoutOrdnanceBtn(tube, rowY)
		if ordBtn.contains(mx, my) {
			if a.loadoutOrdnanceMenuTube == tube {
				a.loadoutOrdnanceMenuTube = 0
			} else {
				a.loadoutOrdnanceMenuTube = tube
			}
			a.uiPressedID = ordBtn.ID
			a.uiPressedAt = time.Now()
			return true
		}
	}
	if a.loadoutOrdnanceMenuTube != 0 {
		a.loadoutOrdnanceMenuTube = 0
	}
	return true
}

func (a *App) drawScenarioLoadoutGroup(screen *ebiten.Image, x, y, w, h int) {
	render.FillRect(screen, x, y, w, h, render.ColorPanel)
	border := render.ColorPanelStroke
	render.FillRect(screen, x, y, w, 1, border)
	render.FillRect(screen, x, y+h-1, w, 1, border)
	render.FillRect(screen, x, y, 1, h, border)
	render.FillRect(screen, x+w-1, y, 1, h, border)
	render.DrawTextLarge(screen, "WEAPON LOADOUT", x+10, y+30, render.ColorPlateLabel)
}

func (a *App) drawScenarioLoadoutOrdnanceMenu(screen *ebiten.Image, mx, my int) {
	if a.loadoutOrdnanceMenuTube == 0 {
		return
	}
	tube := a.loadoutOrdnanceMenuTube
	rowY := a.scenarioLoadoutTubeY(tube)
	btns := a.scenarioLoadoutOrdnanceMenu(tube, rowY)
	if len(btns) == 0 {
		return
	}
	menuX := btns[0].X - 2
	menuY := btns[0].Y - 2
	menuW := btns[0].W + 4
	menuH := btns[len(btns)-1].Y + btns[len(btns)-1].H - menuY + 2
	render.DrawMonitor(screen, menuX, menuY, menuW, menuH)
	render.FillRect(screen, menuX+2, menuY+2, menuW-4, menuH-4, render.ColorPanel)
	for _, b := range btns {
		h := b.contains(mx, my)
		p := a.uiPressedID == b.ID && time.Since(a.uiPressedAt) < 120*time.Millisecond
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, h, p)
	}
}

func (a *App) drawScenarioLoadout(screen *ebiten.Image) {
	a.ensureLoadoutTubes()
	fc := campaign.PreviewFireControl(a.LoadoutTubes, a.LoadoutMix)
	lx, ly, lw, lh := a.scenarioLoadoutRect()
	padX := lx + scenarioDetailPad
	a.drawScenarioLoadoutGroup(screen, lx, ly, lw, lh)

	mx, my := ebiten.CursorPosition()
	for tube := 1; tube <= 4; tube++ {
		rowY := a.scenarioLoadoutTubeY(tube)
		render.DrawText(screen, fmt.Sprintf("TUBE %d", tube), padX, rowY+4, render.ColorText, true)
		ordBtn := a.scenarioLoadoutOrdnanceBtn(tube, rowY)
		hover := ordBtn.contains(mx, my)
		pressed := a.uiPressedID == ordBtn.ID && time.Since(a.uiPressedAt) < 120*time.Millisecond
		render.DrawBevelButton(screen, ordBtn.X, ordBtn.Y, ordBtn.W, ordBtn.H, ordBtn.Label, hover, pressed)
		render.DrawText(screen, "LOADED", ordBtn.X+ordBtn.W+10, rowY+4, render.ColorDim, true)
	}

	sx, sy, sw, sh := a.scenarioLoadoutSliderTrack()
	render.DrawText(screen, fmt.Sprintf("MAGAZINE: %d Mk48  ·  %d Harpoon", fc.MagazineLeft, fc.HarpoonMagLeft),
		padX, sy-10, render.ColorPlateLabel, true)
	render.DrawText(screen, "Mk48", padX, sy+12, render.ColorSonar, true)
	render.DrawText(screen, "Harpoon", sx+sw+12, sy+12, render.ColorActive, true)
	render.DrawMonitor(screen, sx-4, sy-4, sw+8, sh+8)
	render.FillRect(screen, sx, sy, sw, sh, render.ColorPanelInset)
	for i := 0; i <= 4; i++ {
		tx := sx + i*sw/4
		render.DrawLine(screen, float64(tx), float64(sy), float64(tx), float64(sy+sh), render.ColorGrid)
	}
	knobW := 12
	knobX := sx + int(a.LoadoutMix*float64(sw-knobW))
	if knobX < sx {
		knobX = sx
	}
	if knobX > sx+sw-knobW {
		knobX = sx + sw - knobW
	}
	render.FillRect(screen, knobX, sy-2, knobW, sh+4, render.ColorHighlight)

	a.drawScenarioLoadoutOrdnanceMenu(screen, mx, my)
}
