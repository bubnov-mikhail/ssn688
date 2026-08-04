package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/world"
)

const mapRangeLabelBearingDeg = 45.0

func formatMapRangeLabel(rangeYd float64) string {
	nm := rangeYd / world.YardsPerNM
	if rangeYd >= 1000 {
		return fmt.Sprintf("%.0fk yd / %.1f nm", rangeYd/1000, nm)
	}
	return fmt.Sprintf("%.0f yd / %.2f nm", rangeYd, nm)
}

func drawMapRangeRingLabel(screen *ebiten.Image, cx, cy, radiusPx, rangeYd float64, clr color.Color) {
	if screen == nil || radiusPx < 8 {
		return
	}
	rad := mapRangeLabelBearingDeg * math.Pi / 180
	lx := int(cx + radiusPx*math.Sin(rad) + 5)
	ly := int(cy - radiusPx*math.Cos(rad) + 4)
	render.DrawText(screen, formatMapRangeLabel(rangeYd), lx, ly, clr, true)
}
