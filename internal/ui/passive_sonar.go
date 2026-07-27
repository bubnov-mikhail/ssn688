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
	player := a.Engine.Scenario.Player
	best := -1
	bestR := 1e9
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if !a.contactAudibleOnArray(c, player, sonar) {
			continue
		}
		if !a.contactPlotHit(mx, my, c) {
			continue
		}
		r := contactBlobRadiusPx(c)
		if r < bestR {
			bestR = r
			best = i
		}
	}
	if best >= 0 {
		a.selectContact(sonar, &sonar.Contacts[best])
		return
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
		panelX = 952
		labelW = 28
		panelW = 316
		chipY0 = 250
		chipY1 = 268
		chipH  = 16
		gap    = 4
	)
	plotX := panelX + labelW
	plotW := panelW - labelW

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
		cx := plotX + acoustics.HeadingToWaterfallX(c.BearingDeg, plotW)
		items = append(items, pending{id: c.SourceEntityID, cx: cx, w: w, brg: c.BearingDeg})
	}
	// Left-to-right so we can pack without overlap.
	sort.Slice(items, func(i, j int) bool {
		if items[i].cx == items[j].cx {
			return items[i].brg < items[j].brg
		}
		return items[i].cx < items[j].cx
	})

	rowEnd := [2]int{panelX - gap, panelX - gap}
	rowY := [2]int{chipY0, chipY1}
	var chips []contactChip
	for _, it := range items {
		chipX := it.cx - it.w/2
		if chipX < panelX {
			chipX = panelX
		}
		if chipX+it.w > panelX+panelW {
			chipX = panelX + panelW - it.w
		}
		// Prefer the row that can fit without overlap; else the one with more room.
		row := 0
		fits0 := chipX >= rowEnd[0]+gap
		fits1 := chipX >= rowEnd[1]+gap
		switch {
		case fits0 && fits1:
			if rowEnd[0] <= rowEnd[1] {
				row = 0
			} else {
				row = 1
			}
		case fits0:
			row = 0
		case fits1:
			row = 1
		default:
			// Both crowded: nudge to the right of the less-extended row.
			if rowEnd[0] <= rowEnd[1] {
				row = 0
			} else {
				row = 1
			}
			chipX = rowEnd[row] + gap
			if chipX+it.w > panelX+panelW {
				chipX = panelX + panelW - it.w
			}
		}
		if chipX < panelX {
			chipX = panelX
		}
		chips = append(chips, contactChip{
			SourceID: it.id,
			X: chipX, Y: rowY[row], W: it.w, H: chipH,
		})
		end := chipX + it.w
		if end > rowEnd[row] {
			rowEnd[row] = end
		}
	}
	return chips
}

// drawSoftDisk draws a softly dissolved disk; pixels outside clipR from (clipCX,clipCY) are skipped.
func drawSoftDisk(screen *ebiten.Image, cx, cy, r float64, clr color.RGBA, clipCX, clipCY, clipR float64) {
	if r < 1 {
		return
	}
	ri := int(math.Ceil(r * 1.15))
	clipR2 := clipR * clipR
	for dy := -ri; dy <= ri; dy++ {
		for dx := -ri; dx <= ri; dx++ {
			px := cx + float64(dx)
			py := cy + float64(dy)
			if clipR > 0 {
				cdx := px - clipCX
				cdy := py - clipCY
				if cdx*cdx+cdy*cdy > clipR2 {
					continue
				}
			}
			dist := math.Sqrt(float64(dx*dx + dy*dy))
			t := dist / r
			if t > 1.25 {
				continue
			}
			// Smooth dissolve: soft core, long exponential tail (no hard rim).
			fall := math.Exp(-2.8 * t * t)
			if t > 0.85 {
				fall *= math.Max(0, 1-(t-0.85)/0.4)
			}
			alpha := uint8(float64(clr.A) * fall)
			if alpha < 3 {
				continue
			}
			render.FillRect(screen, int(px), int(py), 1, 1, color.RGBA{clr.R, clr.G, clr.B, alpha})
		}
	}
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
	bracket := render.ColorAmber
	if hover && !selected {
		bracket = color.RGBA{255, 255, 120, 200}
	}
	arm := 5
	render.DrawLine(screen, float64(x-half), float64(y-half), float64(x-half+arm), float64(y-half), bracket)
	render.DrawLine(screen, float64(x-half), float64(y-half), float64(x-half), float64(y-half+arm), bracket)
	render.DrawLine(screen, float64(x+half), float64(y-half), float64(x+half-arm), float64(y-half), bracket)
	render.DrawLine(screen, float64(x+half), float64(y-half), float64(x+half), float64(y-half+arm), bracket)
	render.DrawLine(screen, float64(x-half), float64(y+half), float64(x-half+arm), float64(y+half), bracket)
	render.DrawLine(screen, float64(x-half), float64(y+half), float64(x-half), float64(y+half-arm), bracket)
	render.DrawLine(screen, float64(x+half), float64(y+half), float64(x+half-arm), float64(y+half), bracket)
	render.DrawLine(screen, float64(x+half), float64(y+half), float64(x+half), float64(y+half-arm), bracket)
	render.DrawText(screen, c.ID, x+half+4, y+4, bracket, true)
}

func aContactCenter(c *acoustics.Contact) (float64, float64) {
	rad := c.BearingDeg * math.Pi / 180
	rngFrac := math.Min(1, c.EstimatedRangeYd/passivePlotMaxRangeYd)
	dist := rngFrac * (passivePlotR - 28)
	return passivePlotCX + math.Sin(rad)*dist, passivePlotCY - math.Cos(rad)*dist
}

func (a *App) drawPassiveBearingPlot(screen *ebiten.Image, player *world.Entity, sonar *acoustics.SonarState) {
	cx, cy, radius := passivePlotCX, passivePlotCY, passivePlotR
	arrayLabel := "HULL"
	if sonar.PassiveArray == acoustics.PassiveArrayTowed {
		arrayLabel = "TOWED"
	}
	render.DrawText(screen, fmt.Sprintf("BEARING PPI — %s", arrayLabel), int(cx)-70, int(cy-radius)-42, render.ColorPhosphorDim, true)

	// Dark navy face like the marine electronics reference.
	drawFilledCircle(screen, cx, cy, radius, color.RGBA{0, 2, 18, 255})

	// Polar BTR: same energy history as the waterfall for the selected array.
	a.drawPolarBearingField(screen, sonar, cx, cy, radius)

	for _, rngYd := range []float64{2000, 4000, 6000, 8000} {
		r := rngYd / passivePlotMaxRangeYd * (radius - 28)
		drawCircle(screen, cx, cy, r, color.RGBA{0, 120, 70, 160})
		lbl := fmt.Sprintf("%.0fk", rngYd/1000)
		render.DrawText(screen, lbl, int(cx)+6, int(cy-r)-4, render.ColorPhosphorDim, true)
	}

	for deg := 0; deg < 360; deg += 30 {
		rad := float64(deg) * math.Pi / 180
		inner := radius - 8
		render.DrawLine(screen, cx+math.Sin(rad)*14, cy-math.Cos(rad)*14,
			cx+math.Sin(rad)*inner, cy-math.Cos(rad)*inner, color.RGBA{0, 100, 60, 140})
	}
	drawCircle(screen, cx, cy, radius, color.RGBA{0, 180, 110, 220})

	labels := map[int]string{0: "000", 90: "090", 180: "180", 270: "270"}
	for deg, lbl := range labels {
		rad := float64(deg) * math.Pi / 180
		lx := cx + math.Sin(rad)*(radius+14)
		ly := cy - math.Cos(rad)*(radius+14)
		render.DrawText(screen, lbl, int(lx)-12, int(ly)+4, render.ColorPhosphor, false)
	}

	hrad := player.HeadingDeg * math.Pi / 180
	render.DrawLine(screen, cx, cy, cx+math.Sin(hrad)*(radius-36), cy-math.Cos(hrad)*(radius-36), render.ColorAmber)
	render.DrawLine(screen, cx, cy, cx+math.Sin(hrad)*42, cy-math.Cos(hrad)*42, render.ColorHighlight)

	mx, my := ebiten.CursorPosition()
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if !a.contactAudibleOnArray(c, player, sonar) {
			continue
		}
		selected := c.SourceEntityID == a.selectedContactID
		hover := a.contactPlotHit(mx, my, c)
		drawContactMarker(screen, c, selected, hover)
	}

	a.drawActiveSonarWashCircle(screen, sonar, cx, cy, radius)
}

// drawPolarBearingField paints the selected array's current bearing energy as a
// grainy heat PPI (blue→red), matching the waterfall for that array.
func (a *App) drawPolarBearingField(screen *ebiten.Image, sonar *acoustics.SonarState, cx, cy, radius float64) {
	wf := a.bearingWaterfalls.ForArray(sonar.PassiveArray)
	latest := wf.Latest()
	if latest == nil || len(latest.Bearings) == 0 {
		return
	}

	innerR := 16.0
	outerR := radius - 4
	bins := len(latest.Bearings)
	step := 2
	noiseRng := rand.New(rand.NewSource(phosphorNoiseSeed() ^ int64(sonar.PassiveArray)*0x9e37 ^ int64(latest.Heading*10)))

	// Optional short persistence from a few recent rows (phosphor trail).
	persist := wf.rows
	if len(persist) > 6 {
		persist = persist[:6]
	}

	r2max := (radius - 1) * (radius - 1)
	for bi := 0; bi < bins; bi += step {
		peak := 0.0
		for _, row := range persist {
			if bi >= len(row.Bearings) {
				continue
			}
			v := row.Bearings[bi]
			for k := 1; k < step && bi+k < len(row.Bearings); k++ {
				if row.Bearings[bi+k] > v {
					v = row.Bearings[bi+k]
				}
			}
			if v > peak {
				peak = v
			}
		}
		floor := 0.25 + noiseRng.Float64()*0.65
		if peak < floor {
			peak = floor
		}
		peak *= 0.85 + 0.28*noiseRng.Float64()
		baseI := snrToIntensity(peak)
		if baseI < 0.025 {
			continue
		}

		bearing := float64(bi) / float64(bins) * 360
		rad := bearing * math.Pi / 180
		sinB, cosB := math.Sin(rad), math.Cos(rad)

		// Radial cells: stronger energy extends farther and runs hotter (reference look).
		reach := innerR + (outerR-innerR)*(0.35+0.65*baseI)
		for r := innerR; r < reach; r += 3.0 {
			rngFrac := (r - innerR) / (outerR - innerR)
			// Near-field clutter bias + peak mid/far for strong contacts.
			shape := 0.45 + 0.55*rngFrac
			if baseI > 0.55 {
				shape = 0.35 + 0.9*math.Exp(-math.Pow((rngFrac-0.55)/0.28, 2))
			}
			intensity := baseI * shape * (0.75 + 0.4*noiseRng.Float64())
			if intensity < 0.03 {
				continue
			}
			px := cx + sinB*r
			py := cy - cosB*r
			dx := px - cx
			dy := py - cy
			if dx*dx+dy*dy > r2max {
				continue
			}
			arcW := math.Max(2, r*float64(step)*math.Pi/180*1.2)
			render.FillRect(screen, int(px-arcW*0.5), int(py-1.5), int(arcW+0.5), 3, sonarHeatColor(intensity))
		}
	}
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

// activeSonarWashIntensity returns 0..1 bloom from ownship active ping deafening passive arrays.
func (a *App) activeSonarWashIntensity(sonar *acoustics.SonarState) float64 {
	if sonar == nil || !sonar.ActiveEnabled || a.Engine == nil {
		return 0
	}
	gt := a.Engine.Clock.GameTime
	age := gt - sonar.LastPingTime
	if age < 0 || age > 4.5 {
		// Idle active mode: faint persistent wash.
		return 0.12 * sonar.ActivePower
	}
	// Strong flash that decays after each ping.
	flash := math.Exp(-age * 1.15)
	return math.Min(1, (0.18+0.82*flash)*sonar.ActivePower)
}

func (a *App) drawActiveSonarWashCircle(screen *ebiten.Image, sonar *acoustics.SonarState, cx, cy, radius float64) {
	wash := a.activeSonarWashIntensity(sonar)
	if wash < 0.02 {
		return
	}
	baseA := uint8(40 + wash*90)
	drawSoftDisk(screen, cx, cy, radius*0.95, color.RGBA{180, 230, 255, baseA}, cx, cy, radius)
	drawSoftDisk(screen, cx, cy, radius*0.55, color.RGBA{210, 245, 255, uint8(float64(baseA)*0.7)}, cx, cy, radius)
	if wash > 0.35 {
		drawSoftDisk(screen, cx, cy, radius*0.22, color.RGBA{240, 250, 255, uint8(wash*70)}, cx, cy, radius)
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
