package ui

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/render"
)

const (
	navBarH         = 58
	navIconGap      = 6
	navSlotPad      = 4
	navTooltipDelay = 400 * time.Millisecond
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
		{ScreenMast, render.IconMast, "MAST", "ESM / COMM / periscope — intercept, traffic, optic", "F8"},
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

func navItemContentLayout(slotX, slotW, barY int, title string) (iconCX, iconCY, labelX, labelY int) {
	iconSz := render.NavBarIconSize
	labelW := render.SmallLabelWidth(title)
	contentW := iconSz + navIconGap + labelW
	left := slotX + (slotW-contentW)/2
	minLeft := slotX + navSlotPad
	if left < minLeft {
		left = minLeft
	}
	maxLeft := slotX + slotW - navSlotPad - contentW
	if left > maxLeft {
		left = maxLeft
	}
	iconCX = left + iconSz/2
	iconCY = barY + navBarH/2
	labelX = left + iconSz + navIconGap
	labelY = render.SmallLabelBaseline(barY, navBarH)
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
		a.clearDCTabAlertIfOnDamage()
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
		dcAlert := it.Screen == ScreenDamage && a.dcTabAlert
		dcBlink := dcAlert && a.dcTabBlinkOn()

		if active {
			render.FillRect(screen, x+2, y+4, slotW-4, navBarH-6, render.ColorPanelMid)
			render.DrawLine(screen, float64(x+2), float64(y+4), float64(x+slotW-2), float64(y+4), render.ColorPhosphor)
		} else if dcBlink {
			render.FillRect(screen, x+2, y+4, slotW-4, navBarH-6, render.ColorDanger)
			render.DrawLine(screen, float64(x+2), float64(y+4), float64(x+slotW-2), float64(y+4), render.ColorWarn)
		} else if dcAlert {
			render.FillRect(screen, x+4, y+6, slotW-8, navBarH-10, render.ColorPanelMid)
			render.DrawLine(screen, float64(x+4), float64(y+6), float64(x+slotW-4), float64(y+6), render.ColorDanger)
		} else if hover {
			render.FillRect(screen, x+4, y+6, slotW-8, navBarH-10, render.ColorPanelMid)
		}

		clr := render.ColorPhosphorDim
		if active || hover {
			clr = render.ColorPhosphor
		}
		if dcAlert {
			if dcBlink {
				clr = render.ColorWarn
			} else {
				clr = render.ColorDanger
			}
		}

		iconCX, iconCY, labelX, labelY := navItemContentLayout(x, slotW, y, it.Title)
		render.DrawScreenIcon(screen, it.Icon, iconCX, iconCY, render.NavBarIconSize, clr)
		render.DrawText(screen, it.Title, labelX, labelY, clr, true)
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
