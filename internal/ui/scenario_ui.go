package ui

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/render"
)

const (
	scenarioPanelX = 40
	scenarioPanelY = 60
	scenarioPanelH = 740

	scenarioListW        = 320
	scenarioDetailW      = 480
	scenarioDetailPad    = 16
	scenarioCoverH       = 320 // 3:2 at 480px (demo cover 1536×1024)
	scenarioColumnGap    = 24
	scenarioTitleGap     = 40 // title + version block above cover
	scenarioCoverTextGap = 28 // clear font baselines into cover
	scenarioBriefMapAspectW = 880
	scenarioBriefMapAspectH = 1040
)

func scenarioPanelW() int {
	return 16 + scenarioListW + scenarioColumnGap + scenarioDetailW + 16
}

func scenarioDetailX() int {
	return scenarioPanelX + 16 + scenarioListW + scenarioColumnGap
}

func scenarioBriefPanelW(mapW int) int {
	if mapW <= 0 {
		return scenarioPanelW()
	}
	return scenarioPanelW() + scenarioColumnGap + mapW
}

func scenarioBriefMapSize(descH int) (w, h int) {
	if descH < 40 {
		descH = 40
	}
	h = descH
	w = h * scenarioBriefMapAspectW / scenarioBriefMapAspectH
	return w, h
}

func scenarioBriefShowsMap(cur *campaign.MissionDef, debrief bool) bool {
	if cur == nil || debrief {
		return false
	}
	_, key := campaign.MissionBriefMap(cur)
	return key != ""
}

func scenarioBriefMapRect(aboveLoadout bool) (x, y, w, h int) {
	_, ty, _, th := scenarioBriefDescRect(aboveLoadout)
	w, h = scenarioBriefMapSize(th)
	x = scenarioDetailX() + scenarioDetailW + scenarioColumnGap
	y = ty
	return x, y, w, h
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

func (a *App) disposeScenarioPanelCaches() {
	if a.scenarioListDetailImg != nil {
		a.scenarioListDetailImg.Dispose()
		a.scenarioListDetailImg = nil
	}
	a.scenarioListDetailKey = ""
	if a.scenarioBriefDetailImg != nil {
		a.scenarioBriefDetailImg.Dispose()
		a.scenarioBriefDetailImg = nil
	}
	a.scenarioBriefDetailKey = ""
	a.briefMapCacheKey = ""
}

func scenarioListDetailCacheSize() (w, h int) {
	titleY := scenarioPanelY + 48
	_, by, _, bh := scenarioListBackstoryRect()
	w = scenarioDetailW
	h = by + bh - titleY
	if h < 1 {
		h = 1
	}
	return w, h
}

func (a *App) scenarioListDetailCacheKey(sc *campaign.ScenarioDef, tw int) string {
	if sc == nil {
		return ""
	}
	body := sc.Backstory.GetText(a.Lang())
	if !sc.Compatible {
		body = a.L(i18n.UIScenarioIncompatBody) + "\n\n" + sc.IncompatibleReason
	}
	return string(sc.ID) + "\x00" + strconv.Itoa(a.scenarioBackstoryScroll) + "\x00" + strconv.Itoa(tw) + "\x00" + body
}

func (a *App) ensureScenarioListDetailCache(sc *campaign.ScenarioDef) {
	if sc == nil {
		return
	}
	_, _, tw, _ := scenarioListBackstoryRect()
	key := a.scenarioListDetailCacheKey(sc, tw)
	w, h := scenarioListDetailCacheSize()
	if a.scenarioListDetailImg != nil && a.scenarioListDetailKey == key &&
		a.scenarioListDetailImg.Bounds().Dx() == w && a.scenarioListDetailImg.Bounds().Dy() == h {
		return
	}
	if a.scenarioListDetailImg != nil {
		a.scenarioListDetailImg.Dispose()
		a.scenarioListDetailImg = nil
	}
	a.scenarioListDetailKey = key
	img := ebiten.NewImage(w, h)
	titleY := scenarioPanelY + 48
	coverY := scenarioListCoverY()
	render.DrawTextLarge(img, sc.Title.GetText(a.Lang()), 0, 28, render.ColorText)
	render.DrawText(img, scenarioVersionLine(sc), 0, 52, render.ColorPhosphorDim, true)
	drawScenarioCoverImage(a, img, sc, nil, 0, coverY-titleY, scenarioDetailW, scenarioCoverH)
	body := sc.Backstory.GetText(a.Lang())
	if !sc.Compatible {
		body = a.L(i18n.UIScenarioIncompatBody) + "\n\n" + sc.IncompatibleReason
	}
	lines := a.scenarioListMarkdownLines(body, tw)
	_, by, _, bh := scenarioListBackstoryRect()
	vis := bh / 18
	if vis < 1 {
		vis = 1
	}
	a.scenarioBackstoryScroll = clampContactTableScroll(a.scenarioBackstoryScroll, len(lines), vis)
	start, end := contactTableWindow(len(lines), a.scenarioBackstoryScroll, vis)
	render.DrawMDLines(img, lines, start, end, 0, by-titleY, false)
	drawContactTableScrollbar(img, tw+4, by-titleY, bh, len(lines), vis, a.scenarioBackstoryScroll)
	a.scenarioListDetailImg = img
}

func scenarioBriefDetailCacheSize(aboveLoadout bool, mapW int) (w, h int) {
	titleY := scenarioPanelY + 64
	_, ty, tw, th := scenarioBriefDescRect(aboveLoadout)
	h = ty + th - titleY
	w = tw + 14
	if mapW > 0 {
		w += scenarioColumnGap + mapW
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

func (a *App) scenarioBriefDetailCacheKey(sc *campaign.ScenarioDef, cur *campaign.MissionDef, aboveLoadout bool, mapW int) string {
	if sc == nil || cur == nil {
		return ""
	}
	_, _, tw, _ := scenarioBriefDescRect(aboveLoadout)
	body := cur.Description.GetText(a.Lang())
	small := aboveLoadout
	if !sc.Compatible {
		body = a.L(i18n.UIScenarioIncompatBody) + "\n\n" + sc.IncompatibleReason
		small = false
	} else if a.briefDebrief {
		prog := a.cachedScenarioProgress(sc.ID)
		body = campaign.ComposeMissionDebrief(*cur, prog.DebriefOutcomes, a.Lang())
		small = false
	}
	return string(sc.ID) + "\x00" + string(cur.ID) + "\x00" + strconv.FormatBool(a.briefDebrief) + "\x00" +
		strconv.Itoa(a.scenarioBriefDescScroll) + "\x00" + strconv.Itoa(tw) + "\x00" + strconv.Itoa(mapW) + "\x00" +
		strconv.FormatBool(small) + "\x00" + body + "\x00" + a.briefMapCacheKey
}

func (a *App) ensureScenarioBriefDetailCache(sc *campaign.ScenarioDef, cur *campaign.MissionDef, aboveLoadout bool, mapW int) {
	if sc == nil || cur == nil {
		return
	}
	key := a.scenarioBriefDetailCacheKey(sc, cur, aboveLoadout, mapW)
	w, h := scenarioBriefDetailCacheSize(aboveLoadout, mapW)
	if a.scenarioBriefDetailImg != nil && a.scenarioBriefDetailKey == key &&
		a.scenarioBriefDetailImg.Bounds().Dx() == w && a.scenarioBriefDetailImg.Bounds().Dy() == h {
		return
	}
	if a.scenarioBriefDetailImg != nil {
		a.scenarioBriefDetailImg.Dispose()
		a.scenarioBriefDetailImg = nil
	}
	a.scenarioBriefDetailKey = key
	img := ebiten.NewImage(w, h)
	titleY := scenarioPanelY + 64
	_, ty, tw, th := scenarioBriefDescRect(aboveLoadout)
	render.DrawTextLarge(img, cur.Title.GetText(a.Lang()), 0, 0, render.ColorText)
	body := cur.Description.GetText(a.Lang())
	small := true
	if !sc.Compatible {
		body = a.L(i18n.UIScenarioIncompatBody) + "\n\n" + sc.IncompatibleReason
		small = false
	} else if a.briefDebrief {
		prog := a.cachedScenarioProgress(sc.ID)
		body = campaign.ComposeMissionDebrief(*cur, prog.DebriefOutcomes, a.Lang())
		small = false
	}
	lines := a.scenarioBriefMarkdownLines(body, tw, small)
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
	render.DrawMDLines(img, lines, start, end, 0, ty-titleY, small)
	drawContactTableScrollbar(img, tw+4, ty-titleY, th, len(lines), vis, a.scenarioBriefDescScroll)
	if mapW > 0 && a.briefMapCacheKey != "" {
		mx := tw + 14 + scenarioColumnGap
		_, mh := scenarioBriefMapSize(th)
		render.DrawScenarioCoverImage(img, a.briefMapCacheKey, mx, ty-titleY, mapW, mh)
	}
	a.scenarioBriefDetailImg = img
}

func drawScenarioCoverImage(a *App, screen *ebiten.Image, sc *campaign.ScenarioDef, m *campaign.MissionDef, x, y, w, h int) {
	if sc == nil {
		return
	}
	data, key := campaign.MissionCover(sc, m)
	if len(data) > 0 && key != "" {
		render.EnsureScenarioCoverImage(key, data)
	}
	if key != "" {
		render.DrawScenarioCoverImage(screen, key, x, y, w, h)
		return
	}
	render.FillRect(screen, x, y, w, h, render.ColorPanelInset)
	render.DrawText(screen, a.L(i18n.UINoArt), x+w/2-28, y+h/2, render.ColorDim, true)
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
		a.scenarioListMDKey = ""
		a.scenarioListDetailKey = ""
		a.briefMapCacheKey = ""
		a.scenarioBriefDetailKey = ""
		if a.scenarioBriefDetailImg != nil {
			a.scenarioBriefDetailImg.Dispose()
			a.scenarioBriefDetailImg = nil
		}
		a.markScenarioUIDirty()
		if sc := campaign.ScenarioByID(id); sc != nil {
			campaign.WarmScenarioCover(sc, nil)
		}
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
		a.endScenarioUI()
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

	if sc := a.selectedScenarioDef(); sc != nil {
		tx, ty, tw, th := scenarioListBackstoryRect()
		body := sc.Backstory.GetText(a.Lang())
		if !sc.Compatible {
			body = "This scenario is incompatible with this game version.\n\n" + sc.IncompatibleReason
		}
		lines := a.scenarioListMarkdownLines(body, tw)
		vis := th / 18
		if vis < 1 {
			vis = 1
		}
		prevScroll := a.scenarioBackstoryScroll
		scrollContactTableWheel(mx, my, tx, ty, tw+10, th, len(lines), vis, &a.scenarioBackstoryScroll)
		if a.scenarioBackstoryScroll != prevScroll {
			a.scenarioListDetailKey = ""
			a.markScenarioUIDirty()
		}
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}

	listX, listY, listW, rowH := scenarioListRect()
	for i := range defs {
		y := listY + i*rowH
		if hitRect(mx, my, listX, y, listW, rowH) {
			a.ScenarioListIndex = i
			a.ensureScenarioSelection()
			return nil
		}
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
		a.showConfirm(confirmRestartScenario, a.L(i18n.UIConfirmRestartTitle), a.L(i18n.UIConfirmRestartBody))
		return nil
	}
	dx, dy, dw, dh := a.scenarioListDeleteRect()
	if a.selectedScenarioImported() && hitRect(mx, my, dx, dy, dw, dh) {
		a.showConfirm(confirmDeleteScenario, a.L(i18n.UIConfirmDeleteTitle), a.L(i18n.UIConfirmDeleteBody))
		return nil
	}
	sx, sy, sw, sh := a.scenarioListSelectRect()
	if a.selectedScenarioPlayable() && hitRect(mx, my, sx, sy, sw, sh) {
		a.briefDebrief = false
		a.briefMissionID = ""
		a.Mode = ModeScenarioBrief
		a.markScenarioUIDirty()
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
		if a.scenarioBriefDetailImg != nil {
			a.scenarioBriefDetailImg.Dispose()
			a.scenarioBriefDetailImg = nil
		}
		a.scenarioBriefDetailKey = ""
		a.briefMapCacheKey = ""
		a.markScenarioUIDirty()
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
			prog := a.cachedScenarioProgress(sc.ID)
			body = campaign.ComposeMissionDebrief(*cur, prog.DebriefOutcomes, a.Lang())
		} else if cur != nil {
			body = cur.Description.GetText(a.Lang())
		}
		if body != "" {
			small := aboveLoadout
			lines := a.scenarioBriefMarkdownLines(body, tw, small)
			lineH := 18
			if small {
				lineH = 14
			}
			vis := th / lineH
			if vis < 1 {
				vis = 1
			}
			prevScroll := a.scenarioBriefDescScroll
			scrollContactTableWheel(mx, my, tx, ty, tw+10, th, len(lines), vis, &a.scenarioBriefDescScroll)
			if a.scenarioBriefDescScroll != prevScroll {
				a.scenarioBriefDetailKey = ""
				a.markScenarioUIDirty()
			}
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
		a.briefMapCacheKey = ""
		a.markScenarioUIDirty()
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
	w = render.ButtonWidth(a.L(i18n.UIContinueScenario), 20)
	h = 40
	x = scenarioDetailX() + scenarioDetailW - w
	y = scenarioPanelY + scenarioPanelH - h - 24
	return x, y, w, h
}

func (a *App) scenarioListSelectRect() (x, y, w, h int) {
	w = render.ButtonWidth(a.L(i18n.UIOpenScenario), 20)
	h = 40
	x = scenarioDetailX()
	y = scenarioPanelY + scenarioPanelH - h - 24
	return x, y, w, h
}

func (a *App) scenarioListRestartRect() (x, y, w, h int) {
	w = render.ButtonWidth(a.L(i18n.UIRestartScenario), 20)
	h = 40
	ox, oy, ow, _ := a.scenarioListSelectRect()
	x = ox + ow + 16
	y = oy
	return x, y, w, h
}

func (a *App) scenarioListDeleteRect() (x, y, w, h int) {
	w = render.ButtonWidth(a.L(i18n.UIDeleteScenario), 20)
	h = 40
	ox, oy, ow, _ := a.scenarioListRestartRect()
	x = ox + ow + 16
	y = oy
	return x, y, w, h
}

func (a *App) selectedScenarioImported() bool {
	sc := a.selectedScenarioDef()
	return sc != nil && campaign.HasUserScenarioFile(sc.ID)
}

func (a *App) scenarioListBackRect() (x, y, w, h int) {
	w = render.ButtonWidth(a.L(i18n.UIBack), 20)
	h = 40
	x = scenarioPanelX + 16
	_, y, _, _ = a.scenarioListSelectRect()
	return x, y, w, h
}

func (a *App) scenarioBriefBackRect() (x, y, w, h int) {
	w = render.ButtonWidth(a.L(i18n.UIBack), 20)
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
		return a.L(i18n.UINextMission)
	}
	sc := a.selectedScenarioDef()
	if sc != nil && a.scenarioProgress(sc.ID).ScenarioComplete(sc) {
		return a.L(i18n.UIScenarioComplete)
	}
	return a.L(i18n.UIStartMission)
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

func (a *App) scenarioBriefMarkdownLines(body string, tw int, small bool) []render.MDLine {
	key := body + "\x00" + strconv.Itoa(tw) + "\x00" + strconv.FormatBool(small)
	if key == a.scenarioBriefMDKey && len(a.scenarioBriefMDLines) > 0 {
		return a.scenarioBriefMDLines
	}
	a.scenarioBriefMDKey = key
	a.scenarioBriefMDLines = render.ParseMarkdown(body, tw, small)
	return a.scenarioBriefMDLines
}

func (a *App) scenarioListMarkdownLines(body string, tw int) []render.MDLine {
	key := body + "\x00" + strconv.Itoa(tw)
	if key == a.scenarioListMDKey && len(a.scenarioListMDLines) > 0 {
		return a.scenarioListMDLines
	}
	a.scenarioListMDKey = key
	a.scenarioListMDLines = render.ParseMarkdown(body, tw, false)
	return a.scenarioListMDLines
}

func (a *App) cachedScenarioProgress(id campaign.ScenarioID) campaign.Progress {
	if a.briefProgressOK && a.briefProgressID == id {
		return a.briefProgress
	}
	a.briefProgress = a.scenarioProgress(id)
	a.briefProgressID = id
	a.briefProgressOK = true
	return a.briefProgress
}

func (a *App) invalidateScenarioProgressCache() {
	a.briefProgressOK = false
}

func (a *App) ensureScenarioBriefMap(cur *campaign.MissionDef) {
	want := ""
	if cur != nil && scenarioBriefShowsMap(cur, a.briefDebrief) {
		want = campaign.WarmMissionBriefMap(cur)
	}
	if want != a.briefMapCacheKey {
		a.briefMapCacheKey = want
		a.scenarioBriefDetailKey = ""
	}
}

func (a *App) syncScenarioBriefAssets(sc *campaign.ScenarioDef) {
	a.scenarioBriefMDKey = ""
	a.scenarioBriefDetailKey = ""
	a.invalidateScenarioProgressCache()
	if sc == nil {
		a.briefMapCacheKey = ""
		return
	}
	a.ensureScenarioBriefMap(a.briefDisplayedMission(sc))
}

func (a *App) initScenarioBrief() {
	a.scenarioBriefDescScroll = 0
	a.scenarioBriefDetailKey = ""
	a.markScenarioUIDirty()
	a.invalidateScenarioProgressCache()
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
			a.syncScenarioBriefAssets(sc)
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
	a.syncScenarioBriefAssets(sc)
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
	defs := a.scenarioDefs()
	a.ensureScenarioSelection()
	sc := a.selectedScenarioDef()
	mx, my := ebiten.CursorPosition()
	hover := a.scenarioListHoverKey(mx, my)
	if !a.scenarioUIDirty && hover == a.scenarioUIHoverKey {
		return
	}

	a.drawScenarioScreenBackground(screen)
	panelW := scenarioPanelW()
	render.DrawConsolePanel(screen, scenarioPanelX, scenarioPanelY, panelW, scenarioPanelH)
	render.DrawTextLarge(screen, a.L(i18n.UISelectScenario), scenarioPanelX+20, scenarioPanelY+28, render.ColorText)

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
		label := d.Title.GetText(a.Lang())
		if !d.Compatible {
			label += a.L(i18n.UIIncompatibleBadge)
		}
		render.DrawText(screen, label, listX+8, y+24, clr, false)
	}

	if sc != nil {
		a.ensureScenarioListDetailCache(sc)
		if a.scenarioListDetailImg != nil {
			dx := scenarioDetailX()
			dy := scenarioPanelY + 48
			var opts ebiten.DrawImageOptions
			opts.GeoM.Translate(float64(dx), float64(dy))
			screen.DrawImage(a.scenarioListDetailImg, &opts)
		}
	}

	mx, my = ebiten.CursorPosition()
	cx, cy, cw, ch := a.scenarioListContinueRect()
	hasSave := false
	if sc != nil && sc.Compatible {
		_, err := campaign.LatestSaveForScenario(sc.ID)
		hasSave = err == nil
	}
	if hasSave {
		render.DrawBevelButton(screen, cx, cy, cw, ch, a.L(i18n.UIContinueScenario), hitRect(mx, my, cx, cy, cw, ch), false)
	} else {
		render.DrawBevelButtonDisabled(screen, cx, cy, cw, ch, a.L(i18n.UIContinueScenario))
	}
	rx, ry, rw, rh := a.scenarioListRestartRect()
	if sc != nil && sc.Compatible {
		render.DrawBevelButton(screen, rx, ry, rw, rh, a.L(i18n.UIRestartScenario), hitRect(mx, my, rx, ry, rw, rh), false)
	} else {
		render.DrawBevelButtonDisabled(screen, rx, ry, rw, rh, a.L(i18n.UIRestartScenario))
	}
	dx, dy, dw, dh := a.scenarioListDeleteRect()
	if a.selectedScenarioImported() {
		render.DrawBevelButton(screen, dx, dy, dw, dh, a.L(i18n.UIDeleteScenario), hitRect(mx, my, dx, dy, dw, dh), false)
	}
	sx, sy, sw, sh := a.scenarioListSelectRect()
	if sc != nil && sc.Compatible {
		render.DrawBevelButton(screen, sx, sy, sw, sh, a.L(i18n.UIOpenScenario), hitRect(mx, my, sx, sy, sw, sh), false)
	} else {
		render.DrawBevelButtonDisabled(screen, sx, sy, sw, sh, a.L(i18n.UIOpenScenario))
	}
	bx, by, bw, bh := a.scenarioListBackRect()
	render.DrawBevelButton(screen, bx, by, bw, bh, a.L(i18n.UIBack), hitRect(mx, my, bx, by, bw, bh), false)

	if a.StatusMessage != "" {
		render.DrawText(screen, a.displayStatus(), scenarioPanelX+20, scenarioPanelY+scenarioPanelH+12, render.ColorWarn, false)
	}
	a.drawConfirmDialog(screen)
	a.scenarioUIDirty = false
	a.scenarioUIHoverKey = hover
}

func (a *App) drawScenarioBrief(screen *ebiten.Image) {
	sc := a.selectedScenarioDef()
	if sc == nil {
		return
	}
	cur := a.briefDisplayedMission(sc)
	a.ensureScenarioBriefMap(cur)
	aboveLoadout := sc.Compatible && !a.briefDebrief
	mapW := 0
	if scenarioBriefShowsMap(cur, a.briefDebrief) {
		_, _, mapW, _ = scenarioBriefMapRect(aboveLoadout)
	}
	prog := a.cachedScenarioProgress(sc.ID)
	complete := prog.ScenarioComplete(sc)
	mx, my := ebiten.CursorPosition()
	hover := a.scenarioBriefHoverKey(mx, my, sc, complete)
	if !a.scenarioUIDirty && hover == a.scenarioUIHoverKey && !a.loadoutDragging && a.loadoutOrdnanceMenuTube == 0 {
		return
	}

	a.drawScenarioScreenBackground(screen)
	panelW := scenarioBriefPanelW(mapW)
	render.DrawConsolePanel(screen, scenarioPanelX, scenarioPanelY, panelW, scenarioPanelH)
	render.DrawTextLarge(screen, sc.Title.GetText(a.Lang()), scenarioPanelX+20, scenarioPanelY+28, render.ColorText)
	render.DrawText(screen, scenarioVersionLine(sc), scenarioPanelX+20, scenarioPanelY+52, render.ColorPhosphorDim, true)

	listX := scenarioPanelX + 16
	listY := scenarioPanelY + 64
	for i, m := range sc.Missions {
		y := listY + i*32
		done := prog.CompletedMissions[m.ID]
		selected := cur != nil && m.ID == cur.ID
		label := m.Title.GetText(a.Lang())
		if done {
			label += a.L(i18n.UIDoneBadge)
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
		a.ensureScenarioBriefDetailCache(sc, cur, aboveLoadout, mapW)
		if a.scenarioBriefDetailImg != nil {
			dx := scenarioDetailX()
			dy := scenarioPanelY + 64
			var opts ebiten.DrawImageOptions
			opts.GeoM.Translate(float64(dx), float64(dy))
			screen.DrawImage(a.scenarioBriefDetailImg, &opts)
		}
		if aboveLoadout {
			a.drawScenarioLoadout(screen)
		}
	}

	mx, my = ebiten.CursorPosition()
	bx, by, bw, bh := a.scenarioBriefBackRect()
	render.DrawBevelButton(screen, bx, by, bw, bh, a.L(i18n.UIBack), hitRect(mx, my, bx, by, bw, bh), false)
	stx, sty, stw, sth := a.scenarioBriefPrimaryRect()
	if a.briefDebrief {
		if a.scenarioBriefHasNext() {
			render.DrawBevelButton(screen, stx, sty, stw, sth, a.L(i18n.UINextMission), hitRect(mx, my, stx, sty, stw, sth), false)
		}
	} else if !complete && sc.Compatible {
		render.DrawBevelButton(screen, stx, sty, stw, sth, a.L(i18n.UIStartMission), hitRect(mx, my, stx, sty, stw, sth), false)
	} else if !sc.Compatible {
		render.DrawBevelButtonDisabled(screen, stx, sty, stw, sth, a.L(i18n.UIStartMission))
	} else {
		render.DrawBevelButtonDisabled(screen, stx, sty, stw, sth, a.L(i18n.UIScenarioComplete))
	}
	a.drawConfirmDialog(screen)
	a.scenarioUIDirty = false
	a.scenarioUIHoverKey = hover
}
