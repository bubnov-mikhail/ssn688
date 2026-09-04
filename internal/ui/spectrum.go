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
	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/audio"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/layout"
	"github.com/bubnov-mikhail/ssn688/internal/platform"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const (
	spectrumScreenPanelX = 20
	spectrumScreenPanelY = 50
	spectrumScreenPanelH = 700

	spectrumTableX           = 40
	spectrumTableY           = 278
	spectrumTableW           = 480
	spectrumTableRow         = 22
	spectrumTableVisibleRows = 15
	spectrumColID            = 8
	spectrumColBRG           = 72
	spectrumColRNG           = 118
	spectrumColSRC           = 200
	spectrumColCLASS         = 290
	spectrumChartX           = 540
	spectrumChartLabelInset  = 14
	spectrumSigPanelH        = 118
	spectrumGap              = 10
	spectrumRefYOffset       = 52
	spectrumObsGap           = 36

	spectrumArrayLabelX = 40
	spectrumArrayLabelY = 106
)

func spectrumScreenPanelW() int { return spectrumPanelW() }

func spectrumChartW() int {
	return spectrumScreenPanelX + spectrumScreenPanelW() - spectrumChartX - 20
}

func spectrumTableHeight() int {
	return spectrumTableRow*(spectrumTableVisibleRows+1) + 8
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
	refNavBtnW    = 32
	refNavBtnH    = 28
	refLabelPad   = 16
	refNavGap     = 4
	refProfileGap = 12
)

func (a *App) spectrumProfileLabel() string {
	return a.L(i18n.UIProfile)
}

func (a *App) spectrumFilterLabel(filter acoustics.ClassifyFilter) string {
	switch filter {
	case acoustics.ClassifyIndistinct:
		return a.L(i18n.UIInsufficientTonal)
	case acoustics.ClassifyTorpedoOnly:
		return a.L(i18n.UIHFTorpedoSet)
	case acoustics.ClassifyPlatformOnly:
		return a.L(i18n.UILFPlatformSet)
	default:
		return a.L(i18n.UIFullLibrary)
	}
}

var referenceLabelWidthOnce struct {
	once sync.Once
	w    int
}

func referenceLabelWidth() int {
	referenceLabelWidthOnce.once.Do(func() {
		w := 160
		for _, p := range world.SignatureLibrary {
			for _, lang := range i18n.SupportedLangs {
				tw := render.ButtonLabelWidth(p.Name.GetText(lang)) + refLabelPad
				if tw > w {
					w = tw
				}
			}
		}
		referenceLabelWidthOnce.w = w
	})
	return referenceLabelWidthOnce.w
}

func referenceProfileWidth() int {
	w := 0
	for _, lang := range i18n.SupportedLangs {
		tw := render.ButtonLabelWidth(i18n.UIProfile.GetText(lang))
		if tw > w {
			w = tw
		}
	}
	return w
}

func (a *App) referenceNavLayout(x, y int) (prev, next sonarUIButton, labelX, labelW int) {
	labelW = referenceLabelWidth()
	prev = sonarUIButton{ID: "ref_prev", Label: "<", Tooltip: a.L(i18n.UITipPrevLib), X: x, Y: y, W: refNavBtnW, H: refNavBtnH}
	labelX = x + refNavBtnW + refNavGap
	next = sonarUIButton{
		ID: "ref_next", Label: ">", Tooltip: a.L(i18n.UITipNextLib),
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
	f := acoustics.AnalyzeClassifyFilter(bins, mix)
	if f != acoustics.ClassifyIndistinct {
		return f
	}
	// RF intercept on the selected track unlocks the platform library without tonals,
	// so the player can hull-ID from ESM equipment type via LIBRARY.
	c := a.selectedContact(&a.Engine.Sonar)
	if c != nil && a.Engine.ESM.HasRecentRF(c.SourceEntityID, a.Engine.Clock.GameTime) {
		return acoustics.ClassifyPlatformOnly
	}
	return f
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
	w = render.ButtonWidth(a.L(i18n.UIClassify), 20)
	h = refNavBtnH
	navX := spectrumChartX + spectrumChartLabelInset + referenceProfileWidth() + refProfileGap
	navY := spectrumRefNavY()
	_, next, _, _ := a.referenceNavLayout(navX, navY)
	x = next.X + next.W + refProfileGap
	y = navY
	return x, y, w, h
}

func (a *App) drawClassifyButton(screen *ebiten.Image, filter acoustics.ClassifyFilter, offX, offY int) {
	cx, cy, cw, ch := a.classifyButtonRect()
	cx += offX
	cy += offY
	mx, my := ebiten.CursorPosition()
	canClassify := filter != acoustics.ClassifyIndistinct
	label := a.L(i18n.UIClassify)
	if !canClassify {
		render.DrawBevelButtonDisabled(screen, cx, cy, cw, ch, label)
		return
	}
	hover := mx >= cx && mx < cx+cw && my >= cy && my < cy+ch
	pressed := a.uiPressedID == "classify" && time.Since(a.uiPressedAt) < 120*time.Millisecond
	render.DrawBevelButton(screen, cx, cy, cw, ch, label, hover, pressed)
}

func (a *App) updateSpectrumScreen(sonar *acoustics.SonarState) {
	a.validateSelectedContact(sonar)
	mx, my := ebiten.CursorPosition()
	scrollContactTableWheel(mx, my, spectrumTableX, spectrumTableY+spectrumTableRow, spectrumTableW, spectrumTableVisibleRows*spectrumTableRow, len(sonar.Contacts), spectrumTableVisibleRows, &a.contactTableScroll.spectrum)

	bins, bearing := a.spectrumBinsForUI(sonar)
	filter := a.spectrumClassifyFilter(bins, bearing)
	if c := a.selectedContact(sonar); c != nil && (c.ConfirmedID != "" || c.ConfirmedClass != "") {
		// Keep the plate on the contact's class — do not snap back to library[0].
		a.syncReferenceToContact(c)
	} else {
		a.ensureReferenceInFilter(filter)
	}
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
	c.ConfirmedClass = p.DisplayName()
	c.Kind = p.Kind
	if c.Confidence < 0.85 {
		c.Confidence = math.Min(0.95, c.Confidence+0.2)
	}
	a.Audio.PlayClip(audio.ClipSonarContactClassified, a.Lf(i18n.StatusVoiceClassified, c.ID, p.DisplayName()))
}

func (a *App) drawSpectrumContactTable(screen *ebiten.Image, sonar *acoustics.SonarState, offX, offY int) {
	tx, ty := spectrumTableX+offX, spectrumTableY+offY
	render.FillRect(screen, tx, ty, spectrumTableW, spectrumTableHeight(), render.ColorPanelInset)
	render.DrawText(screen, a.L(i18n.UIColID), tx+spectrumColID, ty+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, a.L(i18n.UIColBRGDeg), tx+spectrumColBRG, ty+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, a.L(i18n.UIColRNG), tx+spectrumColRNG, ty+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, a.L(i18n.UIColSource), tx+spectrumColSRC, ty+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, a.L(i18n.UIColClass), tx+spectrumColCLASS, ty+16, render.ColorPhosphorDim, true)

	mx, my := ebiten.CursorPosition()
	player := a.Engine.Scenario.Player
	a.contactTableScroll.spectrum = clampContactTableScroll(a.contactTableScroll.spectrum, len(sonar.Contacts), spectrumTableVisibleRows)
	start, end := contactTableWindow(len(sonar.Contacts), a.contactTableScroll.spectrum, spectrumTableVisibleRows)
	y := ty + spectrumTableRow
	for i := start; i < end; i++ {
		c := &sonar.Contacts[i]
		selected := c.SourceEntityID == a.selectedContactID
		hover := mx >= tx && mx < tx+spectrumTableW && my >= y && my < y+spectrumTableRow
		if selected {
			render.FillRect(screen, tx+2, y, spectrumTableW-4, spectrumTableRow, color.RGBA{80, 60, 0, 180})
		} else if hover {
			render.FillRect(screen, tx+2, y, spectrumTableW-4, spectrumTableRow, render.ColorPanelMid)
		}
		clr := render.ColorPhosphor
		if c.ConfirmedClass != "" {
			clr = render.ColorAmber
		}
		render.DrawText(screen, c.ID, tx+spectrumColID, y+16, clr, true)
		render.DrawText(screen, contactBearingLabel(c), tx+spectrumColBRG, y+16, clr, true)
		render.DrawText(screen, contactRangeLabel(c), tx+spectrumColRNG, y+16, clr, true)
		render.DrawText(screen, contactSourceLabel(c, player, sonar), tx+spectrumColSRC, y+16, clr, true)
		render.DrawText(screen, contactClassLabel(c), tx+spectrumColCLASS, y+16, clr, true)
		y += spectrumTableRow
	}
	drawContactTableScrollbar(screen, tx+spectrumTableW+4, ty+spectrumTableRow, spectrumTableVisibleRows*spectrumTableRow, len(sonar.Contacts), spectrumTableVisibleRows, a.contactTableScroll.spectrum)
}

func (a *App) drawReferenceNav(screen *ebiten.Image, x, y int, profile world.SignatureProfile, filter acoustics.ClassifyFilter, offX, offY int) {
	mx, my := ebiten.CursorPosition()
	prev, next, labelX, labelW := a.referenceNavLayout(x, y)
	prev.X += offX
	prev.Y += offY
	next.X += offX
	next.Y += offY
	labelX += offX
	y += offY
	active := filter != acoustics.ClassifyIndistinct
	textY := render.ButtonLabelBaseline(y, refNavBtnH)

	labelClr := cwSigGreen
	if !active {
		labelClr = render.ColorDim
	}
	render.DrawButtonText(screen, a.spectrumProfileLabel(), spectrumChartX+spectrumChartLabelInset+offX, textY, labelClr)
	if !active {
		render.DrawBevelButtonDisabled(screen, prev.X, prev.Y, prev.W, prev.H, prev.Label)
		render.FillRect(screen, labelX, y, labelW, refNavBtnH, render.ColorPanelInset)
		name := a.L(i18n.UINoTonalLock)
		tw := render.ButtonLabelWidth(name)
		render.DrawButtonText(screen, name, labelX+(labelW-tw)/2, textY, render.ColorDim)
		render.DrawBevelButtonDisabled(screen, next.X, next.Y, next.W, next.H, next.Label)
		return
	}
	render.DrawBevelButton(screen, prev.X, prev.Y, prev.W, prev.H, prev.Label,
		prev.contains(mx, my), a.uiPressedID == prev.ID && time.Since(a.uiPressedAt) < 120*time.Millisecond)
	render.FillRect(screen, labelX, y, labelW, refNavBtnH, render.ColorPanelInset)
	tw := render.ButtonLabelWidth(profile.DisplayName())
	render.DrawButtonText(screen, profile.DisplayName(), labelX+(labelW-tw)/2, textY, cwSigGreen)
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
			disposeImage(&a.spectrumFuzzyImg)
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

func (a *App) drawSpectrumChart(screen *ebiten.Image, bins []float64, profile world.SignatureProfile, bearing float64, filter acoustics.ClassifyFilter, offX, offY int) {
	navX := spectrumChartX + spectrumChartLabelInset + referenceProfileWidth() + refProfileGap
	refY := spectrumRefY() + offY
	obsY := spectrumObsY() + offY
	chartX := spectrumChartX + offX

	// Gray-plate row above the green Hz ruler; black reference panel below it.
	a.drawReferenceNav(screen, navX, spectrumRefNavY(), profile, filter, offX, offY)
	a.drawClassifyButton(screen, filter, offX, offY)
	a.drawFreqScale(screen, chartX, refY-12, spectrumChartW())
	if filter == acoustics.ClassifyIndistinct {
		render.FillRect(screen, chartX, refY, spectrumChartW(), spectrumSigPanelH, cwSigPanelBG)
		render.DrawText(screen, a.L(i18n.UIRefUnavailable), chartX+24, refY+spectrumSigPanelH/2, render.ColorDim, true)
	} else {
		refPeaks := acoustics.ProfileReferencePeaks(profile)
		a.drawSharpSignature(screen, chartX, refY, spectrumChartW(), spectrumSigPanelH, refPeaks)
	}

	render.DrawText(screen, fmt.Sprintf("%s %.0f°", a.L(i18n.UIContactSignalAt), bearing), chartX+spectrumChartLabelInset, obsY-18, cwSigGreenDim, true)
	filterLbl := a.spectrumFilterLabel(filter)
	filterClr := cwSigGreenDim
	switch filter {
	case acoustics.ClassifyIndistinct:
		filterClr = render.ColorWarn
	case acoustics.ClassifyFull:
		if acoustics.CountSpectrumMixContacts(&a.Engine.Sonar, bearing) >= 2 {
			filterLbl = a.L(i18n.UIBearingMixFull)
			filterClr = render.ColorAmber
		}
	}
	render.DrawText(screen, filterLbl, chartX+220, obsY-18, filterClr, true)
	obsPeaks := acoustics.ObservedPeaksFromBins(bins)
	a.drawFuzzySignature(screen, chartX, obsY, spectrumChartW(), spectrumSigPanelH+20, obsPeaks)
	a.drawFreqScale(screen, chartX, obsY+spectrumSigPanelH+28, spectrumChartW())
}

func (a *App) drawSpectrumPanelContent(screen *ebiten.Image, sonar *acoustics.SonarState, bins []float64, profile world.SignatureProfile, bearing float64, filter acoustics.ClassifyFilter, offX, offY int) {
	render.DrawConsolePanel(screen, spectrumScreenPanelX+offX, spectrumScreenPanelY+offY, spectrumScreenPanelW(), spectrumScreenPanelH)
	tableH := spectrumTableHeight()
	render.DrawMonitor(screen, spectrumTableX+offX, spectrumTableY+offY, spectrumTableW, tableH)
	render.DrawMonitor(screen, spectrumChartX+offX, spectrumTableY+offY, spectrumChartW(), tableH)
	a.drawArraySelector(screen, sonar, spectrumArrayLabelX+offX, spectrumArrayLabelY+offY, a.cachedSpectrumArrayButtons())
	a.drawSpectrumContactTable(screen, sonar, offX, offY)
	a.drawSpectrumChart(screen, bins, profile, bearing, filter, offX, offY)

	if sonar.PassiveArray == acoustics.PassiveArrayTowed && sonar.TowedDamaged {
		render.DrawText(screen, a.L(i18n.UITowedDamagedData), spectrumChartX+120+offX, spectrumObsY()+40+offY, render.ColorWarn, false)
	} else if sonar.PassiveArray == acoustics.PassiveArrayHull && a.hullArrayDamaged() {
		render.DrawText(screen, a.L(i18n.UIHullDamagedData), spectrumChartX+120+offX, spectrumObsY()+40+offY, render.ColorWarn, false)
	}
}

func (a *App) drawSpectrum(screen *ebiten.Image) {
	sonar := &a.Engine.Sonar
	player := a.Engine.Scenario.Player
	bins, bearing := a.spectrumBinsForUI(sonar)
	filter := a.spectrumClassifyFilter(bins, bearing)
	a.ensureReferenceInFilter(filter)
	profile := a.referenceProfile()

	a.drawSpectrumPanelContent(screen, sonar, bins, profile, bearing, filter, 0, 0)

	render.DrawScreenTitle(screen, a.L(i18n.UITitleSpectrum), layout.PassiveTitleLabelX, layout.PassiveTitleLabelY+20)

	if c := a.selectedContact(sonar); c != nil {
		idTxt := "ID: —"
		if c.Identified {
			idTxt = "ID: " + contactClassLabel(c) + " (" + c.IdentifiedBy + ")"
		} else if c.HarmonicHoldSec > 0 || c.HarmonicMatch > 0 {
			idTxt = fmt.Sprintf("ID HOLD %.0f/120s  MATCH %.0f%%", c.HarmonicHoldSec, c.HarmonicMatch*100)
		}
		render.DrawText(screen, fmt.Sprintf("%s: %s  |  %s %.0f°  |  R %.1f kyd  |  SPD %.1f kts  |  %s  |  %s",
			a.L(i18n.UISelected), contactLongLabel(c), a.L(i18n.UIColBRG), c.BearingDeg, c.EstimatedRangeYd/1000, player.SpeedKts, c.DetectedBy, idTxt),
			40, 232, render.ColorAmber, false)
	} else {
		render.DrawText(screen, fmt.Sprintf("%s: %.0f°  |  SPD %.1f kts  — select a contact in the table or on PASSIVE",
			a.L(i18n.UIManualBearing), bearing, player.SpeedKts), 40, 232, render.ColorDim, false)
	}
	render.DrawText(screen, fmt.Sprintf("%s: %s  |  %s  |  %s: %s  |  Speed %.1f kts",
		a.L(i18n.UIArray), a.sonarArrayLabel(sonar), a.towedCableStatus(sonar),
		a.L(i18n.UILayer), i18n.LocalizeLayerName(a.Engine.Acoustics.Env.LayerNameKnown(player.DepthFt), a.Lang()), player.SpeedKts), 40, 256, render.ColorPhosphorDim, true)

	mx, my := ebiten.CursorPosition()
	if a.uiTooltip != "" {
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
	if !platform.Mobile() {
		help := "[B] array  < > cycle profile  CLASSIFY  LEFT/RIGHT bearing  — ID: peri <3000 yd or 80% harmonics × 2 min"
		if filter == acoustics.ClassifyIndistinct {
			help = "[B] array  LEFT/RIGHT bearing  — classify locked (no clear harmonics). ID still via peri <3000 yd"
		}
		render.DrawText(screen, help, 40, 720, render.ColorDim, true)
	}
}
