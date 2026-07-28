package ui

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
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

func layoutButtonRow(x, y, h, gap int, specs []buttonSpec) []sonarUIButton {
	out := make([]sonarUIButton, len(specs))
	xPos := x
	for i, s := range specs {
		w := render.ButtonWidth(s.label, 14)
		out[i] = sonarUIButton{ID: s.id, Label: s.label, Tooltip: s.tooltip, X: xPos, Y: y, W: w, H: h}
		xPos += w + gap
	}
	return out
}

func passiveArrayButtons(x, y int) []sonarUIButton {
	return layoutButtonRow(x, y+10, 34, 6, []buttonSpec{
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
	case "towed_deploy":
		if sonar.TowedDeployed() || (sonar.TowedInMotion() && sonar.TowedCableRate > 0) {
			return
		}
		sonar.StartDeploy()
		a.Audio.PlayClip(audio.ClipSonarPassiveOn, "Deploy towed array.")
	case "towed_stop":
		if !sonar.TowedInMotion() {
			return
		}
		sonar.StopTowed()
		a.Audio.PlayClip(audio.ClipSonarActiveStandby, fmt.Sprintf("Towed array held at %d percent.", int(sonar.TowedCablePct*100)))
	case "towed_retract":
		if sonar.TowedStowed() || (sonar.TowedInMotion() && sonar.TowedCableRate < 0) {
			return
		}
		sonar.StartRetract()
		a.Audio.PlayClip(audio.ClipSonarPassiveOff, "Retract towed array.")
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

func (a *App) drawArraySelector(screen *ebiten.Image, sonar *acoustics.SonarState, x, y int) {
	render.DrawText(screen, "PASSIVE ARRAY", x, y, render.ColorPhosphorDim, true)
	buttons := passiveArrayButtons(x, y)
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

func (a *App) drawTowedControls(screen *ebiten.Image, sonar *acoustics.SonarState, x, y int) {
	render.DrawText(screen, "TOWED ARRAY", x, y, render.ColorPhosphorDim, true)
	status := a.towedCableStatus(sonar)
	clr := render.ColorPhosphor
	if sonar.PassiveArray == acoustics.PassiveArrayTowed && sonar.TowedStowed() {
		clr = render.ColorWarn
	}
	render.DrawText(screen, status, x, y+18, clr, true)

	const barH = 12
	barY := y + 34
	barW := 0
	buttons := towedControlButtons(x, barY+22)
	for _, b := range buttons {
		barW += b.W + 4
	}
	if barW > 4 {
		barW -= 4
	}
	render.FillRect(screen, x, barY, barW, barH, render.ColorPanelInset)
	fillW := int(sonar.TowedCablePct * float64(barW))
	if fillW > 0 {
		render.FillRect(screen, x, barY, fillW, barH, render.ColorSonar)
	}

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
