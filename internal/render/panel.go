package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// 688(I)-style phosphor palette for helm panels.
var (
	ColorPanelDark   = color.RGBA{6, 14, 12, 255}
	ColorPanelMid    = color.RGBA{14, 36, 30, 255}
	ColorPanelBezel  = color.RGBA{24, 58, 48, 255}
	ColorPanelInset  = color.RGBA{4, 10, 8, 255}
	ColorPhosphor    = color.RGBA{0, 255, 153, 255}
	ColorPhosphorDim = color.RGBA{0, 170, 110, 255}
	ColorAmber       = color.RGBA{255, 191, 64, 255}
	ColorBevelLight  = color.RGBA{70, 130, 110, 255}
	ColorBevelDark   = color.RGBA{8, 20, 16, 255}
)

// DrawPanel draws a recessed instrument panel with bezel.
func DrawPanel(screen *ebiten.Image, x, y, w, h int) {
	FillRect(screen, x, y, w, h, ColorPanelBezel)
	FillRect(screen, x+3, y+3, w-6, h-6, ColorPanelDark)
	FillRect(screen, x+6, y+6, w-12, h-12, ColorPanelMid)
	DrawLine(screen, float64(x+6), float64(y+6), float64(x+w-7), float64(y+6), ColorBevelLight)
	DrawLine(screen, float64(x+6), float64(y+6), float64(x+6), float64(y+h-7), ColorBevelLight)
	DrawLine(screen, float64(x+w-7), float64(y+6), float64(x+w-7), float64(y+h-7), ColorBevelDark)
	DrawLine(screen, float64(x+6), float64(y+h-7), float64(x+w-7), float64(y+h-7), ColorBevelDark)
}

// DrawBevelButton draws a clickable 688-style push button.
func DrawBevelButton(screen *ebiten.Image, x, y, w, h int, label string, hovered, pressed bool) {
	face := ColorPanelMid
	if hovered {
		face = color.RGBA{30, 68, 54, 255}
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
	ty := y + h/2 + 4
	DrawButtonText(screen, label, tx, ty, clr)
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
