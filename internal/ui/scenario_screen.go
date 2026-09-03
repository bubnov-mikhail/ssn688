package ui

import (
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/render"
)

func (a *App) beginScenarioUI() {
	if !a.scenarioScreenActive {
		ebiten.SetScreenClearedEveryFrame(false)
		a.scenarioScreenActive = true
	}
	a.markScenarioUIDirty()
}

func (a *App) endScenarioUI() {
	if a.scenarioScreenActive {
		ebiten.SetScreenClearedEveryFrame(true)
		a.scenarioScreenActive = false
	}
	a.disposeScenarioPanelCaches()
}

func (a *App) markScenarioUIDirty() {
	a.scenarioUIDirty = true
}

func (a *App) scenarioListHoverKey(mx, my int) string {
	if a.confirmActive() {
		return "confirm"
	}
	var b strings.Builder
	b.WriteString(strconv.Itoa(mx))
	b.WriteByte(',')
	b.WriteString(strconv.Itoa(my))
	cx, cy, cw, ch := a.scenarioListContinueRect()
	rx, ry, rw, rh := a.scenarioListRestartRect()
	dx, dy, dw, dh := a.scenarioListDeleteRect()
	sx, sy, sw, sh := a.scenarioListSelectRect()
	bx, by, bw, bh := a.scenarioListBackRect()
	for _, hit := range []struct {
		tag        string
		x, y, w, h int
	}{
		{"cont", cx, cy, cw, ch},
		{"rest", rx, ry, rw, rh},
		{"del", dx, dy, dw, dh},
		{"open", sx, sy, sw, sh},
		{"back", bx, by, bw, bh},
	} {
		if hitRect(mx, my, hit.x, hit.y, hit.w, hit.h) {
			b.WriteByte('|')
			b.WriteString(hit.tag)
		}
	}
	return b.String()
}

func (a *App) scenarioBriefHoverKey(mx, my int, sc *campaign.ScenarioDef, complete bool) string {
	if a.confirmActive() || a.loadoutDragging || a.loadoutOrdnanceMenuTube > 0 {
		return "interactive"
	}
	var b strings.Builder
	b.WriteString(strconv.Itoa(mx))
	b.WriteByte(',')
	b.WriteString(strconv.Itoa(my))
	bx, by, bw, bh := a.scenarioBriefBackRect()
	if hitRect(mx, my, bx, by, bw, bh) {
		b.WriteString("|back")
	}
	if a.briefDebrief && a.scenarioBriefHasNext() {
		stx, sty, stw, sth := a.scenarioBriefPrimaryRect()
		if hitRect(mx, my, stx, sty, stw, sth) {
			b.WriteString("|next")
		}
	} else if !a.briefDebrief && sc != nil && sc.Compatible && !complete {
		stx, sty, stw, sth := a.scenarioBriefPrimaryRect()
		if hitRect(mx, my, stx, sty, stw, sth) {
			b.WriteString("|start")
		}
	}
	_ = sc
	return b.String()
}

func (a *App) drawScenarioScreenBackground(screen *ebiten.Image) {
	render.DrawMenuBackground(screen)
}

// EnterScenarioListForProfile opens the scenario list (tools/profile_scenario_ui).
func (a *App) EnterScenarioListForProfile(prefer campaign.ScenarioID) {
	a.beginScenarioUI()
	a.Mode = ModeScenarioList
	a.ScenarioListIndex = 0
	for i, d := range campaign.AllScenarios() {
		if prefer != "" && d.ID == prefer {
			a.ScenarioListIndex = i
			break
		}
	}
	a.ensureScenarioSelection()
}
