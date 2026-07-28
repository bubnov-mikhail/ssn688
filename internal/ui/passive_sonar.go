package ui

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/world"
)

const (
	passivePlotCX         = 470.0
	passivePlotCY         = 430.0
	passivePlotR          = 222.0
	passivePlotMaxRangeYd = 8000.0
	passiveListX          = 950
	passiveListY          = 278
	passiveListW          = 310
	passiveListRow        = 22
)

type contactChip struct {
	SourceID   string
	X, Y, W, H int
}

func (a *App) selectContact(sonar *acoustics.SonarState, c *acoustics.Contact) {
	a.selectedContactID = c.SourceEntityID
	sonar.SpectrumBearing = c.BearingDeg
	if c.BestMatchID != "" {
		for i, p := range world.SignatureLibrary {
			if p.ID == c.BestMatchID {
				a.referenceProfileIdx = i
				break
			}
		}
	}
}

func (a *App) selectedContact(sonar *acoustics.SonarState) *acoustics.Contact {
	for i := range sonar.Contacts {
		if sonar.Contacts[i].SourceEntityID == a.selectedContactID {
			return &sonar.Contacts[i]
		}
	}
	return nil
}

func (a *App) validateSelectedContact(sonar *acoustics.SonarState) {
	if a.selectedContactID == "" {
		return
	}
	if a.selectedContact(sonar) == nil {
		a.selectedContactID = ""
	}
}

func (a *App) contactPlotPos(c *acoustics.Contact) (x, y int) {
	rad := c.BearingDeg * math.Pi / 180
	rngFrac := math.Min(1, c.EstimatedRangeYd/passivePlotMaxRangeYd)
	dist := rngFrac * (passivePlotR - 28)
	return int(passivePlotCX + math.Sin(rad)*dist), int(passivePlotCY - math.Cos(rad)*dist)
}

func contactBlobRadiusPx(c *acoustics.Contact) float64 {
	uncR := c.UncRangeYd
	if uncR <= 0 {
		uncR = math.Max(500, c.EstimatedRangeYd*0.35)
	}
	uncB := c.UncBearingDeg
	if uncB <= 0 {
		uncB = 18
	}
	rangePx := uncR / passivePlotMaxRangeYd * (passivePlotR - 28)
	arcPx := uncB * math.Pi / 180 * math.Min(passivePlotR-28, math.Max(40, c.EstimatedRangeYd/passivePlotMaxRangeYd*(passivePlotR-28)))
	r := math.Max(rangePx, arcPx)
	if r < 10 {
		r = 10
	}
	if r > 70 {
		r = 70
	}
	return r
}

func (a *App) contactPlotHit(mx, my int, c *acoustics.Contact) bool {
	x, y := a.contactPlotPos(c)
	r := contactBlobRadiusPx(c) + 4
	dx := float64(mx - x)
	dy := float64(my - y)
	return dx*dx+dy*dy <= r*r
}

func (a *App) spectrumAnalysisBearing(sonar *acoustics.SonarState) float64 {
	if c := a.selectedContact(sonar); c != nil {
		return c.BearingDeg
	}
	return sonar.SpectrumBearing
}

func (a *App) contactAudibleOnArray(c *acoustics.Contact, player *world.Entity, sonar *acoustics.SonarState) bool {
	if player == nil || sonar == nil {
		return true
	}
	rel := acoustics.AngleDiffDeg(c.BearingDeg, player.HeadingDeg)
	sens := acoustics.PassiveArraySensitivity(sonar.PassiveArray, rel, sonar.TowedCablePct)
	return sens >= 0.18
}

func (a *App) updatePassiveInput(sonar *acoustics.SonarState) {
	a.validateSelectedContact(sonar)
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	mx, my := ebiten.CursorPosition()
	y := passiveListY + passiveListRow
	for i := range sonar.Contacts {
		if mx >= passiveListX && mx < passiveListX+passiveListW && my >= y && my < y+passiveListRow {
			a.selectContact(sonar, &sonar.Contacts[i])
			return
		}
		y += passiveListRow
	}
	for _, chip := range a.waterfallContactChips(sonar) {
		if mx >= chip.X && mx < chip.X+chip.W && my >= chip.Y && my < chip.Y+chip.H {
			for j := range sonar.Contacts {
				if sonar.Contacts[j].SourceEntityID == chip.SourceID {
					a.selectContact(sonar, &sonar.Contacts[j])
					return
				}
			}
		}
	}
}

func (a *App) waterfallContactChips(sonar *acoustics.SonarState) []contactChip {
	const (
		panelX = waterfallPlotX
		panelW = waterfallPlotW
		chipY0 = waterfallPanelY + 34
		chipH  = 16
		gap    = 6
	)

	type pending struct {
		id   string
		cx   int
		w    int
		brg  float64
	}
	var items []pending
	player := a.Engine.Scenario.Player
	for _, c := range sonar.Contacts {
		if !a.contactAudibleOnArray(&c, player, sonar) {
			continue
		}
		w := len(c.ID)*7 + 10
		cx := panelX + waterfallBearingDisplayX(c.BearingDeg, panelW)
		items = append(items, pending{id: c.SourceEntityID, cx: cx, w: w, brg: c.BearingDeg})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].cx == items[j].cx {
			return items[i].brg < items[j].brg
		}
		return items[i].cx < items[j].cx
	})

	rowEnd := panelX - gap
	var chips []contactChip
	for _, it := range items {
		chipX := it.cx - it.w/2
		if chipX < panelX {
			chipX = panelX
		}
		if chipX+it.w > panelX+panelW {
			chipX = panelX + panelW - it.w
		}
		if chipX < rowEnd+gap {
			chipX = rowEnd + gap
			if chipX+it.w > panelX+panelW {
				chipX = panelX + panelW - it.w
			}
		}
		if chipX < panelX {
			chipX = panelX
		}
		chips = append(chips, contactChip{
			SourceID: it.id,
			X: chipX, Y: chipY0, W: it.w, H: chipH,
		})
		end := chipX + it.w
		if end > rowEnd {
			rowEnd = end
		}
	}
	return chips
}

func drawContactMarker(screen *ebiten.Image, c *acoustics.Contact, selected, hover bool) {
	if !selected && !hover {
		return
	}
	cx, cy := aContactCenter(c)
	r := contactBlobRadiusPx(c) * 0.45
	if r < 10 {
		r = 10
	}
	if r > 28 {
		r = 28
	}
	half := int(r)
	x, y := int(cx), int(cy)
	arm := 5
	// High-contrast frame on heat map: white corners with black outline.
	drawCornerBracket(screen, x, y, half, arm, color.RGBA{0, 0, 0, 255}, 1)
	fg := color.RGBA{255, 255, 255, 255}
	if hover && !selected {
		fg = color.RGBA{220, 220, 220, 255}
	}
	drawCornerBracket(screen, x, y, half, arm, fg, 0)
	render.DrawText(screen, c.ID, x+half+4, y+4, color.RGBA{0, 0, 0, 255}, true)
	render.DrawText(screen, c.ID, x+half+3, y+3, fg, true)
}

func drawCornerBracket(screen *ebiten.Image, x, y, half, arm int, clr color.RGBA, outset int) {
	h := half + outset
	a := arm + outset
	render.DrawLine(screen, float64(x-h), float64(y-h), float64(x-h+a), float64(y-h), clr)
	render.DrawLine(screen, float64(x-h), float64(y-h), float64(x-h), float64(y-h+a), clr)
	render.DrawLine(screen, float64(x+h), float64(y-h), float64(x+h-a), float64(y-h), clr)
	render.DrawLine(screen, float64(x+h), float64(y-h), float64(x+h), float64(y-h+a), clr)
	render.DrawLine(screen, float64(x-h), float64(y+h), float64(x-h+a), float64(y+h), clr)
	render.DrawLine(screen, float64(x-h), float64(y+h), float64(x-h), float64(y+h-a), clr)
	render.DrawLine(screen, float64(x+h), float64(y+h), float64(x+h-a), float64(y+h), clr)
	render.DrawLine(screen, float64(x+h), float64(y+h), float64(x+h), float64(y+h-a), clr)
}

func aContactCenter(c *acoustics.Contact) (float64, float64) {
	rad := c.BearingDeg * math.Pi / 180
	rngFrac := math.Min(1, c.EstimatedRangeYd/passivePlotMaxRangeYd)
	dist := rngFrac * (passivePlotR - 28)
	return passivePlotCX + math.Sin(rad)*dist, passivePlotCY - math.Cos(rad)*dist
}

func (a *App) drawPassiveContactTable(screen *ebiten.Image, sonar *acoustics.SonarState) {
	render.FillRect(screen, passiveListX, passiveListY, passiveListW, passiveListRow*max(1, len(sonar.Contacts)+1)+14, render.ColorPanelInset)
	render.DrawText(screen, "CONTACT", passiveListX+8, passiveListY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "BRG", passiveListX+72, passiveListY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "RNG", passiveListX+112, passiveListY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "SRC", passiveListX+158, passiveListY+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "CLASS", passiveListX+200, passiveListY+16, render.ColorPhosphorDim, true)

	mx, my := ebiten.CursorPosition()
	player := a.Engine.Scenario.Player
	y := passiveListY + passiveListRow
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		selected := c.SourceEntityID == a.selectedContactID
		hover := mx >= passiveListX && mx < passiveListX+passiveListW && my >= y && my < y+passiveListRow
		if selected {
			render.FillRect(screen, passiveListX+2, y, passiveListW-4, passiveListRow, color.RGBA{80, 60, 0, 180})
		} else if hover {
			render.FillRect(screen, passiveListX+2, y, passiveListW-4, passiveListRow, render.ColorPanelMid)
		}
		clr := render.ColorPhosphor
		if selected {
			clr = render.ColorAmber
		}
		render.DrawText(screen, c.ID, passiveListX+8, y+16, clr, true)
		render.DrawText(screen, fmt.Sprintf("%03.0f", c.BearingDeg), passiveListX+72, y+16, clr, true)
		render.DrawText(screen, contactRangeLabel(c), passiveListX+112, y+16, clr, true)
		render.DrawText(screen, contactSourceLabel(c, player, sonar), passiveListX+158, y+16, clr, true)
		render.DrawText(screen, contactClassLabel(c), passiveListX+200, y+16, clr, true)
		y += passiveListRow
	}
}

// ppiPixelLUT caches polar mapping for the PPI raster (avoids atan2 per rebuild).
type ppiPixelLUT struct {
	bi0    uint16
	frac8  uint8 // 0..255 blend to next bin
	rFrac8 uint8 // 0..255 radial fraction
	inside bool
}

// drawPolarBearingField paints the selected array's current bearing energy as a
// full-disk heat PPI. Low-res cached raster, rebuilt only when waterfall advances.
func (a *App) drawPolarBearingField(screen *ebiten.Image, sonar *acoustics.SonarState, cx, cy, radius float64) {
	const scale = 3.0
	size := int(radius*2/scale) + 1
	if size < 4 {
		return
	}
	if a.passivePPI == nil || a.passivePPI.Bounds().Dx() != size {
		a.passivePPI = ebiten.NewImage(size, size)
		a.passivePPIPixels = make([]byte, size*size*4)
		a.buildPPILUT(size)
		a.passivePPIPending = true
	}

	if a.passivePPIPending || a.passivePPIArray != sonar.PassiveArray {
		a.rebuildPassivePPI(sonar, size)
		a.passivePPIPending = false
		a.passivePPIArray = sonar.PassiveArray
		a.passivePPIStamp = a.lastWaterfallSample
	}

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(cx-radius, cy-radius)
	screen.DrawImage(a.passivePPI, op)
}

func (a *App) buildPPILUT(size int) {
	a.ppiLUTSize = size
	a.ppiLUT = make([]ppiPixelLUT, size*size)
	bins := acoustics.BearingWaterfallBins
	cx := float64(size) / 2
	cy := float64(size) / 2
	rMax := cx - 0.5
	rMax2 := rMax * rMax
	innerR := 1.5
	innerR2 := innerR * innerR
	for py := 0; py < size; py++ {
		dy := float64(py) + 0.5 - cy
		for px := 0; px < size; px++ {
			dx := float64(px) + 0.5 - cx
			r2 := dx*dx + dy*dy
			e := &a.ppiLUT[py*size+px]
			if r2 > rMax2 || r2 < innerR2 {
				continue
			}
			r := math.Sqrt(r2)
			bearing := math.Atan2(dx, -dy) * 180 / math.Pi
			if bearing < 0 {
				bearing += 360
			}
			bf := bearing / 360 * float64(bins)
			bi0 := int(bf) % bins
			if bi0 < 0 {
				bi0 += bins
			}
			frac := bf - math.Floor(bf)
			e.bi0 = uint16(bi0)
			e.frac8 = uint8(frac * 255)
			e.rFrac8 = uint8((r - innerR) / (rMax - innerR) * 255)
			e.inside = true
		}
	}
}

func (a *App) ensurePPIScratch(bins int) {
	if cap(a.ppiEnergies) < bins {
		a.ppiEnergies = make([]float64, bins)
		a.ppiSmoothed = make([]float64, bins)
		a.ppiFloorN = make([]float64, bins)
		a.ppiGrainN = make([]float64, bins)
	} else {
		a.ppiEnergies = a.ppiEnergies[:bins]
		a.ppiSmoothed = a.ppiSmoothed[:bins]
		a.ppiFloorN = a.ppiFloorN[:bins]
		a.ppiGrainN = a.ppiGrainN[:bins]
	}
}

func (a *App) rebuildPassivePPI(sonar *acoustics.SonarState, size int) {
	if a.ppiLUTSize != size || len(a.ppiLUT) != size*size {
		a.buildPPILUT(size)
	}
	wf := a.bearingWaterfalls.ForArray(sonar.PassiveArray)
	bins := acoustics.BearingWaterfallBins
	a.ensurePPIScratch(bins)
	energies := a.ppiEnergies
	for i := range energies {
		energies[i] = 0
	}
	heading := 0.0
	if a.Engine != nil && a.Engine.Scenario.Player != nil {
		heading = a.Engine.Scenario.Player.HeadingDeg
	}
	if latest := wf.Latest(); latest != nil && len(latest.Bearings) > 0 {
		heading = latest.Heading
		persistN := wf.Len()
		if persistN > 5 {
			persistN = 5
		}
		for bi := 0; bi < bins; bi++ {
			peak := 0.0
			for age := 0; age < persistN; age++ {
				row := wf.Row(age)
				if row == nil || bi >= len(row.Bearings) {
					continue
				}
				if row.Bearings[bi] > peak {
					peak = row.Bearings[bi]
				}
			}
			energies[bi] = peak
		}
	}

	smoothed := a.ppiSmoothed
	for bi := 0; bi < bins; bi++ {
		l := energies[(bi-1+bins)%bins]
		r := energies[(bi+1)%bins]
		smoothed[bi] = energies[bi]*0.6 + l*0.2 + r*0.2
	}

	noiseRng := rand.New(rand.NewSource(int64(a.lastWaterfallSample*1000) ^ int64(sonar.PassiveArray)*0x9e37 ^ int64(heading*10)))
	floorN := a.ppiFloorN
	grainN := a.ppiGrainN
	for bi := 0; bi < bins; bi++ {
		floorN[bi] = 0.32 + noiseRng.Float64()*0.38
		grainN[bi] = 0.92 + 0.12*noiseRng.Float64()
	}

	// Precompute sensitivity per bearing bin for current heading/array.
	if cap(a.ppiSens) < bins {
		a.ppiSens = make([]float64, bins)
	} else {
		a.ppiSens = a.ppiSens[:bins]
	}
	sensBin := a.ppiSens
	for bi := 0; bi < bins; bi++ {
		bearing := float64(bi) / float64(bins) * 360
		rel := acoustics.AngleDiffDeg(bearing, heading)
		sensBin[bi] = acoustics.PassiveArraySensitivity(sonar.PassiveArray, rel, sonar.TowedCablePct)
	}

	pix := a.passivePPIPixels
	for i := range pix {
		pix[i] = 0
	}
	lut := a.ppiLUT
	for i, e := range lut {
		if !e.inside {
			continue
		}
		bi0 := int(e.bi0)
		bi1 := (bi0 + 1) % bins
		frac := float64(e.frac8) / 255
		peak := smoothed[bi0]*(1-frac) + smoothed[bi1]*frac
		sens := sensBin[bi0]*(1-frac) + sensBin[bi1]*frac

		floor := (floorN[bi0]*(1-frac) + floorN[bi1]*frac) * (0.1 + 0.9*sens)
		if peak < floor {
			peak = floor
		}
		peak *= grainN[bi0]*(1-frac) + grainN[bi1]*frac
		baseI := snrToIntensity(peak) * (0.2 + 0.8*sens)
		if baseI < 0.015 {
			baseI = 0.015 * sens
		}
		rngFrac := float64(e.rFrac8) / 255
		shape := 0.55 + 0.45*rngFrac
		if baseI > 0.55 {
			dr := rngFrac - 0.55
			shape = 0.4 + 0.85*math.Exp(-(dr*dr)/(0.3*0.3))
		}
		intensity := baseI * shape
		if sens < 0.25 {
			intensity *= 0.35 + 0.4*sens
		}
		if sens < 0.28 {
			intensity *= 0.45 + 0.55*(sens/0.28)
		}
		if intensity < 0.012 {
			intensity = 0.012
		}
		clr := sonarHeatColorFast(intensity)
		off := i * 4
		pix[off] = clr.R
		pix[off+1] = clr.G
		pix[off+2] = clr.B
		pix[off+3] = 255
	}
	a.passivePPI.WritePixels(pix)
}

func drawFilledCircle(screen *ebiten.Image, cx, cy, r float64, clr color.RGBA) {
	ri := int(math.Ceil(r))
	r2 := r * r
	for dy := -ri; dy <= ri; dy++ {
		xx := r2 - float64(dy*dy)
		if xx < 0 {
			continue
		}
		dx := int(math.Sqrt(xx))
		render.FillRect(screen, int(cx)-dx, int(cy)+dy, 2*dx+1, 1, clr)
	}
}

func (a *App) drawWaterfallContactChips(screen *ebiten.Image, sonar *acoustics.SonarState) {
	mx, my := ebiten.CursorPosition()
	for _, chip := range a.waterfallContactChips(sonar) {
		selected := false
		for i := range sonar.Contacts {
			if sonar.Contacts[i].SourceEntityID == chip.SourceID {
				selected = sonar.Contacts[i].SourceEntityID == a.selectedContactID
				break
			}
		}
		hover := mx >= chip.X && mx < chip.X+chip.W && my >= chip.Y && my < chip.Y+chip.H
		bg := render.ColorPanelInset
		if selected {
			bg = color.RGBA{80, 60, 0, 255}
		} else if hover {
			bg = render.ColorPanelMid
		}
		render.FillRect(screen, chip.X, chip.Y, chip.W, chip.H, bg)
		if selected {
			render.DrawLine(screen, float64(chip.X), float64(chip.Y), float64(chip.X+chip.W), float64(chip.Y), render.ColorAmber)
		}
		label := ""
		for i := range sonar.Contacts {
			if sonar.Contacts[i].SourceEntityID == chip.SourceID {
				label = sonar.Contacts[i].ID
				break
			}
		}
		clr := render.ColorPhosphor
		if selected {
			clr = render.ColorAmber
		}
		render.DrawText(screen, label, chip.X+5, chip.Y+12, clr, true)
	}
}

func (a *App) updatePassiveScreen(sonar *acoustics.SonarState) {
	a.updatePassiveInput(sonar)
}
