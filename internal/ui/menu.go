package ui

import (
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/version"
)

type menuItem struct {
	Label  string
	Action int
}

const (
	menuActionScenarios = iota
	menuActionImportScenario
	menuActionLoad
	menuActionSettings
	menuActionQuit
)

func (a *App) menuItems() []menuItem {
	return []menuItem{
		{Label: a.L(i18n.UIMenuNewGame), Action: menuActionScenarios},
		{Label: a.L(i18n.UIMenuImportScenario), Action: menuActionImportScenario},
		{Label: a.L(i18n.UIMenuLoadGame), Action: menuActionLoad},
		{Label: a.L(i18n.UIMenuSettings), Action: menuActionSettings},
		{Label: a.L(i18n.UIMenuQuit), Action: menuActionQuit},
	}
}

func (a *App) menuButtonWidth() int {
	w := 0
	for _, item := range a.menuItems() {
		bw := render.ButtonWidth(item.Label, 40)
		if bw > w {
			w = bw
		}
	}
	return w
}

func (a *App) menuButtonRect(index int) (x, y, w, h int) {
	const (
		btnH   = 48
		gap    = 14
		startY = 360
	)
	w = a.menuButtonWidth()
	h = btnH
	x = (render.ScreenW - w) / 2
	y = startY + index*(btnH+gap)
	return x, y, w, h
}

func (a *App) menuIndexAt(mx, my int) int {
	for i := range a.menuItems() {
		x, y, w, h := a.menuButtonRect(i)
		if mx >= x && mx < x+w && my >= y && my < y+h {
			return i
		}
	}
	return -1
}

func (a *App) activateMenuItem(index int) error {
	switch a.menuItems()[index].Action {
	case menuActionScenarios:
		a.beginScenarioUI()
		a.Mode = ModeScenarioList
		a.ScenarioListIndex = 0
		a.ensureScenarioSelection()
		a.StatusMessage = ""
	case menuActionImportScenario:
		a.importScenarioFromOS()
	case menuActionLoad:
		a.refreshLoadList()
		a.Mode = ModeLoad
		a.LoadIndex = 0
		a.StatusMessage = ""
	case menuActionSettings:
		a.Mode = ModeSettings
	case menuActionQuit:
		return errQuit
	}
	return nil
}

func (a *App) updateMenu() error {
	items := a.menuItems()
	mx, my := ebiten.CursorPosition()
	hover := a.menuIndexAt(mx, my)
	if hover >= 0 {
		a.MenuIndex = hover
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		a.MenuIndex = (a.MenuIndex + 1) % len(items)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		a.MenuIndex = (a.MenuIndex + len(items) - 1) % len(items)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		return a.activateMenuItem(a.MenuIndex)
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && hover >= 0 {
		a.uiPressedID = "menu"
		a.uiPressedAt = time.Now()
		return a.activateMenuItem(hover)
	}
	return nil
}

func menuTitleLines() []string {
	const breakAt = "Submarine "
	if i := strings.Index(version.Title, breakAt); i >= 0 {
		return []string{
			version.Title[:i+len("Submarine")],
			version.Title[i+len(breakAt):],
		}
	}
	return []string{version.Title}
}

func (a *App) drawMenu(screen *ebiten.Image) {
	render.DrawMenuBackground(screen)

	const titleLineH = 44
	titleY := 88
	for _, line := range menuTitleLines() {
		lineW := render.TitleWidth(line)
		render.DrawTextTitle(screen, line, (render.ScreenW-lineW)/2, titleY, render.ColorText)
		titleY += titleLineH
	}

	mx, my := ebiten.CursorPosition()
	for i, item := range a.menuItems() {
		x, y, w, h := a.menuButtonRect(i)
		hover := i == a.MenuIndex || a.menuIndexAt(mx, my) == i
		pressed := a.uiPressedID == "menu" && i == a.MenuIndex && time.Since(a.uiPressedAt) < 120*time.Millisecond
		if i == a.MenuIndex {
			render.FillRect(screen, x-4, y-4, w+8, h+8, color.RGBA{0, 180, 140, 60})
		}
		render.DrawBevelButton(screen, x, y, w, h, item.Label, hover, pressed)
	}

	ver := version.Display()
	verW := render.SmallLabelWidth(ver)
	render.DrawText(screen, ver, (render.ScreenW-verW)/2, render.ScreenH-20, render.ColorPhosphorDim, true)
}
