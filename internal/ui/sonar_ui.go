package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/i18n"
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

func passiveArrayButtons(x, y int, lang string) []sonarUIButton {
	L := func(t i18n.TranslatedText) string { return t.GetText(lang) }
	return layoutButtonRow(x, y, 34, 6, []buttonSpec{
		{"array_hull", L(i18n.UIHull), L(i18n.UITipHullArray)},
		{"array_towed", L(i18n.UITowed), L(i18n.UITipTowedArray)},
	})
}

func towedControlButtons(x, y int, lang string) []sonarUIButton {
	L := func(t i18n.TranslatedText) string { return t.GetText(lang) }
	return layoutButtonRow(x, y, 38, 4, []buttonSpec{
		{"towed_deploy", L(i18n.UIDeploy), L(i18n.UITipDeploy)},
		{"towed_stop", L(i18n.UIStop), "Stop cable motion at present length"},
		{"towed_retract", L(i18n.UIRetract), L(i18n.UITipRetract)},
	})
}

var cachedSonarUIButtons struct {
	mu       sync.Mutex
	lang     string
	passive  []sonarUIButton
	towed    []sonarUIButton
	spectrum []sonarUIButton
	band     []sonarUIButton
}

func (a *App) ensureSonarUIButtons() {
	lang := a.Lang()
	cachedSonarUIButtons.mu.Lock()
	defer cachedSonarUIButtons.mu.Unlock()
	if cachedSonarUIButtons.lang == lang && cachedSonarUIButtons.passive != nil {
		return
	}
	L := func(t i18n.TranslatedText) string { return t.GetText(lang) }
	cachedSonarUIButtons.passive = passiveArrayButtons(layout.PassiveArrayPlateX, layout.PassiveArrayButtonsY, lang)
	cachedSonarUIButtons.towed = towedControlButtons(layout.PassiveTowedPlateX, layout.PassiveTowedButtonsY, lang)
	cachedSonarUIButtons.spectrum = passiveArrayButtons(spectrumArrayLabelX, spectrumArrayLabelY+22, lang)
	cachedSonarUIButtons.band = layoutButtonRow(layout.PassiveArrayPlateX, layout.PassiveBandButtonsY, 34, 6, []buttonSpec{
		{"band_bb", L(i18n.UIShipBand), L(i18n.UITipShipBand)},
		{"band_hf", L(i18n.UITorpBand), L(i18n.UITipTorpBand)},
	})
	cachedSonarUIButtons.lang = lang
}

func (a *App) cachedPassiveArrayButtons() []sonarUIButton {
	a.ensureSonarUIButtons()
	return cachedSonarUIButtons.passive
}

func (a *App) cachedPassiveTowedButtons() []sonarUIButton {
	a.ensureSonarUIButtons()
	return cachedSonarUIButtons.towed
}

func (a *App) cachedSpectrumArrayButtons() []sonarUIButton {
	a.ensureSonarUIButtons()
	return cachedSonarUIButtons.spectrum
}

func (a *App) cachedPassiveBandButtons() []sonarUIButton {
	a.ensureSonarUIButtons()
	return cachedSonarUIButtons.band
}

func (a *App) sonarArrayLabel(sonar *acoustics.SonarState) string {
	if sonar.PassiveArray == acoustics.PassiveArrayTowed {
		return a.L(i18n.UITowed)
	}
	return a.L(i18n.UIHull)
}

func (a *App) towedCableStatus(sonar *acoustics.SonarState) string {
	if sonar.TowedDamaged {
		return a.L(i18n.UIDamagedNoData)
	}
	player := a.Engine.Scenario.Player
	if player != nil && sonar.TowedCablePct >= 0.20 {
		if player.SpeedKts >= acoustics.TowedWarnSpeedKts(sonar.TowedCablePct) {
			return fmt.Sprintf("%s %.0f%%", a.L(i18n.UICableStress), sonar.TowedCablePct*100)
		}
	}
	switch {
	case sonar.TowedInMotion() && sonar.TowedCableRate > 0:
		return fmt.Sprintf("%s %.0f%%", a.L(i18n.UIDeploying), sonar.TowedCablePct*100)
	case sonar.TowedInMotion() && sonar.TowedCableRate < 0:
		return fmt.Sprintf("%s %.0f%%", a.L(i18n.UIRetracting), sonar.TowedCablePct*100)
	case !sonar.TowedDeployed() && !sonar.TowedStowed():
		return fmt.Sprintf("%s %.0f%%", a.L(i18n.UIHeld), sonar.TowedCablePct*100)
	case sonar.TowedDeployed():
		return a.L(i18n.UIDeployed)
	case sonar.TowedStowed():
		return a.L(i18n.UIStowed)
	default:
		return fmt.Sprintf("%s %.0f%%", a.L(i18n.UITowedArray), sonar.TowedCablePct*100)
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
		if sonar.TowedDamaged {
			a.Status(i18n.StatusTowedDamaged)
			return
		}
		if sonar.TowedDeployed() || (sonar.TowedInMotion() && sonar.TowedCableRate > 0) {
			return
		}
		sonar.StartDeploy()
		a.Audio.PlayClip(audio.ClipSonarDeployTowed, i18n.LocalizeRuntimeMessage("Deploy towed array.", a.Lang()))
	case "towed_stop":
		if !sonar.TowedInMotion() {
			return
		}
		sonar.StopTowed()
		a.Audio.PlayClip(audio.ClipSonarTowedHeld, a.Lf(i18n.StatusVoiceTowedHeld, int(sonar.TowedCablePct*100)))
	case "towed_retract":
		if sonar.TowedStowed() || (sonar.TowedInMotion() && sonar.TowedCableRate < 0) {
			return
		}
		sonar.StartRetract()
		a.Audio.PlayClip(audio.ClipSonarRetractTowed, i18n.LocalizeRuntimeMessage("Retract towed array.", a.Lang()))
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
	if inpututil.IsKeyJustPressed(ebiten.KeyH) && !a.debugPeriHitStealsH() {
		a.sonarButtonAction("towed_stop", sonar)
	}
}

func (a *App) drawArraySelector(screen *ebiten.Image, sonar *acoustics.SonarState, labelX, labelY int, buttons []sonarUIButton) {
	render.DrawText(screen, a.L(i18n.UIPassiveArray), labelX, labelY+12, render.ColorPlateLabel, true)
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
	render.DrawText(screen, a.L(i18n.UIListenBand), layout.PassiveBandLabelX, layout.PassiveBandLabelY+12, render.ColorPlateLabel, true)
	buttons := a.cachedPassiveBandButtons()
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
	render.DrawText(screen, a.L(i18n.UITowedArray), layout.PassiveTowedLabelX, layout.PassiveTowedLabelY+12, render.ColorPlateLabel, true)

	status := a.towedCableStatus(sonar)
	clr := render.ColorPhosphor
	if sonar.TowedDamaged {
		clr = render.ColorWarn
	} else if sonar.PassiveArray == acoustics.PassiveArrayTowed && sonar.TowedStowed() {
		clr = render.ColorWarn
	} else if player := a.Engine.Scenario.Player; player != nil && sonar.TowedCablePct >= 0.20 &&
		player.SpeedKts >= acoustics.TowedWarnSpeedKts(sonar.TowedCablePct) {
		clr = render.ColorAmber
	}
	render.DrawText(screen, status, layout.PassiveTowedStatusX+8, layout.PassiveTowedStatusY+18, clr, true)

	buttons := a.cachedPassiveTowedButtons()
	mx, my := ebiten.CursorPosition()
	for _, b := range buttons {
		disabled := sonar.TowedDamaged
		switch b.ID {
		case "towed_deploy":
			disabled = disabled || sonar.TowedDeployed() || (sonar.TowedInMotion() && sonar.TowedCableRate > 0)
		case "towed_stop":
			disabled = disabled || !sonar.TowedInMotion()
		case "towed_retract":
			disabled = disabled || sonar.TowedStowed() || (sonar.TowedInMotion() && sonar.TowedCableRate < 0)
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
