package ui

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/layout"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const (
	waterfallMaxRows   = layout.WaterfallMaxRows
	waterfallSampleSec = layout.WaterfallSampleSec
	waterfallRowH      = layout.WaterfallRowH
	waterfallLabelW    = layout.WaterfallLabelW
	waterfallHeaderH   = layout.WaterfallHeaderH
	waterfallAxisH     = layout.WaterfallAxisH
)

func waterfallMonitorRGBA() (r, g, b, a byte) {
	c := render.ColorMonitorFace
	return c.R, c.G, c.B, 255
}

func fillWaterfallMonitorBG(pix []byte) {
	r, g, b, a := waterfallMonitorRGBA()
	for i := 0; i < len(pix)/4; i++ {
		off := i * 4
		pix[off] = r
		pix[off+1] = g
		pix[off+2] = b
		pix[off+3] = a
	}
}

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
	// Keep backing rows for reuse, but do not shrink len to 0 — PushCopy indexes
	// w.buf[w.pos] and treats a non-nil buf as already allocated.
	if w.buf != nil {
		w.buf = w.buf[:cap(w.buf)]
	}
	w.pos = 0
	w.n = 0
}

// PushCopy stores a copy of bearings into the ring (reuses per-slot slices).
func (w *BearingWaterfall) PushCopy(bearings []float64, heading float64) {
	if len(bearings) == 0 {
		return
	}
	if w.buf == nil || len(w.buf) == 0 {
		w.buf = make([]acoustics.BearingWaterfallRow, waterfallMaxRows)
		w.pos = 0
		w.n = 0
	}
	if w.pos < 0 || w.pos >= len(w.buf) {
		w.pos = 0
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
	onFocus := a.CurrentScreen == ScreenPassive || a.CurrentScreen == ScreenSpectrum
	// Off PASSIVE/SPECTRUM keep a thinner history so other screens aren't taxed.
	sampleInterval := waterfallSampleSec
	if !onFocus {
		sampleInterval = waterfallSampleSec * 3
	}
	if t-a.lastWaterfallSample < sampleInterval {
		return
	}
	// At most one sample per Update tick (high time-scale must not storm the GPU).
	a.lastWaterfallSample = t

	sonar := &a.Engine.Sonar
	player := a.Engine.Scenario.Player
	emitters := a.Engine.AcousticEmitters()
	if a.waterfallScratch == nil || len(a.waterfallScratch) != acoustics.BearingWaterfallBins {
		a.waterfallScratch = make([]float64, acoustics.BearingWaterfallBins)
	}

	// Sample both arrays every tick while on PASSIVE/SPECTRUM so switching
	// HULL↔TOWED does not rebuild from a thinner history (looks "half height").
	// Off those screens, keep the non-selected array warm at a lower rate.
	primary := sonar.PassiveArray
	secondary := acoustics.PassiveArrayHull
	if primary == acoustics.PassiveArrayHull {
		secondary = acoustics.PassiveArrayTowed
	}
	acoustics.BearingWaterfallInto(a.waterfallScratch, a.Engine.Acoustics, player, emitters, sonar, primary, t)
	a.bearingWaterfalls.ForArray(primary).PushCopy(a.waterfallScratch, player.HeadingDeg)

	sampleSecondary := onFocus
	if !onFocus {
		a.waterfallAltCounter++
		sampleSecondary = a.waterfallAltCounter%4 == 0
	}
	if sampleSecondary {
		acoustics.BearingWaterfallInto(a.waterfallScratch, a.Engine.Acoustics, player, emitters, sonar, secondary, t)
		a.bearingWaterfalls.ForArray(secondary).PushCopy(a.waterfallScratch, player.HeadingDeg)
	}

	a.waterfallPendingScroll = true
}

func (a *App) disposeWaterfallImages() {
	if a.waterfallImg != nil {
		a.waterfallImg.Dispose()
		a.waterfallImg = nil
	}
	a.waterfallPix = nil
}

func (a *App) scrollWaterfallPix(pix []byte, w, h int) {
	rowBytes := w * 4 * waterfallRowH
	rows := h / waterfallRowH
	if rows <= 1 {
		return
	}
	copy(pix[rowBytes:], pix[:rowBytes*(rows-1)])
	fillWaterfallMonitorBG(pix[:rowBytes])
}

func (a *App) ensureWaterfallImage() {
	w, h := layout.WaterfallPlotW, layout.WaterfallPlotH
	if a.waterfallImg == nil || a.waterfallImg.Bounds().Dx() != w || a.waterfallImg.Bounds().Dy() != h {
		a.disposeWaterfallImages()
		a.waterfallImg = ebiten.NewImage(w, h)
		a.waterfallPix = make([]byte, w*h*4)
		fillWaterfallMonitorBG(a.waterfallPix)
		a.waterfallImg.WritePixels(a.waterfallPix)
		a.waterfallPendingScroll = true
		a.waterfallFullRebuild = true
	}
}

func (a *App) waterfallRNG() *rand.Rand {
	if a.waterfallRng == nil {
		a.waterfallRng = rand.New(rand.NewSource(1))
	}
	seed := int64(a.lastWaterfallSample*1000) ^ int64(a.waterfallArray)*0x51f0
	a.waterfallRng.Seed(seed)
	return a.waterfallRng
}

func (a *App) paintWaterfallRow(pix []byte, w, py int, row *acoustics.BearingWaterfallRow, rng *rand.Rand) {
	bgR, bgG, bgB, bgA := waterfallMonitorRGBA()
	if row == nil || len(row.Bearings) == 0 {
		rowBytes := w * 4 * waterfallRowH
		fillWaterfallMonitorBG(pix[py*w*4 : py*w*4+rowBytes])
		return
	}
	bins := acoustics.BearingWaterfallBins
	for px := 0; px < w; px++ {
		off := (py*w + px) * 4
		bearing := waterfallDisplayBearingDeg(px, w)
		bi := int(bearing / 360 * float64(bins))
		if bi >= bins {
			bi = bins - 1
		}
		if bi >= len(row.Bearings) {
			pix[off] = bgR
			pix[off+1] = bgG
			pix[off+2] = bgB
			pix[off+3] = bgA
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
		smooth := power*0.78 + left*0.11 + right*0.11
		smooth *= 0.92 + 0.16*rng.Float64()
		intensity := waterfallSNRToIntensity(smooth)
		if intensity <= 0 {
			pix[off] = bgR
			pix[off+1] = bgG
			pix[off+2] = bgB
			pix[off+3] = bgA
			continue
		}
		clr := sonarHeatColorFast(intensity)
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
		pix[off+3] = 255
	}
}

func (a *App) rebuildWaterfallImage(sonar *acoustics.SonarState) {
	a.ensureWaterfallImage()
	w, h := layout.WaterfallPlotW, layout.WaterfallPlotH
	pix := a.waterfallPix
	wf := a.bearingWaterfalls.ForArray(sonar.PassiveArray)
	rng := a.waterfallRNG()

	if a.waterfallFullRebuild || a.waterfallArray != sonar.PassiveArray {
		fillWaterfallMonitorBG(pix)
		n := wf.Len()
		if n > h {
			n = h
		}
		for ri := 0; ri < n; ri++ {
			a.paintWaterfallRow(pix, w, ri, wf.Row(ri), rng)
		}
		a.waterfallFullRebuild = false
		a.waterfallImg.WritePixels(pix)
	} else if a.waterfallPendingScroll {
		a.scrollWaterfallPix(pix, w, h)
		a.paintWaterfallRow(pix, w, 0, wf.Latest(), rng)
		a.waterfallImg.WritePixels(pix)
	}

	a.waterfallPendingScroll = false
	a.waterfallArray = sonar.PassiveArray
	a.waterfallStamp = a.lastWaterfallSample
}

func (a *App) drawBearingWaterfall(screen *ebiten.Image, sonar *acoustics.SonarState) {
	x := layout.WaterfallPanelX
	y := layout.WaterfallPanelY
	h := layout.WaterfallPanelH
	plotX := layout.WaterfallPlotX
	plotY := layout.WaterfallPlotY
	plotW := layout.WaterfallPlotW
	plotH := layout.WaterfallPlotH

	if a.waterfallPendingScroll || a.waterfallFullRebuild || a.waterfallImg == nil || a.waterfallArray != sonar.PassiveArray {
		a.rebuildWaterfallImage(sonar)
	}
	if a.waterfallImg != nil {
		render.DrawImageAt(screen, a.waterfallImg, plotX, plotY)
	}

	totalMin := float64(plotH*waterfallRowH) * waterfallSampleSec / 60
	midMin := totalMin * 0.5
	render.DrawText(screen, a.L(i18n.UINow), x+4, plotY+10, render.ColorPhosphorDim, true)
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

	if sonar.PassiveArray == acoustics.PassiveArrayTowed && sonar.TowedDamaged {
		msg := "TOWED ARRAY DAMAGED — NO DATA"
		tw := len(msg) * 6
		render.DrawText(screen, msg, plotX+(plotW-tw)/2, plotY+plotH/2, render.ColorWarn, false)
	}
	if a.Engine != nil && a.Engine.Scenario.Player != nil {
		p := a.Engine.Scenario.Player
		p.EnsureDamage()
		if sonar.PassiveArray == acoustics.PassiveArrayHull && p.Damage.Destroyed(world.SysPassiveHull) {
			msg := "HULL ARRAY DAMAGED — NO DATA"
			tw := len(msg) * 6
			render.DrawText(screen, msg, plotX+(plotW-tw)/2, plotY+plotH/2, render.ColorWarn, false)
		}
	}
}
