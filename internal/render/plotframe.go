package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Neutral gray behind the chart raster (PLOT / sim player).
var ColorPlotBackground = color.RGBA{108, 108, 110, 255}

// FillPlotBackground clears the full map panel to neutral gray.
func FillPlotBackground(screen *ebiten.Image, outerX, outerY, outerW, outerH int) {
	FillRect(screen, outerX, outerY, outerW, outerH, ColorPlotBackground)
}
