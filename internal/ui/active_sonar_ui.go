package ui

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/audio"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/render"
)

const activeEchoMarkerFadeSec = acoustics.ActiveFixHoldSec // dissolve time; matches tactical active-fix hold

// activeEchoFlash is a fixed range-bearing snapshot from one active echo return.
type activeEchoFlash struct {
	ID             uint64
	SourceEntityID string
	ContactID      string
	BearingDeg     float64
	RangeYd        float64
	SNR            float64
	PingTime       float64
	LastEchoAt     float64
	Strength       float64 // 1 = bright; stepped down on newer echoes from same contact
}

var cachedActiveRangeScale struct {
	mu     sync.Mutex
	lang   string
	btnX   int
	btns   []sonarUIButton
}

var cachedActivePingInterval struct {
	mu     sync.Mutex
	lang   string
	btnX   int
	btns   []sonarUIButton
}

var cachedActivePower struct {
	mu     sync.Mutex
	lang   string
	btnX   int
	btns   []sonarUIButton
}

func (a *App) cachedActivePingIntervalButtons() []sonarUIButton {
	lang := a.Lang()
	btnX := activeControlBtnX()
	cachedActivePingInterval.mu.Lock()
	defer cachedActivePingInterval.mu.Unlock()
	if cachedActivePingInterval.lang == lang && cachedActivePingInterval.btnX == btnX && cachedActivePingInterval.btns != nil {
		return cachedActivePingInterval.btns
	}
	cachedActivePingInterval.btns = layoutButtonRow(btnX, activePingRowY, 28, 4, []buttonSpec{
		{"active_ping_3", "3", "Auto ping every 3 seconds"},
		{"active_ping_6", "6", "Auto ping every 6 seconds"},
		{"active_ping_12", "12", "Auto ping every 12 seconds"},
		{"active_ping_24", "24", "Auto ping every 24 seconds"},
	})
	cachedActivePingInterval.lang = lang
	cachedActivePingInterval.btnX = btnX
	return cachedActivePingInterval.btns
}

func (a *App) cachedActivePowerButtons() []sonarUIButton {
	lang := a.Lang()
	btnX := activeControlBtnX()
	cachedActivePower.mu.Lock()
	defer cachedActivePower.mu.Unlock()
	if cachedActivePower.lang == lang && cachedActivePower.btnX == btnX && cachedActivePower.btns != nil {
		return cachedActivePower.btns
	}
	cachedActivePower.btns = layoutButtonRow(btnX, activePowerRowY, 28, 4, []buttonSpec{
		{"active_power_15", "15%", "Transmit power 15%"},
		{"active_power_25", "25%", "Transmit power 25%"},
		{"active_power_75", "75%", "Transmit power 75%"},
		{"active_power_100", "100%", "Transmit power 100%"},
	})
	cachedActivePower.lang = lang
	cachedActivePower.btnX = btnX
	return cachedActivePower.btns
}

func (a *App) cachedActiveRangeScaleButtons() []sonarUIButton {
	lang := a.Lang()
	btnX := activeControlBtnX()
	cachedActiveRangeScale.mu.Lock()
	defer cachedActiveRangeScale.mu.Unlock()
	if cachedActiveRangeScale.lang == lang && cachedActiveRangeScale.btnX == btnX && cachedActiveRangeScale.btns != nil {
		return cachedActiveRangeScale.btns
	}
	cachedActiveRangeScale.btns = layoutButtonRow(btnX, activeRangeScaleY, 28, 4, []buttonSpec{
		{"active_range_2k", "2k", "Range scale 2 kyd"},
		{"active_range_6k", "6k", "Range scale 6 kyd"},
		{"active_range_12k", "12k", "Range scale 12 kyd"},
	})
	cachedActiveRangeScale.lang = lang
	cachedActiveRangeScale.btnX = btnX
	return cachedActiveRangeScale.btns
}

func (a *App) activeRangeMaxYd() float64 {
	if a.activeRangeScaleYd <= 0 {
		return acoustics.ActiveDisplayMaxRangeYd
	}
	return a.activeRangeScaleYd
}

func (a *App) markActivePlotGridDirty() {
	a.activePlotGridDirty = true
}

func (a *App) activePlotNeedsGridRebuild() bool {
	return a.activePlotGridDirty || a.activePlotImg == nil
}

func (a *App) setActiveRangeScale(yd float64) {
	if a.activeRangeScaleYd == yd {
		return
	}
	a.activeRangeScaleYd = yd
	a.activeEchoFlashes = nil
	a.activePlotGridScaleYd = 0
	a.markActivePlotGridDirty()
}

func (a *App) activeRangeButtonAction(id string) {
	switch id {
	case "active_range_2k":
		a.setActiveRangeScale(2000)
	case "active_range_6k":
		a.setActiveRangeScale(6000)
	case "active_range_12k":
		a.setActiveRangeScale(12000)
	}
}

func (a *App) activeSettingButtonAction(id string, sonar *acoustics.SonarState) {
	switch id {
	case "active_ping_3":
		sonar.PingInterval = 3
	case "active_ping_6":
		sonar.PingInterval = 6
	case "active_ping_12":
		sonar.PingInterval = 12
	case "active_ping_24":
		sonar.PingInterval = 24
	case "active_power_15":
		sonar.ActivePower = 0.15
	case "active_power_25":
		sonar.ActivePower = 0.25
	case "active_power_75":
		sonar.ActivePower = 0.75
	case "active_power_100":
		sonar.ActivePower = 1.0
	}
}

func activePingIntervalSelected(id string, interval float64) bool {
	switch id {
	case "active_ping_3":
		return interval == 3
	case "active_ping_6":
		return interval == 6
	case "active_ping_12":
		return interval == 12
	case "active_ping_24":
		return interval == 24
	default:
		return false
	}
}

func activePowerSelected(id string, power float64) bool {
	switch id {
	case "active_power_15":
		return math.Abs(power-0.15) < 0.01
	case "active_power_25":
		return math.Abs(power-0.25) < 0.01
	case "active_power_75":
		return math.Abs(power-0.75) < 0.01
	case "active_power_100":
		return math.Abs(power-1.0) < 0.01
	default:
		return false
	}
}

func (a *App) activeControlButtons(sonar *acoustics.SonarState) []sonarUIButton {
	toggleLabel := a.L(i18n.UIStandby)
	if sonar.ActiveEnabled {
		toggleLabel = a.L(i18n.UIActiveOn)
	}
	specs := []buttonSpec{
		{"active_toggle", toggleLabel, "Enable or disable active sonar transmit mode"},
		{"active_ping", a.L(i18n.UIPingNow), "Fire one immediate pulse (works in standby)"},
	}
	a.sonarBtnScratch = layoutButtonRowInto(a.sonarBtnScratch[:0], activeControlsX(), activeControlsY+10, 34, 6, specs)
	btns := append(a.sonarBtnScratch, a.cachedActivePingIntervalButtons()...)
	btns = append(btns, a.cachedActivePowerButtons()...)
	return append(btns, a.cachedActiveRangeScaleButtons()...)
}

func activeContactTableHeight() int {
	return passiveListRow*(activeListVisibleRows+1) + 14
}

func activeEchoFlashVisible(f activeEchoFlash, gameTime float64) bool {
	if f.Strength <= 0 {
		return false
	}
	return gameTime-f.LastEchoAt <= activeEchoMarkerFadeSec
}

func activeEchoDecayStrength(strength, dtSec float64) float64 {
	if dtSec <= 0 {
		return strength
	}
	return strength - dtSec/activeEchoMarkerFadeSec
}

func activeEchoSNRIntensity(snr float64) float64 {
	return snrToIntensity(snr)
}

func (a *App) activeFlashPlotPos(f activeEchoFlash, plotW, plotH int, maxR float64) (px, py int) {
	px = waterfallBearingDisplayX(f.BearingDeg, plotW)
	frac := f.RangeYd / maxR
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	py = plotH - 8 - int(frac*float64(plotH-16))
	return px, py
}

func activeFlashCrossHalfLen(snr float64) float64 {
	t := activeEchoSNRIntensity(snr)
	return activeFlashCrossMin + t*(activeFlashCrossMax-activeFlashCrossMin)
}

func (a *App) updateActiveScreen(sonar *acoustics.SonarState) {
	a.validateSelectedContact(sonar)
	buttons := a.activeControlButtons(sonar)
	mx, my := ebiten.CursorPosition()
	a.updateSonarTooltips(buttons, mx, my)
	listW := activeSideW - 40
	scrollContactTableWheel(mx, my, activeControlsX(), activeListY+passiveListRow, listW, activeListVisibleRows*passiveListRow, len(sonar.Contacts), activeListVisibleRows, &a.contactTableScroll.active)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for _, b := range buttons {
			if b.contains(mx, my) {
				switch {
				case b.ID == "active_toggle" || b.ID == "active_ping":
					a.activeSonarButtonAction(b.ID, sonar)
				case strings.HasPrefix(b.ID, "active_range_"):
					a.activeRangeButtonAction(b.ID)
				default:
					a.activeSettingButtonAction(b.ID, sonar)
				}
				a.uiPressedID = b.ID
				a.uiPressedAt = time.Now()
				return
			}
		}
		if a.activePlotClick(sonar, mx, my) {
			return
		}
		a.contactTableScroll.active = clampContactTableScroll(a.contactTableScroll.active, len(sonar.Contacts), activeListVisibleRows)
		start, end := contactTableWindow(len(sonar.Contacts), a.contactTableScroll.active, activeListVisibleRows)
		y := activeListY + passiveListRow
		for i := start; i < end; i++ {
			if mx >= activeControlsX() && mx < activeControlsX()+listW && my >= y && my < y+passiveListRow {
				a.selectContact(sonar, &sonar.Contacts[i])
				return
			}
			y += passiveListRow
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		a.activeSonarButtonAction("active_toggle", sonar)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		a.activeSonarButtonAction("active_ping", sonar)
	}
	a.updateActiveEchoFlashes(sonar)
}

func (a *App) activePlotClick(sonar *acoustics.SonarState, mx, my int) bool {
	plotX, plotY := activePlotX, activePlotY
	if !inRect(mx, my, plotX, plotY, activePlotW(), activePlotH) {
		return false
	}
	lx, ly := mx-plotX, my-plotY
	gameTime := a.activeVisualTime()
	bestID := uint64(0)
	bestDist := math.MaxFloat64
	bestEchoAt := -math.MaxFloat64
	for _, f := range a.activeEchoFlashes {
		if !activeEchoFlashVisible(f, gameTime) {
			continue
		}
		px, py := a.activeFlashPlotPos(f, activePlotW(), activePlotH, a.activeRangeMaxYd())
		dx := float64(lx - px)
		dy := float64(ly - py)
		r := activeFlashCrossHalfLen(f.SNR) + 4
		if dx*dx+dy*dy > r*r {
			continue
		}
		dist := dx*dx + dy*dy
		if f.LastEchoAt > bestEchoAt || (f.LastEchoAt == bestEchoAt && dist < bestDist) {
			bestEchoAt = f.LastEchoAt
			bestDist = dist
			bestID = f.ID
		}
	}
	if bestID == 0 {
		return false
	}
	var bestSourceID string
	for _, f := range a.activeEchoFlashes {
		if f.ID == bestID {
			bestSourceID = f.SourceEntityID
			break
		}
	}
	if bestSourceID == "" {
		return false
	}
	for i := range sonar.Contacts {
		if sonar.Contacts[i].SourceEntityID == bestSourceID {
			a.selectContact(sonar, &sonar.Contacts[i])
			return true
		}
	}
	return false
}

func (a *App) decayActiveEchoFlashesForContact(sourceID string, echoAt float64) {
	for i := range a.activeEchoFlashes {
		if a.activeEchoFlashes[i].SourceEntityID != sourceID {
			continue
		}
		dt := echoAt - a.activeEchoFlashes[i].LastEchoAt
		a.activeEchoFlashes[i].Strength = activeEchoDecayStrength(a.activeEchoFlashes[i].Strength, dt)
		a.activeEchoFlashes[i].LastEchoAt = echoAt
	}
}

func (a *App) purgeActiveEchoFlashes(gameTime float64) {
	kept := a.activeEchoFlashes[:0]
	for _, f := range a.activeEchoFlashes {
		if activeEchoFlashVisible(f, gameTime) {
			kept = append(kept, f)
		}
	}
	a.activeEchoFlashes = kept
}

func (a *App) updateActiveEchoFlashes(sonar *acoustics.SonarState) {
	if a.Engine == nil || sonar == nil {
		return
	}
	gameTime := a.activeVisualTime()
	a.purgeActiveEchoFlashes(gameTime)

	if sonar.LastPingTime <= 0 {
		return
	}

	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if !acoustics.ContactHasActiveRange(c) || c.DetectedBy != "active" {
			continue
		}
		if c.LastUpdate < sonar.LastPingTime {
			continue
		}
		crossAt := sonar.LastPingTime + acoustics.TwoWayTravelSec(c.EstimatedRangeYd)
		if gameTime < crossAt {
			continue
		}
		duplicate := false
		for _, f := range a.activeEchoFlashes {
			if f.SourceEntityID == c.SourceEntityID && f.PingTime == sonar.LastPingTime {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		a.decayActiveEchoFlashesForContact(c.SourceEntityID, crossAt)
		a.activeEchoFlashSeq++
		a.activeEchoFlashes = append(a.activeEchoFlashes, activeEchoFlash{
			ID:             a.activeEchoFlashSeq,
			SourceEntityID: c.SourceEntityID,
			ContactID:      c.ID,
			BearingDeg:     c.BearingDeg,
			RangeYd:        c.EstimatedRangeYd,
			SNR:            c.SNR,
			PingTime:       sonar.LastPingTime,
			LastEchoAt:     crossAt,
			Strength:       1,
		})
	}
	a.purgeActiveEchoFlashes(gameTime)
}

func (a *App) activeSonarButtonAction(id string, sonar *acoustics.SonarState) {
	switch id {
	case "active_toggle":
		sonar.ActiveEnabled = !sonar.ActiveEnabled
		if sonar.ActiveEnabled {
			a.Audio.PlayClip(audio.ClipSonarActiveOnline, a.L(i18n.VoiceActiveOnline))
		} else {
			a.Audio.PlayClip(audio.ClipSonarActiveStandby, a.L(i18n.VoiceActiveStandby))
		}
	case "active_ping":
		if a.Engine == nil || a.Engine.Scenario == nil {
			return
		}
		player := a.Engine.Scenario.Player
		emitters := a.Engine.AcousticEmitters()
		if acoustics.FireActivePingNow(a.Engine.Acoustics, player, emitters, sonar, a.Engine.Clock.GameTime) {
			a.Audio.PlayEnemyPing()
			if a.lastPingPlayed < sonar.LastPingTime {
				a.lastPingPlayed = sonar.LastPingTime
			}
		}
	}
}

func (a *App) drawActiveControls(screen *ebiten.Image, sonar *acoustics.SonarState) {
	const rowBtnH = 28
	render.DrawText(screen, a.L(i18n.UIActiveTX), activeControlsX(), activeSideLabelY+12, render.ColorPlateLabel, true)
	render.DrawText(screen, a.L(i18n.UIAutoPing), activeControlsX(), render.SmallLabelBaseline(activePingRowY, rowBtnH), render.ColorPhosphorDim, true)
	render.DrawText(screen, a.L(i18n.UIPower), activeControlsX(), render.SmallLabelBaseline(activePowerRowY, rowBtnH), render.ColorPhosphorDim, true)
	render.DrawText(screen, a.L(i18n.UIRange), activeControlsX(), render.SmallLabelBaseline(activeRangeScaleY, rowBtnH), render.ColorPhosphorDim, true)

	mx, my := ebiten.CursorPosition()
	buttons := a.activeControlButtons(sonar)
	scaleYd := a.activeRangeMaxYd()
	for _, b := range buttons {
		hover := b.contains(mx, my)
		pressed := a.uiPressedID == b.ID && time.Since(a.uiPressedAt) < 120*time.Millisecond
		selected := (b.ID == "active_range_2k" && scaleYd == 2000) ||
			(b.ID == "active_range_6k" && scaleYd == 6000) ||
			(b.ID == "active_range_12k" && scaleYd == 12000) ||
			activePingIntervalSelected(b.ID, sonar.PingInterval) ||
			activePowerSelected(b.ID, sonar.ActivePower)
		if b.ID == "active_toggle" && sonar.ActiveEnabled {
			render.FillRect(screen, b.X-2, b.Y-2, b.W+4, b.H+4, render.ColorActive)
		}
		if selected {
			render.FillRect(screen, b.X-2, b.Y-2, b.W+4, b.H+4, render.ColorActive)
		}
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	}
}

func (a *App) drawActiveContactTable(screen *ebiten.Image, sonar *acoustics.SonarState) {
	x, y0, w := activeControlsX(), activeListY, activeSideW-40
	mx, my := ebiten.CursorPosition()
	render.FillRect(screen, x, y0, w, activeContactTableHeight(), render.ColorPanelInset)
	render.DrawText(screen, a.L(i18n.UIContactCol), x+8, y0+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, a.L(i18n.UIColBRGDeg), x+72, y0+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, a.L(i18n.UIColRNG), x+118, y0+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, a.L(i18n.UIColClass), x+176, y0+16, render.ColorPhosphorDim, true)

	a.contactTableScroll.active = clampContactTableScroll(a.contactTableScroll.active, len(sonar.Contacts), activeListVisibleRows)
	start, end := contactTableWindow(len(sonar.Contacts), a.contactTableScroll.active, activeListVisibleRows)
	y := y0 + passiveListRow
	for i := start; i < end; i++ {
		c := &sonar.Contacts[i]
		selected := c.SourceEntityID == a.selectedContactID
		hover := mx >= x && mx < x+w && my >= y && my < y+passiveListRow
		if selected {
			render.FillRect(screen, x+2, y, w-4, passiveListRow, color.RGBA{80, 60, 0, 180})
		} else if hover {
			render.FillRect(screen, x+2, y, w-4, passiveListRow, render.ColorPanelMid)
		}
		clr := render.ColorPhosphor
		if acoustics.ContactHasActiveRange(c) {
			clr = render.ColorActive
		}
		if selected {
			clr = render.ColorAmber
		}
		render.DrawText(screen, c.ID, x+8, y+16, clr, true)
		render.DrawText(screen, contactBearingLabel(c), x+72, y+16, clr, true)
		render.DrawText(screen, contactRangeLabel(c), x+118, y+16, clr, true)
		render.DrawText(screen, contactClassLabel(c), x+176, y+16, clr, true)
		y += passiveListRow
	}
	drawContactTableScrollbar(screen, x+w+4, y0+passiveListRow, activeListVisibleRows*passiveListRow, len(sonar.Contacts), activeListVisibleRows, a.contactTableScroll.active)
}

func (a *App) activeVisualTime() float64 {
	if a.Engine == nil {
		return 0
	}
	return a.Engine.VisualGameTime()
}

func (a *App) activeEchoReachYd(sonar *acoustics.SonarState) float64 {
	if a.Engine == nil || sonar == nil || sonar.LastPingTime <= 0 {
		return 0
	}
	age := a.activeVisualTime() - sonar.LastPingTime
	reach := acoustics.EchoRangeYd(age)
	if reach > acoustics.ActiveDisplayMaxRangeYd {
		return 0
	}
	return reach
}

func (a *App) ensureActivePlotImage() {
	w, h := activePlotW(), activePlotH
	if a.activePlotImg != nil && a.activePlotImg.Bounds().Dx() == w {
		return
	}
	disposeImage(&a.activePlotImg)
	a.activePlotImg = ebiten.NewImage(w, h)
	a.activePlotPix = make([]byte, w*h*4)
	a.activePlotGridPix = nil
	a.activePlotGridScaleYd = 0
	a.markActivePlotGridDirty()
}

func (a *App) ensureActivePlotGrid(maxR float64) {
	w, h := activePlotW(), activePlotH
	n := w * h * 4
	if a.activePlotGridPix != nil && len(a.activePlotGridPix) == n && a.activePlotGridScaleYd == maxR {
		return
	}
	if a.activePlotGridPix == nil || len(a.activePlotGridPix) != n {
		a.activePlotGridPix = make([]byte, n)
	}
	a.fillActivePlotBackground(a.activePlotGridPix, w, h)
	a.paintActivePlotGrid(a.activePlotGridPix, w, h, maxR)
	ownX := waterfallBearingDisplayX(0, w)
	for dy := 0; dy < 6; dy++ {
		for dx := -2; dx <= 2; dx++ {
			activePlotSetPix(a.activePlotGridPix, w, ownX+dx, h-4-dy, color.RGBA{0, 255, 180, 255}, 0.9-float64(dy)*0.12)
		}
	}
	a.activePlotGridScaleYd = maxR
}

func (a *App) fillActivePlotBackground(pix []byte, w, h int) {
	for i := 0; i < w*h; i++ {
		off := i * 4
		pix[off] = activePlotBgR
		pix[off+1] = activePlotBgG
		pix[off+2] = activePlotBgB
		pix[off+3] = 255
	}
}

func activePlotSetPix(pix []byte, w, px, py int, clr color.RGBA, gain float64) {
	if px < 0 || py < 0 || px >= w {
		return
	}
	off := (py*w + px) * 4
	for c, v := range []uint8{clr.R, clr.G, clr.B} {
		nv := float64(pix[off+c]) + float64(v)*gain
		if nv > 255 {
			nv = 255
		}
		pix[off+c] = uint8(nv)
	}
}

func activeEchoCrossColor(strength float64, selected bool) (color.RGBA, bool) {
	if strength <= 0 {
		return color.RGBA{activePlotBgR, activePlotBgG, activePlotBgB, 255}, false
	}
	if strength > 1 {
		strength = 1
	}
	brightR, brightG, brightB := 100, 220, 255
	if selected {
		brightR, brightG, brightB = 255, 191, 64
	}
	inv := 1 - strength
	return color.RGBA{
		R: uint8(float64(activePlotBgR)*inv + float64(brightR)*strength),
		G: uint8(float64(activePlotBgG)*inv + float64(brightG)*strength),
		B: uint8(float64(activePlotBgB)*inv + float64(brightB)*strength),
		A: 255,
	}, true
}

func drawActiveEchoCross(screen *ebiten.Image, cx, cy, halfLen float64, strength float64, selected bool) {
	clr, visible := activeEchoCrossColor(strength, selected)
	if !visible {
		return
	}
	render.DrawLine(screen, cx-halfLen, cy, cx+halfLen, cy, clr)
	render.DrawLine(screen, cx, cy-halfLen, cx, cy+halfLen, clr)
}

func activeRangeGridKyds(maxR float64) []float64 {
	switch {
	case maxR <= 2500:
		return []float64{0.5, 1, 1.5, 2}
	case maxR <= 7000:
		return []float64{1, 2, 3, 4, 5, 6}
	default:
		return []float64{2, 4, 6, 8, 10, 12}
	}
}

func (a *App) paintActivePlotGrid(pix []byte, w, h int, maxR float64) {
	for _, kyd := range activeRangeGridKyds(maxR) {
		frac := (kyd * 1000) / maxR
		if frac > 1 {
			continue
		}
		py := h - 8 - int(frac*float64(h-16))
		for px := 0; px < w; px++ {
			activePlotSetPix(pix, w, px, py, color.RGBA{0, 55, 45, 255}, 0.18)
		}
	}
	for deg := 0; deg < 360; deg += 10 {
		px := waterfallBearingDisplayX(float64(deg), w)
		gain := 0.07
		if deg%30 == 0 {
			gain = 0.13
		}
		if deg%90 == 0 {
			gain = 0.22
		}
		for py := 0; py < h; py++ {
			activePlotSetPix(pix, w, px, py, color.RGBA{0, 55, 45, 255}, gain)
		}
	}
}

func (a *App) drawActiveBearingRuler(screen *ebiten.Image, x, y, w, h int) {
	// Top bearing scale: ticks every 10°, labels every 30° (000 at center).
	for deg := 0; deg < 360; deg += 10 {
		px := x + waterfallBearingDisplayX(float64(deg), w)
		tickTop := y
		tickBot := y + 6
		clr := color.RGBA{0, 100, 75, 180}
		if deg%30 == 0 {
			tickBot = y + 10
			clr = color.RGBA{0, 140, 100, 210}
		}
		if deg%90 == 0 {
			tickBot = y + 14
			clr = color.RGBA{0, 210, 150, 255}
		}
		render.DrawLine(screen, float64(px), float64(tickTop), float64(px), float64(tickBot), clr)
		if deg%30 == 0 && deg != 180 {
			labelClr := render.ColorPhosphorDim
			if deg%90 == 0 {
				labelClr = render.ColorPhosphor
			}
			render.DrawText(screen, fmt.Sprintf("%03d", deg), px-12, y-16, labelClr, true)
		}
	}
	// Faint ticks along bottom edge for quick read without covering ownship marker.
	for deg := 0; deg < 360; deg += 30 {
		if deg%90 == 0 {
			continue
		}
		px := x + waterfallBearingDisplayX(float64(deg), w)
		render.DrawLine(screen, float64(px), float64(y+h-5), float64(px), float64(y+h), color.RGBA{0, 80, 60, 140})
	}
	render.DrawText(screen, "180", x-8, y-16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "180", x+w-22, y-16, render.ColorPhosphorDim, true)
}

func (a *App) rebuildActivePlotRaster() {
	a.ensureActivePlotImage()
	maxR := a.activeRangeMaxYd()
	a.ensureActivePlotGrid(maxR)
	copy(a.activePlotPix, a.activePlotGridPix)
	a.activePlotImg.WritePixels(a.activePlotPix)
	a.activePlotGridDirty = false
}

func (a *App) drawActiveEchoFlashes(screen *ebiten.Image, maxR float64) {
	plotX, plotY := activePlotX, activePlotY
	w, h := activePlotW(), activePlotH
	for i := len(a.activeEchoFlashes) - 1; i >= 0; i-- {
		f := a.activeEchoFlashes[i]
		if f.Strength <= 0 {
			continue
		}
		px, py := a.activeFlashPlotPos(f, w, h, maxR)
		half := activeFlashCrossHalfLen(f.SNR)
		selected := f.SourceEntityID == a.selectedContactID
		drawActiveEchoCross(screen, float64(plotX+px), float64(plotY+py), half, f.Strength, selected)
	}
}

func (a *App) drawActiveRangeDisplay(screen *ebiten.Image, sonar *acoustics.SonarState) {
	maxR := a.activeRangeMaxYd()
	maxKyd := int(maxR / 1000)
	if a.activePlotNeedsGridRebuild() {
		a.rebuildActivePlotRaster()
	}
	render.DrawImageAt(screen, a.activePlotImg, activePlotX, activePlotY)

	x, y, w, h := activePlotX, activePlotY, activePlotW(), activePlotH
	a.drawActiveEchoFlashes(screen, maxR)
	a.drawActiveBearingRuler(screen, x, y, w, h)
	render.DrawText(screen, a.L(i18n.UIOwn), x+w/2-14, y+h+14, render.ColorPhosphorDim, true)
	topLabel := fmt.Sprintf("%dk", maxKyd)
	render.DrawText(screen, topLabel, x-6, y+18, render.ColorPhosphorDim, true)
	midKyd := maxKyd / 2
	if midKyd > 0 {
		midY := y + h - 8 - int(float64(midKyd*1000)/maxR*float64(h-16))
		render.DrawText(screen, fmt.Sprintf("%dk", midKyd), x-2, midY, render.ColorPhosphorDim, true)
	}
}
