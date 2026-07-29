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
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/world"
)

const (
	spectrumTableX   = 40
	spectrumTableY   = 278
	spectrumTableW   = 450
	spectrumTableRow = 22
	spectrumChartX   = 510
	spectrumChartW   = 590
	spectrumPanelH   = 118
	spectrumGap      = 10
	spectrumRefY     = 330
	spectrumObsY     = spectrumRefY + spectrumPanelH + 40

	spectrumArrayLabelX = 40
	spectrumArrayLabelY = 106
)

var (
	cwSigGreen     = color.RGBA{0, 255, 70, 255}
	cwSigGreenDim  = color.RGBA{0, 160, 50, 180}
	cwSigGreenGlow = color.RGBA{0, 255, 70, 70}
	cwSigNoise     = color.RGBA{0, 90, 35, 40}
	cwSigPanelBG   = color.RGBA{0, 0, 0, 255}
)

func (a *App) spectrumContactAt(mx, my int, sonar *acoustics.SonarState) *acoustics.Contact {
	y := spectrumTableY + spectrumTableRow
	for i := range sonar.Contacts {
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
	refNavBtnW  = 32
	refNavBtnH  = 28
	refLabelPad = 16
	refNavGap   = 4
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

func (a *App) cycleReferenceProfile(delta int) {
	n := len(world.SignatureLibrary)
	if n == 0 {
		return
	}
	a.referenceProfileIdx = (a.referenceProfileIdx + delta) % n
	if a.referenceProfileIdx < 0 {
		a.referenceProfileIdx += n
	}
}

func (a *App) classifyButtonRect() (x, y, w, h int) {
	w = render.ButtonWidth("CLASSIFY", 20)
	h = 36
	x = spectrumTableX
	y = spectrumTableY + spectrumTableRow*(1+max(4, len(a.Engine.Sonar.Contacts))) + 8
	if y > 640 {
		y = 640
	}
	return x, y, w, h
}

func (a *App) updateSpectrumScreen(sonar *acoustics.SonarState) {
	a.validateSelectedContact(sonar)
	mx, my := ebiten.CursorPosition()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if c := a.spectrumContactAt(mx, my, sonar); c != nil {
			a.selectContact(sonar, c)
			return
		}
		navX := spectrumChartX + 78
		prev, next, _, _ := a.referenceNavLayout(navX, spectrumRefY-58)
		if prev.contains(mx, my) {
			a.cycleReferenceProfile(-1)
			a.uiPressedID = prev.ID
			a.uiPressedAt = time.Now()
			return
		}
		if next.contains(mx, my) {
			a.cycleReferenceProfile(1)
			a.uiPressedID = next.ID
			a.uiPressedAt = time.Now()
			return
		}
		cx, cy, cw, ch := a.classifyButtonRect()
		if mx >= cx && mx < cx+cw && my >= cy && my < cy+ch {
			a.confirmClassification(sonar)
			a.uiPressedID = "classify"
			a.uiPressedAt = time.Now()
		}
	}
}

func (a *App) confirmClassification(sonar *acoustics.SonarState) {
	c := a.selectedContact(sonar)
	if c == nil {
		return
	}
	p := a.referenceProfile()
	c.ConfirmedID = p.ID
	c.ConfirmedClass = p.Name
	if c.Confidence < 0.85 {
		c.Confidence = math.Min(0.95, c.Confidence+0.2)
	}
	a.Audio.PlayClip(audio.ClipSonarContactClassified, fmt.Sprintf("Contact %s classified as %s.", c.ID, p.Name))
}

func (a *App) drawSpectrumContactTable(screen *ebiten.Image, sonar *acoustics.SonarState) {
	render.FillRect(screen, spectrumTableX, spectrumTableY, spectrumTableW, spectrumTableRow*max(1, len(sonar.Contacts)+1)+44, render.ColorPanelInset)
	render.DrawText(screen, "CONTACT", spectrumTableX+8, spectrumTableY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "BRG", spectrumTableX+72, spectrumTableY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "RNG", spectrumTableX+118, spectrumTableY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "SOURCE", spectrumTableX+162, spectrumTableY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "CLASS", spectrumTableX+230, spectrumTableY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "TYPE", spectrumTableX+382, spectrumTableY+16, render.ColorPhosphorDim, true)

	mx, my := ebiten.CursorPosition()
	player := a.Engine.Scenario.Player
	y := spectrumTableY + spectrumTableRow
	for i := range sonar.Contacts {
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
		render.DrawText(screen, fmt.Sprintf("%03.0f", c.BearingDeg), spectrumTableX+72, y+16, clr, true)
		render.DrawText(screen, contactRangeLabel(c), spectrumTableX+118, y+16, clr, true)
		render.DrawText(screen, contactSourceLabel(c, player, sonar), spectrumTableX+162, y+16, clr, true)
		render.DrawText(screen, contactClassLabel(c), spectrumTableX+230, y+16, clr, true)
		render.DrawText(screen, contactTypeLabel(c), spectrumTableX+380, y+16, clr, true)
		y += spectrumTableRow
	}

	cx, cy, cw, ch := a.classifyButtonRect()
	hover := mx >= cx && mx < cx+cw && my >= cy && my < cy+ch
	pressed := a.uiPressedID == "classify" && time.Since(a.uiPressedAt) < 120*time.Millisecond
	render.DrawBevelButton(screen, cx, cy, cw, ch, "CLASSIFY", hover, pressed)
}

func (a *App) drawReferenceNav(screen *ebiten.Image, x, y int, profile world.SignatureProfile) {
	mx, my := ebiten.CursorPosition()
	prev, next, labelX, labelW := a.referenceNavLayout(x, y)

	render.DrawText(screen, "PROFILE:", spectrumChartX, y+refNavBtnH/2+4, cwSigGreen, true)
	render.DrawBevelButton(screen, prev.X, prev.Y, prev.W, prev.H, prev.Label,
		prev.contains(mx, my), a.uiPressedID == prev.ID && time.Since(a.uiPressedAt) < 120*time.Millisecond)
	render.FillRect(screen, labelX, y, labelW, refNavBtnH, render.ColorPanelInset)
	tw := render.ButtonLabelWidth(profile.Name)
	render.DrawButtonText(screen, profile.Name, labelX+(labelW-tw)/2, y+refNavBtnH/2+4, cwSigGreen)
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
		label := fmt.Sprintf("%d", hz)
		lw := len(label) * 6
		render.DrawText(screen, label, px-lw/2, y+14, cwSigGreenDim, true)
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

func (a *App) drawSpectrumChart(screen *ebiten.Image, bins []float64, profile world.SignatureProfile, bearing float64) {
	navX := spectrumChartX + 78
	// Keep profile picker and Hz scale clearly above the reference panel.
	a.drawReferenceNav(screen, navX, spectrumRefY-58, profile)

	a.drawFreqScale(screen, spectrumChartX, spectrumRefY-26, spectrumChartW)
	refPeaks := acoustics.ProfileReferencePeaks(profile)
	a.drawSharpSignature(screen, spectrumChartX, spectrumRefY, spectrumChartW, spectrumPanelH, refPeaks)

	render.DrawText(screen, fmt.Sprintf("CONTACT SIGNAL @ %.0f°", bearing), spectrumChartX, spectrumObsY-18, cwSigGreenDim, true)
	if acoustics.CountSpectrumMixContacts(&a.Engine.Sonar, bearing) >= 2 {
		render.DrawText(screen, "BEARING MIX — harmonics may overlap", spectrumChartX+220, spectrumObsY-18, render.ColorAmber, true)
	}
	obsPeaks := acoustics.ObservedPeaksFromBins(bins)
	a.drawFuzzySignature(screen, spectrumChartX, spectrumObsY, spectrumChartW, spectrumPanelH+20, obsPeaks)
	a.drawFreqScale(screen, spectrumChartX, spectrumObsY+spectrumPanelH+28, spectrumChartW)
}

func (a *App) drawSpectrum(screen *ebiten.Image) {
	sonar := &a.Engine.Sonar
	player := a.Engine.Scenario.Player
	bearing := a.spectrumAnalysisBearing(sonar)
	gt := a.Engine.Clock.GameTime
	if a.spectrumCacheBins == nil || math.Abs(bearing-a.spectrumCacheBearing) > 0.5 || gt-a.spectrumCacheAt >= 0.1 {
		a.spectrumCacheBins = acoustics.SpectrumAtBearingInto(a.spectrumCacheBins, a.Engine.Acoustics, player, a.Engine.AcousticEmitters(), sonar, bearing, gt)
		a.spectrumCacheAt = gt
		a.spectrumCacheBearing = bearing
	}
	bins := a.spectrumCacheBins
	profile := a.referenceProfile()

	render.DrawConsolePanel(screen, 20, 50, 1100, 700)
	render.DrawMonitor(screen, spectrumTableX, spectrumTableY, spectrumTableW, spectrumTableRow*6+44)
	render.DrawMonitor(screen, spectrumChartX, spectrumRefY-10, spectrumChartW, spectrumPanelH*2+60)
	render.DrawText(screen, "SPECTRUM ANALYZER", 40, 90, render.ColorPlateLabel, true)
	a.drawArraySelector(screen, sonar, spectrumArrayLabelX, spectrumArrayLabelY, cachedSpectrumArrayButtons())

	if c := a.selectedContact(sonar); c != nil {
		render.DrawText(screen, fmt.Sprintf("SELECTED: %s  |  BRG %.0f°  |  R %.1f kyd  |  SPD %.1f kts  |  %s  |  %s",
			contactLongLabel(c), c.BearingDeg, c.EstimatedRangeYd/1000, player.SpeedKts, contactTypeLabel(c), c.DetectedBy),
			40, 232, render.ColorAmber, false)
	} else {
		render.DrawText(screen, fmt.Sprintf("MANUAL BEARING: %.0f°  |  SPD %.1f kts  — select a contact in the table or on PASSIVE",
			bearing, player.SpeedKts), 40, 232, render.ColorDim, false)
	}
	render.DrawText(screen, fmt.Sprintf("Array: %s  |  %s  |  Layer: %s  |  Speed %.1f kts",
		a.sonarArrayLabel(sonar), a.towedCableStatus(sonar),
		a.Engine.Acoustics.Env.LayerNameKnown(player.DepthFt), player.SpeedKts), 40, 256, render.ColorPhosphorDim, true)

	a.drawSpectrumContactTable(screen, sonar)
	a.drawSpectrumChart(screen, bins, profile, bearing)

	mx, my := ebiten.CursorPosition()
	if a.uiTooltip != "" {
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
	render.DrawText(screen, "[B] array  < > cycle profile  CLASSIFY  LEFT/RIGHT manual bearing", 40, 720, render.ColorDim, true)
}
