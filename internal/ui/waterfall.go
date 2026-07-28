package ui

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/world"
)

const (
	waterfallMaxRows   = 400
	waterfallSampleSec = 0.15
	waterfallRowH      = 1
	waterfallPanelX    = 20
	waterfallPanelY    = 108
	waterfallPanelW    = 900
	waterfallPanelH    = 590
	waterfallLabelW    = 46
	waterfallHeaderH   = 56
	waterfallAxisH     = 22
	waterfallPlotX     = waterfallPanelX + waterfallLabelW
	waterfallPlotY     = waterfallPanelY + waterfallHeaderH
	waterfallPlotW     = waterfallPanelW - waterfallLabelW - 8
	waterfallPlotH     = waterfallPanelH - waterfallHeaderH - waterfallAxisH
)

func waterfallBearingDisplayX(bearing float64, plotW int) int {
	b := math.Mod(bearing+180, 360)
	if b < 0 {
		b += 360
	}
	x := int(b / 360 * float64(plotW))
	if x < 0 {
		return 0
	}
	if x >= plotW {
		return plotW - 1
	}
	return x
}

func waterfallDisplayBearingDeg(px, plotW int) float64 {
	if plotW <= 0 {
		return 0
	}
	b := math.Mod(float64(px)/float64(plotW)*360+180, 360)
	if b < 0 {
		b += 360
	}
	return b
}

// BearingWaterfall stores recent omnidirectional bearing snapshots in a ring
// (index 0 via Row(0) is newest / top of display).
type BearingWaterfall struct {
	buf []acoustics.BearingWaterfallRow
	pos int // next write slot
	n   int
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
	w.buf = w.buf[:0]
	w.pos = 0
	w.n = 0
}

// PushCopy stores a copy of bearings into the ring (reuses per-slot slices).
func (w *BearingWaterfall) PushCopy(bearings []float64, heading float64) {
	if len(bearings) == 0 {
		return
	}
	if w.buf == nil {
		w.buf = make([]acoustics.BearingWaterfallRow, waterfallMaxRows)
	}
	slot := &w.buf[w.pos]
	if slot.Bearings == nil || len(slot.Bearings) != len(bearings) {
		slot.Bearings = make([]float64, len(bearings))
	}
	copy(slot.Bearings, bearings)
	slot.Heading = heading
	w.pos++
	if w.pos >= len(w.buf) {
		w.pos = 0
	}
	if w.n < len(w.buf) {
		w.n++
	}
}

func (w *BearingWaterfall) Len() int { return w.n }

func (w *BearingWaterfall) Latest() *acoustics.BearingWaterfallRow {
	return w.Row(0)
}

// Row returns the snapshot of the given age (0 = newest). Nil if out of range.
func (w *BearingWaterfall) Row(age int) *acoustics.BearingWaterfallRow {
	if age < 0 || age >= w.n {
		return nil
	}
	i := w.pos - 1 - age
	for i < 0 {
		i += len(w.buf)
	}
	return &w.buf[i]
}

func (a *App) updateBearingWaterfall() {
	if a.Engine == nil || a.Mode == ModePaused || !a.Engine.Sonar.PassiveEnabled {
		return
	}
	t := a.Engine.Clock.GameTime
	// Advance the clock even off-screen so we never catch up hundreds of samples in one frame.
	if t-a.lastWaterfallSample < waterfallSampleSec {
		return
	}
	// At most one sample per Update tick (high time-scale must not storm the GPU).
	a.lastWaterfallSample = t

	sonar := &a.Engine.Sonar
	player := a.Engine.Scenario.Player
	emitters := a.Engine.Scenario.AllEntities()
	if a.waterfallScratch == nil || len(a.waterfallScratch) != acoustics.BearingWaterfallBins {
		a.waterfallScratch = make([]float64, acoustics.BearingWaterfallBins)
	}

	// Sample the displayed array every tick; keep the other warm at 1/3 rate.
	primary := sonar.PassiveArray
	secondary := acoustics.PassiveArrayHull
	if primary == acoustics.PassiveArrayHull {
		secondary = acoustics.PassiveArrayTowed
	}
	acoustics.BearingWaterfallInto(a.waterfallScratch, a.Engine.Acoustics, player, emitters, sonar, primary, t)
	a.bearingWaterfalls.ForArray(primary).PushCopy(a.waterfallScratch, player.HeadingDeg)

	a.waterfallAltCounter++
	if a.waterfallAltCounter%3 == 0 {
		acoustics.BearingWaterfallInto(a.waterfallScratch, a.Engine.Acoustics, player, emitters, sonar, secondary, t)
		a.bearingWaterfalls.ForArray(secondary).PushCopy(a.waterfallScratch, player.HeadingDeg)
	}

	a.waterfallPendingScroll = true
	if a.CurrentScreen == ScreenPassive || a.CurrentScreen == ScreenSpectrum {
		a.passivePPIPending = true
	}
}

func (a *App) ensureWaterfallImage() {
	w, h := waterfallPlotW, waterfallPlotH
	if a.waterfallImg == nil || a.waterfallImg.Bounds().Dx() != w || a.waterfallImg.Bounds().Dy() != h {
		a.waterfallImg = ebiten.NewImage(w, h)
		a.waterfallPix = make([]byte, w*h*4)
		for i := 0; i < w*h; i++ {
			off := i * 4
			a.waterfallPix[off+1] = 2
			a.waterfallPix[off+2] = 16
			a.waterfallPix[off+3] = 255
		}
		a.waterfallImg.WritePixels(a.waterfallPix)
		a.waterfallPendingScroll = true
		a.waterfallFullRebuild = true
	}
}

func (a *App) paintWaterfallRow(pix []byte, w, py int, row *acoustics.BearingWaterfallRow, rng *rand.Rand) {
	if row == nil || len(row.Bearings) == 0 {
		return
	}
	bins := acoustics.BearingWaterfallBins
	for px := 0; px < w; px++ {
		bearing := waterfallDisplayBearingDeg(px, w)
		bi := int(bearing / 360 * float64(bins))
		if bi >= bins {
			bi = bins - 1
		}
		if bi >= len(row.Bearings) {
			continue
		}
		power := row.Bearings[bi]
		left, right := power, power
		if bi > 0 {
			left = row.Bearings[bi-1]
		}
		if bi+1 < len(row.Bearings) {
			right = row.Bearings[bi+1]
		}
		smooth := power*0.55 + left*0.225 + right*0.225
		floor := 0.35 + rng.Float64()*0.9
		if smooth < floor {
			smooth = floor
		}
		smooth *= 0.84 + 0.28*rng.Float64()
		clr := sonarHeatColorFast(snrToIntensity(smooth))
		off := (py*w + px) * 4
		pix[off] = clr.R
		pix[off+1] = clr.G
		pix[off+2] = clr.B
		pix[off+3] = 255
	}
	hx := waterfallBearingDisplayX(row.Heading, w)
	if hx < 0 {
		hx = 0
	}
	if hx > w-2 {
		hx = w - 2
	}
	for dx := 0; dx < 2; dx++ {
		off := (py*w + hx + dx) * 4
		pix[off] = 255
		pix[off+1] = 220
		pix[off+2] = 80
		pix[off+3] = 200
	}
}

func (a *App) rebuildWaterfallImage(sonar *acoustics.SonarState) {
	a.ensureWaterfallImage()
	w, h := waterfallPlotW, waterfallPlotH
	pix := a.waterfallPix
	wf := a.bearingWaterfalls.ForArray(sonar.PassiveArray)
	rng := rand.New(rand.NewSource(int64(a.lastWaterfallSample*1000) ^ int64(sonar.PassiveArray)*0x51f0))

	if a.waterfallFullRebuild || a.waterfallArray != sonar.PassiveArray {
		for i := 0; i < w*h; i++ {
			off := i * 4
			pix[off] = 0
			pix[off+1] = 2
			pix[off+2] = 16
			pix[off+3] = 255
		}
		n := wf.Len()
		if n > h {
			n = h
		}
		for ri := 0; ri < n; ri++ {
			a.paintWaterfallRow(pix, w, ri, wf.Row(ri), rng)
		}
		a.waterfallFullRebuild = false
	} else if a.waterfallPendingScroll {
		// Scroll history down one row; paint only the newest line at the top.
		rowBytes := w * 4
		copy(pix[rowBytes:], pix[:rowBytes*(h-1)])
		// Clear top row to navy before paint.
		for px := 0; px < w; px++ {
			off := px * 4
			pix[off] = 0
			pix[off+1] = 2
			pix[off+2] = 16
			pix[off+3] = 255
		}
		a.paintWaterfallRow(pix, w, 0, wf.Latest(), rng)
	}

	a.waterfallImg.WritePixels(pix)
	a.waterfallPendingScroll = false
	a.waterfallArray = sonar.PassiveArray
	a.waterfallStamp = a.lastWaterfallSample
}

func (a *App) drawBearingWaterfall(screen *ebiten.Image, player *world.Entity, sonar *acoustics.SonarState) {
	const (
	x      = waterfallPanelX
	y      = waterfallPanelY
	w      = waterfallPanelW
	h      = waterfallPanelH
	labelW = waterfallLabelW
	)
	plotX := waterfallPlotX
	plotY := waterfallPlotY
	plotW := waterfallPlotW
	plotH := waterfallPlotH

	render.FillRect(screen, x, y, w, h, color.RGBA{0, 2, 16, 255})
	render.DrawLine(screen, float64(x), float64(y), float64(x+w), float64(y), render.ColorBevelLight)
	render.DrawLine(screen, float64(x), float64(y+h), float64(x+w), float64(y+h), render.ColorBevelDark)
	render.DrawLine(screen, float64(x), float64(y), float64(x), float64(y+h), render.ColorBevelLight)
	render.DrawLine(screen, float64(x+w), float64(y), float64(x+w), float64(y+h), render.ColorBevelDark)

	if a.waterfallPendingScroll || a.waterfallFullRebuild || a.waterfallImg == nil || a.waterfallArray != sonar.PassiveArray {
		a.rebuildWaterfallImage(sonar)
	}
	if a.waterfallImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(plotX), float64(plotY))
		screen.DrawImage(a.waterfallImg, op)
	}

	totalMin := float64(plotH*waterfallRowH) * waterfallSampleSec / 60
	midMin := totalMin * 0.5
	render.DrawText(screen, "NOW", x+4, plotY+10, render.ColorPhosphorDim, true)
	render.DrawText(screen, fmt.Sprintf("T-%.1fm", totalMin*0.25), x+2, plotY+plotH/4+4, render.ColorPhosphorDim, true)
	render.DrawText(screen, fmt.Sprintf("T-%.1fm", midMin), x+2, plotY+plotH/2+4, render.ColorPhosphorDim, true)
	render.DrawText(screen, fmt.Sprintf("T-%.1fm", totalMin*0.75), x+2, plotY+3*plotH/4+4, render.ColorPhosphorDim, true)
	render.DrawText(screen, fmt.Sprintf("T-%.1fm", totalMin), x+2, plotY+plotH-8, render.ColorPhosphorDim, true)

	for _, deg := range []int{180, 270, 0, 90} {
		gx := plotX + waterfallBearingDisplayX(float64(deg), plotW)
		render.DrawLine(screen, float64(gx), float64(plotY), float64(gx), float64(plotY+plotH), color.RGBA{0, 90, 50, 140})
	}
	for _, frac := range []float64{0.25, 0.5, 0.75} {
		gy := plotY + int(float64(plotH)*frac)
		render.DrawLine(screen, float64(plotX), float64(gy), float64(plotX+plotW), float64(gy), color.RGBA{0, 85, 55, 90})
	}
	render.DrawText(screen, "180", plotX-10, y+h-4, render.ColorPhosphorDim, true)
	render.DrawText(screen, "270", plotX+plotW/4-10, y+h-4, render.ColorPhosphorDim, true)
	render.DrawText(screen, "000", plotX+plotW/2-10, y+h-4, render.ColorPhosphor, true)
	render.DrawText(screen, "090", plotX+3*plotW/4-10, y+h-4, render.ColorPhosphorDim, true)
	render.DrawText(screen, "180", plotX+plotW-18, y+h-4, render.ColorPhosphorDim, true)

	arrayLabel := "HULL"
	if sonar.PassiveArray == acoustics.PassiveArrayTowed {
		arrayLabel = "TOWED"
	}
	render.DrawText(screen, fmt.Sprintf("BEARING WATERFALL  |  HDG %.0f°  |  SPD %.1f kts  |  %s", player.HeadingDeg, player.SpeedKts, arrayLabel), x+8, y+20, render.ColorAmber, true)
}
