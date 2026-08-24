package ui

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/campaign"
	"github.com/ssn688/sim/internal/render"
)

const (
	scenarioPanelX = 40
	scenarioPanelY = 80
	scenarioPanelH = 600

	scenarioListW     = 320
	scenarioDetailW   = 480
	scenarioDetailPad = 16
	scenarioCoverH    = 270 // 16:9 at 480px
	scenarioColumnGap = 24
)

func scenarioPanelW() int {
	return 16 + scenarioListW + scenarioColumnGap + scenarioDetailW + 16
}

func scenarioDetailX() int {
	return scenarioPanelX + 16 + scenarioListW + scenarioColumnGap
}

func scenarioDetailContentRect() (x, w int) {
	return scenarioDetailX() + 8, scenarioDetailW - 16
}

func drawWrappedText(screen *ebiten.Image, text string, x, y, maxW, lineH int) {
	drawWrappedTextBox(screen, text, x, y, maxW, lineH, 0, false)
}

func wrapLineWidth(line string, small bool) int {
	if small {
		return render.SmallLabelWidth(line)
	}
	return render.LabelWidth(line)
}

func drawWrappedTextBox(screen *ebiten.Image, text string, x, y, maxW, lineH, maxY int, small bool) {
	yy := y
	for _, para := range strings.Split(text, "\n") {
		words := strings.Fields(para)
		line := ""
		flush := func() {
			if line == "" {
				return
			}
			if maxY > 0 && yy+lineH > maxY {
				return
			}
			render.DrawText(screen, line, x, yy, render.ColorDim, small)
			yy += lineH
			line = ""
		}
		for _, w := range words {
			candidate := w
			if line != "" {
				candidate = line + " " + w
			}
			if wrapLineWidth(candidate, small) > maxW {
				flush()
				line = w
			} else {
				line = candidate
			}
		}
		flush()
		if maxY > 0 && yy > maxY {
			break
		}
		if maxY == 0 || yy+lineH/2 <= maxY {
			yy += lineH / 2
		}
	}
}

func (a *App) scenarioDefs() []campaign.ScenarioDef {
	return campaign.AllScenarios()
}

func (a *App) selectedScenarioDef() *campaign.ScenarioDef {
	if a.SelectedScenarioID == "" {
		return nil
	}
	return campaign.ScenarioByID(a.SelectedScenarioID)
}

func (a *App) ensureScenarioSelection() {
	defs := a.scenarioDefs()
	if len(defs) == 0 {
		return
	}
	if a.ScenarioListIndex < 0 || a.ScenarioListIndex >= len(defs) {
		a.ScenarioListIndex = 0
	}
	a.SelectedScenarioID = defs[a.ScenarioListIndex].ID
}

func (a *App) updateScenarioList() error {
	if a.confirmActive() {
		a.updateConfirmDialog()
		return nil
	}
	a.ensureScenarioSelection()
	defs := a.scenarioDefs()
	mx, my := ebiten.CursorPosition()

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.Mode = ModeMenu
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && len(defs) > 0 {
		a.ScenarioListIndex = (a.ScenarioListIndex + 1) % len(defs)
		a.ensureScenarioSelection()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && len(defs) > 0 {
		a.ScenarioListIndex = (a.ScenarioListIndex + len(defs) - 1) % len(defs)
		a.ensureScenarioSelection()
	}

	listX, listY, listW, rowH := scenarioListRect()
	for i := range defs {
		y := listY + i*rowH
		if mx >= listX && mx < listX+listW && my >= y && my < y+rowH {
			a.ScenarioListIndex = i
			a.ensureScenarioSelection()
		}
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}

	cx, cy, cw, ch := a.scenarioListContinueRect()
	if hitRect(mx, my, cx, cy, cw, ch) {
		a.continueScenario()
		return nil
	}
	rx, ry, rw, rh := a.scenarioListRestartRect()
	if hitRect(mx, my, rx, ry, rw, rh) {
		a.showConfirm(confirmRestartScenario, "RESTART SCENARIO",
			"All save files for this scenario will be permanently deleted. Start the campaign from the first mission?")
		return nil
	}
	sx, sy, sw, sh := a.scenarioListSelectRect()
	if hitRect(mx, my, sx, sy, sw, sh) {
		a.Mode = ModeScenarioBrief
		a.initScenarioBrief()
		return nil
	}
	return nil
}

func (a *App) updateScenarioBrief() error {
	if a.confirmActive() {
		a.updateConfirmDialog()
		return nil
	}
	a.ensureScenarioSelection()
	mx, my := ebiten.CursorPosition()

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.Mode = ModeScenarioList
		return nil
	}

	if a.handleScenarioLoadoutInput(mx, my) {
		return nil
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}

	bx, by, bw, bh := a.scenarioBriefBackRect()
	if hitRect(mx, my, bx, by, bw, bh) {
		a.Mode = ModeScenarioList
		return nil
	}
	stx, sty, stw, sth := a.scenarioBriefStartRect()
	if hitRect(mx, my, stx, sty, stw, sth) {
		a.startSelectedMission()
		return nil
	}
	return nil
}

func hitRect(mx, my, x, y, w, h int) bool {
	return mx >= x && mx < x+w && my >= y && my < y+h
}

func scenarioListRect() (x, y, w, rowH int) {
	return scenarioPanelX + 16, scenarioPanelY + 56, scenarioListW, 36
}

func (a *App) scenarioListContinueRect() (x, y, w, h int) {
	w = render.ButtonWidth("CONTINUE SCENARIO", 20)
	h = 40
	x = scenarioPanelX + scenarioPanelW() - w - 24
	y = scenarioPanelY + scenarioPanelH - h - 24
	return x, y, w, h
}

func (a *App) scenarioListRestartRect() (x, y, w, h int) {
	w = render.ButtonWidth("RESTART SCENARIO", 20)
	h = 40
	cx, cy, _, _ := a.scenarioListContinueRect()
	x = cx - w - 16
	y = cy
	return x, y, w, h
}

func (a *App) scenarioListSelectRect() (x, y, w, h int) {
	w = render.ButtonWidth("OPEN SCENARIO", 20)
	h = 40
	cx, cy, _, _ := a.scenarioListContinueRect()
	x = cx - 2*w - 32
	y = cy
	return x, y, w, h
}

func (a *App) scenarioBriefBackRect() (x, y, w, h int) {
	w = render.ButtonWidth("BACK", 20)
	h = 36
	x = scenarioPanelX + 16
	y = scenarioPanelY + scenarioPanelH - h - 16
	return x, y, w, h
}

func (a *App) scenarioBriefStartRect() (x, y, w, h int) {
	w = render.ButtonWidth("START MISSION", 24)
	h = 40
	x = scenarioPanelX + scenarioPanelW() - w - 24
	y = scenarioPanelY + scenarioPanelH - h - 16
	return x, y, w, h
}

func (a *App) initScenarioBrief() {
	sc := a.selectedScenarioDef()
	if sc == nil {
		return
	}
	if path, err := campaign.LatestSaveForScenario(sc.ID); err == nil {
		if meta, ok := campaign.ReadSaveCampaignMeta(path); ok {
			a.LoadoutMix = meta.LoadoutMix
			a.ensureLoadoutTubes()
			return
		}
	}
	if a.LoadoutMix == 0 {
		a.LoadoutMix = 0.25
	}
	a.ensureLoadoutTubes()
}

func (a *App) scenarioProgress(id campaign.ScenarioID) campaign.Progress {
	prog := campaign.Progress{
		ScenarioID:        id,
		CompletedMissions: map[campaign.MissionID]bool{},
		Vars:              map[string]string{},
		LoadoutMix:        a.LoadoutMix,
	}
	if path, err := campaign.LatestSaveForScenario(id); err == nil {
		meta, ok := campaign.ReadSaveCampaignMeta(path)
		if ok {
			prog.CompletedMissions = meta.Completed
			prog.Vars = meta.Vars
			if meta.LoadoutMix > 0 {
				prog.LoadoutMix = meta.LoadoutMix
			}
		}
	}
	return prog
}

func (a *App) drawScenarioList(screen *ebiten.Image) {
	render.DrawMenuBackground(screen)
	panelW := scenarioPanelW()
	render.DrawConsolePanel(screen, scenarioPanelX, scenarioPanelY, panelW, scenarioPanelH)
	render.DrawTextLarge(screen, "SELECT SCENARIO", scenarioPanelX+20, scenarioPanelY+28, render.ColorText)

	defs := a.scenarioDefs()
	a.ensureScenarioSelection()
	sc := a.selectedScenarioDef()
	listX, listY, listW, rowH := scenarioListRect()
	for i, d := range defs {
		y := listY + i*rowH
		selected := i == a.ScenarioListIndex
		if selected {
			render.FillRect(screen, listX, y, listW, rowH, render.ColorPanel)
		}
		clr := render.ColorDim
		if selected {
			clr = render.ColorSonar
		}
		render.DrawText(screen, d.Title, listX+8, y+24, clr, false)
	}

	if sc != nil {
		detailX := scenarioDetailX()
		coverY := scenarioPanelY + 56
		render.DrawScenarioCover(screen, sc.CoverFile, detailX, coverY, scenarioDetailW, scenarioCoverH)
		titleX := detailX + scenarioDetailPad
		render.DrawTextLarge(screen, sc.Title, titleX, coverY+scenarioCoverH+16, render.ColorText)
		textW := scenarioDetailW - 2*scenarioDetailPad
		drawWrappedText(screen, sc.Backstory, titleX, coverY+scenarioCoverH+48, textW, 16)
	}

	mx, my := ebiten.CursorPosition()
	cx, cy, cw, ch := a.scenarioListContinueRect()
	hasSave := false
	if sc != nil {
		_, err := campaign.LatestSaveForScenario(sc.ID)
		hasSave = err == nil
	}
	if hasSave {
		render.DrawBevelButton(screen, cx, cy, cw, ch, "CONTINUE SCENARIO", hitRect(mx, my, cx, cy, cw, ch), false)
	} else {
		render.DrawBevelButtonDisabled(screen, cx, cy, cw, ch, "CONTINUE SCENARIO")
	}
	rx, ry, rw, rh := a.scenarioListRestartRect()
	render.DrawBevelButton(screen, rx, ry, rw, rh, "RESTART SCENARIO", hitRect(mx, my, rx, ry, rw, rh), false)
	sx, sy, sw, sh := a.scenarioListSelectRect()
	render.DrawBevelButton(screen, sx, sy, sw, sh, "OPEN SCENARIO", hitRect(mx, my, sx, sy, sw, sh), false)

	if a.StatusMessage != "" {
		render.DrawText(screen, a.StatusMessage, scenarioPanelX+20, scenarioPanelY+scenarioPanelH+12, render.ColorWarn, false)
	}
	a.drawConfirmDialog(screen)
}

func (a *App) drawScenarioBrief(screen *ebiten.Image) {
	render.DrawMenuBackground(screen)
	panelW := scenarioPanelW()
	render.DrawConsolePanel(screen, scenarioPanelX, scenarioPanelY, panelW, scenarioPanelH)
	sc := a.selectedScenarioDef()
	if sc == nil {
		return
	}
	prog := a.scenarioProgress(sc.ID)
	curIdx := prog.CurrentMissionIndex(sc)
	render.DrawTextLarge(screen, sc.Title, scenarioPanelX+20, scenarioPanelY+28, render.ColorText)

	listX := scenarioPanelX + 16
	listY := scenarioPanelY + 64
	for i, m := range sc.Missions {
		y := listY + i*32
		label := m.Title
		if prog.CompletedMissions[m.ID] {
			label += "  — DONE"
		} else if i == curIdx {
			label += "  — CURRENT"
		}
		clr := render.ColorDim
		if i == curIdx {
			clr = render.ColorSonar
			render.FillRect(screen, listX, y, scenarioListW, 28, render.ColorPanel)
		}
		render.DrawText(screen, label, listX+8, y+20, clr, false)
	}

	if cur := prog.CurrentMission(sc); cur != nil {
		contentX, contentW := scenarioDetailContentRect()
		textX := contentX + 8
		textW := contentW - 16
		descY := scenarioPanelY + 64
		render.DrawTextLarge(screen, cur.Title, textX, descY, render.ColorText)
		descMaxY := scenarioLoadoutY() - 6
		drawWrappedTextBox(screen, cur.Description, textX, descY+36, textW, 13, descMaxY, true)
		a.drawScenarioLoadout(screen)
	}

	mx, my := ebiten.CursorPosition()
	bx, by, bw, bh := a.scenarioBriefBackRect()
	render.DrawBevelButton(screen, bx, by, bw, bh, "BACK", hitRect(mx, my, bx, by, bw, bh), false)
	stx, sty, stw, sth := a.scenarioBriefStartRect()
	canStart := prog.CurrentMission(sc) != nil && !prog.ScenarioComplete(sc)
	if canStart {
		render.DrawBevelButton(screen, stx, sty, stw, sth, "START MISSION", hitRect(mx, my, stx, sty, stw, sth), false)
	} else {
		render.DrawBevelButtonDisabled(screen, stx, sty, stw, sth, "SCENARIO COMPLETE")
	}
	a.drawConfirmDialog(screen)
}
