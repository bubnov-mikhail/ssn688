package ui

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/platform"
	"github.com/bubnov-mikhail/ssn688/internal/render"
)

const (
	loadListX      = 360
	loadListY      = 200
	loadRowH       = 36
	loadMaxVisible = 12
	loadBtnW       = 160
	loadBtnH       = 48
	loadBtnGap     = 16
)

func (a *App) loadActionButtonRects() (loadX, backX, y, w, h int) {
	w = loadBtnW
	h = loadBtnH
	loadX = 980
	backX = loadX
	y = 280
	return loadX, backX, y, w, h
}

func (a *App) loadListHitIndex(mx, my int) int {
	if len(a.LoadFiles) == 0 {
		return -1
	}
	if mx < loadListX || mx > loadListX+520 {
		return -1
	}
	rel := my - loadListY
	if rel < 0 {
		return -1
	}
	idx := rel / loadRowH
	if idx < 0 || idx >= len(a.LoadFiles) || idx >= loadMaxVisible {
		return -1
	}
	return idx
}

func (a *App) updateLoad() {
	mx, my := ebiten.CursorPosition()
	loadX, backX, btnY, btnW, btnH := a.loadActionButtonRects()
	backY := btnY + btnH + loadBtnGap

	if hover := a.loadListHitIndex(mx, my); hover >= 0 {
		a.LoadIndex = hover
	}

	if len(a.LoadFiles) > 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			a.LoadIndex = (a.LoadIndex + 1) % len(a.LoadFiles)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			a.LoadIndex = (a.LoadIndex + len(a.LoadFiles) - 1) % len(a.LoadFiles)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			a.loadSelectedSave()
			return
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.Mode = ModeMenu
		return
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	if hit := a.loadListHitIndex(mx, my); hit >= 0 {
		a.LoadIndex = hit
		a.uiPressedID = "load_row"
		a.uiPressedAt = time.Now()
		return
	}
	if a.headerHit(mx, my, loadX, btnY, btnW, btnH) {
		a.uiPressedID = "load_do"
		a.uiPressedAt = time.Now()
		a.loadSelectedSave()
		return
	}
	if a.headerHit(mx, my, backX, backY, btnW, btnH) {
		a.uiPressedID = "load_back"
		a.uiPressedAt = time.Now()
		a.Mode = ModeMenu
	}
}

func (a *App) drawLoad(screen *ebiten.Image) {
	render.DrawMenuBackground(screen)

	title := a.L(i18n.UILoadGameTitle)
	titleW := len(title) * 14
	render.DrawTextLarge(screen, title, (render.ScreenW-titleW)/2, 100, render.ColorText)

	if len(a.LoadFiles) == 0 {
		render.DrawText(screen, a.L(i18n.UINoSavesFound), loadListX, loadListY, render.ColorWarn, false)
	} else {
		mx, my := ebiten.CursorPosition()
		for i, f := range a.LoadFiles {
			if i >= loadMaxVisible {
				break
			}
			y := loadListY + i*loadRowH
			selected := i == a.LoadIndex
			hovered := a.loadListHitIndex(mx, my) == i
			if selected {
				render.FillRect(screen, loadListX-12, y-8, 540, loadRowH, render.ColorPanelInset)
			} else if hovered {
				render.FillRect(screen, loadListX-12, y-8, 540, loadRowH, render.ColorPanelMid)
			}
			clr := render.ColorDim
			if selected {
				clr = render.ColorHighlight
			}
			render.DrawText(screen, filepath.Base(f), loadListX, y+16, clr, false)
		}
		if len(a.LoadFiles) > loadMaxVisible {
			render.DrawText(screen, fmt.Sprintf("…and %d more", len(a.LoadFiles)-loadMaxVisible),
				loadListX, loadListY+loadMaxVisible*loadRowH+8, render.ColorDim, true)
		}
	}

	loadX, backX, btnY, btnW, btnH := a.loadActionButtonRects()
	backY := btnY + btnH + loadBtnGap
	mx, my := ebiten.CursorPosition()
	loadHover := a.headerHit(mx, my, loadX, btnY, btnW, btnH)
	backHover := a.headerHit(mx, my, backX, backY, btnW, btnH)
	loadPressed := a.uiPressedID == "load_do" && time.Since(a.uiPressedAt) < 120*time.Millisecond
	backPressed := a.uiPressedID == "load_back" && time.Since(a.uiPressedAt) < 120*time.Millisecond
	render.DrawBevelButton(screen, loadX, btnY, btnW, btnH, a.L(i18n.UILoad), loadHover, loadPressed)
	render.DrawBevelButton(screen, backX, backY, btnW, btnH, a.L(i18n.UIBack), backHover, backPressed)

	if a.StatusMessage != "" {
		render.DrawText(screen, a.displayStatus(), loadListX, 700, render.ColorWarn, false)
	}

	hint := "CLICK A SAVE, THEN LOAD  ·  ESC BACK"
	if platform.Mobile() {
		hint = "TAP A SAVE, THEN LOAD"
	}
	hintW := len(hint) * 7
	render.DrawText(screen, hint, (render.ScreenW-hintW)/2, 780, render.ColorPhosphorDim, true)
}
