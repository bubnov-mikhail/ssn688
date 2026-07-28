package ui

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/render"
)

const (
	activePanelX             = 20
	activePanelY             = 50
	activePanelW             = 900
	activeSideX              = 940
	activeSideY              = 50
	activeSideW              = 340
	activeControlsX          = 960
	activeControlsY          = 90
	activeSliderW            = 200
	activeSliderH            = 10
	activeSliderLabelW       = 90
	activeSliderTrackX       = activeControlsX + activeSliderLabelW
	activeSliderRowGap       = 28
	activePingSliderY        = activeControlsY + 88
	activePowerSliderY       = activeControlsY + 88 + activeSliderRowGap
	activeListY              = 280
	activePlotCX             = 470.0
	activePlotCY             = 400.0
	activePlotR              = 260.0
	activePlotBlipFullFadeSec = 60.0
	activePulseMaxOpacity    = 0.15 // 85% transparent at peak
)

type activePlotBlip struct {
	ContactID      string
	SourceEntityID string
	BearingDeg     float64
	RangeYd        float64
	SeenAt         float64 // game time when the echo ring crossed this range
}

func activeSonarButtons(sonar *acoustics.SonarState) []sonarUIButton {
	toggleLabel := "STANDBY"
	if sonar.ActiveEnabled {
		toggleLabel = "ACTIVE ON"
	}
	return layoutButtonRow(activeControlsX, activeControlsY+10, 34, 6, []buttonSpec{
		{"active_toggle", toggleLabel, "Enable or disable active sonar transmit mode"},
		{"active_ping", "PING NOW", "Fire one immediate active sonar pulse"},
	})
}

func (a *App) activePingIntervalSliderRect() (x, y, w, h int) {
	return activeSliderTrackX, activePingSliderY - 8, activeSliderW, activeSliderH
}

func (a *App) activePowerSliderRect() (x, y, w, h int) {
	return activeSliderTrackX, activePowerSliderY - 8, activeSliderW, activeSliderH
}

func (a *App) sliderValueFromMouse(mx, x, w int, min, max float64) float64 {
	if w <= 0 {
		return min
	}
	t := float64(mx-x) / float64(w)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return min + t*(max-min)
}

func activePlotBlipOpacity(age, pingIntervalSec float64) float64 {
	if age >= activePlotBlipFullFadeSec || age < 0 {
		return 0
	}
	interval := pingIntervalSec
	if interval <= 0 {
		interval = 12
	}
	if interval > activePlotBlipFullFadeSec {
		interval = activePlotBlipFullFadeSec
	}
	if age <= interval {
		return 1.0 - 0.8*(age/interval)
	}
	tail := activePlotBlipFullFadeSec - interval
	if tail <= 0 {
		return 0
	}
	return 0.2 * (1.0 - (age-interval)/tail)
}

func (a *App) updateActiveScreen(sonar *acoustics.SonarState) {
	a.validateSelectedContact(sonar)
	buttons := activeSonarButtons(sonar)
	mx, my := ebiten.CursorPosition()
	a.updateSonarTooltips(buttons, mx, my)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for _, b := range buttons {
			if b.contains(mx, my) {
				a.activeSonarButtonAction(b.ID, sonar)
				a.uiPressedID = b.ID
				a.uiPressedAt = time.Now()
				return
			}
		}
		pingX, pingY, pingW, pingH := a.activePingIntervalSliderRect()
		if inRect(mx, my, pingX, pingY, pingW, pingH) {
			a.activeSliderDrag = "ping_interval"
			sonar.PingInterval = math.Round(a.sliderValueFromMouse(mx, activeSliderTrackX, activeSliderW, acoustics.PingIntervalMinSec, acoustics.PingIntervalMaxSec))
			return
		}
		powerX, powerY, powerW, powerH := a.activePowerSliderRect()
		if inRect(mx, my, powerX, powerY, powerW, powerH) {
			a.activeSliderDrag = "power"
			sonar.ActivePower = a.sliderValueFromMouse(mx, activeSliderTrackX, activeSliderW, 0.3, 1.0)
			return
		}
		if a.activePlotBlipClick(sonar, mx, my) {
			return
		}
		y := activeListY + passiveListRow
		listW := activeSideW - 40
		for i := range sonar.Contacts {
			if mx >= activeControlsX && mx < activeControlsX+listW && my >= y && my < y+passiveListRow {
				a.selectContact(sonar, &sonar.Contacts[i])
				return
			}
			y += passiveListRow
		}
	}

	if a.activeSliderDrag != "" && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		switch a.activeSliderDrag {
		case "ping_interval":
			sonar.PingInterval = math.Round(a.sliderValueFromMouse(mx, activeSliderTrackX, activeSliderW, acoustics.PingIntervalMinSec, acoustics.PingIntervalMaxSec))
		case "power":
			sonar.ActivePower = a.sliderValueFromMouse(mx, activeSliderTrackX, activeSliderW, 0.3, 1.0)
		}
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		a.activeSliderDrag = ""
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		a.activeSonarButtonAction("active_toggle", sonar)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		a.activeSonarButtonAction("active_ping", sonar)
	}
	a.updateActivePlotBlips(sonar)
}

func (a *App) activePlotBlipPos(blip activePlotBlip, cx, cy, radius, maxR float64) (x, y float64) {
	rad := blip.BearingDeg * math.Pi / 180
	rng := math.Min(radius-12, blip.RangeYd/maxR*radius)
	return cx + math.Sin(rad)*rng, cy - math.Cos(rad)*rng
}

func (a *App) activePlotBlipHit(mx, my int, blip activePlotBlip, cx, cy, radius, maxR float64) bool {
	x, y := a.activePlotBlipPos(blip, cx, cy, radius, maxR)
	return inRect(mx, my, int(x)-10, int(y)-12, 88, 26)
}

func (a *App) activePlotBlipClick(sonar *acoustics.SonarState, mx, my int) bool {
	cx, cy, radius := activePlotCX, activePlotCY, activePlotR
	dx := float64(mx) - cx
	dy := float64(my) - cy
	if dx*dx+dy*dy > (radius+16)*(radius+16) {
		return false
	}
	gameTime := a.activeVisualTime()
	maxR := acoustics.ActiveDisplayMaxRangeYd
	a.ensureActivePlotBlips()
	for _, blip := range a.activePlotBlips {
		if gameTime < blip.SeenAt {
			continue
		}
		age := gameTime - blip.SeenAt
		if activePlotBlipOpacity(age, sonar.PingInterval) < 0.05 {
			continue
		}
		if !a.activePlotBlipHit(mx, my, blip, cx, cy, radius, maxR) {
			continue
		}
		for i := range sonar.Contacts {
			if sonar.Contacts[i].SourceEntityID == blip.SourceEntityID {
				a.selectContact(sonar, &sonar.Contacts[i])
				return true
			}
		}
	}
	return false
}

func (a *App) ensureActivePlotBlips() {
	if a.activePlotBlips == nil {
		a.activePlotBlips = map[string]activePlotBlip{}
	}
}

func (a *App) syncActivePulseWall(sonar *acoustics.SonarState) {
	if sonar == nil || sonar.LastPingTime <= 0 {
		return
	}
	if a.activePlotPingAt != sonar.LastPingTime {
		a.activePlotPingAt = sonar.LastPingTime
		a.activePulseWallAt = time.Now()
	}
}

func (a *App) activePulseAgeSec(sonar *acoustics.SonarState) float64 {
	if a.Engine == nil || sonar == nil || sonar.LastPingTime <= 0 {
		return 0
	}
	a.syncActivePulseWall(sonar)
	if a.Engine.Clock.Paused || a.activePulseWallAt.IsZero() {
		return a.Engine.Clock.GameTime - sonar.LastPingTime
	}
	scale := a.Engine.Clock.TimeScale
	if scale <= 0 {
		scale = 1
	}
	return time.Since(a.activePulseWallAt).Seconds() * scale
}

func (a *App) updateActivePlotBlips(sonar *acoustics.SonarState) {
	if a.Engine == nil || sonar == nil || sonar.LastPingTime <= 0 {
		return
	}
	a.ensureActivePlotBlips()
	a.syncActivePulseWall(sonar)
	gameTime := a.activeVisualTime()

	for id, blip := range a.activePlotBlips {
		if gameTime-blip.SeenAt > activePlotBlipFullFadeSec {
			delete(a.activePlotBlips, id)
		}
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
		if existing, ok := a.activePlotBlips[c.SourceEntityID]; ok && crossAt <= existing.SeenAt+0.05 {
			continue
		}
		a.activePlotBlips[c.SourceEntityID] = activePlotBlip{
			ContactID:      c.ID,
			SourceEntityID: c.SourceEntityID,
			BearingDeg:     c.BearingDeg,
			RangeYd:        c.EstimatedRangeYd,
			SeenAt:         crossAt,
		}
	}
}

func (a *App) activeSonarButtonAction(id string, sonar *acoustics.SonarState) {
	switch id {
	case "active_toggle":
		sonar.ActiveEnabled = !sonar.ActiveEnabled
		if sonar.ActiveEnabled {
			a.Audio.PlayClip(audio.ClipSonarActiveStandby, "Active sonar online.")
		} else {
			a.Audio.PlayClip(audio.ClipSonarActiveStandby, "Active sonar standby.")
		}
	case "active_ping":
		if a.Engine == nil || a.Engine.Scenario == nil {
			return
		}
		player := a.Engine.Scenario.Player
		emitters := a.Engine.Scenario.AllEntities()
		if acoustics.FireActivePingNow(a.Engine.Acoustics, player, emitters, sonar, a.Engine.Clock.GameTime) {
			a.activePlotPingAt = 0 // force wall-clock resync on next pulse draw
			a.Audio.PlayEnemyPing()
			if a.lastPingPlayed < sonar.LastPingTime {
				a.lastPingPlayed = sonar.LastPingTime
			}
		} else if !sonar.ActiveEnabled {
			a.StatusMessage = "Enable active sonar before transmitting."
		}
	}
}

func (a *App) drawActiveSlider(screen *ebiten.Image, label string, y int, min, max, value float64, mx, my int, dragID string) {
	render.DrawText(screen, label, activeControlsX, y+2, render.ColorPhosphorDim, true)
	x, w, h := activeSliderTrackX, activeSliderW, activeSliderH
	hover := inRect(mx, my, x, y-8, w, h)
	render.FillRect(screen, x, y-8, w, h, render.ColorPanelInset)
	t := (value - min) / (max - min)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	fillW := int(float64(w) * t)
	clr := render.ColorActive
	if hover || a.activeSliderDrag == dragID {
		clr = render.ColorHighlight
	}
	if fillW > 0 {
		render.FillRect(screen, x, y-8, fillW, h, clr)
	}
	knobX := x + fillW - 3
	if knobX < x {
		knobX = x
	}
	if knobX > x+w-6 {
		knobX = x + w - 6
	}
	render.FillRect(screen, knobX, y-10, 6, h+4, render.ColorText)
}

func (a *App) drawActiveControls(screen *ebiten.Image, sonar *acoustics.SonarState) {
	render.DrawText(screen, "ACTIVE TX", activeControlsX, activeControlsY-8, render.ColorPhosphorDim, true)
	mx, my := ebiten.CursorPosition()
	buttons := activeSonarButtons(sonar)
	for _, b := range buttons {
		disabled := b.ID == "active_ping" && !sonar.ActiveEnabled
		hover := !disabled && b.contains(mx, my)
		pressed := a.uiPressedID == b.ID && time.Since(a.uiPressedAt) < 120*time.Millisecond
		if disabled {
			render.FillRect(screen, b.X, b.Y, b.W, b.H, render.ColorPanelInset)
			tw := render.ButtonLabelWidth(b.Label)
			render.DrawButtonText(screen, b.Label, b.X+(b.W-tw)/2, b.Y+b.H/2+4, render.ColorPhosphorDim)
			continue
		}
		if b.ID == "active_toggle" && sonar.ActiveEnabled {
			render.FillRect(screen, b.X-2, b.Y-2, b.W+4, b.H+4, render.ColorActive)
		}
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	}

	a.drawActiveSlider(screen, "AUTO PING", activePingSliderY, acoustics.PingIntervalMinSec, acoustics.PingIntervalMaxSec, sonar.PingInterval, mx, my, "ping_interval")
	a.drawActiveSlider(screen, "POWER", activePowerSliderY, 0.3, 1.0, sonar.ActivePower, mx, my, "power")

	autoLabel := "MANUAL"
	if sonar.PingInterval > 0 {
		autoLabel = fmt.Sprintf("%.0fs", sonar.PingInterval)
	}
	render.DrawText(screen, autoLabel, activeSliderTrackX+activeSliderW+8, activePingSliderY+2, render.ColorDim, true)
	render.DrawText(screen, fmt.Sprintf("%.0f%%", sonar.ActivePower*100), activeSliderTrackX+activeSliderW+8, activePowerSliderY+2, render.ColorDim, true)
}

func (a *App) drawActiveContactTable(screen *ebiten.Image, sonar *acoustics.SonarState) {
	x, y0, w := activeControlsX, activeListY, activeSideW-40
	mx, my := ebiten.CursorPosition()
	render.FillRect(screen, x, y0, w, passiveListRow*max(1, len(sonar.Contacts)+1)+14, render.ColorPanelInset)
	render.DrawText(screen, "CONTACT", x+8, y0+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "BRG", x+72, y0+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "RNG", x+118, y0+16, render.ColorPhosphorDim, true)
	render.DrawText(screen, "CLASS", x+176, y0+16, render.ColorPhosphorDim, true)

	y := y0 + passiveListRow
	for i := range sonar.Contacts {
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
		render.DrawText(screen, fmt.Sprintf("%03.0f", c.BearingDeg), x+72, y+16, clr, true)
		render.DrawText(screen, contactRangeLabel(c), x+118, y+16, clr, true)
		render.DrawText(screen, contactClassLabel(c), x+176, y+16, clr, true)
		y += passiveListRow
	}
}

func (a *App) activeVisualTime() float64 {
	if a.Engine == nil {
		return 0
	}
	return a.Engine.VisualGameTime()
}

func (a *App) activeEchoReachYd(sonar *acoustics.SonarState) float64 {
	return acoustics.EchoRangeYd(a.activePulseAgeSec(sonar))
}

func (a *App) ensureActivePlotBase() {
	pad := 12
	size := int(activePlotR)*2 + pad*2
	if a.activePlotBase != nil && a.activePlotBase.Bounds().Dx() == size {
		return
	}
	img := ebiten.NewImage(size, size)
	cx := float64(size / 2)
	cy := float64(size / 2)
	r := activePlotR
	drawCircle(img, cx, cy, r, color.RGBA{0, 70, 55, 160})
	drawCircle(img, cx, cy, r*0.5, color.RGBA{0, 70, 55, 120})
	for _, deg := range []float64{0, 90, 180, 270} {
		rad := deg * math.Pi / 180
		render.DrawLine(img, cx, cy, cx+math.Sin(rad)*r, cy-math.Cos(rad)*r, color.RGBA{0, 55, 45, 100})
	}
	a.activePlotBase = img
}

func (a *App) drawActivePulseRing(screen *ebiten.Image, cx, cy, plotR, pulseR float64) {
	if pulseR < 2 {
		return
	}
	expand := pulseR / plotR
	opacity := activePulseMaxOpacity * (1 - expand) * (1 - expand)
	if opacity < 0.01 {
		return
	}
	alpha := uint8(opacity * 255)
	vector.StrokeCircle(screen, float32(cx), float32(cy), float32(pulseR), 1.5,
		color.RGBA{170, 230, 255, alpha}, true)
}

func (a *App) drawActiveRangeDisplay(screen *ebiten.Image, sonar *acoustics.SonarState) {
	player := a.Engine.Scenario.Player
	cx, cy, radius := activePlotCX, activePlotCY, activePlotR
	maxR := acoustics.ActiveDisplayMaxRangeYd
	gameTime := a.activeVisualTime()

	a.ensureActivePlotBase()
	if a.activePlotBase != nil {
		b := a.activePlotBase.Bounds()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(cx-float64(b.Dx())/2, cy-float64(b.Dy())/2)
		screen.DrawImage(a.activePlotBase, op)
	}
	render.DrawText(screen, "0", int(cx)-4, int(cy-radius)-14, render.ColorPhosphorDim, true)
	render.DrawText(screen, "12 KYD", int(cx)+int(radius)-42, int(cy)+4, render.ColorPhosphorDim, true)

	echoYd := a.activeEchoReachYd(sonar)
	if sonar.LastPingTime > 0 && echoYd > 0 {
		age := a.activePulseAgeSec(sonar)
		maxPulseAge := acoustics.TwoWayTravelSec(maxR) + 0.05
		if age <= maxPulseAge {
			frac := echoYd / maxR
			if frac > 1 {
				frac = 1
			}
			a.drawActivePulseRing(screen, cx, cy, radius, radius*frac)
		}
	}

	a.ensureActivePlotBlips()
	for _, blip := range a.activePlotBlips {
		if gameTime < blip.SeenAt {
			continue
		}
		age := gameTime - blip.SeenAt
		opacity := activePlotBlipOpacity(age, sonar.PingInterval)
		if opacity < 0.05 {
			continue
		}
		alpha := uint8(opacity*15) * 17
		if alpha < 4 {
			continue
		}

		selected := blip.SourceEntityID == a.selectedContactID
		x, y := a.activePlotBlipPos(blip, cx, cy, radius, maxR)
		lineClr := color.RGBA{80, 200, 255, alpha}
		markClr := color.RGBA{120, 220, 255, alpha}
		textClr := color.RGBA{210, 235, 245, alpha}
		if selected {
			lineClr = color.RGBA{255, 210, 80, alpha}
			markClr = color.RGBA{255, 220, 100, alpha}
			textClr = color.RGBA{255, 235, 200, alpha}
			render.FillRect(screen, int(x)-7, int(y)-7, 15, 15, color.RGBA{255, 180, 40, alpha / 2})
		}
		render.DrawLine(screen, cx, cy, x, y, lineClr)
		render.FillRect(screen, int(x)-4, int(y)-4, 9, 9, markClr)
		render.DrawText(screen, fmt.Sprintf("%s %.1fk", blip.ContactID, blip.RangeYd/1000), int(x)+8, int(y), textClr, true)
	}
	drawOwnshipSymbol(screen, cx, cy, player.HeadingDeg, render.ColorHighlight)
}
