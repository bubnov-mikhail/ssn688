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
	wepsMapH = 590

	wepsTubeY0   = 124
	wepsTubeRowH = 36

	// Combined NEXT SHOT PREP / WIRE GUIDE / HARPOON PREP panel.
	wepsCtrlY = 276
	wepsCtrlH = 58

	wepsCMY = 344
	wepsCMH = 48

	wepsTargetsY   = 402
	wepsTargetsH   = 288
	wepsTargetsRow = 22
)

type wepsCtrlMode int

const (
	wepsCtrlInactive wepsCtrlMode = iota
	wepsCtrlPrep
	wepsCtrlHarpoonPrep
	wepsCtrlWire
)

func (a *App) wepsTubeButtons(tube int, y int) []sonarUIButton {
	const btnH = 24
	btnY := y + 4 - btnH/2
	doorLabel := "OPEN"
	doorTip := "Open outer door"
	fc := &a.Engine.FireControl
	if tube >= 1 && tube <= 4 {
		st := fc.Tubes[tube-1].State
		if st == weapons.TubeDoorOpen || st == weapons.TubeFired {
			doorLabel = "CLOSE"
			doorTip = "Close outer door"
		}
	}
	specs := []buttonSpec{
		{fmt.Sprintf("tube_%d_door", tube), doorLabel, doorTip},
		{fmt.Sprintf("tube_%d_fire", tube), "FIRE", "Launch (door must be open)"},
	}
	x := wepsTubeActionsX(specs, 4)
	return layoutButtonRow(x, btnY, btnH, 4, specs)
}

// wepsTubeActionsX right-aligns OPEN/FIRE within the left WEPS column.
func wepsTubeActionsX(specs []buttonSpec, gap int) int {
	blockRight := wepsLeftX + wepsLeftW - 6
	totalW := 0
	for i, s := range specs {
		if i > 0 {
			totalW += gap
		}
		totalW += render.ButtonWidth(s.label, 14)
	}
	const padR = 10
	return blockRight - padR - totalW
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

func (a *App) wepsControlMode(fc *weapons.FireControl) (wepsCtrlMode, *weapons.Torpedo) {
	if fc == nil || fc.SelectedTube < 1 || fc.SelectedTube > 4 {
		return wepsCtrlInactive, nil
	}
	fish := fc.TorpedoForTube(fc.SelectedTube)
	if fish != nil && fish.Alive && !fish.WireCut {
		tube := fc.TubeByNumber(fc.SelectedTube)
		if tube != nil && tube.WireIntact {
			return wepsCtrlWire, fish
		}
	}
	st := fc.Tubes[fc.SelectedTube-1].State
	tube := &fc.Tubes[fc.SelectedTube-1]
	if st == weapons.TubeFired {
		if fc.HarpoonByTube(fc.SelectedTube) != nil {
			return wepsCtrlInactive, nil
		}
	}
	if st == weapons.TubeLoaded || st == weapons.TubeDoorOpen {
		if weapons.NormalizeOrdnance(tube.TorpedoType) == weapons.OrdnanceHarpoon {
			return wepsCtrlHarpoonPrep, nil
		}
		return wepsCtrlPrep, nil
	}
	if st == weapons.TubeReloading && weapons.NormalizeOrdnance(tube.ReloadOrdnance) == weapons.OrdnanceHarpoon {
		return wepsCtrlHarpoonPrep, nil
	}
	return wepsCtrlInactive, nil
}

func wepsCtrlSpinLayout() (spinY, spinH, btnW, g0, d0, inner, toggleX int) {
	spinY = wepsCtrlY + 28
	spinH = 24
	btnW = 22
	g0 = wepsLeftX + 8
	d0 = wepsLeftX + 140
	inner = 88
	toggleX = d0 + btnW + inner + btnW + 8
	return
}

func (a *App) wepsCtrlButtons(fc *weapons.FireControl) []sonarUIButton {
	mode, fish := a.wepsControlMode(fc)
	spinY, spinH, btnW, g0, d0, inner, toggleX := wepsCtrlSpinLayout()

	seekOn := false
	switch mode {
	case wepsCtrlWire:
		if fish != nil {
			if !fish.TubeCleared() {
				seekOn = fish.EnableSearchAfterClear
			} else {
				seekOn = fish.Mode == weapons.ModeSearch || fish.SeekerOn
			}
		}
	case wepsCtrlPrep:
		seekOn = fc != nil && fc.SeekerEnabled
	}

	seekLabel := "SEEK OFF"
	if seekOn {
		seekLabel = "SEEK ON"
	}

	btns := []sonarUIButton{
		{ID: "gyro_m", Label: "-", Tooltip: "Decrease course 5°", X: g0, Y: spinY, W: btnW, H: spinH},
		{ID: "gyro_p", Label: "+", Tooltip: "Increase course 5°", X: g0 + btnW + inner, Y: spinY, W: btnW, H: spinH},
		{ID: "dep_m", Label: "-", Tooltip: "Shallower 50 ft", X: d0, Y: spinY, W: btnW, H: spinH},
		{ID: "dep_p", Label: "+", Tooltip: "Deeper 50 ft", X: d0 + btnW + inner, Y: spinY, W: btnW, H: spinH},
	}

	lowW := render.ButtonWidth("LOW", 14)
	highW := render.ButtonWidth("HIGH", 14)
	btns = append(btns,
		sonarUIButton{ID: "spd_low", Label: "LOW", Tooltip: "Run speed LOW (28 kts)", X: toggleX, Y: spinY, W: lowW, H: spinH},
		sonarUIButton{ID: "spd_high", Label: "HIGH", Tooltip: "Run speed HIGH (55 kts)", X: toggleX + lowW + 2, Y: spinY, W: highW, H: spinH},
	)
	seekX := toggleX + lowW + 2 + highW + 6
	btns = append(btns, layoutButtonRow(seekX, spinY, spinH, 4, []buttonSpec{
		{"seek", seekLabel, "Toggle seeker / course-hold"},
	})...)

	if mode == wepsCtrlWire {
		btns = append(btns, a.wepsWireAuxButtons()...)
	}
	return btns
}

func (a *App) wepsWireAuxButtons() []sonarUIButton {
	const (
		btnH = 22
		gap  = 6
		padR = 10
	)
	specs := []buttonSpec{
		{"wire_cut", "CUT", "Cut guidance wire"},
		{"wire_sd", "S/D", "Self-destruct while wire intact"},
	}
	totalW := 0
	for i, s := range specs {
		if i > 0 {
			totalW += gap
		}
		totalW += render.ButtonWidth(s.label, 14)
	}
	blockRight := wepsLeftX - 6 + wepsLeftW
	x := blockRight - padR - totalW
	y := wepsCtrlY + (26-btnH)/2 + 4
	return layoutButtonRow(x, y, btnH, gap, specs)
}

func (a *App) wepsCMButtons(decoyN, jitterN int) []sonarUIButton {
	const (
		btnH = 26
		gap  = 8
		padR = 10
	)
	specs := []buttonSpec{
		{"cm_decoy", fmt.Sprintf("DECOY %d", decoyN), "Launch acoustic decoy (ADC) toward threat"},
		{"cm_jitter", fmt.Sprintf("JITTER %d", jitterN), "Launch broadband jammer toward threat"},
	}
	totalW := 0
	for i, s := range specs {
		if i > 0 {
			totalW += gap
		}
		totalW += render.ButtonWidth(s.label, 14)
	}
	blockRight := wepsLeftX - 6 + wepsLeftW
	x := blockRight - padR - totalW
	y := wepsCMY + (wepsCMH-btnH)/2
	return layoutButtonRow(x, y, btnH, gap, specs)
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
	mode, fish := a.wepsControlMode(fc)
	speedHigh := fc.SpeedSetting == "HIGH"
	seekOn := fc.SeekerEnabled
	if mode == wepsCtrlWire && fish != nil {
		speedHigh = fish.CruiseKts >= 40
		if !fish.TubeCleared() {
			seekOn = fish.EnableSearchAfterClear
		} else {
			seekOn = fish.Mode == weapons.ModeSearch || fish.SeekerOn
		}
	}
	switch id {
	case "spd_low":
		return !speedHigh
	case "spd_high":
		return speedHigh
	case "spd":
		return speedHigh
	case "seek", "wire_seek":
		return seekOn
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
			if (action == "open" || action == "door") && tube >= 1 && tube <= 4 {
				st := fc.Tubes[tube-1].State
				return st == weapons.TubeDoorOpen || st == weapons.TubeFired
			}
		}
	}
	return false
}

func (a *App) drawWepsButton(screen *ebiten.Image, b sonarUIButton, mx, my int, fc *weapons.FireControl) {
	a.drawWepsButtonState(screen, b, mx, my, fc, true)
}

func (a *App) drawWepsButtonState(screen *ebiten.Image, b sonarUIButton, mx, my int, fc *weapons.FireControl, enabled bool) {
	if enabled && a.wepsButtonLatched(b.ID, fc) {
		render.FillRect(screen, b.X-2, b.Y-2, b.W+4, b.H+4, render.ColorAmber)
	}
	hover := enabled && b.contains(mx, my)
	pressed := enabled && a.wepsButtonFlash(b.ID)
	render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	if !enabled {
		render.FillRect(screen, b.X, b.Y, b.W, b.H, color.RGBA{12, 12, 14, 140})
	}
}

func (a *App) handleFireControl(fc *weapons.FireControl, player *world.Entity) {
	gt := a.Engine.Clock.GameTime
	sonar := &a.Engine.Sonar
	a.validateSelectedContact(sonar)
	a.validateTubeContactTargets(sonar)
	mx, my := ebiten.CursorPosition()
	scrollContactTableWheel(mx, my, wepsLeftX, wepsTargetsY+40, wepsLeftW-16, wepsVisibleContactRows()*wepsTargetsRow, len(sonar.Contacts), wepsVisibleContactRows(), &a.contactTableScroll.weps)

	var all []sonarUIButton
	for i := 1; i <= 4; i++ {
		y := wepsTubeY0 + (i-1)*wepsTubeRowH
		all = append(all, a.wepsTubePickButton(i, y))
		all = append(all, a.wepsTubeButtons(i, y)...)
	}
	mode, _ := a.wepsControlMode(fc)
	ctrlActive := mode != wepsCtrlInactive
	var ctrlBtns []sonarUIButton
	if mode == wepsCtrlHarpoonPrep {
		ctrlBtns = a.wepsHarpoonCtrlButtons(fc)
	} else if ctrlActive {
		ctrlBtns = a.wepsCtrlButtons(fc)
	}
	all = append(all, a.wepsCollectOrdnanceButtons(fc)...)
	if ctrlActive {
		all = append(all, ctrlBtns...)
	}
	decoyN, jitterN := 0, 0
	if player != nil {
		decoyN = a.Engine.CM.DecoyLeft(player.ID)
		jitterN = a.Engine.CM.JitterLeft(player.ID)
	}
	all = append(all, a.wepsCMButtons(decoyN, jitterN)...)
	a.updateSonarTooltips(all, mx, my)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// Open ordnance popup steals clicks from panels underneath.
		if a.wepsOrdnanceMenuTube != 0 {
			y := wepsTubeY0 + (a.wepsOrdnanceMenuTube-1)*wepsTubeRowH
			menuBtns := a.wepsOrdnanceMenuButtons(a.wepsOrdnanceMenuTube, y)
			pick := a.wepsOrdnancePickButton(a.wepsOrdnanceMenuTube, y)
			for _, b := range menuBtns {
				if b.contains(mx, my) {
					if a.wepsOrdnanceAction(b.ID, fc, gt) {
						a.uiPressedID = b.ID
						a.uiPressedAt = time.Now()
						return
					}
				}
			}
			onPopup := pick.contains(mx, my)
			for _, b := range menuBtns {
				if b.contains(mx, my) {
					onPopup = true
					break
				}
			}
			if !onPopup {
				a.wepsOrdnanceMenuTube = 0
			}
		}
		for _, b := range all {
			if b.contains(mx, my) {
				if a.wepsOrdnanceAction(b.ID, fc, gt) {
					a.uiPressedID = b.ID
					a.uiPressedAt = time.Now()
					return
				}
				a.wepsButtonAction(b.ID, fc, player, gt)
				a.uiPressedID = b.ID
				a.uiPressedAt = time.Now()
				return
			}
		}
		if idx := a.wepsTargetRowAt(mx, my, sonar); idx >= 0 {
			c := &sonar.Contacts[idx]
			a.wepsAssignTubeTarget(fc, player, sonar, c)
			a.wepsFitSelectedContact()
			return
		}
	}
	if a.uiPressedID != "" && time.Since(a.uiPressedAt) > 120*time.Millisecond {
		a.uiPressedID = ""
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyDigit1) {
		a.wepsSelectTube(fc, 1, player)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit2) {
		a.wepsSelectTube(fc, 2, player)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit3) {
		a.wepsSelectTube(fc, 3, player)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit4) {
		a.wepsSelectTube(fc, 4, player)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyO) {
		a.wepsButtonAction(fmt.Sprintf("tube_%d_door", fc.SelectedTube), fc, player, gt)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		a.wepsButtonAction(fmt.Sprintf("tube_%d_door", fc.SelectedTube), fc, player, gt)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		a.wepsButtonAction(fmt.Sprintf("tube_%d_fire", fc.SelectedTube), fc, player, gt)
	}
	if ctrlActive {
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
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
			a.wepsButtonAction("gyro_m", fc, player, gt)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
			a.wepsButtonAction("gyro_p", fc, player, gt)
		}
	}
	if mode == wepsCtrlWire {
		if inpututil.IsKeyJustPressed(ebiten.KeyW) {
			a.wepsButtonAction("wire_cut", fc, player, gt)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyX) {
			a.wepsButtonAction("wire_sd", fc, player, gt)
		}
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
			a.wepsSelectTube(fc, tube, player)
		case "door":
			st := weapons.TubeEmpty
			if tube >= 1 && tube <= 4 {
				st = fc.Tubes[tube-1].State
			}
			if st == weapons.TubeDoorOpen || st == weapons.TubeFired {
				if fc.CloseOuterDoor(tube, gameTime) {
					a.Engine.EmitTubeTransient(player, gameTime, false)
					a.Audio.PlayClip(audio.ClipWepsOuterDoorClosed, "")
				}
			} else {
				if fc.OpenOuterDoor(tube) {
					a.Engine.EmitTubeTransient(player, gameTime, true)
					a.Audio.PlayClip(audio.TubeClip("outer_door_open", tube),
						fmt.Sprintf("Outer door open, tube %d.", tube))
				}
			}
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
			a.wepsFireTube(fc, player, tube, gameTime)
		}
	case id == "harp_beam":
		if mode, _ := a.wepsControlMode(fc); mode == wepsCtrlHarpoonPrep {
			fc.CycleHarpoonBeam()
		}
	case id == "harp_srch":
		if mode, _ := a.wepsControlMode(fc); mode == wepsCtrlHarpoonPrep {
			fc.CycleHarpoonRadarRange()
		}
	case id == "harp_dstr":
		if mode, _ := a.wepsControlMode(fc); mode == wepsCtrlHarpoonPrep {
			fc.CycleHarpoonDestructRange()
		}
	case id == "gyro_m", id == "gyro_p", id == "dep_m", id == "dep_p",
		id == "spd", id == "spd_low", id == "spd_high", id == "seek":
		mode, fish := a.wepsControlMode(fc)
		if mode == wepsCtrlInactive {
			return
		}
		if mode == wepsCtrlHarpoonPrep {
			switch id {
			case "gyro_m":
				fc.GyroAngleDeg = normalizeGyroDeg(fc.GyroAngleDeg - 5)
			case "gyro_p":
				fc.GyroAngleDeg = normalizeGyroDeg(fc.GyroAngleDeg + 5)
			}
			return
		}
		if mode == wepsCtrlWire {
			if fish == nil {
				return
			}
			switch id {
			case "gyro_m":
				fc.WireSteer(fish, -5, 0)
			case "gyro_p":
				fc.WireSteer(fish, 5, 0)
			case "dep_m":
				fc.WireSteer(fish, 0, -50)
				a.Audio.PlayClip(audio.ClipWepsRunDepthSet, fmt.Sprintf("Torpedo run depth %d feet.", int(fish.RunDepthFt)))
			case "dep_p":
				fc.WireSteer(fish, 0, 50)
				a.Audio.PlayClip(audio.ClipWepsRunDepthSet, fmt.Sprintf("Torpedo run depth %d feet.", int(fish.RunDepthFt)))
			case "spd", "spd_low", "spd_high":
				wantHigh := fish.CruiseKts < 40
				if id == "spd_low" {
					wantHigh = false
				} else if id == "spd_high" {
					wantHigh = true
				}
				if wantHigh {
					fish.CruiseKts = weapons.CruiseSpeedKts("HIGH")
					a.Audio.PlayClip(audio.ClipWepsSpeedHigh, "Torpedo speed HIGH.")
				} else {
					fish.CruiseKts = weapons.CruiseSpeedKts("LOW")
					a.Audio.PlayClip(audio.ClipWepsSpeedLow, "Torpedo speed LOW.")
				}
			case "seek":
				wasSearch := fish.Mode == weapons.ModeSearch || fish.SeekerOn || fish.EnableSearchAfterClear
				fc.ToggleSeeker(fish)
				nowSearch := false
				if !fish.TubeCleared() {
					nowSearch = fish.EnableSearchAfterClear
				} else {
					nowSearch = fish.Mode == weapons.ModeSearch || fish.SeekerOn
				}
				if nowSearch && !wasSearch {
					a.Audio.PlayClip(audio.ClipWepsSeekerOn, "")
				} else if wasSearch && !nowSearch {
					a.Audio.PlayClip(audio.ClipWepsSeekerOff, "")
				}
			}
			return
		}
		// Prep mode — next-shot orders.
		switch id {
		case "gyro_m":
			fc.GyroAngleDeg = normalizeGyroDeg(fc.GyroAngleDeg - 5)
		case "gyro_p":
			fc.GyroAngleDeg = normalizeGyroDeg(fc.GyroAngleDeg + 5)
		case "dep_m":
			fc.RunDepthFt = max(40, fc.RunDepthFt-50)
			a.Audio.PlayClip(audio.ClipWepsRunDepthSet, fmt.Sprintf("Torpedo run depth %d feet.", int(fc.RunDepthFt)))
		case "dep_p":
			fc.RunDepthFt += 50
			a.Audio.PlayClip(audio.ClipWepsRunDepthSet, fmt.Sprintf("Torpedo run depth %d feet.", int(fc.RunDepthFt)))
		case "spd", "spd_low", "spd_high":
			wantHigh := fc.SpeedSetting != "HIGH"
			if id == "spd_low" {
				wantHigh = false
			} else if id == "spd_high" {
				wantHigh = true
			}
			if wantHigh {
				fc.SpeedSetting = "HIGH"
				a.Audio.PlayClip(audio.ClipWepsSpeedHigh, "Torpedo speed HIGH.")
			} else {
				fc.SpeedSetting = "LOW"
				a.Audio.PlayClip(audio.ClipWepsSpeedLow, "Torpedo speed LOW.")
			}
			if c := a.selectedContact(&a.Engine.Sonar); c != nil {
				a.wepsApplyContactToPrep(fc, player, c)
			}
		case "seek":
			fc.SeekerEnabled = !fc.SeekerEnabled
			if fc.SeekerEnabled {
				a.Audio.PlayClip(audio.ClipWepsSeekerOn, "")
			} else {
				a.Audio.PlayClip(audio.ClipWepsSeekerOff, "")
			}
		}
	case id == "wire_cut", id == "wire_sd":
		fish := fc.TorpedoForTube(fc.SelectedTube)
		if fish == nil {
			a.StatusMessage = "No wire-guided fish on selected tube."
			return
		}
		switch id {
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
	case id == "cm_decoy":
		_, msg := a.Engine.LaunchPlayerDecoy()
		a.StatusMessage = msg
	case id == "cm_jitter":
		_, msg := a.Engine.LaunchPlayerJitter()
		a.StatusMessage = msg
	}
}

func (a *App) drawFireControl(screen *ebiten.Image) {
	fc := &a.Engine.FireControl
	sonar := &a.Engine.Sonar
	gt := a.Engine.Clock.GameTime
	render.DrawConsolePanel(screen, wepsPanelX, wepsPanelY, wepsPanelW, wepsPanelH)
	render.DrawScreenTitle(screen, "FIRE CONTROL — WEAPONS", wepsLeftX, 78)
	render.DrawText(screen, fmt.Sprintf("MAGAZINE: %d Mk48  ·  %d Harpoon", fc.MagazineLeft, fc.HarpoonMagLeft), wepsLeftX, 100, render.ColorPlateLabel, true)

	mx, my := ebiten.CursorPosition()
	for i := range fc.Tubes {
		t := fc.Tubes[i]
		y := wepsTubeY0 + i*wepsTubeRowH
		titleBtn := a.wepsTubePickButton(t.Number, y)
		ordBtn := a.wepsOrdnancePickButton(t.Number, y)
		statusExtra := wepsTubeRowStatusExtra(t, fc.ReloadRemaining(t, gt))
		if player := a.Engine.Scenario.Player; player != nil {
			player.EnsureDamage()
			sys := world.TubeSys(t.Number)
			if sys != world.SysNone && !player.Damage.Operational(sys) {
				statusExtra = fmt.Sprintf("DAMAGED (%.0f%%)", player.Damage.EffOf(sys))
			}
		}
		if t.Number == fc.SelectedTube {
			render.FillRect(screen, wepsLeftX-4, y-10, wepsLeftW-8, 28, color.RGBA{58, 52, 20, 180})
		}
		clr := render.ColorText
		if titleBtn.contains(mx, my) || t.Number == fc.SelectedTube {
			clr = render.ColorHighlight
		}
		render.DrawText(screen, titleBtn.Label, titleBtn.X, y+4, clr, true)
		a.drawWepsButton(screen, ordBtn, mx, my, fc)
		line := statusExtra
		if t.TargetContactID != "" {
			if line != "" {
				line += " · "
			}
			line += t.TargetContactID
		}
		if line != "" {
			statusX := ordBtn.X + ordBtn.W + 8
			render.DrawText(screen, line, statusX, y+4, render.ColorDim, true)
		}
		for _, b := range a.wepsTubeButtons(t.Number, y) {
			a.drawWepsButton(screen, b, mx, my, fc)
		}
	}

	a.drawWepsControlPanel(screen, fc, mx, my)

	fish := fc.TorpedoForTube(fc.SelectedTube)
	harp := fc.HarpoonByTube(fc.SelectedTube)

	a.drawWepsGroup(screen, wepsLeftX-6, wepsCMY, wepsLeftW, wepsCMH, "COUNTERMEASURES")
	decoyN, jitterN := 0, 0
	if player := a.Engine.Scenario.Player; player != nil {
		decoyN = a.Engine.CM.DecoyLeft(player.ID)
		jitterN = a.Engine.CM.JitterLeft(player.ID)
	}
	for _, b := range a.wepsCMButtons(decoyN, jitterN) {
		a.drawWepsButton(screen, b, mx, my, fc)
	}

	a.drawWepsContactTable(screen, sonar)
	a.drawWepsMap(screen, sonar, fish, harp)

	// Popups last so they sit above panels (tube-4 ordnance opens over fire settings).
	a.drawWepsOrdnanceMenus(screen, fc, mx, my)

	if a.uiTooltip != "" {
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
	help := "[1-4] tube  [O/C] door  [ENTER] fire  [G/←→] course  [D] depth  [S] speed  [H] seek  [W] cut  [X] S/D"
	if mode, _ := a.wepsControlMode(fc); mode == wepsCtrlHarpoonPrep {
		help = "[1-4] tube  [O/C] door  [ENTER] fire  [G/←→] course  — BEAM / SRCH / DSTR set seeker profile"
	}
	render.DrawText(screen, help, wepsLeftX, wepsPanelY+wepsPanelH-16, render.ColorDim, true)
}

func (a *App) drawWepsControlPanel(screen *ebiten.Image, fc *weapons.FireControl, mx, my int) {
	mode, fish := a.wepsControlMode(fc)
	title := "NEXT SHOT PREP"
	switch mode {
	case wepsCtrlWire:
		title = "WIRE GUIDE"
	case wepsCtrlHarpoonPrep:
		title = "HARPOON PREP"
	}
	a.drawWepsGroup(screen, wepsLeftX-6, wepsCtrlY, wepsLeftW, wepsCtrlH, title)
	a.drawWepsSpinLabels(screen, fc, mode, fish)
	enabled := mode != wepsCtrlInactive
	var ctrlBtns []sonarUIButton
	switch mode {
	case wepsCtrlHarpoonPrep:
		ctrlBtns = a.wepsHarpoonCtrlButtons(fc)
	case wepsCtrlInactive:
		ctrlBtns = nil
	default:
		ctrlBtns = a.wepsCtrlButtons(fc)
	}
	for _, b := range ctrlBtns {
		btnOn := enabled
		if b.ID == "wire_cut" || b.ID == "wire_sd" {
			btnOn = mode == wepsCtrlWire
		}
		a.drawWepsButtonState(screen, b, mx, my, fc, btnOn)
	}
}

func (a *App) drawWepsSpinLabels(screen *ebiten.Image, fc *weapons.FireControl, mode wepsCtrlMode, fish *weapons.Torpedo) {
	_, _, btnW, g0, d0, inner, _ := wepsCtrlSpinLayout()
	spinY := wepsCtrlY + 28
	gyroMid := g0 + btnW + inner/2
	depMid := d0 + btnW + inner/2
	gyroTxt := "GYRO ---"
	depTxt := "DEP ---"
	switch mode {
	case wepsCtrlPrep:
		gyroTxt = fmt.Sprintf("GYRO %03.0f°", fc.GyroAngleDeg)
		depTxt = fmt.Sprintf("DEP %d", int(fc.RunDepthFt))
	case wepsCtrlHarpoonPrep:
		gyroTxt = fmt.Sprintf("GYRO %03.0f°", fc.GyroAngleDeg)
		depTxt = ""
	case wepsCtrlWire:
		if fish != nil {
			gyroTxt = fmt.Sprintf("GYRO %03.0f°", fish.GyroCourseDeg)
			depTxt = fmt.Sprintf("DEP %d", int(fish.RunDepthFt))
		}
	}
	clr := render.ColorAmber
	if mode == wepsCtrlInactive {
		clr = render.ColorDim
	}
	render.DrawText(screen, gyroTxt, gyroMid-len(gyroTxt)*3, spinY+16, clr, true)
	if depTxt != "" {
		render.DrawText(screen, depTxt, depMid-len(depTxt)*3, spinY+16, clr, true)
	}
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
	render.DrawText(screen, "BRG°", wepsLeftX+86, hdrY, render.ColorPhosphorDim, true)
	render.DrawText(screen, "RNG kyd", wepsLeftX+126, hdrY, render.ColorPhosphorDim, true)
	render.DrawText(screen, "CLASS", wepsLeftX+178, hdrY, render.ColorPhosphorDim, true)

	mx, my := ebiten.CursorPosition()
	rowY := wepsTargetsY + 40
	visibleRows := wepsVisibleContactRows()
	a.contactTableScroll.weps = clampContactTableScroll(a.contactTableScroll.weps, len(sonar.Contacts), visibleRows)
	start, end := contactTableWindow(len(sonar.Contacts), a.contactTableScroll.weps, visibleRows)
	for i := start; i < end; i++ {
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
		render.DrawText(screen, contactBearingLabel(c), wepsLeftX+86, ty, clr, true)
		render.DrawText(screen, contactRangeLabel(c), wepsLeftX+126, ty, clr, true)
		render.DrawText(screen, contactClassLabel(c), wepsLeftX+178, ty, clr, true)
		rowY += wepsTargetsRow
	}
	drawContactTableScrollbar(screen, wepsLeftX+wepsLeftW-10, wepsTargetsY+40, visibleRows*wepsTargetsRow, len(sonar.Contacts), visibleRows, a.contactTableScroll.weps)
}

func (a *App) wepsTargetRowAt(mx, my int, sonar *acoustics.SonarState) int {
	visibleRows := wepsVisibleContactRows()
	a.contactTableScroll.weps = clampContactTableScroll(a.contactTableScroll.weps, len(sonar.Contacts), visibleRows)
	start, end := contactTableWindow(len(sonar.Contacts), a.contactTableScroll.weps, visibleRows)
	rowY := wepsTargetsY + 40
	for i := start; i < end; i++ {
		if mx >= wepsLeftX && mx < wepsLeftX+wepsLeftW-16 && my >= rowY && my < rowY+wepsTargetsRow {
			return i
		}
		rowY += wepsTargetsRow
	}
	return -1
}

func wepsVisibleContactRows() int {
	rows := (wepsTargetsH - 48) / wepsTargetsRow
	if rows < 1 {
		return 1
	}
	return rows
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

func contactBySonarID(sonar *acoustics.SonarState, contactID string) *acoustics.Contact {
	if sonar == nil || contactID == "" {
		return nil
	}
	for i := range sonar.Contacts {
		if sonar.Contacts[i].ID == contactID {
			return &sonar.Contacts[i]
		}
	}
	return nil
}

func (a *App) validateTubeContactTargets(sonar *acoustics.SonarState) {
	if a.Engine == nil || sonar == nil {
		return
	}
	for i := range a.Engine.FireControl.Tubes {
		id := a.Engine.FireControl.Tubes[i].TargetContactID
		if id == "" {
			continue
		}
		if contactBySonarID(sonar, id) == nil {
			a.Engine.FireControl.Tubes[i].TargetContactID = ""
		}
	}
}

// wepsSelectTube picks a tube and syncs the contact table to its assigned target (WEPS only).
func (a *App) wepsSelectTube(fc *weapons.FireControl, tube int, player *world.Entity) {
	if fc == nil {
		return
	}
	fc.SelectTube(tube)
	if a.Engine == nil || a.CurrentScreen != ScreenFireControl {
		return
	}
	a.wepsSyncSelectionFromTube(fc, player, &a.Engine.Sonar)
}

func (a *App) wepsSyncSelectionFromTube(fc *weapons.FireControl, player *world.Entity, sonar *acoustics.SonarState) {
	if fc == nil || sonar == nil {
		return
	}
	t := fc.TubeByNumber(fc.SelectedTube)
	if t == nil || t.TargetContactID == "" {
		return
	}
	c := contactBySonarID(sonar, t.TargetContactID)
	if c == nil {
		return
	}
	a.selectContact(sonar, c)
	a.wepsApplyContactToPrep(fc, player, c)
}

// wepsAssignTubeTarget binds the selected tube to a contact and updates fire-control prep (WEPS only).
func (a *App) wepsAssignTubeTarget(fc *weapons.FireControl, player *world.Entity, sonar *acoustics.SonarState, c *acoustics.Contact) {
	if fc == nil || c == nil {
		return
	}
	fc.SetTubeTargetContact(fc.SelectedTube, c.ID)
	a.selectContact(sonar, c)
	a.wepsApplyContactToPrep(fc, player, c)
}

func (a *App) wepsApplyContactToPrep(fc *weapons.FireControl, player *world.Entity, c *acoustics.Contact) {
	if fc == nil || c == nil {
		return
	}
	gt := 0.0
	if a.Engine != nil {
		gt = a.Engine.Clock.GameTime
	}
	if mode, fish := a.wepsControlMode(fc); mode == wepsCtrlWire && fish != nil {
		a.wepsApplyContactToWireFish(fc, player, c, fish, gt)
		return
	}
	if mode, _ := a.wepsControlMode(fc); mode == wepsCtrlHarpoonPrep {
		a.wepsApplyContactToHarpoonPrep(fc, player, c)
		return
	}
	fc.GyroAngleDeg = normalizeGyroDeg(c.BearingDeg)
	if player != nil && contactHasKnownRange(c, gt) && contactTMAUsableForLead(c) {
		tx, ty := contactPlotRaw(player, c, gt)
		weaponKts := weapons.CruiseSpeedKts(fc.SpeedSetting)
		if course, ok := weapons.TorpedoInterceptGyro(
			player.X, player.Y, player.HeadingDeg,
			tx, ty, c.TMACourseDeg, c.TMASpeedKts, weaponKts,
		); ok {
			fc.GyroAngleDeg = normalizeGyroDeg(course)
		}
	}
	fc.RunDepthFt = wepsSuggestedRunDepth(player, c)
}

func wepsSuggestedRunDepth(player *world.Entity, c *acoustics.Contact) float64 {
	depth := 200.0
	if player != nil {
		depth = math.Max(80, player.DepthFt)
	}
	if c == nil || c.ConfirmedClass == "" {
		return depth
	}
	switch contactConfirmedKind(c) {
	case world.KindSurfaceShip:
		// Shallow under-keel run for surface targets.
		return 40
	case world.KindSubmarine:
		if player != nil {
			return math.Max(80, player.DepthFt)
		}
	case world.KindTorpedo:
		return 100
	}
	return depth
}

// contactConfirmedKind prefers the signature-library Kind for a confirmed class,
// so SPECTRUM classification drives WEPS depth even if track Kind lagged.
func contactConfirmedKind(c *acoustics.Contact) world.EntityKind {
	if c == nil {
		return world.EntityKind(-1)
	}
	if c.ConfirmedID != "" {
		if p, ok := world.ProfileByID(c.ConfirmedID); ok {
			return p.Kind
		}
	}
	if c.ConfirmedClass != "" {
		for _, p := range world.SignatureLibrary {
			if p.Name == c.ConfirmedClass || p.Class == c.ConfirmedClass {
				return p.Kind
			}
		}
	}
	return c.Kind
}

// wepsApplyContactToWireFish retargets a wire-guided fish (not in Search) onto the contact.
func (a *App) wepsApplyContactToWireFish(fc *weapons.FireControl, player *world.Entity, c *acoustics.Contact, fish *weapons.Torpedo, gameTime float64) {
	if fish == nil || c == nil || player == nil {
		return
	}
	if fish.Mode == weapons.ModeSearch || fish.SeekerOn {
		return
	}
	tx, ty := contactPlotRaw(player, c, gameTime)
	course := normalizeGyroDeg(math.Atan2(tx-fish.X, ty-fish.Y) * 180 / math.Pi)
	weaponKts := fish.CruiseKts
	if weaponKts < 1 {
		weaponKts = weapons.CruiseSpeedKts(fc.SpeedSetting)
	}
	if contactHasKnownRange(c, gameTime) && acoustics.ContactTMAAccurate(c) {
		if ic, ok := weapons.InterceptCourseDeg(tx-fish.X, ty-fish.Y, c.TMACourseDeg, c.TMASpeedKts, weaponKts); ok {
			course = normalizeGyroDeg(ic)
		}
	}
	fc.WireSetCourse(fish, course)
	fish.RunDepthFt = wepsSuggestedRunDepth(player, c)
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

func (a *App) drawWepsMap(screen *ebiten.Image, sonar *acoustics.SonarState, fish *weapons.Torpedo, harp *weapons.HarpoonMissile) {
	fc := &a.Engine.FireControl
	render.DrawText(screen, "TACTICAL MAP", wepsMapX, 86, render.ColorPlateLabel, true)
	render.DrawMonitor(screen, wepsMapX, wepsMapY, wepsMapW, wepsMapH)
	mapBorder := color.RGBA{78, 78, 84, 255}
	render.FillRect(screen, wepsMapX, wepsMapY, wepsMapW, 1, mapBorder)
	render.FillRect(screen, wepsMapX, wepsMapY+wepsMapH-1, wepsMapW, 1, mapBorder)
	render.FillRect(screen, wepsMapX, wepsMapY, 1, wepsMapH, mapBorder)
	render.FillRect(screen, wepsMapX+wepsMapW-1, wepsMapY, 1, wepsMapH, mapBorder)

	img := a.ensureWepsMapImg()
	img.Fill(color.RGBA{4, 18, 28, 255})

	player := a.Engine.Scenario.Player
	px := float64(wepsMapW) / 2
	py := float64(wepsMapH) / 2
	ringLabelClr := color.RGBA{0, 150, 120, 210}
	for _, rYd := range []float64{1000, 2000, 4000, 8000} {
		rad := rYd * a.wepsMapZoom
		if rad > 20 && rad < float64(wepsMapW)/2 {
			drawCircle(img, px, py, rad, color.RGBA{0, 70, 55, 160})
			drawMapRangeRingLabel(img, px, py, rad, rYd, ringLabelClr)
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
		clr := render.ColorActive
		// Orange while seeker has a lock (ship or soft-kill CM); blue on wire / no target.
		if torp.Mode == weapons.ModeSearch && torp.TargetID != "" {
			clr = render.ColorAmber
		}
		render.FillRect(img, int(sx)-3, int(sy)-3, 7, 7, clr)
	}
	if a.Settings.Debug {
		a.drawWepsDebugCMMarkers(img, px, py, player, gt)
	}
	// Contacts use the same plotted positions as TACTICAL PLOT (not true emitter coords).
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if c.Kind == world.KindCountermeasure {
			continue
		}
		if wireFishIDs[c.SourceEntityID] {
			continue
		}
		wx, wy := a.contactPlotWorld(player, c, gt)
		sx := px + (wx-player.X)*a.wepsMapZoom
		sy := py - (wy-player.Y)*a.wepsMapZoom
		if !wepsMapMarkerInside(sx, sy) {
			continue
		}
		if x1, y1, ok := contactTMAWorldLineEnd(c, wx, wy); ok {
			lx := px + (x1-player.X)*a.wepsMapZoom
			ly := py - (y1-player.Y)*a.wepsMapZoom
			render.DrawLine(img, sx, sy, lx, ly, contactTMALineColor)
		}
		clr := color.RGBA{150, 155, 160, 255}
		kind := world.EntityKind(-1)
		if c.ConfirmedClass != "" {
			kind = contactConfirmedKind(c)
			if kind == world.KindTorpedo {
				clr = color.RGBA{220, 60, 50, 255}
			}
		}
		if c.SourceEntityID == a.selectedContactID {
			clr = color.RGBA{255, 200, 60, 255}
		}
		drawContactPictogram(img, int(sx), int(sy), kind, clr)
		if a.torpedoThreatMarkerActive(c.SourceEntityID) {
			drawThreatBlinkMarker(img, sx, sy)
		}
		render.DrawText(img, c.ID, int(sx)+8, int(sy)+4, clr, true)
	}

	drawOwnshipSymbol(img, px, py, player.HeadingDeg, render.ColorHighlight)
	if fish != nil && fish.Alive {
		a.drawWepsTorpedoGeometry(img, px, py, player, fish)
		if a.torpedoThreatMarkerActive(fish.ID) {
			tx := px + (fish.X-player.X)*a.wepsMapZoom
			ty := py - (fish.Y-player.Y)*a.wepsMapZoom
			if wepsMapMarkerInside(tx, ty) {
				drawThreatBlinkMarker(img, tx, ty)
			}
		}
	}
	for _, hm := range fc.ActiveHarpoons {
		if hm != nil && hm.Alive && hm.VisibleOnWEPS {
			a.drawWepsHarpoonGeometry(img, px, py, player, hm)
		}
	}
	if harp != nil && harp.Alive && harp.VisibleOnWEPS {
		a.drawWepsHarpoonGeometry(img, px, py, player, harp)
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

func (a *App) drawWepsDebugCMMarkers(img *ebiten.Image, px, py float64, player *world.Entity, gameTime float64) {
	if a.Engine == nil || player == nil {
		return
	}
	fc := &a.Engine.FireControl
	fc.PruneDebugMapFlashes(gameTime)
	dbgClr := color.RGBA{200, 160, 255, 230}

	for _, cm := range a.Engine.CM.Active {
		if cm == nil || !cm.Alive || cm.Kind == weapons.CMTowedNixie {
			continue
		}
		sx := px + (cm.X-player.X)*a.wepsMapZoom
		sy := py - (cm.Y-player.Y)*a.wepsMapZoom
		if !wepsMapMarkerInside(sx, sy) {
			continue
		}
		label := "d"
		if cm.Kind == weapons.CMExpendableJitter {
			label = "j"
		}
		render.DrawText(img, label, int(sx)-2, int(sy)+4, dbgClr, true)
	}

	for _, flash := range fc.DebugMapFlashes {
		sx := px + (flash.X-player.X)*a.wepsMapZoom
		sy := py - (flash.Y-player.Y)*a.wepsMapZoom
		if !wepsMapMarkerInside(sx, sy) {
			continue
		}
		render.DrawText(img, flash.Label, int(sx)-2, int(sy)+4, dbgClr, true)
	}
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
