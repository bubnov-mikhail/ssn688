package ui

import (
	"fmt"
	"image/color"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/world"
)

func phosphorNoiseSeed() int64 {
	return time.Now().UnixNano() / int64(40*time.Millisecond)
}

const (
	waterfallMaxRows   = 210
	waterfallSampleSec = 0.22
	waterfallRowH      = 2
)

// BearingWaterfall stores recent omnidirectional bearing snapshots; index 0 is newest (top).
type BearingWaterfall struct {
	rows []acoustics.BearingWaterfallRow
}

// BearingWaterfallBank keeps separate histories for hull and towed arrays.
type BearingWaterfallBank struct {
	Hull  BearingWaterfall
	Towed BearingWaterfall
}

func (b *BearingWaterfallBank) Reset() {
	b.Hull.Reset()
	b.Towed.Reset()
}

func (b *BearingWaterfallBank) ForArray(array acoustics.PassiveArrayKind) *BearingWaterfall {
	if array == acoustics.PassiveArrayTowed {
		return &b.Towed
	}
	return &b.Hull
}

func (w *BearingWaterfall) Reset() {
	w.rows = w.rows[:0]
}

func (w *BearingWaterfall) Push(row acoustics.BearingWaterfallRow) {
	if len(row.Bearings) == 0 {
		return
	}
	w.rows = append([]acoustics.BearingWaterfallRow{row}, w.rows...)
	if len(w.rows) > waterfallMaxRows {
		w.rows = w.rows[:waterfallMaxRows]
	}
}

func (w *BearingWaterfall) Latest() *acoustics.BearingWaterfallRow {
	if len(w.rows) == 0 {
		return nil
	}
	return &w.rows[0]
}

func (a *App) updateBearingWaterfall() {
	if a.Engine == nil || a.Mode == ModePaused || !a.Engine.Sonar.PassiveEnabled {
		return
	}
	sonar := &a.Engine.Sonar
	t := a.Engine.Clock.GameTime
	for t-a.lastWaterfallSample >= waterfallSampleSec {
		a.lastWaterfallSample += waterfallSampleSec
		player := a.Engine.Scenario.Player
		emitters := a.Engine.Scenario.AllEntities()
		// Keep both array histories warm so HULL/TOWED switches are instantaneous.
		for _, array := range []acoustics.PassiveArrayKind{acoustics.PassiveArrayHull, acoustics.PassiveArrayTowed} {
			row := acoustics.BearingWaterfallSlice(
				a.Engine.Acoustics, player, emitters, sonar, array, t,
			)
			a.bearingWaterfalls.ForArray(array).Push(row)
		}
	}
}

func (a *App) drawBearingWaterfall(screen *ebiten.Image, player *world.Entity, sonar *acoustics.SonarState) {
	const (
		x      = 952
		y      = 310
		w      = 316
		h      = 412
		axisH  = 18
		labelW = 28
	)
	plotX := x + labelW
	plotY := y
	plotW := w - labelW
	plotH := h - axisH

	render.FillRect(screen, x, y, w, h, color.RGBA{0, 2, 16, 255})
	render.DrawLine(screen, float64(x), float64(y), float64(x+w), float64(y), render.ColorBevelLight)
	render.DrawLine(screen, float64(x), float64(y+h), float64(x+w), float64(y+h), render.ColorBevelDark)
	render.DrawLine(screen, float64(x), float64(y), float64(x), float64(y+h), render.ColorBevelLight)
	render.DrawLine(screen, float64(x+w), float64(y), float64(x+w), float64(y+h), render.ColorBevelDark)

	noiseRng := rand.New(rand.NewSource(phosphorNoiseSeed() ^ 0x51f0))

	render.DrawText(screen, "NOW", x+2, plotY+10, render.ColorPhosphorDim, true)
	render.DrawText(screen, "T-", x+4, plotY+plotH-8, render.ColorPhosphorDim, true)

	bins := acoustics.BearingWaterfallBins
	binW := float64(plotW) / float64(bins)
	maxRows := plotH / waterfallRowH
	rows := a.bearingWaterfalls.ForArray(sonar.PassiveArray).rows
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}

	for ri, row := range rows {
		py := plotY + ri*waterfallRowH
		for bi := 0; bi < bins && bi < len(row.Bearings); bi++ {
			power := row.Bearings[bi]
			floor := 0.35 + noiseRng.Float64()*0.9
			left, right := 0.0, 0.0
			if bi > 0 {
				left = row.Bearings[bi-1]
			}
			if bi+1 < len(row.Bearings) {
				right = row.Bearings[bi+1]
			}
			smooth := power*0.55 + left*0.225 + right*0.225
			if smooth < floor {
				smooth = floor
			}
			smooth *= 0.84 + 0.28*noiseRng.Float64()
			px := plotX + int(float64(bi)*binW)
			intensity := snrToIntensity(smooth)
			bw := int(binW)
			if bw < 1 {
				bw = 1
			}
			render.FillRect(screen, px, py, bw, waterfallRowH, sonarHeatColor(intensity))
		}
		hx := plotX + acoustics.HeadingToWaterfallX(row.Heading, plotW)
		render.FillRect(screen, hx, py, 2, waterfallRowH, color.RGBA{255, 220, 80, 160})
	}

	for _, deg := range []int{0, 90, 180, 270} {
		gx := plotX + acoustics.HeadingToWaterfallX(float64(deg), plotW)
		render.DrawLine(screen, float64(gx), float64(plotY), float64(gx), float64(plotY+plotH), color.RGBA{0, 90, 50, 140})
	}
	render.DrawText(screen, "000", plotX, y+h-4, render.ColorPhosphorDim, true)
	render.DrawText(screen, "090", plotX+plotW/4-10, y+h-4, render.ColorPhosphorDim, true)
	render.DrawText(screen, "180", plotX+plotW/2-10, y+h-4, render.ColorPhosphorDim, true)
	render.DrawText(screen, "270", plotX+3*plotW/4-10, y+h-4, render.ColorPhosphorDim, true)

	arrayLabel := "HULL"
	if sonar.PassiveArray == acoustics.PassiveArrayTowed {
		arrayLabel = "TOWED"
	}
	render.DrawText(screen, fmt.Sprintf("HDG %.0f°  |  %s", player.HeadingDeg, arrayLabel), x+labelW, y-2, render.ColorAmber, true)
	render.DrawText(screen, "BEARING WATERFALL", x, y-18, render.ColorPhosphorDim, true)

	a.drawActiveSonarWashWaterfall(screen, sonar, plotX, plotY, plotW, plotH)
}

func (a *App) drawActiveSonarWashWaterfall(screen *ebiten.Image, sonar *acoustics.SonarState, plotX, plotY, plotW, plotH int) {
	wash := a.activeSonarWashIntensity(sonar)
	if wash < 0.02 {
		return
	}
	for row := 0; row < plotH; row++ {
		rowFade := 1.0 - float64(row)/float64(plotH)*0.55
		aRow := uint8(wash * rowFade * 55)
		if aRow < 4 {
			continue
		}
		render.FillRect(screen, plotX, plotY+row, plotW, 1, color.RGBA{160, 220, 255, aRow})
	}
	if wash > 0.25 {
		bandH := int(28 + wash*40)
		if bandH > plotH {
			bandH = plotH
		}
		render.FillRect(screen, plotX, plotY, plotW, bandH, color.RGBA{200, 240, 255, uint8(wash*70)})
	}
}
