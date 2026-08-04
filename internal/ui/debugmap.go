package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/world"
)

const (
	debugMapW      = 250
	debugMapH      = 250
	debugMapPad    = 16
	debugZoomMin   = 0.4
	debugZoomMax   = 4.0
	debugZoomStep  = 0.25
	debugBaseScale = 0.018 // pixels per yard at zoom 1.0
)

func (a *App) debugMapRect() (x, y, w, h int) {
	x = render.ScreenW - debugMapW - debugMapPad
	y = render.ScreenH - debugMapH - 90 - navBarH
	return x, y, debugMapW, debugMapH
}

func (a *App) debugZoomInRect() (x, y, w, h int) {
	mx, my, _, _ := a.debugMapRect()
	return mx + debugMapW - 56, my + 8, 22, 22
}

func (a *App) debugZoomOutRect() (x, y, w, h int) {
	mx, my, _, _ := a.debugMapRect()
	return mx + debugMapW - 84, my + 8, 22, 22
}

func (a *App) updateDebugMapInput() {
	if a.Engine == nil || !a.Settings.Debug {
		return
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	mx, my := ebiten.CursorPosition()
	zinX, zinY, zinW, zinH := a.debugZoomInRect()
	if inRect(mx, my, zinX, zinY, zinW, zinH) {
		a.debugMapZoom = math.Min(debugZoomMax, a.debugMapZoom+debugZoomStep)
		return
	}
	zoutX, zoutY, zoutW, zoutH := a.debugZoomOutRect()
	if inRect(mx, my, zoutX, zoutY, zoutW, zoutH) {
		a.debugMapZoom = math.Max(debugZoomMin, a.debugMapZoom-debugZoomStep)
	}
}

func inRect(px, py, x, y, w, h int) bool {
	return px >= x && px < x+w && py >= y && py < y+h
}

func (a *App) drawDebugMap(screen *ebiten.Image) {
	if a.Engine == nil || !a.Settings.Debug {
		return
	}

	mx, my, mw, mh := a.debugMapRect()
	render.FillRect(screen, mx, my, mw, mh, render.ColorDebugPanel)
	render.DrawLine(screen, float64(mx), float64(my), float64(mx+mw), float64(my), render.ColorBorder)
	render.DrawLine(screen, float64(mx+mw), float64(my), float64(mx+mw), float64(my+mh), render.ColorBorder)
	render.DrawLine(screen, float64(mx+mw), float64(my+mh), float64(mx), float64(my+mh), render.ColorBorder)
	render.DrawLine(screen, float64(mx), float64(my+mh), float64(mx), float64(my), render.ColorBorder)

	render.DrawText(screen, "DEBUG MAP", mx+10, my+24, render.ColorDim, true)

	ox := float64(mx + mw/2)
	oy := float64(my + mh/2 + 8)
	scale := debugBaseScale * a.debugMapZoom

	for _, r := range []float64{1000, 3000, 6000} {
		rad := r * scale
		if rad < 8 || rad > float64(mw)/2-8 {
			continue
		}
		drawCircle(screen, ox, oy, rad, render.ColorGrid)
	}

	player := a.Engine.Scenario.Player
	a.drawDebugEntityAt(screen, ox, oy, 0, 0, player.HeadingDeg, player.SpeedKts, render.ColorDebugPlayer, player.Alive(), "")

	for _, e := range a.Engine.Scenario.Entities {
		a.drawDebugEntityAt(screen, ox, oy, e.X-player.X, e.Y-player.Y, e.HeadingDeg, e.SpeedKts, debugEntityColor(e), e.Alive(), a.debugEntityLabel(e))
	}

	for _, t := range a.Engine.FireControl.ActiveTorpedoes {
		if !t.Alive {
			continue
		}
		clr := render.ColorDebugAttack
		if t.Side == world.SidePlayer {
			clr = render.ColorActive
		}
		a.drawDebugEntityAt(screen, ox, oy, t.X-player.X, t.Y-player.Y, t.HeadingDeg, t.SpeedKts, clr, true, "MK48")
	}

	a.drawDebugZoomButtons(screen)
}

func debugEntityClass(e *world.Entity) string {
	if p, ok := world.ProfileByID(e.SignatureID); ok {
		return p.Class
	}
	if e.Name != "" {
		return e.Name
	}
	return e.ID
}

func (a *App) debugEntityLabel(e *world.Entity) string {
	label := debugEntityClass(e)
	if a == nil || a.Engine == nil {
		return label
	}
	for i := range a.Engine.Sonar.Contacts {
		c := &a.Engine.Sonar.Contacts[i]
		if c.SourceEntityID == e.ID {
			return fmt.Sprintf("%s %s", c.ID, label)
		}
	}
	return label
}

func (a *App) drawDebugEntityAt(screen *ebiten.Image, ox, oy, dx, dy, heading, speedKts float64, clr color.Color, active bool, classLabel string) {
	scale := debugBaseScale * a.debugMapZoom
	sx := ox + dx*scale
	sy := oy - dy*scale

	mx, my, mw, mh := a.debugMapRect()
	if sx < float64(mx+4) || sx > float64(mx+mw-4) || sy < float64(my+28) || sy > float64(my+mh-4) {
		return
	}

	if !active {
		clr = render.ColorDebugInactive
	}

	render.FillRect(screen, int(sx)-3, int(sy)-3, 7, 7, clr)

	rad := heading * math.Pi / 180
	ln := 14.0
	render.DrawLine(screen, sx, sy, sx+math.Sin(rad)*ln, sy-math.Cos(rad)*ln, clr)

	line1 := fmt.Sprintf("%.0f kt", speedKts)
	if classLabel != "" {
		render.DrawText(screen, classLabel, int(sx)+8, int(sy)-4, clr, true)
		line1 = fmt.Sprintf("%.0f kt", speedKts)
		render.DrawText(screen, line1, int(sx)+8, int(sy)+8, clr, true)
		return
	}
	render.DrawText(screen, line1, int(sx)+8, int(sy)-4, clr, true)
}

func (a *App) drawDebugZoomButtons(screen *ebiten.Image) {
	zoutX, zoutY, zw, zh := a.debugZoomOutRect()
	zinX, zinY, _, _ := a.debugZoomInRect()

	render.FillRect(screen, zoutX, zoutY, zw, zh, render.ColorPanel)
	render.FillRect(screen, zinX, zinY, zw, zh, render.ColorPanel)
	render.DrawText(screen, "-", zoutX+7, zoutY+18, render.ColorText, false)
	render.DrawText(screen, "+", zinX+6, zinY+18, render.ColorText, false)
}

func debugEntityColor(e *world.Entity) color.Color {
	if !e.Alive() {
		return render.ColorDebugInactive
	}
	if e.Side == world.SideNeutral {
		return color.RGBA{180, 180, 200, 255}
	}
	switch e.AIState {
	case "INTERCEPT", "ATTACK", "FIRING", "SHADOW", "CLOSING", "OPENING":
		return render.ColorDebugAttack
	case "SEARCH", "PINGING", "ACTIVE_SEARCH":
		return render.ColorDebugSearch
	default:
		return render.ColorDebugCalm
	}
}

func drawCircle(screen *ebiten.Image, cx, cy, r float64, clr color.Color) {
	const segments = 48
	for i := 0; i < segments; i++ {
		a0 := float64(i) * 2 * math.Pi / segments
		a1 := float64(i+1) * 2 * math.Pi / segments
		render.DrawLine(screen,
			cx+math.Cos(a0)*r, cy+math.Sin(a0)*r,
			cx+math.Cos(a1)*r, cy+math.Sin(a1)*r,
			clr)
	}
}
