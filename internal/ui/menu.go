package ui

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/render"
)

type menuItem struct {
	Label  string
	Action int
}

const (
	menuActionNewGame = iota
	menuActionLoad
	menuActionSettings
	menuActionQuit
)

func (a *App) menuItems() []menuItem {
	return []menuItem{
		{Label: "NEW GAME", Action: menuActionNewGame},
		{Label: "LOAD GAME", Action: menuActionLoad},
		{Label: "SETTINGS", Action: menuActionSettings},
		{Label: "QUIT", Action: menuActionQuit},
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
		startY = 340
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
	case menuActionNewGame:
		a.StartNewGame()
	case menuActionLoad:
		a.refreshLoadList()
		a.Mode = ModeLoad
		a.LoadIndex = 0
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

func (a *App) drawMenu(screen *ebiten.Image) {
	render.DrawMenuBackground(screen)

	title := "SSN-688(I) HUNTER/KILLER"
	titleW := len(title) * 14
	render.DrawTextLarge(screen, title, (render.ScreenW-titleW)/2, 108, render.ColorText)

	subtitle := "MODERN SUBMARINE COMBAT SIMULATOR"
	subW := len(subtitle) * 8
	render.DrawText(screen, subtitle, (render.ScreenW-subW)/2, 158, render.ColorPhosphorDim, false)

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

	hint := "CLICK OR UP/DOWN + ENTER"
	hintW := len(hint) * 7
	render.DrawText(screen, hint, (render.ScreenW-hintW)/2, 620, render.ColorPhosphorDim, true)
}
