package ui

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/render"
)

type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmExitMenu
	confirmEndMission
	confirmRestartScenario
	confirmDeleteScenario
)

type confirmDialog struct {
	Kind    confirmKind
	Title   string
	Message string
}

func (a *App) confirmActive() bool {
	return a.confirm.Kind != confirmNone
}

func (a *App) showConfirm(kind confirmKind, title, message string) {
	a.confirm = confirmDialog{Kind: kind, Title: title, Message: message}
	a.markScenarioUIDirty()
}

func (a *App) dismissConfirm() {
	a.confirm = confirmDialog{}
}

func (a *App) updateConfirmDialog() bool {
	if !a.confirmActive() {
		return false
	}
	mx, my := ebiten.CursorPosition()
	yesX, yesY, noX, noY, w, h := a.confirmButtonRects()
	yesHover := mx >= yesX && mx < yesX+w && my >= yesY && my < yesY+h
	noHover := mx >= noX && mx < noX+w && my >= noY && my < noY+h
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyN) {
		a.dismissConfirm()
		return true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyY) {
		a.executeConfirmYes()
		return true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if yesHover {
			a.uiPressedID = "confirm_yes"
			a.uiPressedAt = time.Now()
			a.executeConfirmYes()
			return true
		}
		if noHover {
			a.uiPressedID = "confirm_no"
			a.uiPressedAt = time.Now()
			a.dismissConfirm()
			return true
		}
	}
	_ = noHover
	return true
}

func (a *App) executeConfirmYes() {
	kind := a.confirm.Kind
	a.dismissConfirm()
	switch kind {
	case confirmExitMenu:
		a.exitToMenuConfirmed()
	case confirmEndMission:
		a.endMissionConfirmed()
	case confirmRestartScenario:
		a.restartScenarioConfirmed()
	case confirmDeleteScenario:
		a.deleteScenarioConfirmed()
	}
}

func (a *App) confirmButtonRects() (yesX, yesY, noX, noY, w, h int) {
	w = render.ButtonWidth(a.L(i18n.UIYes), 24)
	if nw := render.ButtonWidth(a.L(i18n.UINo), 24); nw > w {
		w = nw
	}
	if cw := render.ButtonWidth(a.L(i18n.UICancel), 24); cw > w {
		w = cw
	}
	h = 36
	cx := render.ScreenW / 2
	yesX = cx - w - 12
	noX = cx + 12
	yesY = render.ScreenH/2 + 60
	noY = yesY
	return yesX, yesY, noX, noY, w, h
}

func (a *App) confirmPanelRect() (x, y, w, h int) {
	w = 640
	h = 260
	x = (render.ScreenW - w) / 2
	y = (render.ScreenH - h) / 2
	return x, y, w, h
}

func (a *App) drawConfirmDialog(screen *ebiten.Image) {
	if !a.confirmActive() {
		return
	}
	render.FillRect(screen, 0, 0, render.ScreenW, render.ScreenH, color.RGBA{0, 0, 0, 180})
	px, py, pw, ph := a.confirmPanelRect()
	render.DrawConsolePanel(screen, px, py, pw, ph)
	render.DrawTextLarge(screen, a.confirm.Title, px+24, py+36, render.ColorText)
	drawWrappedText(screen, a.confirm.Message, px+24, py+72, pw-48, 16)
	yesX, yesY, noX, noY, bw, bh := a.confirmButtonRects()
	mx, my := ebiten.CursorPosition()
	yesHover := mx >= yesX && mx < yesX+bw && my >= yesY && my < yesY+bh
	noHover := mx >= noX && mx < noX+bw && my >= noY && my < noY+bh
	yesPressed := a.uiPressedID == "confirm_yes" && time.Since(a.uiPressedAt) < 120*time.Millisecond
	noPressed := a.uiPressedID == "confirm_no" && time.Since(a.uiPressedAt) < 120*time.Millisecond
	render.DrawBevelButton(screen, yesX, yesY, bw, bh, a.L(i18n.UIYes), yesHover, yesPressed)
	render.DrawBevelButton(screen, noX, noY, bw, bh, a.L(i18n.UINo), noHover, noPressed)
}
