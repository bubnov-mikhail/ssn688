package ui

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/render"
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
		{ScreenPassive, render.IconPassive, a.L(i18n.UINavPassive), a.L(i18n.UINavTipPassive), "F1"},
		{ScreenActive, render.IconActive, a.L(i18n.UINavActive), a.L(i18n.UINavTipActive), "F2"},
		{ScreenSpectrum, render.IconSpectrum, a.L(i18n.UINavSpectrum), a.L(i18n.UINavTipSpectrum), "F3"},
		{ScreenLibrary, render.IconLibrary, a.L(i18n.UINavLibrary), a.L(i18n.UINavTipLibrary), "F4"},
		{ScreenFireControl, render.IconWeapons, a.L(i18n.UINavWeps), a.L(i18n.UINavTipWeps), "F5"},
		{ScreenManeuver, render.IconManeuver, a.L(i18n.UINavHelm), a.L(i18n.UINavTipHelm), "F6"},
		{ScreenMast, render.IconMast, a.L(i18n.UINavMast), a.L(i18n.UINavTipMast), "F8"},
		{ScreenDamage, render.IconDamage, a.L(i18n.UINavDC), a.L(i18n.UINavTipDC), "F7"},
		{ScreenTactical, render.IconTactical, a.L(i18n.UINavPlot), a.L(i18n.UINavTipPlot), "M"},
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
		mastAlert := it.Screen == ScreenMast && a.mastTabAlert
		mastBlink := mastAlert && a.mastTabBlinkOn()
		tabAlert := dcAlert || mastAlert
		tabBlink := dcBlink || mastBlink

		if active {
			render.FillRect(screen, x+2, y+4, slotW-4, navBarH-6, render.ColorPanelMid)
			render.DrawLine(screen, float64(x+2), float64(y+4), float64(x+slotW-2), float64(y+4), render.ColorPhosphor)
		} else if tabBlink {
			render.FillRect(screen, x+2, y+4, slotW-4, navBarH-6, render.ColorDanger)
			render.DrawLine(screen, float64(x+2), float64(y+4), float64(x+slotW-2), float64(y+4), render.ColorWarn)
		} else if tabAlert {
			render.FillRect(screen, x+4, y+6, slotW-8, navBarH-10, render.ColorPanelMid)
			render.DrawLine(screen, float64(x+4), float64(y+6), float64(x+slotW-4), float64(y+6), render.ColorDanger)
		} else if hover {
			render.FillRect(screen, x+4, y+6, slotW-8, navBarH-10, render.ColorPanelMid)
		}

		clr := render.ColorPhosphorDim
		if active || hover {
			clr = render.ColorPhosphor
		}
		if tabAlert {
			if tabBlink {
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
