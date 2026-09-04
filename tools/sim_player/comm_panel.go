package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/simreplay"
)

const commPanelW = 360

func (g *playerGame) commRect() (x, y, w, h int) {
	return screenW - commPanelW, topH, commPanelW, screenH - topH - controlsH
}

func (g *playerGame) commMessageLines() []render.MDLine {
	_, _, w, _ := g.commRect()
	maxW := w - 28
	if maxW < 80 {
		maxW = 80
	}
	g.syncCommPlayhead()
	return simreplay.CommLines(g.comm.Inbox(), g.replay.MissionStartSec, g.curTime, g.lang, maxW)
}

func (g *playerGame) scrollCommWheel(mx, my int) bool {
	x, y, w, h := g.commRect()
	if mx < x || mx >= x+w || my < y || my >= y+h {
		return false
	}
	lines := g.commMessageLines()
	vis := commVisibleRows(h)
	if len(lines) <= vis {
		return true
	}
	_, wheelY := ebiten.Wheel()
	if math.Abs(wheelY) < 0.01 {
		return true
	}
	step := int(math.Ceil(math.Abs(wheelY)))
	if wheelY < 0 {
		g.commScroll += step
	} else {
		g.commScroll -= step
	}
	g.commScroll = clampScroll(g.commScroll, len(lines), vis)
	return true
}

func (g *playerGame) drawCommPanel(screen *ebiten.Image) {
	x, y, w, h := g.commRect()
	render.FillRect(screen, x, y, w, h, color.RGBA{14, 16, 22, 255})
	render.DrawLine(screen, float64(x), float64(y), float64(x), float64(y+h), color.RGBA{50, 60, 80, 255})

	title := i18n.UICOMContacts.GetText(g.lang)
	render.DrawText(screen, title, x+12, y+20, render.ColorPhosphor, true)
	render.DrawLine(screen, float64(x+8), float64(y+28), float64(x+w-8), float64(y+28), color.RGBA{60, 70, 90, 255})

	msgX, msgY := x+10, y+36
	msgW, msgH := w-20, h-44
	render.FillRect(screen, msgX, msgY, msgW, msgH, render.ColorPanelInset)

	lines := g.commMessageLines()
	vis := commVisibleRows(msgH)
	g.commScroll = clampScroll(g.commScroll, len(lines), vis)
	start, end := scrollWindow(len(lines), g.commScroll, vis)
	if len(lines) == 0 {
		noTraffic := i18n.UINoTraffic.GetText(g.lang)
		render.DrawText(screen, noTraffic, msgX+6, msgY+16, render.ColorDim, true)
	} else {
		render.DrawMDLines(screen, lines, start, end, msgX+6, msgY+12, true)
	}
	drawScrollBar(screen, msgX+msgW-6, msgY+8, msgH-12, len(lines), vis, g.commScroll)
}

func commVisibleRows(panelH int) int {
	vis := panelH / 14
	if vis < 1 {
		return 1
	}
	return vis
}

func clampScroll(offset, total, visible int) int {
	if visible < 1 {
		visible = 1
	}
	maxOff := total - visible
	if maxOff < 0 {
		maxOff = 0
	}
	if offset < 0 {
		return 0
	}
	if offset > maxOff {
		return maxOff
	}
	return offset
}

func scrollWindow(total, offset, visible int) (start, end int) {
	offset = clampScroll(offset, total, visible)
	end = offset + visible
	if end > total {
		end = total
	}
	return offset, end
}

func drawScrollBar(screen *ebiten.Image, x, y, h, total, visible, offset int) {
	if total <= visible || visible < 1 || h <= 0 {
		return
	}
	trackW := 4
	render.FillRect(screen, x, y, trackW, h, color.RGBA{42, 46, 52, 255})
	thumbH := int(float64(h) * float64(visible) / float64(total))
	if thumbH < 10 {
		thumbH = 10
	}
	maxOffset := total - visible
	top := y
	if maxOffset > 0 && h-thumbH > 0 {
		top += int(float64(h-thumbH) * float64(offset) / float64(maxOffset))
	}
	render.FillRect(screen, x, top, trackW, thumbH, color.RGBA{115, 120, 128, 255})
}
