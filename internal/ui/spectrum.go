package ui

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/layout"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/world"
)

const (
	spectrumScreenPanelX = 20
	spectrumScreenPanelY = 50
	spectrumScreenPanelW = 1260
	spectrumScreenPanelH = 700

	spectrumTableX           = 40
	spectrumTableY           = 278
	spectrumTableW           = 450
	spectrumTableRow         = 22
	spectrumTableVisibleRows = 15
	spectrumChartX           = 510
	spectrumChartW           = spectrumScreenPanelX + spectrumScreenPanelW - spectrumChartX - 20
	spectrumChartLabelInset  = 14
	spectrumSigPanelH        = 118
	spectrumGap              = 10
	spectrumRefYOffset       = 52
	spectrumObsGap           = 36

	spectrumArrayLabelX = 40
	spectrumArrayLabelY = 106
)

func spectrumTableHeight() int {
	return spectrumTableRow*(spectrumTableVisibleRows+1) + 44
}

func spectrumRefY() int {
	return spectrumTableY + spectrumRefYOffset
}

func spectrumObsY() int {
	return spectrumRefY() + spectrumSigPanelH + spectrumObsGap
}

// spectrumRefNavY places PROFILE / < > / nameplate in the gray plate above the Hz ruler.
func spectrumRefNavY() int {
	return spectrumRefY() - 12 - 6 - refNavBtnH
}

var (
	cwSigGreen     = color.RGBA{0, 255, 70, 255}
	cwSigGreenDim  = color.RGBA{0, 160, 50, 180}
	cwSigGreenGlow = color.RGBA{0, 255, 70, 70}
	cwSigNoise     = color.RGBA{0, 90, 35, 40}
	cwSigPanelBG   = color.RGBA{0, 0, 0, 255}
)

func (a *App) spectrumContactAt(mx, my int, sonar *acoustics.SonarState) *acoustics.Contact {
	a.contactTableScroll.spectrum = clampContactTableScroll(a.contactTableScroll.spectrum, len(sonar.Contacts), spectrumTableVisibleRows)
	start, end := contactTableWindow(len(sonar.Contacts), a.contactTableScroll.spectrum, spectrumTableVisibleRows)
	y := spectrumTableY + spectrumTableRow
	for i := start; i < end; i++ {
		if mx >= spectrumTableX && mx < spectrumTableX+spectrumTableW && my >= y && my < y+spectrumTableRow {
			return &sonar.Contacts[i]
		}
		y += spectrumTableRow
	}
	return nil
}

func (a *App) referenceProfile() world.SignatureProfile {
	if a.referenceProfileIdx < 0 || a.referenceProfileIdx >= len(world.SignatureLibrary) {
		return world.SignatureLibrary[0]
	}
	return world.SignatureLibrary[a.referenceProfileIdx]
}

const (
	refNavBtnW     = 32
	refNavBtnH     = 28
	refLabelPad    = 16
	refNavGap      = 4
	refProfileGap  = 12
	refProfileText = "PROFILE:"
)

var referenceLabelWidthOnce struct {
	once sync.Once
	w    int
}

func referenceLabelWidth() int {
	referenceLabelWidthOnce.once.Do(func() {
		w := 160
		for _, p := range world.SignatureLibrary {
			tw := render.ButtonLabelWidth(p.Name) + refLabelPad
			if tw > w {
				w = tw
			}
		}
		referenceLabelWidthOnce.w = w
	})
	return referenceLabelWidthOnce.w
}

func referenceProfileWidth() int {
	return render.ButtonLabelWidth(refProfileText)
}

func (a *App) referenceNavLayout(x, y int) (prev, next sonarUIButton, labelX, labelW int) {
	labelW = referenceLabelWidth()
	prev = sonarUIButton{ID: "ref_prev", Label: "<", Tooltip: "Previous library target", X: x, Y: y, W: refNavBtnW, H: refNavBtnH}
	labelX = x + refNavBtnW + refNavGap
	next = sonarUIButton{
		ID: "ref_next", Label: ">", Tooltip: "Next library target",
		X: labelX + labelW + refNavGap, Y: y, W: refNavBtnW, H: refNavBtnH,
	}
	return prev, next, labelX, labelW
}

// spectrumBinsForUI refreshes the cached analyzer look used by SPECTRUM draw/input.
func (a *App) spectrumBinsForUI(sonar *acoustics.SonarState) (bins []float64, bearing float64) {
	bearing = a.spectrumAnalysisBearing(sonar)
	player := a.Engine.Scenario.Player
	gt := a.Engine.Clock.GameTime
	if a.spectrumCacheBins == nil || math.Abs(bearing-a.spectrumCacheBearing) > 0.5 || gt-a.spectrumCacheAt >= 0.1 {
		a.spectrumCacheBins = acoustics.SpectrumAtBearingInto(a.spectrumCacheBins, a.Engine.Acoustics, player, a.Engine.AcousticEmitters(), sonar, bearing, gt)
		a.spectrumCacheAt = gt
		a.spectrumCacheBearing = bearing
	}
	return a.spectrumCacheBins, bearing
}

func (a *App) spectrumClassifyFilter(bins []float64, bearing float64) acoustics.ClassifyFilter {
	mix := acoustics.CountSpectrumMixContacts(&a.Engine.Sonar, bearing)
	return acoustics.AnalyzeClassifyFilter(bins, mix)
}

func (a *App) ensureReferenceInFilter(f acoustics.ClassifyFilter) {
	idxs := acoustics.ClassificationLibraryIndices(f)
	if len(idxs) == 0 {
		return
	}
	for _, i := range idxs {
		if i == a.referenceProfileIdx {
			return
		}
	}
	a.referenceProfileIdx = idxs[0]
}

func (a *App) cycleReferenceProfile(delta int, f acoustics.ClassifyFilter) {
	idxs := acoustics.ClassificationLibraryIndices(f)
	if len(idxs) == 0 {
		return
	}
	cur := 0
	for i, idx := range idxs {
		if idx == a.referenceProfileIdx {
			cur = i
			break
		}
	}
	n := len(idxs)
	cur = (cur + delta) % n
	if cur < 0 {
		cur += n
	}
	a.referenceProfileIdx = idxs[cur]
}

func (a *App) classifyButtonRect() (x, y, w, h int) {
	w = render.ButtonWidth("CLASSIFY", 20)
	h = 36
	x = spectrumTableX
	y = spectrumTableY + spectrumTableRow*(1+max(4, spectrumTableVisibleRows)) + 8
	if y > 640 {
		y = 640
	}
	return x, y, w, h
}

func (a *App) updateSpectrumScreen(sonar *acoustics.SonarState) {
	a.validateSelectedContact(sonar)
	mx, my := ebiten.CursorPosition()
	scrollContactTableWheel(mx, my, spectrumTableX, spectrumTableY+spectrumTableRow, spectrumTableW, spectrumTableVisibleRows*spectrumTableRow, len(sonar.Contacts), spectrumTableVisibleRows, &a.contactTableScroll.spectrum)

	bins, bearing := a.spectrumBinsForUI(sonar)
	filter := a.spectrumClassifyFilter(bins, bearing)
	a.ensureReferenceInFilter(filter)
	canClassify := filter != acoustics.ClassifyIndistinct

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if c := a.spectrumContactAt(mx, my, sonar); c != nil {
			a.selectContact(sonar, c)
			return
		}
		if !canClassify {
			return
		}
		navX := spectrumChartX + spectrumChartLabelInset + referenceProfileWidth() + refProfileGap
		prev, next, _, _ := a.referenceNavLayout(navX, spectrumRefNavY())
		if prev.contains(mx, my) {
			a.cycleReferenceProfile(-1, filter)
			a.uiPressedID = prev.ID
			a.uiPressedAt = time.Now()
			return
		}
		if next.contains(mx, my) {
			a.cycleReferenceProfile(1, filter)
			a.uiPressedID = next.ID
			a.uiPressedAt = time.Now()
			return
		}
		cx, cy, cw, ch := a.classifyButtonRect()
		if mx >= cx && mx < cx+cw && my >= cy && my < cy+ch {
			a.confirmClassification(sonar, filter)
			a.uiPressedID = "classify"
			a.uiPressedAt = time.Now()
		}
	}
}

func (a *App) confirmClassification(sonar *acoustics.SonarState, filter acoustics.ClassifyFilter) {
	if filter == acoustics.ClassifyIndistinct {
		return
	}
	c := a.selectedContact(sonar)
	if c == nil {
		return
	}
	p := a.referenceProfile()
	if !acoustics.ProfileAllowedByFilter(p, filter) {
		return
	}
	c.ConfirmedID = p.ID
	c.ConfirmedClass = p.Name
	c.Kind = p.Kind
	if c.Confidence < 0.85 {
		c.Confidence = math.Min(0.95, c.Confidence+0.2)
	}
	a.Audio.PlayClip(audio.ClipSonarContactClassified, fmt.Sprintf("Contact %s classified as %s.", c.ID, p.Name))
}

func (a *App) drawSpectrumContactTable(screen *ebiten.Image, sonar *acoustics.SonarState) {
	render.FillRect(screen, spectrumTableX, spectrumTableY, spectrumTableW, spectrumTableHeight(), render.ColorPanelInset)
	render.DrawText(screen, "CONTACT", spectrumTableX+8, spectrumTableY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "BRG°", spectrumTableX+72, spectrumTableY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "RNG kyd", spectrumTableX+118, spectrumTableY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "SOURCE", spectrumTableX+162, spectrumTableY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "CLASS", spectrumTableX+230, spectrumTableY+16, render.ColorPhosphorDim, true)

	mx, my := ebiten.CursorPosition()
	player := a.Engine.Scenario.Player
	a.contactTableScroll.spectrum = clampContactTableScroll(a.contactTableScroll.spectrum, len(sonar.Contacts), spectrumTableVisibleRows)
	start, end := contactTableWindow(len(sonar.Contacts), a.contactTableScroll.spectrum, spectrumTableVisibleRows)
	y := spectrumTableY + spectrumTableRow
	for i := start; i < end; i++ {
		c := &sonar.Contacts[i]
		selected := c.SourceEntityID == a.selectedContactID
		hover := mx >= spectrumTableX && mx < spectrumTableX+spectrumTableW && my >= y && my < y+spectrumTableRow
		if selected {
			render.FillRect(screen, spectrumTableX+2, y, spectrumTableW-4, spectrumTableRow, color.RGBA{80, 60, 0, 180})
		} else if hover {
			render.FillRect(screen, spectrumTableX+2, y, spectrumTableW-4, spectrumTableRow, render.ColorPanelMid)
		}
		clr := render.ColorPhosphor
		if c.ConfirmedClass != "" {
			clr = render.ColorAmber
		}
		render.DrawText(screen, c.ID, spectrumTableX+8, y+16, clr, true)
		render.DrawText(screen, contactBearingLabel(c), spectrumTableX+72, y+16, clr, true)
		render.DrawText(screen, contactRangeLabel(c), spectrumTableX+118, y+16, clr, true)
		render.DrawText(screen, contactSourceLabel(c, player, sonar), spectrumTableX+162, y+16, clr, true)
		render.DrawText(screen, contactClassLabel(c), spectrumTableX+230, y+16, clr, true)
		y += spectrumTableRow
	}
	drawContactTableScrollbar(screen, spectrumTableX+spectrumTableW+4, spectrumTableY+spectrumTableRow, spectrumTableVisibleRows*spectrumTableRow, len(sonar.Contacts), spectrumTableVisibleRows, a.contactTableScroll.spectrum)

	cx, cy, cw, ch := a.classifyButtonRect()
	bins, bearing := a.spectrumBinsForUI(sonar)
	canClassify := a.spectrumClassifyFilter(bins, bearing) != acoustics.ClassifyIndistinct
	if !canClassify {
		render.DrawBevelButtonDisabled(screen, cx, cy, cw, ch, "CLASSIFY")
	} else {
		hover := mx >= cx && mx < cx+cw && my >= cy && my < cy+ch
		pressed := a.uiPressedID == "classify" && time.Since(a.uiPressedAt) < 120*time.Millisecond
		render.DrawBevelButton(screen, cx, cy, cw, ch, "CLASSIFY", hover, pressed)
	}
}

func (a *App) drawReferenceNav(screen *ebiten.Image, x, y int, profile world.SignatureProfile, filter acoustics.ClassifyFilter) {
	mx, my := ebiten.CursorPosition()
	prev, next, labelX, labelW := a.referenceNavLayout(x, y)
	active := filter != acoustics.ClassifyIndistinct
	textY := render.ButtonLabelBaseline(y, refNavBtnH)

	labelClr := cwSigGreen
	if !active {
		labelClr = render.ColorDim
	}
	render.DrawButtonText(screen, refProfileText, spectrumChartX+spectrumChartLabelInset, textY, labelClr)
	if !active {
		render.DrawBevelButtonDisabled(screen, prev.X, prev.Y, prev.W, prev.H, prev.Label)
		render.FillRect(screen, labelX, y, labelW, refNavBtnH, render.ColorPanelInset)
		name := "NO TONAL LOCK"
		tw := render.ButtonLabelWidth(name)
		render.DrawButtonText(screen, name, labelX+(labelW-tw)/2, textY, render.ColorDim)
		render.DrawBevelButtonDisabled(screen, next.X, next.Y, next.W, next.H, next.Label)
		return
	}
	render.DrawBevelButton(screen, prev.X, prev.Y, prev.W, prev.H, prev.Label,
		prev.contains(mx, my), a.uiPressedID == prev.ID && time.Since(a.uiPressedAt) < 120*time.Millisecond)
	render.FillRect(screen, labelX, y, labelW, refNavBtnH, render.ColorPanelInset)
	tw := render.ButtonLabelWidth(profile.Name)
	render.DrawButtonText(screen, profile.Name, labelX+(labelW-tw)/2, textY, cwSigGreen)
	render.DrawBevelButton(screen, next.X, next.Y, next.W, next.H, next.Label,
		next.contains(mx, my), a.uiPressedID == next.ID && time.Since(a.uiPressedAt) < 120*time.Millisecond)
}

func freqToPanelX(freq float64, panelW int) int {
	span := acoustics.MaxFreqHz - acoustics.MinFreqHz
	if span <= 0 {
		return 0
	}
	t := (freq - acoustics.MinFreqHz) / span
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return int(t * float64(panelW-1))
}

func (a *App) drawFreqScale(screen *ebiten.Image, x, y, w int) {
	for _, hz := range []int{0, 500, 1000, 1500, 2000} {
		px := x + freqToPanelX(float64(hz), w)
		render.DrawLine(screen, float64(px), float64(y-3), float64(px), float64(y+3), cwSigGreenDim)
	}
	render.DrawLine(screen, float64(x), float64(y), float64(x+w), float64(y), cwSigGreenDim)
}

func (a *App) drawSharpSignature(screen *ebiten.Image, x, y, w, h int, peaks []acoustics.SpectrumPeak) {
	render.FillRect(screen, x, y, w, h, cwSigPanelBG)
	base := y + h - 2
	for _, p := range peaks {
		px := x + freqToPanelX(p.FreqHz, w)
		lh := int(float64(h-6) * p.Level)
		if lh < 4 {
			lh = 4
		}
		render.DrawLine(screen, float64(px), float64(base), float64(px), float64(base-lh), cwSigGreen)
		if p.Level > 0.55 {
			render.DrawLine(screen, float64(px-1), float64(base), float64(px-1), float64(base-lh/2), cwSigGreenGlow)
			render.DrawLine(screen, float64(px+1), float64(base), float64(px+1), float64(base-lh/2), cwSigGreenGlow)
		}
	}
}

func (a *App) drawFuzzySignature(screen *ebiten.Image, x, y, w, h int, peaks []acoustics.SpectrumPeak) {
	if w < 2 || h < 2 {
		return
	}
	key := 0.0
	if a.Engine != nil {
		key = a.Engine.Clock.GameTime
	}
	for _, p := range peaks {
		key += p.FreqHz*0.001 + p.Level
	}
	stamp := time.Now().UnixNano() / int64(80*time.Millisecond)
	need := a.spectrumFuzzyImg == nil ||
		a.spectrumFuzzyImg.Bounds().Dx() != w ||
		a.spectrumFuzzyImg.Bounds().Dy() != h ||
		a.spectrumFuzzyStamp != stamp ||
		math.Abs(a.spectrumFuzzyKey-key) > 0.02

	if need {
		if a.spectrumFuzzyImg == nil || a.spectrumFuzzyImg.Bounds().Dx() != w || a.spectrumFuzzyImg.Bounds().Dy() != h {
			a.spectrumFuzzyImg = ebiten.NewImage(w, h)
			a.spectrumFuzzyPix = make([]byte, w*h*4)
		}
		pix := a.spectrumFuzzyPix
		for i := 0; i < w*h; i++ {
			off := i * 4
			pix[off] = 0
			pix[off+1] = 2
			pix[off+2] = 16
			pix[off+3] = 255
		}
		if len(peaks) > 0 {
			if cap(a.spectrumFuzzyLevels) < w {
				a.spectrumFuzzyLevels = make([]float64, w)
			} else {
				a.spectrumFuzzyLevels = a.spectrumFuzzyLevels[:w]
			}
			levels := a.spectrumFuzzyLevels
			for px := 0; px < w; px++ {
				freq := acoustics.MinFreqHz + (float64(px)/float64(w-1))*(acoustics.MaxFreqHz-acoustics.MinFreqHz)
				i := 0
				for i < len(peaks)-1 && peaks[i+1].FreqHz < freq {
					i++
				}
				if i >= len(peaks)-1 {
					levels[px] = peaks[len(peaks)-1].Level
				} else {
					f0, f1 := peaks[i].FreqHz, peaks[i+1].FreqHz
					t := 0.0
					if f1 > f0 {
						t = (freq - f0) / (f1 - f0)
					}
					levels[px] = peaks[i].Level*(1-t) + peaks[i+1].Level*t
				}
			}
			rng := rand.New(rand.NewSource(stamp ^ 0xc0ffee))
			clarity := 0.0
			for _, p := range peaks {
				if p.Level > clarity {
					clarity = p.Level
				}
			}
			// More grain / hash when harmonics are weak (matches faint waterfall).
			grainN := w * h / 40
			if clarity < 0.55 {
				grainN = w * h / (12 + int(clarity*20))
			}
			for i := 0; i < grainN; i++ {
				gx := rng.Intn(w)
				gy := rng.Intn(h)
				v := rng.Float64()
				off := (gy*w + gx) * 4
				pix[off] = 0
				pix[off+1] = uint8(4 + v*20)
				pix[off+2] = uint8(12 + v*40)
				pix[off+3] = 255
			}
			base := h - 2
			for px := 0; px < w; px++ {
				lvl := levels[px]
				jitter := 0.78 + 0.32*rng.Float64()
				if clarity < 0.5 {
					// Extra flutter — lines break up instead of reading as clean spikes.
					jitter = 0.45 + 0.70*rng.Float64()
					if rng.Float64() > 0.35+clarity {
						lvl *= 0.25 + rng.Float64()*0.4
					}
				}
				lvl *= jitter
				floor := 0.04 + rng.Float64()*0.05
				if clarity < 0.45 {
					floor = 0.12 + rng.Float64()*(0.22-clarity*0.15)
				}
				if lvl < floor {
					lvl = floor
				}
				lh := int(float64(h-4) * lvl)
				if lh < 2 {
					lh = 2
				}
				for dy := 0; dy < lh; dy++ {
					t := float64(dy) / float64(lh)
					intensity := lvl * (0.35 + 0.65*t)
					if clarity < 0.4 {
						intensity *= 0.55 + 0.45*clarity
					}
					if intensity < 0.03 {
						continue
					}
					clr := sonarHeatColorFast(intensity)
					yy := base - dy
					if yy < 0 || yy >= h {
						continue
					}
					off := (yy*w + px) * 4
					pix[off] = clr.R
					pix[off+1] = clr.G
					pix[off+2] = clr.B
					pix[off+3] = 255
				}
			}
		}
		a.spectrumFuzzyImg.WritePixels(pix)
		a.spectrumFuzzyStamp = stamp
		a.spectrumFuzzyKey = key
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(a.spectrumFuzzyImg, op)
}

func (a *App) drawSpectrumChart(screen *ebiten.Image, bins []float64, profile world.SignatureProfile, bearing float64, filter acoustics.ClassifyFilter) {
	navX := spectrumChartX + spectrumChartLabelInset + referenceProfileWidth() + refProfileGap
	refY := spectrumRefY()
	obsY := spectrumObsY()
	navY := spectrumRefNavY()

	// Gray-plate row above the green Hz ruler; black reference panel below it.
	a.drawReferenceNav(screen, navX, navY, profile, filter)
	a.drawFreqScale(screen, spectrumChartX, refY-12, spectrumChartW)
	if filter == acoustics.ClassifyIndistinct {
		render.FillRect(screen, spectrumChartX, refY, spectrumChartW, spectrumSigPanelH, cwSigPanelBG)
		render.DrawText(screen, "REFERENCE UNAVAILABLE — insufficient harmonics", spectrumChartX+24, refY+spectrumSigPanelH/2, render.ColorDim, true)
	} else {
		refPeaks := acoustics.ProfileReferencePeaks(profile)
		a.drawSharpSignature(screen, spectrumChartX, refY, spectrumChartW, spectrumSigPanelH, refPeaks)
	}

	render.DrawText(screen, fmt.Sprintf("CONTACT SIGNAL @ %.0f°", bearing), spectrumChartX+spectrumChartLabelInset, obsY-18, cwSigGreenDim, true)
	filterLbl := filter.Label()
	filterClr := cwSigGreenDim
	switch filter {
	case acoustics.ClassifyIndistinct:
		filterClr = render.ColorWarn
	case acoustics.ClassifyFull:
		if acoustics.CountSpectrumMixContacts(&a.Engine.Sonar, bearing) >= 2 {
			filterLbl = "BEARING MIX — FULL LIBRARY"
			filterClr = render.ColorAmber
		}
	}
	render.DrawText(screen, filterLbl, spectrumChartX+220, obsY-18, filterClr, true)
	obsPeaks := acoustics.ObservedPeaksFromBins(bins)
	a.drawFuzzySignature(screen, spectrumChartX, obsY, spectrumChartW, spectrumSigPanelH+20, obsPeaks)
	a.drawFreqScale(screen, spectrumChartX, obsY+spectrumSigPanelH+28, spectrumChartW)
}

func (a *App) drawSpectrum(screen *ebiten.Image) {
	sonar := &a.Engine.Sonar
	player := a.Engine.Scenario.Player
	bins, bearing := a.spectrumBinsForUI(sonar)
	filter := a.spectrumClassifyFilter(bins, bearing)
	a.ensureReferenceInFilter(filter)
	profile := a.referenceProfile()

	render.DrawConsolePanel(screen, spectrumScreenPanelX, spectrumScreenPanelY, spectrumScreenPanelW, spectrumScreenPanelH)
	tableH := spectrumTableHeight()
	render.DrawMonitor(screen, spectrumTableX, spectrumTableY, spectrumTableW, tableH)
	render.DrawMonitor(screen, spectrumChartX, spectrumTableY, spectrumChartW, tableH)
	render.DrawScreenTitle(screen, "SPECTRUM ANALYZER", layout.PassiveTitleLabelX, layout.PassiveTitleLabelY+20)
	a.drawArraySelector(screen, sonar, spectrumArrayLabelX, spectrumArrayLabelY, cachedSpectrumArrayButtons())

	if c := a.selectedContact(sonar); c != nil {
		render.DrawText(screen, fmt.Sprintf("SELECTED: %s  |  BRG %.0f°  |  R %.1f kyd  |  SPD %.1f kts  |  %s",
			contactLongLabel(c), c.BearingDeg, c.EstimatedRangeYd/1000, player.SpeedKts, c.DetectedBy),
			40, 232, render.ColorAmber, false)
	} else {
		render.DrawText(screen, fmt.Sprintf("MANUAL BEARING: %.0f°  |  SPD %.1f kts  — select a contact in the table or on PASSIVE",
			bearing, player.SpeedKts), 40, 232, render.ColorDim, false)
	}
	render.DrawText(screen, fmt.Sprintf("Array: %s  |  %s  |  Layer: %s  |  Speed %.1f kts",
		a.sonarArrayLabel(sonar), a.towedCableStatus(sonar),
		a.Engine.Acoustics.Env.LayerNameKnown(player.DepthFt), player.SpeedKts), 40, 256, render.ColorPhosphorDim, true)

	a.drawSpectrumContactTable(screen, sonar)
	a.drawSpectrumChart(screen, bins, profile, bearing, filter)

	if sonar.PassiveArray == acoustics.PassiveArrayTowed && sonar.TowedDamaged {
		render.DrawText(screen, "TOWED ARRAY DAMAGED — NO DATA", spectrumChartX+120, spectrumObsY()+40, render.ColorWarn, false)
	} else if sonar.PassiveArray == acoustics.PassiveArrayHull && a.hullArrayDamaged() {
		render.DrawText(screen, "HULL ARRAY DAMAGED — NO DATA", spectrumChartX+120, spectrumObsY()+40, render.ColorWarn, false)
	}

	mx, my := ebiten.CursorPosition()
	if a.uiTooltip != "" {
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
	help := "[B] array  < > cycle profile  CLASSIFY  LEFT/RIGHT manual bearing"
	if filter == acoustics.ClassifyIndistinct {
		help = "[B] array  LEFT/RIGHT manual bearing  — classify locked (no clear harmonics)"
	}
	render.DrawText(screen, help, 40, 720, render.ColorDim, true)
}
