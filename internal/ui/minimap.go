package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ssn688/sim/internal/i18n"
	"github.com/ssn688/sim/internal/layout"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

const (
	objectivesPanelX = 300 // inset from right: ScreenW - 300
	objectivesPanelW = 290

	minimapW      = objectivesPanelW
	minimapH      = objectivesPanelW // square
	minimapTitleH = 28
)

func inRect(px, py, x, y, w, h int) bool {
	return px >= x && px < x+w && py >= y && py < y+h
}

func (a *App) minimapRect() (x, y, w, h int) {
	x = render.ScreenW - objectivesPanelX
	// Bottom edge matches the main console plates (y=50, h=700 → bottom 750).
	y = layout.PassiveMainPanelY + layout.PassiveMainPanelH - minimapH
	return x, y, minimapW, minimapH
}

func (a *App) drawTacticalMinimap(screen *ebiten.Image) {
	if a.Engine == nil || a.CurrentScreen == ScreenTactical {
		return
	}
	a.ensureTactical()

	mx, my, mw, mh := a.minimapRect()
	render.FillRect(screen, mx, my, mw, mh, render.ColorDebugPanel)
	border := render.ColorBorder
	render.DrawLine(screen, float64(mx), float64(my), float64(mx+mw), float64(my), border)
	render.DrawLine(screen, float64(mx+mw), float64(my), float64(mx+mw), float64(my+mh), border)
	render.DrawLine(screen, float64(mx+mw), float64(my+mh), float64(mx), float64(my+mh), border)
	render.DrawLine(screen, float64(mx), float64(my+mh), float64(mx), float64(my), border)

	title := a.L(i18n.UINavPlot)
	if a.Settings.Debug {
		title = a.L(i18n.UINavPlot) + " · DEBUG"
	}
	render.DrawText(screen, title, mx+10, my+20, render.ColorDim, true)

	// Map content below the title strip; SubImage clips rings/LOB/icons to the panel.
	mapX := mx + 1
	mapY := my + minimapTitleH
	mapW := mw - 2
	mapH := mh - minimapTitleH - 1
	if mapW <= 0 || mapH <= 0 {
		return
	}
	clip := screen.SubImage(image.Rect(mapX, mapY, mapX+mapW, mapY+mapH)).(*ebiten.Image)

	opts := tacticalMapOpts{
		minimap:       true,
		debugOverlay:  a.Settings.Debug,
		showSelection: false,
		showChrome:    false,
	}
	a.drawTacticalMap(clip, mapX, mapY, mapW, mapH, opts)
}

func (a *App) drawTacticalDebugOverlay(screen *ebiten.Image, view tacticalMapView) {
	player := a.Engine.Scenario.Player
	if player == nil {
		return
	}

	a.drawDebugRoutes(screen, view)

	for _, e := range a.Engine.Scenario.Entities {
		a.drawDebugEntityAt(screen, view, e.X, e.Y, e.HeadingDeg, e.SpeedKts, debugEntityColor(e), e.Alive(), a.debugEntityLabel(e))
		if e.Side == world.SideEnemy && e.Alive() {
			sx, sy := view.worldToScreen(e.X, e.Y)
			if view.containsScreen(int(sx), int(sy)) {
				render.DrawText(screen, fmt.Sprintf("%d", e.Defcon), int(sx)-6, int(sy)+16, render.ColorAmber, true)
			}
		}
		if e.Side == world.SidePlayer && e.Alive() && e.ID != player.ID {
			sx, sy := view.worldToScreen(e.X, e.Y)
			if view.containsScreen(int(sx), int(sy)) {
				render.DrawText(screen, fmt.Sprintf("F%d", e.Defcon), int(sx)-8, int(sy)+16, color.RGBA{80, 200, 255, 255}, true)
			}
		}
	}

	for _, t := range a.Engine.FireControl.ActiveTorpedoes {
		if t == nil || !t.Alive {
			continue
		}
		clr := render.ColorDebugAttack
		label := "TORP"
		if t.Class == weapons.ClassUMGT1 {
			switch t.AcousticSig {
			case "set40":
				label = "SET40"
			case "mk46":
				label = "MK46"
			default:
				label = "LW"
			}
		}
		if t.Side == world.SidePlayer {
			clr = render.ColorActive
			label = "MK48"
			if t.WireCut {
				label = "MK48 AUTO"
				clr = color.RGBA{0, 180, 200, 255}
			}
		}
		a.drawDebugEntityAt(screen, view, t.X, t.Y, t.HeadingDeg, t.SpeedKts, clr, true, label)
	}

	for _, h := range a.Engine.FireControl.ActiveHarpoons {
		if h == nil || !h.Alive {
			continue
		}
		clr := color.RGBA{255, 140, 40, 255}
		if h.Side != world.SidePlayer {
			clr = render.ColorDebugAttack
		}
		label := "HSM"
		switch {
		case h.Phase == weapons.HarpoonUnderwater:
			label = "HSM UW"
			clr = color.RGBA{200, 110, 40, 220}
		case h.LockedTargetID != "":
			label = "HSM LCK"
		case h.RadarOn:
			label = "HSM RDR"
		}
		a.drawDebugEntityAt(screen, view, h.X, h.Y, h.HeadingDeg, h.SpeedKts, clr, true, label)
	}

	gt := a.Engine.Clock.GameTime
	for _, rbu := range a.Engine.FireControl.ActiveRBU {
		if rbu == nil || !rbu.Alive {
			continue
		}
		ax, ay := rbu.Pos(gt)
		clr := color.RGBA{255, 90, 40, 255}
		a.drawDebugEntityAt(screen, view, ax, ay, bearingDeg(rbu.X0, rbu.Y0, rbu.X1, rbu.Y1), 0, clr, true, "RBU")
		sx0, sy0 := view.worldToScreen(ax, ay)
		sx1, sy1 := view.worldToScreen(rbu.X1, rbu.Y1)
		render.DrawLine(screen, sx0, sy0, sx1, sy1, color.RGBA{255, 120, 50, 140})
		render.DrawText(screen, a.L(i18n.UISplash), int(sx1)-12, int(sy1)-6, render.ColorAmber, true)
	}
}

func (a *App) drawDebugRoutes(screen *ebiten.Image, view tacticalMapView) {
	routes := a.Engine.Scenario.Routes
	if len(routes) == 0 {
		return
	}
	lineClr := render.ColorDebugRoute
	wpClr := render.ColorDebugRouteWP
	for _, r := range routes {
		if r == nil || len(r.Waypoints) < 2 {
			continue
		}
		n := r.UniqueCount()
		for i := 1; i < n; i++ {
			a0, a1 := r.Waypoints[i-1], r.Waypoints[i]
			x0, y0 := view.worldToScreen(a0.X, a0.Y)
			x1, y1 := view.worldToScreen(a1.X, a1.Y)
			render.DrawLine(screen, x0, y0, x1, y1, lineClr)
		}
		if r.Looped && !r.PingPong && n >= 2 {
			a0, a1 := r.Waypoints[n-1], r.Waypoints[0]
			x0, y0 := view.worldToScreen(a0.X, a0.Y)
			x1, y1 := view.worldToScreen(a1.X, a1.Y)
			render.DrawLine(screen, x0, y0, x1, y1, lineClr)
		}
		for i := 0; i < n; i++ {
			wp := r.Waypoints[i]
			sx, sy := view.worldToScreen(wp.X, wp.Y)
			if !view.containsScreen(int(sx), int(sy)) {
				continue
			}
			render.FillRect(screen, int(sx)-2, int(sy)-2, 5, 5, wpClr)
			if i == 0 || (r.PingPong && i == n-1) {
				label := r.ID
				if r.PingPong && i == n-1 && i != 0 {
					label = r.ID + "⇄"
				}
				render.DrawText(screen, label, int(sx)+6, int(sy)-4, lineClr, true)
			}
		}
	}
}

func debugEntityClass(e *world.Entity) string {
	if p, ok := world.ProfileByID(e.SignatureID); ok {
		return p.DisplayClass()
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

func (a *App) drawDebugEntityAt(screen *ebiten.Image, view tacticalMapView, wx, wy, heading, speedKts float64, clr color.Color, active bool, classLabel string) {
	sx, sy := view.worldToScreen(wx, wy)
	if !view.containsScreen(int(sx), int(sy)) {
		return
	}
	if !active {
		clr = render.ColorDebugInactive
	}

	render.FillRect(screen, int(sx)-3, int(sy)-3, 7, 7, clr)

	rad := heading * math.Pi / 180
	ln := 14.0
	render.DrawLine(screen, sx, sy, sx+math.Sin(rad)*ln, sy-math.Cos(rad)*ln, clr)

	if classLabel != "" {
		render.DrawText(screen, classLabel, int(sx)+8, int(sy)-4, clr, true)
		render.DrawText(screen, fmt.Sprintf("%.0f kt", speedKts), int(sx)+8, int(sy)+8, clr, true)
		return
	}
	render.DrawText(screen, fmt.Sprintf("%.0f kt", speedKts), int(sx)+8, int(sy)-4, clr, true)
}

func debugEntityColor(e *world.Entity) color.Color {
	if !e.Alive() {
		return render.ColorDebugInactive
	}
	if e.Side == world.SideNeutral {
		return color.RGBA{180, 180, 200, 255}
	}
	if e.Side == world.SidePlayer {
		// Ally AI (ownship is drawn separately as the ownship glyph).
		switch e.AIState {
		case "INTERCEPT", "ATTACK", "FIRING", "SHADOW", "CLOSING", "OPENING", "TORPEDO_EVADE", "RBU", "RASTRUB", "SHIP_TUBE", "DATUM":
			return color.RGBA{80, 200, 255, 255}
		default:
			return color.RGBA{40, 160, 255, 255}
		}
	}
	switch e.AIState {
	case "INTERCEPT", "ATTACK", "FIRING", "SHADOW", "CLOSING", "OPENING", "TORPEDO_EVADE", "RBU", "RASTRUB", "SHIP_TUBE":
		return render.ColorDebugAttack
	case "SEARCH", "PINGING", "ACTIVE_SEARCH", "PING_ALERT", "TRACKING", "RADAR_TRACK", "DATUM":
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
