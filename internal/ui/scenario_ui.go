package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/campaign"
	"github.com/ssn688/sim/internal/render"
)

const (
	scenarioPanelX = 40
	scenarioPanelY = 60
	scenarioPanelH = 740

	scenarioListW       = 320
	scenarioDetailW     = 480
	scenarioDetailPad   = 16
	scenarioCoverH      = 320 // 3:2 at 480px (demo cover 1536×1024)
	scenarioColumnGap   = 24
	scenarioTitleGap    = 40 // title + version block above cover
	scenarioCoverTextGap = 28 // clear font baselines into cover
)

func scenarioPanelW() int {
	return 16 + scenarioListW + scenarioColumnGap + scenarioDetailW + 16
}

func scenarioDetailX() int {
	return scenarioPanelX + 16 + scenarioListW + scenarioColumnGap
}

// scenarioBriefDescRect is the scrollable mission description area above loadout (or buttons in debrief).
func scenarioBriefDescRect(aboveLoadout bool) (x, y, w, h int) {
	x = scenarioDetailX()
	y = scenarioPanelY + 64 + 36
	w = scenarioDetailW - 10
	maxY := scenarioPanelY + scenarioPanelH - 16 - 40 - 8
	if aboveLoadout {
		maxY = scenarioLoadoutY() - 8
	}
	h = maxY - y
	if h < 40 {
		h = 40
	}
	return x, y, w, h
}

func drawWrappedText(screen *ebiten.Image, text string, x, y, maxW, lineH int) {
	_ = lineH
	render.DrawMarkdown(screen, text, x, y, maxW, 0, false)
}

func drawWrappedTextBox(screen *ebiten.Image, text string, x, y, maxW, lineH, maxY int, small bool) {
	_ = lineH
	render.DrawMarkdown(screen, text, x, y, maxW, maxY, small)
}

func drawScenarioCoverImage(screen *ebiten.Image, sc *campaign.ScenarioDef, m *campaign.MissionDef, x, y, w, h int) {
	if sc == nil {
		return
	}
	if data, key := campaign.MissionCover(sc, m); len(data) > 0 {
		render.DrawScenarioCoverBytes(screen, key, data, x, y, w, h)
		return
	}
	render.FillRect(screen, x, y, w, h, render.ColorPanelInset)
	render.DrawText(screen, "NO ART", x+w/2-28, y+h/2, render.ColorDim, true)
}

func scenarioVersionLine(sc *campaign.ScenarioDef) string {
	if sc == nil {
		return ""
	}
	return "v" + sc.Version.String()
}

func scenarioListCoverY() int {
	return scenarioPanelY + 48 + scenarioTitleGap
}

func scenarioListBackstoryRect() (x, y, w, h int) {
	x = scenarioDetailX()
	y = scenarioListCoverY() + scenarioCoverH + scenarioCoverTextGap
	w = scenarioDetailW - 10 // room for scrollbar
	btnY := scenarioPanelY + scenarioPanelH - 24 - 40
	h = btnY - y - 8
	if h < 40 {
		h = 40
	}
	return x, y, w, h
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

func (a *App) selectedScenarioPlayable() bool {
	sc := a.selectedScenarioDef()
	return sc != nil && sc.Compatible
}

func (a *App) ensureScenarioSelection() {
	defs := a.scenarioDefs()
	if len(defs) == 0 {
		return
	}
	if a.ScenarioListIndex < 0 || a.ScenarioListIndex >= len(defs) {
		a.ScenarioListIndex = 0
	}
	id := defs[a.ScenarioListIndex].ID
	if id != a.SelectedScenarioID {
		a.scenarioBackstoryScroll = 0
	}
	a.SelectedScenarioID = id
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

	if sc := a.selectedScenarioDef(); sc != nil {
		tx, ty, tw, th := scenarioListBackstoryRect()
		body := sc.Backstory
		if !sc.Compatible {
			body = "This scenario is incompatible with this game version.\n\n" + sc.IncompatibleReason
		}
		lines := render.ParseMarkdown(body, tw, false)
		vis := th / 18
		if vis < 1 {
			vis = 1
		}
		scrollContactTableWheel(mx, my, tx, ty, tw+10, th, len(lines), vis, &a.scenarioBackstoryScroll)
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}

	bx, by, bw, bh := a.scenarioListBackRect()
	if hitRect(mx, my, bx, by, bw, bh) {
		a.Mode = ModeMenu
		return nil
	}
	cx, cy, cw, ch := a.scenarioListContinueRect()
	if a.selectedScenarioPlayable() && hitRect(mx, my, cx, cy, cw, ch) {
		a.continueScenario()
		return nil
	}
	rx, ry, rw, rh := a.scenarioListRestartRect()
	if a.selectedScenarioPlayable() && hitRect(mx, my, rx, ry, rw, rh) {
		a.showConfirm(confirmRestartScenario, "RESTART SCENARIO",
			"All save files for this scenario will be permanently deleted. Start the campaign from the first mission?")
		return nil
	}
	sx, sy, sw, sh := a.scenarioListSelectRect()
	if a.selectedScenarioPlayable() && hitRect(mx, my, sx, sy, sw, sh) {
		a.briefDebrief = false
		a.briefMissionID = ""
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

	if sc := a.selectedScenarioDef(); sc != nil {
		aboveLoadout := sc.Compatible && !a.briefDebrief
		tx, ty, tw, th := scenarioBriefDescRect(aboveLoadout)
		body := ""
		cur := a.briefDisplayedMission(sc)
		if !sc.Compatible {
			body = "This scenario is incompatible with this game version.\n\n" + sc.IncompatibleReason
		} else if a.briefDebrief && cur != nil {
			prog := a.scenarioProgress(sc.ID)
			body = campaign.ComposeMissionDebrief(*cur, prog.DebriefOutcomes)
		} else if cur != nil {
			body = cur.Description
		}
		if body != "" {
			small := aboveLoadout
			lines := render.ParseMarkdown(body, tw, small)
			lineH := 18
			if small {
				lineH = 14
			}
			vis := th / lineH
			if vis < 1 {
				vis = 1
			}
			scrollContactTableWheel(mx, my, tx, ty, tw+10, th, len(lines), vis, &a.scenarioBriefDescScroll)
		}
	}

	if !a.briefDebrief && a.handleScenarioLoadoutInput(mx, my) {
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
	if a.briefDebrief {
		if a.scenarioBriefHasNext() {
			nx, ny, nw, nh := a.scenarioBriefPrimaryRect()
			if hitRect(mx, my, nx, ny, nw, nh) {
				a.acknowledgeDebriefAndSelectNext()
			}
		}
		return nil
	}
	stx, sty, stw, sth := a.scenarioBriefPrimaryRect()
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
	x = scenarioDetailX() + scenarioDetailW - w
	y = scenarioPanelY + scenarioPanelH - h - 24
	return x, y, w, h
}

func (a *App) scenarioListSelectRect() (x, y, w, h int) {
	w = render.ButtonWidth("OPEN SCENARIO", 20)
	h = 40
	x = scenarioDetailX()
	y = scenarioPanelY + scenarioPanelH - h - 24
	return x, y, w, h
}

func (a *App) scenarioListRestartRect() (x, y, w, h int) {
	w = render.ButtonWidth("RESTART SCENARIO", 20)
	h = 40
	ox, oy, ow, _ := a.scenarioListSelectRect()
	x = ox + ow + 16
	y = oy
	return x, y, w, h
}

func (a *App) scenarioListBackRect() (x, y, w, h int) {
	w = render.ButtonWidth("BACK", 20)
	h = 40
	x = scenarioPanelX + 16
	_, y, _, _ = a.scenarioListSelectRect()
	return x, y, w, h
}

func (a *App) scenarioBriefBackRect() (x, y, w, h int) {
	w = render.ButtonWidth("BACK", 20)
	h = 40
	x = scenarioPanelX + 16
	y = scenarioPanelY + scenarioPanelH - h - 24
	return x, y, w, h
}

func (a *App) scenarioBriefPrimaryRect() (x, y, w, h int) {
	w = render.ButtonWidth(a.scenarioBriefPrimaryLabel(), 24)
	h = 40
	x = scenarioPanelX + scenarioPanelW() - w - 24
	y = scenarioPanelY + scenarioPanelH - h - 24
	return x, y, w, h
}

func (a *App) scenarioBriefPrimaryLabel() string {
	if a.briefDebrief {
		return "NEXT MISSION"
	}
	sc := a.selectedScenarioDef()
	if sc != nil && a.scenarioProgress(sc.ID).ScenarioComplete(sc) {
		return "SCENARIO COMPLETE"
	}
	return "START MISSION"
}

func (a *App) scenarioBriefHasNext() bool {
	sc := a.selectedScenarioDef()
	if sc == nil || !a.briefDebrief {
		return false
	}
	return campaign.NextMission(sc, a.briefMissionID) != nil
}

func (a *App) briefDisplayedMission(sc *campaign.ScenarioDef) *campaign.MissionDef {
	if sc == nil {
		return nil
	}
	if m := campaign.FindMission(sc, a.briefMissionID); m != nil {
		return m
	}
	prog := a.scenarioProgress(sc.ID)
	return prog.CurrentMission(sc)
}

func (a *App) initScenarioBrief() {
	a.scenarioBriefDescScroll = 0
	sc := a.selectedScenarioDef()
	if sc == nil {
		return
	}
	if path, err := campaign.LatestSaveForScenario(sc.ID); err == nil {
		if meta, ok := campaign.ReadSaveCampaignMeta(path); ok {
			if meta.LoadoutMix > 0 {
				a.LoadoutMix = meta.LoadoutMix
			}
			a.ensureLoadoutTubes()
			if meta.DebriefPending {
				a.briefDebrief = true
				if meta.DebriefMission != "" {
					a.briefMissionID = meta.DebriefMission
				} else {
					a.briefMissionID = meta.MissionID
				}
			} else if a.briefMissionID == "" {
				prog := meta.ToProgress()
				if !prog.ScenarioComplete(sc) {
					if cur := prog.CurrentMission(sc); cur != nil {
						a.briefMissionID = cur.ID
					}
				} else if len(sc.Missions) > 0 {
					a.briefMissionID = sc.Missions[len(sc.Missions)-1].ID
				}
			}
			return
		}
	}
	if a.LoadoutMix == 0 {
		a.LoadoutMix = 0.25
	}
	a.ensureLoadoutTubes()
	if a.briefMissionID == "" && len(sc.Missions) > 0 {
		a.briefMissionID = sc.Missions[0].ID
	}
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
			prog.DebriefPending = meta.DebriefPending
			prog.DebriefMission = meta.DebriefMission
			prog.DebriefOutcomes = meta.DebriefOutcomes
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
		if !d.Compatible {
			clr = render.ColorDanger
		} else if selected {
			clr = render.ColorSonar
		}
		label := d.Title
		if !d.Compatible {
			label += "  — INCOMPATIBLE"
		}
		render.DrawText(screen, label, listX+8, y+24, clr, false)
	}

	if sc != nil {
		detailX := scenarioDetailX()
		titleX := detailX
		titleY := scenarioPanelY + 48
		render.DrawTextLarge(screen, sc.Title, titleX, titleY, render.ColorText)
		render.DrawText(screen, scenarioVersionLine(sc), titleX, titleY+24, render.ColorPhosphorDim, true)
		coverY := scenarioListCoverY()
		drawScenarioCoverImage(screen, sc, nil, detailX, coverY, scenarioDetailW, scenarioCoverH)

		tx, ty, tw, th := scenarioListBackstoryRect()
		body := sc.Backstory
		if !sc.Compatible {
			body = "This scenario is incompatible with this game version.\n\n" + sc.IncompatibleReason
		}
		lines := render.ParseMarkdown(body, tw, false)
		vis := th / 18
		if vis < 1 {
			vis = 1
		}
		a.scenarioBackstoryScroll = clampContactTableScroll(a.scenarioBackstoryScroll, len(lines), vis)
		start, end := contactTableWindow(len(lines), a.scenarioBackstoryScroll, vis)
		render.DrawMDLines(screen, lines, start, end, tx, ty, false)
		drawContactTableScrollbar(screen, tx+tw+4, ty, th, len(lines), vis, a.scenarioBackstoryScroll)
	}

	mx, my := ebiten.CursorPosition()
	cx, cy, cw, ch := a.scenarioListContinueRect()
	hasSave := false
	if sc != nil && sc.Compatible {
		_, err := campaign.LatestSaveForScenario(sc.ID)
		hasSave = err == nil
	}
	if hasSave {
		render.DrawBevelButton(screen, cx, cy, cw, ch, "CONTINUE SCENARIO", hitRect(mx, my, cx, cy, cw, ch), false)
	} else {
		render.DrawBevelButtonDisabled(screen, cx, cy, cw, ch, "CONTINUE SCENARIO")
	}
	rx, ry, rw, rh := a.scenarioListRestartRect()
	if sc != nil && sc.Compatible {
		render.DrawBevelButton(screen, rx, ry, rw, rh, "RESTART SCENARIO", hitRect(mx, my, rx, ry, rw, rh), false)
	} else {
		render.DrawBevelButtonDisabled(screen, rx, ry, rw, rh, "RESTART SCENARIO")
	}
	sx, sy, sw, sh := a.scenarioListSelectRect()
	if sc != nil && sc.Compatible {
		render.DrawBevelButton(screen, sx, sy, sw, sh, "OPEN SCENARIO", hitRect(mx, my, sx, sy, sw, sh), false)
	} else {
		render.DrawBevelButtonDisabled(screen, sx, sy, sw, sh, "OPEN SCENARIO")
	}
	bx, by, bw, bh := a.scenarioListBackRect()
	render.DrawBevelButton(screen, bx, by, bw, bh, "BACK", hitRect(mx, my, bx, by, bw, bh), false)

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
	complete := prog.ScenarioComplete(sc)
	cur := a.briefDisplayedMission(sc)
	render.DrawTextLarge(screen, sc.Title, scenarioPanelX+20, scenarioPanelY+28, render.ColorText)
	render.DrawText(screen, scenarioVersionLine(sc), scenarioPanelX+20, scenarioPanelY+52, render.ColorPhosphorDim, true)

	listX := scenarioPanelX + 16
	listY := scenarioPanelY + 64
	for i, m := range sc.Missions {
		y := listY + i*32
		done := prog.CompletedMissions[m.ID]
		selected := cur != nil && m.ID == cur.ID
		label := m.Title
		if done {
			label += "  — DONE"
		} else if !complete && selected {
			label += "  — CURRENT"
		}
		if selected {
			render.FillRect(screen, listX, y, scenarioListW, 28, render.ColorPanel)
		}
		clr := render.ColorDim
		if done {
			clr = render.ColorMuted
		} else if selected {
			clr = render.ColorSonar
		}
		render.DrawText(screen, label, listX+8, y+20, clr, false)
	}

	if cur != nil {
		textX := scenarioDetailX()
		descY := scenarioPanelY + 64
		render.DrawTextLarge(screen, cur.Title, textX, descY, render.ColorText)

		aboveLoadout := sc.Compatible && !a.briefDebrief
		tx, ty, tw, th := scenarioBriefDescRect(aboveLoadout)
		body := cur.Description
		small := true
		if !sc.Compatible {
			body = "This scenario is incompatible with this game version.\n\n" + sc.IncompatibleReason
			small = false
		} else if a.briefDebrief {
			body = campaign.ComposeMissionDebrief(*cur, prog.DebriefOutcomes)
			small = false
		}
		lines := render.ParseMarkdown(body, tw, small)
		lineH := 18
		if small {
			lineH = 14
		}
		vis := th / lineH
		if vis < 1 {
			vis = 1
		}
		a.scenarioBriefDescScroll = clampContactTableScroll(a.scenarioBriefDescScroll, len(lines), vis)
		start, end := contactTableWindow(len(lines), a.scenarioBriefDescScroll, vis)
		render.DrawMDLines(screen, lines, start, end, tx, ty, small)
		drawContactTableScrollbar(screen, tx+tw+4, ty, th, len(lines), vis, a.scenarioBriefDescScroll)

		if aboveLoadout {
			a.drawScenarioLoadout(screen)
		}
	}

	mx, my := ebiten.CursorPosition()
	bx, by, bw, bh := a.scenarioBriefBackRect()
	render.DrawBevelButton(screen, bx, by, bw, bh, "BACK", hitRect(mx, my, bx, by, bw, bh), false)
	stx, sty, stw, sth := a.scenarioBriefPrimaryRect()
	if a.briefDebrief {
		if a.scenarioBriefHasNext() {
			render.DrawBevelButton(screen, stx, sty, stw, sth, "NEXT MISSION", hitRect(mx, my, stx, sty, stw, sth), false)
		}
	} else if !complete && sc.Compatible {
		render.DrawBevelButton(screen, stx, sty, stw, sth, "START MISSION", hitRect(mx, my, stx, sty, stw, sth), false)
	} else if !sc.Compatible {
		render.DrawBevelButtonDisabled(screen, stx, sty, stw, sth, "START MISSION")
	} else {
		render.DrawBevelButtonDisabled(screen, stx, sty, stw, sth, "SCENARIO COMPLETE")
	}
	a.drawConfirmDialog(screen)
}
