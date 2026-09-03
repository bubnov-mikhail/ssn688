package ui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const (
	dcPanelX = 20
	dcPanelY = 50
	dcPanelW = 1260
	dcPanelH = 700
	dcRowH   = 32
)

func (a *App) updateDamageUI() {
	if a.Engine == nil || a.Engine.Scenario.Player == nil {
		return
	}
	player := a.Engine.Scenario.Player
	player.EnsureDamage()
	mx, my := ebiten.CursorPosition()

	buttons := a.damageButtons()
	a.updateButtonTooltips(buttons, mx, my)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for _, b := range buttons {
			if b.contains(mx, my) {
				a.handleDamageButton(b.ID)
				return
			}
		}
	}
}

func (a *App) damageButtons() []uiButton {
	player := a.Engine.Scenario.Player
	d := &player.Damage
	btns := make([]uiButton, 0, world.SysCount)
	tableY := dcPanelY + 90
	row := 0
	for sys := 0; sys < world.SysCount; sys++ {
		if player.Kind != world.KindSubmarine && (sys == world.SysTowed || sys == world.SysDepth || sys == world.SysESM || sys == world.SysCOMM || sys == world.SysPeriscope) {
			continue
		}
		label := a.L(i18n.UIRepair)
		tip := a.L(i18n.UITipRepair)
		if d.Repairing == sys {
			label = a.L(i18n.UIRepairing)
			tip = a.L(i18n.UITipRepairing)
		} else if !d.Repairable(sys) {
			if d.EffOf(sys) >= 100 {
				label = a.L(i18n.UIOK)
				tip = "System nominal"
			} else {
				label = a.L(i18n.UINA)
				tip = "Destroyed beyond repair"
			}
		}
		y := tableY + row*dcRowH
		row++
		btns = append(btns, uiButton{
			ID:      fmt.Sprintf("dc_repair_%d", sys),
			Label:   label,
			Tooltip: tip,
			X:       dcPanelX + dcPanelW - 160,
			Y:       y,
			W:       120,
			H:       28,
		})
	}
	return btns
}

func (a *App) handleDamageButton(id string) {
	var sys int
	if _, err := fmt.Sscanf(id, "dc_repair_%d", &sys); err != nil {
		return
	}
	player := a.Engine.Scenario.Player
	ok, reason := player.Damage.StartRepair(sys)
	if !ok {
		a.StatusMessage = reason
		return
	}
	a.Statusf(i18n.StatusRepairing, i18n.LocalizeSystemName(world.SystemName(sys), a.Lang()))
}

func (a *App) drawDamage(screen *ebiten.Image) {
	if a.Engine == nil || a.Engine.Scenario.Player == nil {
		return
	}
	player := a.Engine.Scenario.Player
	player.EnsureDamage()
	d := &player.Damage
	lang := a.Lang()

	render.DrawConsolePanel(screen, dcPanelX, dcPanelY, dcPanelW, dcPanelH)
	render.DrawScreenTitle(screen, a.L(i18n.UITitleDamage), dcPanelX+20, dcPanelY+28)
	render.DrawText(screen, a.L(i18n.UIDamageHint), dcPanelX+280, dcPanelY+26, render.ColorPlateLabel, true)

	headers := []string{a.L(i18n.UISystem), a.L(i18n.UIStatusCol), a.L(i18n.UIEfficiency), ""}
	xs := []int{dcPanelX + 40, dcPanelX + 280, dcPanelX + 480, dcPanelX + dcPanelW - 160}
	hy := dcPanelY + 60
	for i, h := range headers {
		render.DrawText(screen, h, xs[i], hy, render.ColorPhosphorDim, true)
	}
	render.DrawLine(screen, float64(dcPanelX+30), float64(hy+8), float64(dcPanelX+dcPanelW-30), float64(hy+8), render.ColorBevelLight)

	tableY := dcPanelY + 90
	row := 0
	for sys := 0; sys < world.SysCount; sys++ {
		if player.Kind != world.KindSubmarine && (sys == world.SysTowed || sys == world.SysDepth || sys == world.SysESM || sys == world.SysCOMM || sys == world.SysPeriscope) {
			continue
		}
		y := tableY + row*dcRowH
		row++
		eff := d.EffOf(sys)
		status := i18n.LocalizeSystemStatus(world.SystemStatusLabel(d, sys), lang)
		clr := color.RGBA{0, 200, 120, 255}
		switch {
		case eff <= world.RepairThresholdPct:
			clr = render.ColorDanger
		case eff < 100:
			clr = render.ColorAmber
		}
		if d.Repairing == sys {
			status = a.L(i18n.UIRepairing)
			clr = render.ColorHighlight
		}
		render.DrawText(screen, i18n.LocalizeSystemName(world.SystemName(sys), lang), xs[0], y+18, render.ColorPhosphor, true)
		render.DrawText(screen, status, xs[1], y+18, clr, true)
		render.DrawText(screen, fmt.Sprintf("%.0f%%", eff), xs[2], y+18, clr, true)

		barX, barY, barW, barH := xs[2]+70, y+8, 200, 14
		render.FillRect(screen, barX, barY, barW, barH, color.RGBA{20, 30, 28, 255})
		fill := int(float64(barW) * eff / 100)
		if fill > 0 {
			render.FillRect(screen, barX, barY, fill, barH, clr)
		}
	}

	okLbl := a.L(i18n.UIOK)
	naLbl := a.L(i18n.UINA)
	repairingLbl := a.L(i18n.UIRepairing)
	for _, b := range a.damageButtons() {
		disabled := b.Label == naLbl || b.Label == okLbl || b.Label == repairingLbl
		hover := a.uiHoverID == b.ID && !disabled
		pressed := a.uiPressedID == b.ID && !disabled
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	}

	fy := dcPanelY + dcPanelH - 70
	if d.Destroyed(world.SysDepth) && d.DepthRunawayFPM != 0 {
		render.DrawText(screen, a.L(i18n.UIDepthLost),
			dcPanelX+40, fy, render.ColorDanger, true)
		fy += 18
	}
	if d.SteeringJammed || d.Destroyed(world.SysSteering) {
		render.DrawText(screen, a.L(i18n.UIRudderJam),
			dcPanelX+40, fy, render.ColorDanger, true)
	}

	if a.uiTooltip != "" {
		mx, my := ebiten.CursorPosition()
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
}

// hullArrayDamaged reports player hull-passive loss for UI overlays.
func (a *App) hullArrayDamaged() bool {
	if a.Engine == nil || a.Engine.Scenario.Player == nil {
		return false
	}
	p := a.Engine.Scenario.Player
	p.EnsureDamage()
	return p.Damage.Destroyed(world.SysPassiveHull)
}

func (a *App) activeSonarDamaged() bool {
	if a.Engine == nil || a.Engine.Scenario.Player == nil {
		return false
	}
	p := a.Engine.Scenario.Player
	p.EnsureDamage()
	return p.Damage.Destroyed(world.SysActive)
}
