package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ssn688/sim/internal/layout"
	"image/color"
)

// DrawConsoleBackdrop fills the screen with the submarine console backdrop.
func DrawConsoleBackdrop(screen *ebiten.Image) {
	screen.Fill(ColorBG)
}

// DrawConsolePanel draws a flat dark-gray instrument panel.
func DrawConsolePanel(screen *ebiten.Image, x, y, w, h int) {
	FillRect(screen, x, y, w, h, ColorPanel)
	strokeRect(screen, x, y, w, h, ColorPanelStroke)
}

// DrawMonitor draws a recessed display area (no bright bezel).
func DrawMonitor(screen *ebiten.Image, x, y, w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	FillRect(screen, x, y, w, h, ColorMonitorFace)
}

// DrawPassiveConsole draws the PASSIVE screen panel chrome (no textures, no leader lines).
func DrawPassiveConsole(screen *ebiten.Image) {
	DrawConsolePanel(screen, layout.PassiveMainPanelX, layout.PassiveMainPanelY, layout.PassiveMainPanelW, layout.PassiveMainPanelH)
	DrawConsolePanel(screen, layout.PassiveSidePanelX, layout.PassiveSidePanelY, layout.PassiveSidePanelW, layout.PassiveSidePanelH)
	DrawMonitor(screen, layout.WaterfallPanelX, layout.WaterfallPanelY, layout.WaterfallPanelW, layout.WaterfallPanelH)
	DrawMonitor(screen, layout.PassiveListX, layout.PassiveListY, layout.PassiveListW, layout.PassiveListH)
	DrawMonitor(screen, layout.PassiveTowedStatusX, layout.PassiveTowedStatusY, layout.PassiveTowedStatusW, layout.PassiveTowedStatusH)
}

func strokeRect(screen *ebiten.Image, x, y, w, h int, clr color.Color) {
	FillRect(screen, x, y, w, 1, clr)
	FillRect(screen, x, y+h-1, w, 1, clr)
	FillRect(screen, x, y, 1, h, clr)
	FillRect(screen, x+w-1, y, 1, h, clr)
}
