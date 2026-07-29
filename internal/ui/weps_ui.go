package ui

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

const (
	wepsPanelX = 20
	wepsPanelY = 50
	wepsPanelW = 1260
	wepsPanelH = 700

	wepsLeftX = 36
	wepsLeftW = 460

	wepsMapX = 520
	wepsMapY = 100
	wepsMapW = 740
	wepsMapH = 620

	wepsTubeY0   = 124
	wepsTubeRowH = 36

	wepsPrepY = 276
	wepsPrepH = 58

	wepsGuideY = 342
	wepsGuideH = 72

	wepsTargetsY   = 422
	wepsTargetsH   = 258
	wepsTargetsRow = 22
)

func (a *App) wepsTubeButtons(tube int, y int) []sonarUIButton {
	const btnH = 24
	// Same Y for all three; center on the TUBE/status text baseline (drawn at y+4).
	btnY := y + 4 - btnH/2
	x := wepsLeftX + 200
	return layoutButtonRow(x, btnY, btnH, 4, []buttonSpec{
		{fmt.Sprintf("tube_%d_open", tube), "OPEN", "Open outer door"},
		{fmt.Sprintf("tube_%d_close", tube), "CLOSE", "Close door"},
		{fmt.Sprintf("tube_%d_fire", tube), "FIRE", "Launch (door must be open)"},
	})
}

func (a *App) wepsTubePickButton(tube int, y int) sonarUIButton {
	return sonarUIButton{
		ID:      fmt.Sprintf("tube_%d_pick", tube),
		Label:   fmt.Sprintf("TUBE %d", tube),
		Tooltip: "Select this tube",
		X:       wepsLeftX,
		Y:       y - 8,
		W:       78,
		H:       20,
	}
}

func (a *App) wepsPrepButtons(fc *weapons.FireControl) []sonarUIButton {
	// One row: [-] GYRO [+]  [-] DEPTH [+]  [SPD …] [SEEK …]
	const (
		spinY = wepsPrepY + 28
		spinH = 24
		btnW  = 22
		g0    = wepsLeftX + 8
		d0    = wepsLeftX + 148
		inner = 96
	)
	spdLabel := "SPD LOW"
	if fc != nil && fc.SpeedSetting == "HIGH" {
		spdLabel = "SPD HIGH"
	}
	seekLabel := "SEEK OFF"
	if fc != nil && fc.SeekerEnabled {
		seekLabel = "SEEK ON"
	}
	btns := []sonarUIButton{
		{ID: "gyro_m", Label: "-", Tooltip: "Decrease gyro angle", X: g0, Y: spinY, W: btnW, H: spinH},
		{ID: "gyro_p", Label: "+", Tooltip: "Increase gyro angle", X: g0 + btnW + inner, Y: spinY, W: btnW, H: spinH},
		{ID: "dep_m", Label: "-", Tooltip: "Shallower run depth", X: d0, Y: spinY, W: btnW, H: spinH},
		{ID: "dep_p", Label: "+", Tooltip: "Deeper run depth", X: d0 + btnW + inner, Y: spinY, W: btnW, H: spinH},
	}
	toggleX := d0 + btnW + inner + btnW + 10
	btns = append(btns, layoutButtonRow(toggleX, spinY, spinH, 4, []buttonSpec{
		{"spd", spdLabel, "Toggle LOW/HIGH launch speed"},
		{"seek", seekLabel, "Toggle seeker enable for next shot"},
	})...)
	return btns
}

func (a *App) wepsGuideButtons() []sonarUIButton {
	return layoutButtonRow(wepsLeftX+8, wepsGuideY+40, 26, 3, []buttonSpec{
		{"wire_l", "L 10", "Wire steer left 10°"},
		{"wire_r", "R 10", "Wire steer right 10°"},
		{"wire_up", "UP", "Wire shallower 50 ft"},
		{"wire_dn", "DN", "Wire deeper 50 ft"},
		{"wire_seek", "SEEK", "Toggle ModeSearch (wire steer cancels SEEK)"},
		{"wire_cut", "CUT", "Cut guidance wire"},
		{"wire_sd", "S/D", "Self-destruct while wire intact (unavailable after CUT)"},
	})
}

func (a *App) wepsButtonFlash(id string) bool {
	return a.uiPressedID == id && time.Since(a.uiPressedAt) < 120*time.Millisecond
}

// wepsButtonLatched is a sticky "on" state drawn as an outline, not a pressed inset
// (pressed inset makes OPEN look vertically offset from CLOSE/FIRE).
func (a *App) wepsButtonLatched(id string, fc *weapons.FireControl) bool {
	if fc == nil {
		return false
	}
	switch id {
	case "spd":
		return fc.SpeedSetting == "HIGH"
	case "seek":
		return fc.SeekerEnabled
	case "wire_seek":
		if fish := fc.TorpedoForTube(fc.SelectedTube); fish != nil {
			if !fish.TubeCleared() {
				return fish.EnableSearchAfterClear
			}
			return fish.Mode == weapons.ModeSearch || fish.SeekerOn
		}
	}
	if len(id) > 5 && id[:5] == "tube_" {
		rest := id[5:]
		us := -1
		for i := 0; i < len(rest); i++ {
			if rest[i] == '_' {
				us = i
				break
			}
		}
		if us > 0 {
			tube, _ := strconv.Atoi(rest[:us])
			action := rest[us+1:]
			if action == "open" && tube >= 1 && tube <= 4 {
				st := fc.Tubes[tube-1].State
				return st == weapons.TubeDoorOpen || st == weapons.TubeFired
			}
		}
	}
	return false
}

func (a *App) drawWepsButton(screen *ebiten.Image, b sonarUIButton, mx, my int, fc *weapons.FireControl) {
	if a.wepsButtonLatched(b.ID, fc) {
		render.FillRect(screen, b.X-2, b.Y-2, b.W+4, b.H+4, render.ColorAmber)
	}
	render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, b.contains(mx, my), a.wepsButtonFlash(b.ID))
}

func (a *App) handleFireControl(fc *weapons.FireControl, player *world.Entity) {
	gt := a.Engine.Clock.GameTime
	sonar := &a.Engine.Sonar
	a.validateSelectedContact(sonar)
	mx, my := ebiten.CursorPosition()

	var all []sonarUIButton
	for i := 1; i <= 4; i++ {
		y := wepsTubeY0 + (i-1)*wepsTubeRowH
		all = append(all, a.wepsTubePickButton(i, y))
		all = append(all, a.wepsTubeButtons(i, y)...)
	}
	all = append(all, a.wepsPrepButtons(fc)...)
	all = append(all, a.wepsGuideButtons()...)
	a.updateSonarTooltips(all, mx, my)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for _, b := range all {
			if b.contains(mx, my) {
				a.wepsButtonAction(b.ID, fc, player, gt)
				a.uiPressedID = b.ID
				a.uiPressedAt = time.Now()
				return
			}
		}
		if idx := a.wepsTargetRowAt(mx, my, sonar); idx >= 0 {
			c := &sonar.Contacts[idx]
			a.selectContact(sonar, c)
			a.wepsApplyContactToPrep(fc, player, c)
			a.wepsFitSelectedContact()
			return
		}
	}
	if a.uiPressedID != "" && time.Since(a.uiPressedAt) > 120*time.Millisecond {
		a.uiPressedID = ""
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyDigit1) {
		fc.SelectTube(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit2) {
		fc.SelectTube(2)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit3) {
		fc.SelectTube(3)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit4) {
		fc.SelectTube(4)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyO) {
		a.wepsButtonAction(fmt.Sprintf("tube_%d_open", fc.SelectedTube), fc, player, gt)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		a.wepsButtonAction(fmt.Sprintf("tube_%d_close", fc.SelectedTube), fc, player, gt)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		a.wepsButtonAction(fmt.Sprintf("tube_%d_fire", fc.SelectedTube), fc, player, gt)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		a.wepsButtonAction("gyro_p", fc, player, gt)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		a.wepsButtonAction("dep_p", fc, player, gt)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && !ebiten.IsKeyPressed(ebiten.KeyControl) {
		a.wepsButtonAction("spd", fc, player, gt)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		a.wepsButtonAction("seek", fc, player, gt)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		a.wepsButtonAction("wire_cut", fc, player, gt)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		a.wepsButtonAction("wire_sd", fc, player, gt)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		a.wepsButtonAction("wire_l", fc, player, gt)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		a.wepsButtonAction("wire_r", fc, player, gt)
	}
}

func (a *App) wepsButtonAction(id string, fc *weapons.FireControl, player *world.Entity, gameTime float64) {
	switch {
	case len(id) > 5 && id[:5] == "tube_":
		rest := id[5:]
		us := -1
		for i := 0; i < len(rest); i++ {
			if rest[i] == '_' {
				us = i
				break
			}
		}
		if us <= 0 {
			return
		}
		tube, _ := strconv.Atoi(rest[:us])
		action := rest[us+1:]
		switch action {
		case "pick":
			fc.SelectTube(tube)
		case "open":
			if fc.OpenOuterDoor(tube) {
				a.Engine.EmitTubeTransient(player, gameTime, true)
				a.Audio.PlayClip(audio.TubeClip("outer_door_open", tube),
					fmt.Sprintf("Outer door open, tube %d.", tube))
			}
		case "close":
			if fc.CloseOuterDoor(tube, gameTime) {
				a.Engine.EmitTubeTransient(player, gameTime, false)
				a.Audio.PlayClip(audio.ClipWepsOuterDoorClosed, "")
			}
		case "fire":
			fc.SelectTube(tube)
			if torp := fc.Shoot(player, tube); torp != nil {
				a.Audio.PlayTorpedoLaunch()
				a.Audio.PlayClip(audio.TubeClip("torpedo_away", tube),
					fmt.Sprintf("Torpedo away, tube %d.", tube))
			} else {
				a.StatusMessage = "Cannot fire — open outer door first."
			}
		}
	case id == "gyro_m":
		fc.GyroAngleDeg = normalizeGyroDeg(fc.GyroAngleDeg - 5)
	case id == "gyro_p":
		fc.GyroAngleDeg = normalizeGyroDeg(fc.GyroAngleDeg + 5)
	case id == "dep_m":
		fc.RunDepthFt = max(40, fc.RunDepthFt-50)
		a.Audio.PlayClip(audio.ClipWepsRunDepthSet, fmt.Sprintf("Torpedo run depth %d feet.", int(fc.RunDepthFt)))
	case id == "dep_p":
		fc.RunDepthFt += 50
		a.Audio.PlayClip(audio.ClipWepsRunDepthSet, fmt.Sprintf("Torpedo run depth %d feet.", int(fc.RunDepthFt)))
	case id == "spd":
		if fc.SpeedSetting == "HIGH" {
			fc.SpeedSetting = "LOW"
			a.Audio.PlayClip(audio.ClipWepsSpeedLow, "Torpedo speed LOW.")
		} else {
			fc.SpeedSetting = "HIGH"
			a.Audio.PlayClip(audio.ClipWepsSpeedHigh, "Torpedo speed HIGH.")
		}
	case id == "seek":
		fc.SeekerEnabled = !fc.SeekerEnabled
		if fc.SeekerEnabled {
			a.Audio.PlayClip(audio.ClipWepsSeekerOn, "")
		} else {
			a.Audio.PlayClip(audio.ClipWepsSeekerOff, "")
		}
	case id == "wire_l", id == "wire_r", id == "wire_up", id == "wire_dn", id == "wire_seek", id == "wire_cut", id == "wire_sd":
		fish := fc.TorpedoForTube(fc.SelectedTube)
		if fish == nil {
			a.StatusMessage = "No wire-guided fish on selected tube."
			return
		}
		switch id {
		case "wire_l":
			fc.WireSteer(fish, -10, 0)
		case "wire_r":
			fc.WireSteer(fish, 10, 0)
		case "wire_up":
			fc.WireSteer(fish, 0, -50)
		case "wire_dn":
			fc.WireSteer(fish, 0, 50)
		case "wire_seek":
			wasSearch := fish.Mode == weapons.ModeSearch || fish.SeekerOn
			fc.ToggleSeeker(fish)
			nowSearch := fish.Mode == weapons.ModeSearch || fish.SeekerOn
			if nowSearch && !wasSearch {
				a.Audio.PlayClip(audio.ClipWepsSeekerOn, "")
			} else if wasSearch && !nowSearch {
				a.Audio.PlayClip(audio.ClipWepsSeekerOff, "")
			}
		case "wire_cut":
			fc.CutWire(fish)
			a.Audio.PlayClip(audio.ClipWepsWireCut, "")
		case "wire_sd":
			if det := fc.SelfDestruct(fish); det != nil {
				a.Engine.Events = append(a.Engine.Events, "Torpedo self-destructed")
				a.Audio.PlayClip(audio.ClipWepsWireCut, "Torpedo self-destruct.")
				a.StatusMessage = "Torpedo self-destructed (safe abort)."
			} else if fish.WireCut {
				a.StatusMessage = "Wire cut — cannot self-destruct."
			}
		}
	}
}

func (a *App) drawFireControl(screen *ebiten.Image) {
	fc := &a.Engine.FireControl
	sonar := &a.Engine.Sonar
	gt := a.Engine.Clock.GameTime
	render.DrawConsolePanel(screen, wepsPanelX, wepsPanelY, wepsPanelW, wepsPanelH)
	render.DrawText(screen, "FIRE CONTROL — MK48 ADCAP", wepsLeftX, 78, render.ColorPlateLabel, true)
	render.DrawText(screen, fmt.Sprintf("MAGAZINE: %d Mk48 remaining", fc.MagazineLeft), wepsLeftX, 100, render.ColorPhosphorDim, true)

	mx, my := ebiten.CursorPosition()
	for i := range fc.Tubes {
		t := fc.Tubes[i]
		y := wepsTubeY0 + i*wepsTubeRowH
		titleBtn := a.wepsTubePickButton(t.Number, y)
		state := weapons.TubeStateName(t.State)
		extra := ""
		if t.State == weapons.TubeReloading {
			extra = fmt.Sprintf("  %ds", int(fc.ReloadRemaining(t, gt)+0.5))
		}
		if t.TorpedoID != "" {
			extra += "  " + t.TorpedoID
		}
		if t.Number == fc.SelectedTube {
			render.FillRect(screen, wepsLeftX-4, y-10, wepsLeftW-8, 28, color.RGBA{58, 52, 20, 180})
		}
		clr := render.ColorText
		if titleBtn.contains(mx, my) || t.Number == fc.SelectedTube {
			clr = render.ColorHighlight
		}
		render.DrawText(screen, titleBtn.Label, titleBtn.X, y+4, clr, true)
		render.DrawText(screen, fmt.Sprintf("%s%s", state, extra), wepsLeftX+82, y+4, render.ColorDim, true)
		for _, b := range a.wepsTubeButtons(t.Number, y) {
			a.drawWepsButton(screen, b, mx, my, fc)
		}
	}

	a.drawWepsGroup(screen, wepsLeftX-6, wepsPrepY, wepsLeftW, wepsPrepH, "NEXT SHOT PREP")
	a.drawWepsSpinLabels(screen, fc)
	for _, b := range a.wepsPrepButtons(fc) {
		a.drawWepsButton(screen, b, mx, my, fc)
	}

	a.drawWepsGroup(screen, wepsLeftX-6, wepsGuideY, wepsLeftW, wepsGuideH, "WIRE GUIDE")
	fish := fc.TorpedoForTube(fc.SelectedTube)
	if fish != nil && fish.Alive {
		wire := "WIRE"
		if fish.WireCut {
			wire = "CUT"
		}
		mode := "WIRE"
		if !fish.TubeCleared() {
			mode = "CLEAR"
		} else if fish.Mode == weapons.ModeSearch {
			mode = "SEEK"
		}
		render.DrawText(screen, fmt.Sprintf("%s  hdg %.0f (gyro %.0f)  d %.0f  spd %.0f/%.0f  %s  %s",
			fish.ID, fish.HeadingDeg, fish.GyroCourseDeg, fish.RunDepthFt, fish.SpeedKts, fish.CruiseKts, wire, mode),
			wepsLeftX+8, wepsGuideY+28, render.ColorAmber, true)
	} else {
		render.DrawText(screen, "No active fish on selected tube.", wepsLeftX+8, wepsGuideY+28, render.ColorDim, true)
	}
	for _, b := range a.wepsGuideButtons() {
		a.drawWepsButton(screen, b, mx, my, fc)
	}

	a.drawWepsContactTable(screen, sonar)
	a.drawWepsMap(screen, sonar, fish)

	if a.uiTooltip != "" {
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
	render.DrawText(screen, "[1-4] tube  [O/C] door  [ENTER] fire  [G/D/S/H] prep  arrows wire  [W] cut  [X] self-destruct",
		wepsLeftX, wepsPanelY+wepsPanelH-16, render.ColorDim, true)
}

func (a *App) drawWepsSpinLabels(screen *ebiten.Image, fc *weapons.FireControl) {
	const (
		spinY = wepsPrepY + 28
		btnW  = 22
		g0    = wepsLeftX + 8
		d0    = wepsLeftX + 148
		inner = 96
	)
	gyroMid := g0 + btnW + inner/2
	depMid := d0 + btnW + inner/2
	gyroTxt := fmt.Sprintf("GYRO %03.0f°", fc.GyroAngleDeg)
	depTxt := fmt.Sprintf("DEP %d", int(fc.RunDepthFt))
	render.DrawText(screen, gyroTxt, gyroMid-len(gyroTxt)*3, spinY+16, render.ColorAmber, true)
	render.DrawText(screen, depTxt, depMid-len(depTxt)*3, spinY+16, render.ColorAmber, true)
}

func (a *App) drawWepsGroup(screen *ebiten.Image, x, y, w, h int, title string) {
	render.FillRect(screen, x, y, w, h, color.RGBA{18, 18, 20, 255})
	border := color.RGBA{78, 78, 84, 255}
	render.FillRect(screen, x, y, w, 1, border)
	render.FillRect(screen, x, y+h-1, w, 1, border)
	render.FillRect(screen, x, y, 1, h, border)
	render.FillRect(screen, x+w-1, y, 1, h, border)
	render.DrawText(screen, title, x+10, y+16, render.ColorPlateLabel, true)
}

func (a *App) drawWepsContactTable(screen *ebiten.Image, sonar *acoustics.SonarState) {
	a.drawWepsGroup(screen, wepsLeftX-6, wepsTargetsY, wepsLeftW, wepsTargetsH, "RECOGNIZED TARGETS")
	hdrY := wepsTargetsY + 32
	render.DrawText(screen, "CONTACT", wepsLeftX+6, hdrY, render.ColorPhosphorDim, true)
	render.DrawText(screen, "BRG", wepsLeftX+86, hdrY, render.ColorPhosphorDim, true)
	render.DrawText(screen, "RNG", wepsLeftX+126, hdrY, render.ColorPhosphorDim, true)
	render.DrawText(screen, "TYPE", wepsLeftX+178, hdrY, render.ColorPhosphorDim, true)
	render.DrawText(screen, "CLASS", wepsLeftX+226, hdrY, render.ColorPhosphorDim, true)

	mx, my := ebiten.CursorPosition()
	rowY := wepsTargetsY + 40
	maxY := wepsTargetsY + wepsTargetsH - 8
	for i := range sonar.Contacts {
		if rowY+wepsTargetsRow > maxY {
			break
		}
		c := &sonar.Contacts[i]
		selected := c.SourceEntityID == a.selectedContactID
		hover := mx >= wepsLeftX && mx < wepsLeftX+wepsLeftW-16 && my >= rowY && my < rowY+wepsTargetsRow
		if selected {
			render.FillRect(screen, wepsLeftX, rowY, wepsLeftW-20, wepsTargetsRow, color.RGBA{80, 60, 0, 180})
		} else if hover {
			render.FillRect(screen, wepsLeftX, rowY, wepsLeftW-20, wepsTargetsRow, color.RGBA{34, 34, 38, 255})
		}
		clr := render.ColorPhosphor
		if selected {
			clr = render.ColorAmber
		}
		ty := rowY + 15
		render.DrawText(screen, c.ID, wepsLeftX+6, ty, clr, true)
		render.DrawText(screen, fmt.Sprintf("%03.0f", c.BearingDeg), wepsLeftX+86, ty, clr, true)
		render.DrawText(screen, contactRangeLabel(c), wepsLeftX+126, ty, clr, true)
		render.DrawText(screen, contactTypeLabel(c), wepsLeftX+178, ty, clr, true)
		render.DrawText(screen, contactClassLabel(c), wepsLeftX+226, ty, clr, true)
		rowY += wepsTargetsRow
	}
}

func (a *App) wepsTargetRowAt(mx, my int, sonar *acoustics.SonarState) int {
	rowY := wepsTargetsY + 40
	maxY := wepsTargetsY + wepsTargetsH - 8
	for i := range sonar.Contacts {
		if rowY+wepsTargetsRow > maxY {
			break
		}
		if mx >= wepsLeftX && mx < wepsLeftX+wepsLeftW-16 && my >= rowY && my < rowY+wepsTargetsRow {
			return i
		}
		rowY += wepsTargetsRow
	}
	return -1
}

func normalizeGyroDeg(deg float64) float64 {
	for deg < 0 {
		deg += 360
	}
	for deg >= 360 {
		deg -= 360
	}
	return deg
}

func (a *App) wepsApplyContactToPrep(fc *weapons.FireControl, player *world.Entity, c *acoustics.Contact) {
	if fc == nil || c == nil {
		return
	}
	fc.GyroAngleDeg = normalizeGyroDeg(c.BearingDeg)
	depth := 200.0
	if player != nil {
		depth = math.Max(80, player.DepthFt)
	}
	if c.ConfirmedClass != "" {
		switch c.Kind {
		case world.KindSurfaceShip:
			depth = 40
		case world.KindSubmarine:
			if player != nil {
				depth = math.Max(80, player.DepthFt)
			}
		case world.KindTorpedo:
			depth = 100
		}
	}
	fc.RunDepthFt = depth
}

func (a *App) wepsFitSelectedContact() {
	c := a.selectedContact(&a.Engine.Sonar)
	if c == nil {
		return
	}
	rng := math.Max(1200, c.EstimatedRangeYd)
	usable := math.Min(float64(wepsMapW), float64(wepsMapH)) - 80
	zoom := usable / (rng * 2.35)
	if zoom < 0.015 {
		zoom = 0.015
	}
	if zoom > 0.11 {
		zoom = 0.11
	}
	a.wepsMapZoom = zoom
}

func (a *App) ensureWepsMapImg() *ebiten.Image {
	if a.wepsMapImg == nil || a.wepsMapImg.Bounds().Dx() != wepsMapW || a.wepsMapImg.Bounds().Dy() != wepsMapH {
		a.wepsMapImg = ebiten.NewImage(wepsMapW, wepsMapH)
	}
	return a.wepsMapImg
}

func (a *App) drawWepsMap(screen *ebiten.Image, sonar *acoustics.SonarState, fish *weapons.Torpedo) {
	fc := &a.Engine.FireControl
	render.DrawText(screen, "TACTICAL MAP", wepsMapX, 86, render.ColorPlateLabel, true)
	render.DrawMonitor(screen, wepsMapX, wepsMapY, wepsMapW, wepsMapH)

	img := a.ensureWepsMapImg()
	img.Fill(color.RGBA{4, 18, 28, 255})

	player := a.Engine.Scenario.Player
	px := float64(wepsMapW) / 2
	py := float64(wepsMapH) / 2
	for _, rYd := range []float64{1000, 2000, 4000, 8000} {
		rad := rYd * a.wepsMapZoom
		if rad > 20 && rad < float64(wepsMapW)/2 {
			drawCircle(img, px, py, rad, color.RGBA{0, 70, 55, 160})
		}
	}

	gt := a.Engine.Clock.GameTime
	// Own fish still on the wire: true geographic position (wire telemetry).
	wireFishIDs := map[string]bool{}
	for _, torp := range a.Engine.FireControl.ActiveTorpedoes {
		if torp == nil || !torp.Alive || torp.Side != world.SidePlayer || torp.WireCut {
			continue
		}
		wireFishIDs[torp.ID] = true
		sx := px + (torp.X-player.X)*a.wepsMapZoom
		sy := py - (torp.Y-player.Y)*a.wepsMapZoom
		if !wepsMapMarkerInside(sx, sy) {
			continue
		}
		render.FillRect(img, int(sx)-3, int(sy)-3, 7, 7, render.ColorActive)
	}
	// Contacts use the same plotted positions as TACTICAL PLOT (not true emitter coords).
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if wireFishIDs[c.SourceEntityID] {
			continue
		}
		wx, wy := a.contactPlotWorld(player, c, gt)
		sx := px + (wx-player.X)*a.wepsMapZoom
		sy := py - (wy-player.Y)*a.wepsMapZoom
		if !wepsMapMarkerInside(sx, sy) {
			continue
		}
		clr := color.RGBA{150, 155, 160, 255}
		kind := world.EntityKind(-1)
		if c.ConfirmedClass != "" {
			kind = c.Kind
			if c.Kind == world.KindTorpedo {
				clr = color.RGBA{220, 60, 50, 255}
			}
		}
		if c.SourceEntityID == a.selectedContactID {
			clr = color.RGBA{255, 200, 60, 255}
		}
		drawContactPictogram(img, int(sx), int(sy), kind, clr)
		render.DrawText(img, c.ID, int(sx)+8, int(sy)+4, clr, true)
	}

	drawOwnshipSymbol(img, px, py, player.HeadingDeg, render.ColorHighlight)
	if fish != nil && fish.Alive {
		a.drawWepsTorpedoGeometry(img, px, py, player, fish)
	}

	render.DrawText(img, fmt.Sprintf("ZOOM %.3f  GYRO %03.0f", a.wepsMapZoom, fc.GyroAngleDeg), 10, 16, render.ColorPhosphorDim, true)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(wepsMapX), float64(wepsMapY))
	screen.DrawImage(img, op)
}

// wepsMapMarkerInside requires icon + label to stay inside the map buffer.
func wepsMapMarkerInside(sx, sy float64) bool {
	const (
		left   = 14
		top    = 14
		right  = 48 // room for "C0x" label
		bottom = 14
	)
	return sx >= left && sy >= top &&
		sx <= float64(wepsMapW)-right &&
		sy <= float64(wepsMapH)-bottom
}

func (a *App) drawWepsTorpedoGeometry(screen *ebiten.Image, px, py float64, player *world.Entity, fish *weapons.Torpedo) {
	tx := px + (fish.X-player.X)*a.wepsMapZoom
	ty := py - (fish.Y-player.Y)*a.wepsMapZoom
	if !wepsMapMarkerInside(tx, ty) {
		return
	}
	head := fish.HeadingDeg
	// Endurance range at HIGH cruise — reference for marker lengths.
	const enduranceSec = 600.0
	maxSpd := 55.0 // HIGH
	minSpd := 28.0 // LOW
	maxRangeYd := enduranceSec * maxSpd * world.KnotsToYPS
	coneYd := maxRangeYd / 20 // fixed seek cone length
	// Course line scales with speed: 1/50 at LOW (or below) → 1/20 at HIGH.
	span := maxSpd - minSpd
	t := 0.0
	if span > 0 {
		t = (fish.SpeedKts - minSpd) / span
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	lineLo := maxRangeYd / 50
	lineHi := maxRangeYd / 20
	lineYd := lineLo + t*(lineHi-lineLo)

	rad := head * math.Pi / 180
	lx := tx + math.Sin(rad)*lineYd*a.wepsMapZoom
	ly := ty - math.Cos(rad)*lineYd*a.wepsMapZoom
	render.DrawLine(screen, tx, ty, lx, ly, color.RGBA{100, 200, 255, 255})

	coneR := coneYd * a.wepsMapZoom
	for _, ang := range []float64{head - weapons.SeekConeHalfAngleDeg, head + weapons.SeekConeHalfAngleDeg} {
		ar := ang * math.Pi / 180
		render.DrawLine(screen, tx, ty, tx+math.Sin(ar)*coneR, ty-math.Cos(ar)*coneR, color.RGBA{255, 200, 60, 180})
	}
}
