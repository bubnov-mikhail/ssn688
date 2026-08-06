package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Ballast-board console palette (dark gray panels, light engraved labels, green sonar data).
var (
	ColorPanelDark   = color.RGBA{18, 18, 20, 255}
	ColorPanelMid    = color.RGBA{28, 28, 30, 255}
	ColorPanelBezel  = color.RGBA{34, 34, 36, 255}
	ColorPanelInset  = color.RGBA{10, 10, 12, 255}
	ColorPhosphor    = color.RGBA{0, 255, 153, 255}
	ColorPhosphorDim = color.RGBA{0, 170, 110, 255}
	ColorAmber       = color.RGBA{255, 191, 64, 255}
	ColorBevelLight  = color.RGBA{58, 58, 62, 255}
	ColorBevelDark   = color.RGBA{22, 22, 24, 255}
	ColorPlateLabel  = color.RGBA{178, 180, 186, 255}
)

// DrawPanel draws a flat console panel.
func DrawPanel(screen *ebiten.Image, x, y, w, h int) {
	DrawConsolePanel(screen, x, y, w, h)
}

// DrawBevelButton draws a clickable 688-style push button.
func DrawBevelButton(screen *ebiten.Image, x, y, w, h int, label string, hovered, pressed bool) {
	face := ColorPanelMid
	if hovered {
		face = color.RGBA{38, 38, 42, 255}
	}
	if pressed {
		face = ColorPanelInset
	}
	FillRect(screen, x, y, w, h, ColorBevelDark)
	inset := 2
	if pressed {
		inset = 3
	}
	FillRect(screen, x+inset, y+inset, w-inset*2, h-inset*2, face)
	if !pressed {
		DrawLine(screen, float64(x+inset), float64(y+inset), float64(x+w-inset-1), float64(y+inset), ColorBevelLight)
		DrawLine(screen, float64(x+inset), float64(y+inset), float64(x+inset), float64(y+h-inset-1), ColorBevelLight)
	}
	clr := ColorPhosphor
	if pressed {
		clr = ColorPhosphorDim
	}
	tw := ButtonLabelWidth(label)
	tx := x + (w-tw)/2
	ty := ButtonLabelBaseline(y, h)
	DrawButtonText(screen, label, tx, ty, clr)
}

// DrawBevelButtonDisabled draws a non-interactive button face.
func DrawBevelButtonDisabled(screen *ebiten.Image, x, y, w, h int, label string) {
	FillRect(screen, x, y, w, h, ColorBevelDark)
	FillRect(screen, x+2, y+2, w-4, h-4, ColorPanelInset)
	tw := ButtonLabelWidth(label)
	DrawButtonText(screen, label, x+(w-tw)/2, ButtonLabelBaseline(y, h), ColorDim)
}

// DrawTooltip shows a help bubble near the cursor.
func DrawTooltip(screen *ebiten.Image, mx, my int, text string) {
	if text == "" {
		return
	}
	pad := 8
	tw := len(text)*7 + pad*2
	th := 28
	tx := mx + 14
	ty := my - th - 8
	if tx+tw > ScreenW-10 {
		tx = ScreenW - tw - 10
	}
	if ty < 50 {
		ty = my + 20
	}
	FillRect(screen, tx, ty, tw, th, color.RGBA{0, 0, 0, 220})
	DrawLine(screen, float64(tx), float64(ty), float64(tx+tw), float64(ty), ColorAmber)
	DrawText(screen, text, tx+pad, ty+20, ColorAmber, true)
}
