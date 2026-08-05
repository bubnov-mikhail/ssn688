package ui

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/render"
)

const (
	navBarH           = 58
	navTooltipDelay   = 400 * time.Millisecond
)

type navItem struct {
	Screen  Screen
	Icon    int
	Title   string
	Tooltip string
	Hotkey  string
}

func (a *App) navBarY() int {
	return render.ScreenH - navBarH
}

func (a *App) navItems() []navItem {
	return []navItem{
		{ScreenPassive, render.IconPassive, "PASSIVE", "Passive sonar — towed array display", "F1"},
		{ScreenActive, render.IconActive, "ACTIVE", "Active sonar — transmit and receive", "F2"},
		{ScreenSpectrum, render.IconSpectrum, "SPECTRUM", "Spectrum analyzer — classify by frequency", "F3"},
		{ScreenLibrary, render.IconLibrary, "LIBRARY", "Threat library — platforms, weapons, tactics", "F4"},
		{ScreenFireControl, render.IconWeapons, "WEPS", "Fire control — torpedo tubes and launch", "F5"},
		{ScreenManeuver, render.IconManeuver, "HELM", "Maneuvering room — speed, course, depth", "F6"},
		{ScreenDamage, render.IconDamage, "DC", "Damage control — systems status and repair", "F7"},
		{ScreenTactical, render.IconTactical, "PLOT", "Tactical plot — overview of the battlespace", "M"},
	}
}

func (a *App) navSlotWidth() int {
	return render.ScreenW / len(a.navItems())
}

func (a *App) navItemRect(index int) (x, y, w, h int) {
	w = a.navSlotWidth()
	x = index * w
	y = a.navBarY()
	h = navBarH
	return
}

func (a *App) updateNavBar() {
	if a.Engine == nil {
		return
	}
	mx, my := ebiten.CursorPosition()
	hoverIdx := -1
	for i := range a.navItems() {
		x, y, w, h := a.navItemRect(i)
		if mx >= x && mx < x+w && my >= y && my < y+h {
			hoverIdx = i
			break
		}
	}

	now := time.Now()
	if hoverIdx != a.navHoverIdx {
		a.navHoverIdx = hoverIdx
		a.navHoverSince = now
		a.navTooltip = ""
	}
	if hoverIdx >= 0 && now.Sub(a.navHoverSince) >= navTooltipDelay {
		it := a.navItems()[hoverIdx]
		a.navTooltip = it.Title + " — " + it.Tooltip + " (" + it.Hotkey + ")"
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && hoverIdx >= 0 {
		a.CurrentScreen = a.navItems()[hoverIdx].Screen
	}
}

func (a *App) drawNavBar(screen *ebiten.Image) {
	if a.Engine == nil {
		return
	}

	y := a.navBarY()
	render.FillRect(screen, 0, y, render.ScreenW, navBarH, render.ColorPanelBezel)
	render.FillRect(screen, 0, y+2, render.ScreenW, navBarH-2, render.ColorPanelDark)
	render.DrawLine(screen, 0, float64(y), float64(render.ScreenW), float64(y), render.ColorBevelLight)

	items := a.navItems()
	mx, my := ebiten.CursorPosition()
	slotW := a.navSlotWidth()

	for i, it := range items {
		x := i * slotW
		active := a.CurrentScreen == it.Screen
		hover := i == a.navHoverIdx && my >= y

		if active {
			render.FillRect(screen, x+2, y+4, slotW-4, navBarH-6, render.ColorPanelMid)
			render.DrawLine(screen, float64(x+2), float64(y+4), float64(x+slotW-2), float64(y+4), render.ColorPhosphor)
		} else if hover {
			render.FillRect(screen, x+4, y+6, slotW-8, navBarH-10, render.ColorPanelMid)
		}

		clr := render.ColorPhosphorDim
		if active || hover {
			clr = render.ColorPhosphor
		}

		cx := x + slotW/2
		cy := y + navBarH/2 - 4
		render.DrawScreenIcon(screen, it.Icon, cx, cy, 26, clr)

		labelW := len(it.Title) * 6
		render.DrawText(screen, it.Title, cx-labelW/2, y+navBarH-12, clr, true)
	}

	if a.navTooltip != "" {
		tx := mx + 12
		ty := y - 36
		if tx+len(a.navTooltip)*6 > render.ScreenW-20 {
			tx = render.ScreenW - len(a.navTooltip)*6 - 20
		}
		render.DrawTooltip(screen, tx, ty, a.navTooltip)
	}
}
