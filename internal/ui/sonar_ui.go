package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/layout"
	"github.com/ssn688/sim/internal/render"
)

type sonarUIButton struct {
	ID         string
	Label      string
	Tooltip    string
	X, Y, W, H int
}

func (b sonarUIButton) contains(mx, my int) bool {
	return mx >= b.X && mx < b.X+b.W && my >= b.Y && my < b.Y+b.H
}

type buttonSpec struct {
	id, label, tooltip string
}

func layoutButtonRowInto(dst []sonarUIButton, x, y, h, gap int, specs []buttonSpec) []sonarUIButton {
	if cap(dst) < len(specs) {
		dst = make([]sonarUIButton, 0, len(specs))
	} else {
		dst = dst[:0]
	}
	xPos := x
	for _, s := range specs {
		w := render.ButtonWidth(s.label, 14)
		dst = append(dst, sonarUIButton{ID: s.id, Label: s.label, Tooltip: s.tooltip, X: xPos, Y: y, W: w, H: h})
		xPos += w + gap
	}
	return dst
}

func layoutButtonRow(x, y, h, gap int, specs []buttonSpec) []sonarUIButton {
	return layoutButtonRowInto(nil, x, y, h, gap, specs)
}

func passiveArrayButtons(x, y int) []sonarUIButton {
	return layoutButtonRow(x, y, 34, 6, []buttonSpec{
		{"array_hull", "HULL", "Hull-mounted spherical array"},
		{"array_towed", "TOWED", "Towed array (TB-16) — deploy cable first"},
	})
}

func towedControlButtons(x, y int) []sonarUIButton {
	return layoutButtonRow(x, y, 38, 4, []buttonSpec{
		{"towed_deploy", "DEPLOY", "Pay out towed array cable"},
		{"towed_stop", "STOP", "Stop cable motion at present length"},
		{"towed_retract", "RETRACT", "Recover towed array"},
	})
}

var cachedSonarUIButtons struct {
	once     sync.Once
	passive  []sonarUIButton
	towed    []sonarUIButton
	spectrum []sonarUIButton
	band     []sonarUIButton
}

func initCachedSonarUIButtons() {
	cachedSonarUIButtons.passive = passiveArrayButtons(layout.PassiveArrayPlateX, layout.PassiveArrayButtonsY)
	cachedSonarUIButtons.towed = towedControlButtons(layout.PassiveTowedPlateX, layout.PassiveTowedButtonsY)
	cachedSonarUIButtons.spectrum = passiveArrayButtons(spectrumArrayLabelX, spectrumArrayLabelY+22)
	cachedSonarUIButtons.band = layoutButtonRow(layout.PassiveArrayPlateX, layout.PassiveBandButtonsY, 34, 6, []buttonSpec{
		{"band_bb", "SHIP", "Broadband listen — ships and subs; torpedo HF attenuated"},
		{"band_hf", "TORP", "High-frequency listen — torpedo motors; ship LF/MF cut"},
	})
}

func cachedPassiveArrayButtons() []sonarUIButton {
	cachedSonarUIButtons.once.Do(initCachedSonarUIButtons)
	return cachedSonarUIButtons.passive
}

func cachedPassiveTowedButtons() []sonarUIButton {
	cachedSonarUIButtons.once.Do(initCachedSonarUIButtons)
	return cachedSonarUIButtons.towed
}

func cachedSpectrumArrayButtons() []sonarUIButton {
	cachedSonarUIButtons.once.Do(initCachedSonarUIButtons)
	return cachedSonarUIButtons.spectrum
}

func cachedPassiveBandButtons() []sonarUIButton {
	cachedSonarUIButtons.once.Do(initCachedSonarUIButtons)
	return cachedSonarUIButtons.band
}

func (a *App) sonarArrayLabel(sonar *acoustics.SonarState) string {
	if sonar.PassiveArray == acoustics.PassiveArrayTowed {
		return "TOWED"
	}
	return "HULL"
}

func (a *App) towedCableStatus(sonar *acoustics.SonarState) string {
	switch {
	case sonar.TowedInMotion() && sonar.TowedCableRate > 0:
		return fmt.Sprintf("DEPLOYING %.0f%%", sonar.TowedCablePct*100)
	case sonar.TowedInMotion() && sonar.TowedCableRate < 0:
		return fmt.Sprintf("RETRACTING %.0f%%", sonar.TowedCablePct*100)
	case !sonar.TowedDeployed() && !sonar.TowedStowed():
		return fmt.Sprintf("HELD %.0f%%", sonar.TowedCablePct*100)
	case sonar.TowedDeployed():
		return "DEPLOYED"
	case sonar.TowedStowed():
		return "STOWED"
	default:
		return fmt.Sprintf("CABLE %.0f%%", sonar.TowedCablePct*100)
	}
}

func (a *App) updateSonarUIButtons(buttons []sonarUIButton, sonar *acoustics.SonarState) {
	mx, my := ebiten.CursorPosition()
	a.updateSonarTooltips(buttons, mx, my)

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	for _, b := range buttons {
		if b.contains(mx, my) {
			a.sonarButtonAction(b.ID, sonar)
			a.uiPressedID = b.ID
			a.uiPressedAt = time.Now()
		}
	}
}

func (a *App) updateSonarTooltips(buttons []sonarUIButton, mx, my int) {
	hoverID := ""
	for _, b := range buttons {
		if b.contains(mx, my) {
			hoverID = b.ID
			break
		}
	}
	now := time.Now()
	if hoverID != a.uiHoverID {
		a.uiHoverID = hoverID
		a.uiHoverSince = now
		a.uiTooltip = ""
	}
	if hoverID != "" && now.Sub(a.uiHoverSince) >= 400*time.Millisecond {
		for _, b := range buttons {
			if b.ID == hoverID {
				a.uiTooltip = b.Tooltip
				break
			}
		}
	}
}

func (a *App) sonarButtonAction(id string, sonar *acoustics.SonarState) {
	switch id {
	case "array_hull":
		sonar.PassiveArray = acoustics.PassiveArrayHull
		a.waterfallFullRebuild = true
		a.passivePPIPending = true
	case "array_towed":
		sonar.PassiveArray = acoustics.PassiveArrayTowed
		a.waterfallFullRebuild = true
		a.passivePPIPending = true
	case "band_bb":
		sonar.ListenBand = acoustics.ListenBroadband
		a.waterfallFullRebuild = true
	case "band_hf":
		sonar.ListenBand = acoustics.ListenHF
		a.waterfallFullRebuild = true
	case "towed_deploy":
		if sonar.TowedDeployed() || (sonar.TowedInMotion() && sonar.TowedCableRate > 0) {
			return
		}
		sonar.StartDeploy()
		a.Audio.PlayClip(audio.ClipSonarDeployTowed, "Deploy towed array.")
	case "towed_stop":
		if !sonar.TowedInMotion() {
			return
		}
		sonar.StopTowed()
		a.Audio.PlayClip(audio.ClipSonarTowedHeld, fmt.Sprintf("Towed array held at %d percent.", int(sonar.TowedCablePct*100)))
	case "towed_retract":
		if sonar.TowedStowed() || (sonar.TowedInMotion() && sonar.TowedCableRate < 0) {
			return
		}
		sonar.StartRetract()
		a.Audio.PlayClip(audio.ClipSonarRetractTowed, "Retract towed array.")
	}
}

func (a *App) handleSonarArrayKeys(sonar *acoustics.SonarState, allowTowedMotion bool) {
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		if sonar.PassiveArray == acoustics.PassiveArrayHull {
			sonar.PassiveArray = acoustics.PassiveArrayTowed
		} else {
			sonar.PassiveArray = acoustics.PassiveArrayHull
		}
		a.waterfallFullRebuild = true
		a.passivePPIPending = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		if sonar.ListenBand == acoustics.ListenBroadband {
			sonar.ListenBand = acoustics.ListenHF
		} else {
			sonar.ListenBand = acoustics.ListenBroadband
		}
		a.waterfallFullRebuild = true
	}
	if !allowTowedMotion {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyU) {
		a.sonarButtonAction("towed_deploy", sonar)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyY) {
		a.sonarButtonAction("towed_retract", sonar)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		a.sonarButtonAction("towed_stop", sonar)
	}
}

func (a *App) drawArraySelector(screen *ebiten.Image, sonar *acoustics.SonarState, labelX, labelY int, buttons []sonarUIButton) {
	render.DrawText(screen, "PASSIVE ARRAY", labelX, labelY+12, render.ColorPlateLabel, true)
	mx, my := ebiten.CursorPosition()
	for _, b := range buttons {
		active := (b.ID == "array_hull" && sonar.PassiveArray == acoustics.PassiveArrayHull) ||
			(b.ID == "array_towed" && sonar.PassiveArray == acoustics.PassiveArrayTowed)
		hover := b.contains(mx, my)
		pressed := a.uiPressedID == b.ID
		if active {
			render.FillRect(screen, b.X-2, b.Y-2, b.W+4, b.H+4, render.ColorAmber)
		}
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	}
}

func (a *App) drawListenBandSelector(screen *ebiten.Image, sonar *acoustics.SonarState) {
	render.DrawText(screen, "LISTEN BAND", layout.PassiveBandLabelX, layout.PassiveBandLabelY+12, render.ColorPlateLabel, true)
	buttons := cachedPassiveBandButtons()
	mx, my := ebiten.CursorPosition()
	for _, b := range buttons {
		active := (b.ID == "band_bb" && sonar.ListenBand == acoustics.ListenBroadband) ||
			(b.ID == "band_hf" && sonar.ListenBand == acoustics.ListenHF)
		hover := b.contains(mx, my)
		pressed := a.uiPressedID == b.ID
		if active {
			render.FillRect(screen, b.X-2, b.Y-2, b.W+4, b.H+4, render.ColorAmber)
		}
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	}
}

func (a *App) drawTowedControls(screen *ebiten.Image, sonar *acoustics.SonarState) {
	render.DrawText(screen, "TOWED ARRAY", layout.PassiveTowedLabelX, layout.PassiveTowedLabelY+12, render.ColorPlateLabel, true)

	status := a.towedCableStatus(sonar)
	clr := render.ColorPhosphor
	if sonar.PassiveArray == acoustics.PassiveArrayTowed && sonar.TowedStowed() {
		clr = render.ColorWarn
	}
	render.DrawText(screen, status, layout.PassiveTowedStatusX+8, layout.PassiveTowedStatusY+18, clr, true)

	buttons := cachedPassiveTowedButtons()
	mx, my := ebiten.CursorPosition()
	for _, b := range buttons {
		disabled := false
		switch b.ID {
		case "towed_deploy":
			disabled = sonar.TowedDeployed() || (sonar.TowedInMotion() && sonar.TowedCableRate > 0)
		case "towed_stop":
			disabled = !sonar.TowedInMotion()
		case "towed_retract":
			disabled = sonar.TowedStowed() || (sonar.TowedInMotion() && sonar.TowedCableRate < 0)
		}
		hover := !disabled && b.contains(mx, my)
		pressed := a.uiPressedID == b.ID
		if disabled {
			render.FillRect(screen, b.X, b.Y, b.W, b.H, render.ColorPanelInset)
			tw := render.ButtonLabelWidth(b.Label)
			render.DrawButtonText(screen, b.Label, b.X+(b.W-tw)/2, b.Y+b.H/2+4, render.ColorPhosphorDim)
			continue
		}
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	}
}
