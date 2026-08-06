package ui

import (
	"fmt"
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/layout"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/world"
)

type uiButton struct {
	ID         string
	Label      string
	Tooltip    string
	X, Y, W, H int
}

func (b uiButton) contains(mx, my int) bool {
	return mx >= b.X && mx < b.X+b.W && my >= b.Y && my < b.Y+b.H
}

const (
	helmPanelX = 20
	helmPanelY = 50
	helmPanelW = 1260
	helmPanelH = 700

	helmStationY   = 118
	helmStationH   = 510
	helmStationGap = 16

	helmSpeedX  = 40
	helmSpeedW  = 280
	helmCourseX = helmSpeedX + helmSpeedW + helmStationGap // 336
	helmCourseW = 520
	helmDepthX  = helmCourseX + helmCourseW + helmStationGap // 872
	helmDepthW  = 1260 - helmDepthX                        // 388

	helmRailX = 40
	helmRailY = 644
	helmRailW = 1220
	helmRailH = 86

	depthGaugeW      = 90
	depthMarkPad     = 28
	depthGaugeGap    = 20
	depthTop         = helmStationY + 56
	depthH           = 360
	depthMaxFt       = 800.0
	depthMinOrder    = 60.0
	depthSurfaceFt   = 0.0
	depthPeriscopeFt = depthMinOrder // ~60 ft — periscope depth

	compassCX = float64(helmCourseX + helmCourseW/2) // 596
	compassCY = float64(helmStationY + 220)         // 338
	compassR  = 132.0

	eotBtnH   = 34
	eotBtnGap = 4
)

// depthGaugeX / depthBtn* are centered inside the DEPTH station.
var (
	depthGaugeX int
	depthBtnX   int
	depthBtnW   int
)

// Engine-order telegraph bells. Frac is a signed fraction of MaxSpeedKts
// (negative = astern; clamped by MaxAsternKts in world.Entity.Advance).
type eotBell struct {
	ID, Label, Tooltip string
	Frac               float64
}

var eotBells = []eotBell{
	{ID: "eot_flank", Label: "FLANK", Tooltip: "Flank ahead — maximum speed", Frac: 1.0},
	{ID: "eot_full", Label: "FULL", Tooltip: "Full ahead", Frac: 0.75},
	{ID: "eot_23", Label: "2/3", Tooltip: "Two-thirds ahead", Frac: 0.50},
	{ID: "eot_13", Label: "1/3", Tooltip: "One-third ahead", Frac: 0.25},
	{ID: "eot_stop", Label: "STOP", Tooltip: "All stop", Frac: 0},
	{ID: "eot_13a", Label: "1/3 AST", Tooltip: "One-third astern", Frac: -0.25},
	{ID: "eot_23a", Label: "2/3 AST", Tooltip: "Two-thirds astern", Frac: -0.40},
	{ID: "eot_fulla", Label: "FULL AST", Tooltip: "Full astern", Frac: -0.55},
}

func eotSpeedKts(player *world.Entity, frac float64) float64 {
	if player == nil {
		return 0
	}
	if frac >= 0 {
		return frac * player.MaxSpeedKts()
	}
	return frac / 0.55 * player.MaxAsternKts()
}

func nearestEOTIndex(ordered float64, player *world.Entity) int {
	best, bestDist := 0, math.MaxFloat64
	for i, b := range eotBells {
		d := math.Abs(ordered - eotSpeedKts(player, b.Frac))
		if d < bestDist {
			best, bestDist = i, d
		}
	}
	return best
}

func layoutDepthStation(btnW int) {
	contentW := depthMarkPad + depthGaugeW + depthGaugeGap + btnW
	pad := (helmDepthW - contentW) / 2
	if pad < 10 {
		pad = 10
	}
	depthGaugeX = helmDepthX + pad + depthMarkPad
	depthBtnX = depthGaugeX + depthGaugeW + depthGaugeGap
	depthBtnW = btnW
}

var cachedManeuverButtons struct {
	once sync.Once
	btns []uiButton
}

func maneuverButtons() []uiButton {
	cachedManeuverButtons.once.Do(func() {
		depColW := render.ButtonWidth("PERISCOPE", 12)
		for _, lab := range []string{"▲", "SURFACE", "HOLD", "▼"} {
			if w := render.ButtonWidth(lab, 12); w > depColW {
				depColW = w
			}
		}
		layoutDepthStation(depColW)

		eotW := helmSpeedW - 48
		for _, b := range eotBells {
			if w := render.ButtonWidth(b.Label, 12); w+8 > eotW {
				eotW = w + 8
			}
		}
		eotX := helmSpeedX + (helmSpeedW-eotW)/2
		eotTotalH := len(eotBells)*eotBtnH + (len(eotBells)-1)*eotBtnGap
		eotTop := helmStationY + 56
		if eotTop+eotTotalH > helmStationY+helmStationH-36 {
			eotTop = helmStationY + helmStationH - 36 - eotTotalH
		}

		hdgBtnH := 40
		hdgY := int(compassCY+compassR) + 36

		depTopH, depMidH, depBotH := 44, 30, 44
		depShallowY := depthTop
		depSurfaceY := depShallowY + depTopH + 8
		depPeriscopeY := depSurfaceY + depMidH + 6
		depDeepY := depthTop + depthH - depBotH
		depHoldY := depPeriscopeY + depMidH + 20
		if depHoldY+depTopH > depDeepY-12 {
			depHoldY = depDeepY - depTopH - 12
		}

		btW := render.ButtonWidth("BT CAST", 12)
		btX := helmRailX + helmRailW - btW - 20
		btY := helmRailY + (helmRailH-36)/2

		buttons := make([]uiButton, 0, len(eotBells)+10)
		for i, bell := range eotBells {
			buttons = append(buttons, uiButton{
				ID: bell.ID, Label: bell.Label, Tooltip: bell.Tooltip,
				X: eotX, Y: eotTop + i*(eotBtnH+eotBtnGap), W: eotW, H: eotBtnH,
			})
		}
		buttons = append(buttons,
			uiButton{ID: "hdg_port10", Label: "◄◄", Tooltip: "Come left 10 degrees", Y: hdgY, H: hdgBtnH},
			uiButton{ID: "hdg_port", Label: "◄ PORT", Tooltip: "Come left 5 degrees", Y: hdgY, H: hdgBtnH},
			uiButton{ID: "hdg_stbd", Label: "STBD ►", Tooltip: "Come right 5 degrees", Y: hdgY, H: hdgBtnH},
			uiButton{ID: "hdg_stbd10", Label: "►►", Tooltip: "Come right 10 degrees", Y: hdgY, H: hdgBtnH},
			uiButton{ID: "dep_shallow", Label: "▲", Tooltip: "Rise 20 feet (shallower)", X: depthBtnX, Y: depShallowY, H: depTopH},
			uiButton{ID: "dep_surface", Label: "SURFACE", Tooltip: "Surface — order zero depth", X: depthBtnX, Y: depSurfaceY, H: depMidH},
			uiButton{ID: "dep_periscope", Label: "PERISCOPE", Tooltip: "Periscope depth — order 60 feet", X: depthBtnX, Y: depPeriscopeY, H: depMidH},
			uiButton{ID: "dep_hold", Label: "HOLD", Tooltip: "Hold present depth", X: depthBtnX, Y: depHoldY, H: depTopH},
			uiButton{ID: "dep_deep", Label: "▼", Tooltip: "Dive 20 feet (deeper)", X: depthBtnX, Y: depDeepY, H: depBotH},
			uiButton{ID: "bt_cast", Label: "BT CAST", Tooltip: "Launch SSXBT — survey thermocline (~15 s sim time)", X: btX, Y: btY, H: 36},
		)
		for i := range buttons {
			if buttons[i].W == 0 {
				buttons[i].W = render.ButtonWidth(buttons[i].Label, 12)
			}
		}

		hdgIDs := []string{"hdg_port10", "hdg_port", "hdg_stbd", "hdg_stbd10"}
		hdgTotal := 0
		for _, id := range hdgIDs {
			for i := range buttons {
				if buttons[i].ID == id {
					hdgTotal += buttons[i].W
					break
				}
			}
		}
		hdgTotal += 6 * (len(hdgIDs) - 1)
		x := helmCourseX + (helmCourseW-hdgTotal)/2
		for i := range buttons {
			switch buttons[i].ID {
			case "hdg_port10", "hdg_port", "hdg_stbd", "hdg_stbd10":
				buttons[i].X = x
				x += buttons[i].W + 6
			case "dep_shallow", "dep_hold", "dep_deep", "dep_surface", "dep_periscope":
				buttons[i].X = depthBtnX
				buttons[i].W = depthBtnW
			}
		}
		cachedManeuverButtons.btns = buttons
	})
	return cachedManeuverButtons.btns
}

func compassHeadingAt(mx, my int) (float64, bool) {
	dx := float64(mx) - compassCX
	dy := compassCY - float64(my)
	dist := math.Hypot(dx, dy)
	if dist < 24 || dist > compassR+8 {
		return 0, false
	}
	deg := math.Atan2(dx, dy) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg, true
}

func depthGaugeContains(mx, my int) bool {
	return mx >= depthGaugeX && mx < depthGaugeX+depthGaugeW+30 &&
		my >= depthTop && my <= depthTop+depthH
}

func depthFromGaugeY(my int) float64 {
	ft := float64(my-depthTop) / float64(depthH) * depthMaxFt
	if ft < depthMinOrder {
		ft = depthMinOrder
	}
	if ft > depthMaxFt {
		ft = depthMaxFt
	}
	return math.Round(ft/5) * 5
}

func depthToGaugeY(ft float64) int {
	if ft < 0 {
		ft = 0
	}
	if ft > depthMaxFt {
		ft = depthMaxFt
	}
	return depthTop + int(ft/depthMaxFt*float64(depthH))
}

func (a *App) clampOrderedDepth(player *world.Entity, want float64) (depth float64, limited bool, bottomFt float64) {
	if want < depthMinOrder {
		want = depthMinOrder
	}
	if want > depthMaxFt {
		want = depthMaxFt
	}
	if a.Engine == nil || a.Engine.Scenario == nil || a.Engine.Scenario.Bathy == nil || !a.Engine.Scenario.Bathy.Valid() || player == nil {
		return want, false, 0
	}
	bot := a.Engine.Scenario.Bathy.DepthAtFt(player.X, player.Y)
	if bot <= 0 {
		return want, false, 0
	}
	maxDepth := bot - 50
	if maxDepth < depthMinOrder {
		maxDepth = depthMinOrder
	}
	if want > maxDepth {
		return maxDepth, true, bot
	}
	return want, false, bot
}

func (a *App) orderMakeDepth(player *world.Entity, want float64) {
	depth, limited, bottomFt := a.clampOrderedDepth(player, want)
	player.OrderedDepth = depth
	if limited {
		a.Audio.PlayClip(audio.ClipDiveUnableDeeper,
			fmt.Sprintf("Unable to dive deeper here — bottom %.0f ft, max safe depth %.0f ft.", bottomFt, depth))
		return
	}
	a.Audio.PlayClip(audio.ClipDiveMakeDepth, fmt.Sprintf("Make depth %d feet.", int(depth)))
}

func drawDashedHLine(screen *ebiten.Image, x1, x2, y float64, clr color.Color, dash, gap float64) {
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	x := x1
	for x < x2 {
		xEnd := x + dash
		if xEnd > x2 {
			xEnd = x2
		}
		render.DrawLine(screen, x, y, xEnd, y, clr)
		x = xEnd + gap
	}
}

func (a *App) playOrderedHeadingVoice(player *world.Entity) {
	head := int(player.OrderedHead)
	diff := player.OrderedHead - player.HeadingDeg
	for diff > 180 {
		diff -= 360
	}
	for diff < -180 {
		diff += 360
	}
	switch {
	case diff < -0.5:
		a.Audio.PlayClip(audio.ClipDiveComeLeft, fmt.Sprintf("Come left to %d.", head))
	case diff > 0.5:
		a.Audio.PlayClip(audio.ClipDiveComeRight, fmt.Sprintf("Come right to %d.", head))
	}
}

func (a *App) stepEOT(player *world.Entity, delta int) {
	idx := nearestEOTIndex(player.OrderedSpeed, player) + delta
	if idx < 0 {
		idx = 0
	}
	if idx >= len(eotBells) {
		idx = len(eotBells) - 1
	}
	a.maneuverButtonAction(eotBells[idx].ID, player)
}

func (a *App) updateManeuverUI(player *world.Entity) {
	buttons := maneuverButtons()
	mx, my := ebiten.CursorPosition()
	a.updateButtonTooltips(buttons, mx, my)

	clickedButton := false
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for _, b := range buttons {
			if b.contains(mx, my) {
				a.maneuverButtonAction(b.ID, player)
				a.uiPressedID = b.ID
				a.uiPressedAt = time.Now()
				clickedButton = true
				break
			}
		}
		if !clickedButton {
			if depthGaugeContains(mx, my) {
				a.orderMakeDepth(player, depthFromGaugeY(my))
				clickedButton = true
			} else if hdg, ok := compassHeadingAt(mx, my); ok {
				a.compassDrag = true
				player.OrderedHead = hdg
			}
		}
	}
	if a.compassDrag && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if hdg, ok := compassHeadingAt(mx, my); ok {
			player.OrderedHead = hdg
		}
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && a.compassDrag {
		a.compassDrag = false
		a.playOrderedHeadingVoice(player)
	}
	if a.uiPressedID != "" && time.Since(a.uiPressedAt) > 120*time.Millisecond {
		a.uiPressedID = ""
	}

	// Arrow keys step the engine-order telegraph (ahead ↑ / astern ↓).
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		a.stepEOT(player, -1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		a.stepEOT(player, +1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		a.maneuverButtonAction("hdg_port", player)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		a.maneuverButtonAction("hdg_stbd", player)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		a.maneuverButtonAction("dep_shallow", player)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		a.maneuverButtonAction("dep_deep", player)
	}
}

func (a *App) ringEOT(player *world.Entity, frac float64) {
	next := eotSpeedKts(player, frac)
	sonar := &a.Engine.Sonar
	if next > 0 && !sonar.TowedDamaged && sonar.TowedCablePct >= 0.20 {
		warnAt := acoustics.TowedWarnSpeedKts(sonar.TowedCablePct)
		shearAt := acoustics.TowedShearSpeedKts(sonar.TowedCablePct)
		if next >= shearAt || math.Abs(player.SpeedKts) >= warnAt {
			a.StatusMessage = fmt.Sprintf(
				"WARNING: towed cable stress — shear risk above %.0f kn (now %.0f ordered).",
				shearAt, next)
		}
	}
	player.OrderedSpeed = next
}

func (a *App) maneuverButtonAction(id string, player *world.Entity) {
	switch id {
	case "eot_flank", "eot_full", "eot_23", "eot_13", "eot_stop", "eot_13a", "eot_23a", "eot_fulla":
		for _, b := range eotBells {
			if b.ID == id {
				a.ringEOT(player, b.Frac)
				return
			}
		}
	case "hdg_port":
		player.OrderedHead -= 5
		if player.OrderedHead < 0 {
			player.OrderedHead += 360
		}
		a.Audio.PlayClip(audio.ClipDiveComeLeft, fmt.Sprintf("Come left to %d.", int(player.OrderedHead)))
	case "hdg_stbd":
		player.OrderedHead += 5
		if player.OrderedHead >= 360 {
			player.OrderedHead -= 360
		}
		a.Audio.PlayClip(audio.ClipDiveComeRight, fmt.Sprintf("Come right to %d.", int(player.OrderedHead)))
	case "hdg_port10":
		player.OrderedHead -= 10
		if player.OrderedHead < 0 {
			player.OrderedHead += 360
		}
		a.Audio.PlayClip(audio.ClipDiveComeLeft, fmt.Sprintf("Come left to %d.", int(player.OrderedHead)))
	case "hdg_stbd10":
		player.OrderedHead += 10
		if player.OrderedHead >= 360 {
			player.OrderedHead -= 360
		}
		a.Audio.PlayClip(audio.ClipDiveComeRight, fmt.Sprintf("Come right to %d.", int(player.OrderedHead)))
	case "dep_shallow":
		a.orderMakeDepth(player, math.Max(depthMinOrder, player.OrderedDepth-20))
	case "dep_deep":
		a.orderMakeDepth(player, math.Min(depthMaxFt, player.OrderedDepth+20))
	case "dep_hold":
		player.OrderedDepth = player.DepthFt
		a.Audio.PlayClip(audio.ClipDiveHoldDepth, fmt.Sprintf("Hold depth %d feet.", int(player.OrderedDepth)))
	case "dep_surface":
		player.OrderedDepth = depthSurfaceFt
		a.Audio.PlayClip(audio.ClipDiveMakeDepth, "Surface the ship.")
	case "dep_periscope":
		a.orderMakeDepth(player, depthPeriscopeFt)
	case "bt_cast":
		env := &a.Engine.Acoustics.Env
		gt := a.Engine.Clock.GameTime
		if env.LayerSurveyActive(gt) {
			a.StatusMessage = fmt.Sprintf("BT cast in progress — %.0fs remaining.", env.LayerSurveyRemainingSec(gt))
			return
		}
		wasKnown := env.LayerSurveyKnown
		if env.StartLayerSurvey(gt) {
			msg := fmt.Sprintf("SSXBT launched — surveying layers (~%.0fs).", acoustics.LayerSurveyDurationSec)
			if wasKnown {
				msg = fmt.Sprintf("SSXBT re-cast — refreshing layer profile (~%.0fs).", acoustics.LayerSurveyDurationSec)
			}
			a.StatusMessage = msg
			a.Audio.PlayClip(audio.ClipSonarBTLaunch, "Launching bathythermograph.")
		} else {
			a.StatusMessage = "Unable to start BT cast."
		}
	}
}

func (a *App) updateButtonTooltips(buttons []uiButton, mx, my int) {
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
	if hoverID != "" && now.Sub(a.uiHoverSince) >= 2*time.Second {
		for _, b := range buttons {
			if b.ID == hoverID {
				a.uiTooltip = b.Tooltip
				break
			}
		}
	}
}

func drawHelmStationHeader(screen *ebiten.Image, x, y, w int, title, hint string) {
	render.DrawText(screen, title, x+16, y+22, render.ColorPlateLabel, true)
	render.DrawText(screen, hint, x+16, y+40, render.ColorPhosphorDim, true)
	_ = w
}

func (a *App) drawManeuver(screen *ebiten.Image) {
	p := a.Engine.Scenario.Player
	env := a.Engine.Acoustics.Env
	gt := a.Engine.Clock.GameTime
	_ = maneuverButtons() // ensure depth layout vars are initialized

	render.DrawConsolePanel(screen, helmPanelX, helmPanelY, helmPanelW, helmPanelH)
	render.DrawScreenTitle(screen, "MANEUVERING ROOM — HELM", layout.PassiveTitleLabelX, layout.PassiveTitleLabelY+20)

	render.DrawMonitor(screen, helmSpeedX, helmStationY, helmSpeedW, helmStationH)
	render.DrawMonitor(screen, helmCourseX, helmStationY, helmCourseW, helmStationH)
	render.DrawMonitor(screen, helmDepthX, helmStationY, helmDepthW, helmStationH)
	render.DrawMonitor(screen, helmRailX, helmRailY, helmRailW, helmRailH)

	drawHelmStationHeader(screen, helmSpeedX, helmStationY, helmSpeedW, "SPEED", "ENGINE ORDER")
	drawHelmStationHeader(screen, helmCourseX, helmStationY, helmCourseW, "COURSE", "CLICK OR DRAG")
	render.DrawText(screen, fmt.Sprintf("ACT %.0f°", p.HeadingDeg), helmCourseX+16, helmStationY+56, render.ColorPhosphor, true)
	render.DrawText(screen, fmt.Sprintf("ORD %.0f°", p.OrderedHead), helmCourseX+16, helmStationY+74, render.ColorAmber, true)
	drawHelmStationHeader(screen, helmDepthX, helmStationY, helmDepthW, "DEPTH", "CLICK SCALE")

	mx, my := ebiten.CursorPosition()

	// --- COURSE gyro ---
	drawCompassRose(screen, compassCX, compassCY, compassR, p.HeadingDeg, p.OrderedHead)
	if hdg, ok := compassHeadingAt(mx, my); ok {
		rad := hdg * math.Pi / 180
		ix := compassCX + math.Sin(rad)*(compassR-6)
		iy := compassCY - math.Cos(rad)*(compassR-6)
		render.FillRect(screen, int(ix)-4, int(iy)-4, 8, 8, render.ColorHighlight)
	}

	// --- DEPTH gauge ---
	render.FillRect(screen, depthGaugeX, depthTop, depthGaugeW, depthH+16, render.ColorPanelInset)
	for _, mark := range []float64{0, 200, 400, 600, 800} {
		y := depthToGaugeY(mark)
		render.DrawLine(screen, float64(depthGaugeX), float64(y), float64(depthGaugeX+15), float64(y), render.ColorGrid)
		render.DrawText(screen, fmt.Sprintf("%.0f", mark), depthGaugeX-depthMarkPad, y+4, render.ColorPhosphorDim, true)
	}
	for _, bd := range env.KnownBoundaryDepthsFt() {
		if bd > depthMaxFt {
			continue
		}
		y := float64(depthToGaugeY(bd))
		drawDashedHLine(screen, float64(depthGaugeX+4), float64(depthGaugeX+depthGaugeW-4), y, color.RGBA{0, 200, 160, 200}, 6, 4)
	}
	oy := float64(depthToGaugeY(p.OrderedDepth))
	render.DrawLine(screen, float64(depthGaugeX+20), oy, float64(depthGaugeX+depthGaugeW-4), oy, render.ColorAmber)
	drawDepthSubIcon(screen, depthGaugeX+depthGaugeW/2, depthToGaugeY(p.DepthFt))
	if depthGaugeContains(mx, my) {
		hy := float64(depthToGaugeY(depthFromGaugeY(my)))
		render.DrawLine(screen, float64(depthGaugeX+4), hy, float64(depthGaugeX+depthGaugeW-4), hy, color.RGBA{255, 255, 100, 120})
	}
	depthReadY := depthTop + depthH + 28
	render.DrawText(screen, fmt.Sprintf("%.0f FT", p.DepthFt), depthGaugeX, depthReadY, render.ColorPhosphor, false)
	render.DrawText(screen, fmt.Sprintf("ORD %.0f", p.OrderedDepth), depthGaugeX, depthReadY+18, render.ColorAmber, true)
	clearance := env.KeelClearanceFt(p.DepthFt)
	render.DrawText(screen, fmt.Sprintf("KEEL %.0f FT", clearance), depthGaugeX, depthReadY+36, render.ColorPhosphorDim, true)

	buttons := maneuverButtons()
	activeBell := eotBells[nearestEOTIndex(p.OrderedSpeed, p)].ID
	for _, b := range buttons {
		hover := b.contains(mx, my)
		pressed := a.uiPressedID == b.ID || b.ID == activeBell
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	}

	// ACT / ORD under the telegraph dial.
	actY := helmStationY + helmStationH - 18
	actLbl := fmt.Sprintf("ACT %.1f", p.SpeedKts)
	if p.SpeedKts < -0.05 {
		actLbl = fmt.Sprintf("ACT %.1f AST", -p.SpeedKts)
	}
	ordLbl := fmt.Sprintf("ORD %.0f", p.OrderedSpeed)
	if p.OrderedSpeed < -0.05 {
		ordLbl = fmt.Sprintf("ORD %.0f AST", -p.OrderedSpeed)
	}
	render.DrawText(screen, actLbl, helmSpeedX+16, actY, render.ColorPhosphor, true)
	render.DrawText(screen, ordLbl, helmSpeedX+16+render.ButtonLabelWidth(actLbl)+16, actY, render.ColorAmber, true)

	// --- Status rail ---
	railTextX := helmRailX + 16
	railTextY := helmRailY + 22
	layer := env.LayerNameKnown(p.DepthFt)
	render.DrawText(screen, "WATER LAYER: "+layer, railTextX, railTextY, render.ColorPhosphorDim, false)
	if env.LayerSurveyActive(gt) {
		render.DrawText(screen, fmt.Sprintf("BT CAST: %.0fs remaining", env.LayerSurveyRemainingSec(gt)), railTextX, railTextY+18, render.ColorAmber, true)
	} else if env.LayerSurveyKnown {
		render.DrawText(screen, "LAYER PROFILE: ON FILE", railTextX, railTextY+18, render.ColorPhosphorDim, true)
	} else {
		render.DrawText(screen, "LAYER PROFILE: UNKNOWN — launch BT CAST", railTextX, railTextY+18, render.ColorWarn, true)
	}
	cav := acoustics.CavitationSeverity(p.DepthFt, p.SpeedKts)
	warnY := railTextY + 36
	if cav > 0.15 {
		render.DrawText(screen, fmt.Sprintf("CAVITATION RISK %.0f%%", cav*100), railTextX, warnY, render.ColorWarn, false)
		warnY += 16
	}
	p.EnsureDamage()
	if p.Damage.Destroyed(world.SysSteering) {
		render.DrawText(screen, "STEERING DAMAGED — RUDDER JAMMED", railTextX+280, railTextY, render.ColorDanger, true)
	}
	if p.Damage.Destroyed(world.SysDepth) {
		render.DrawText(screen, "DEPTH CONTROL DAMAGED — UNCONTROLLED TRIM", railTextX+280, railTextY+16, render.ColorDanger, true)
	}
	if p.Damage.Destroyed(world.SysPropulsion) {
		render.DrawText(screen, "PROPULSION DESTROYED — NO THRUST", railTextX+280, railTextY+32, render.ColorDanger, true)
	} else if !p.Damage.Operational(world.SysPropulsion) {
		render.DrawText(screen, fmt.Sprintf("PROPULSION DEGRADED — MAX %.0f KTS", p.MaxSpeedKts()), railTextX+280, railTextY+32, render.ColorWarn, true)
	}
	_ = warnY

	a.drawBTProgress(screen, env, gt)

	if a.uiTooltip != "" {
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
}

func (a *App) drawBTProgress(screen *ebiten.Image, env acoustics.Environment, gt float64) {
	btW := render.ButtonWidth("BT CAST", 12)
	btX := helmRailX + helmRailW - btW - 20
	const (
		barW = 160
		barH = 22
	)
	barX := btX - barW - 16
	barY := helmRailY + (helmRailH-barH)/2

	if !env.LayerSurveyActive(gt) {
		render.DrawText(screen, fmt.Sprintf("BOTTOM %.0f FT", env.BottomDepthFt), barX, barY+barH/2+4, render.ColorPhosphorDim, true)
		return
	}
	prog := env.LayerSurveyProgress(gt)
	render.DrawText(screen, "BT / LAYERS", barX, barY-2, render.ColorPhosphorDim, true)
	render.FillRect(screen, barX, barY+2, barW, barH, render.ColorPanelInset)
	fill := int(float64(barW-4) * prog)
	if fill > 0 {
		render.FillRect(screen, barX+2, barY+4, fill, barH-4, render.ColorAmber)
	}
	render.DrawText(screen, fmt.Sprintf("%.0f%%", prog*100), barX+barW/2-16, barY+barH/2+6, render.ColorPhosphor, true)
}

func drawDepthSubIcon(screen *ebiten.Image, cx, cy int) {
	// Cold Waters CONDITIONS-panel style: long low profile, bow right,
	// cylindrical top-to-bottom phosphor shading.
	hi := color.RGBA{90, 255, 170, 255}
	mid := color.RGBA{0, 210, 110, 255}
	lo := color.RGBA{0, 130, 75, 255}
	dim := color.RGBA{0, 95, 55, 255}

	type span struct{ x0, x1 int }
	rows := []struct {
		s span
		c color.RGBA
	}{
		{span{-22, 22}, lo},
		{span{-26, 26}, hi},
		{span{-28, 28}, mid},
		{span{-27, 27}, mid},
		{span{-25, 25}, lo},
		{span{-18, 20}, dim},
	}
	for i, r := range rows {
		y := cy - 2 + i
		render.FillRect(screen, cx+r.s.x0, y, r.s.x1-r.s.x0+1, 1, r.c)
	}
	render.FillRect(screen, cx+28, cy-1, 3, 3, mid)
	render.FillRect(screen, cx+31, cy, 2, 2, hi)
	render.FillRect(screen, cx+33, cy, 1, 1, lo)
	render.FillRect(screen, cx-30, cy, 4, 2, mid)
	render.FillRect(screen, cx-32, cy+1, 3, 1, lo)

	sx := cx + 10
	render.FillRect(screen, sx, cy-8, 6, 6, mid)
	render.FillRect(screen, sx, cy-8, 6, 2, hi)
	render.FillRect(screen, sx, cy-4, 6, 2, lo)
	render.FillRect(screen, sx-1, cy-6, 8, 1, mid)
	render.FillRect(screen, sx+4, cy-9, 1, 1, hi)

	render.FillRect(screen, cx-28, cy-4, 2, 9, mid)
	render.FillRect(screen, cx-31, cy+1, 7, 1, lo)
}

func drawCompassRose(screen *ebiten.Image, cx, cy, radius float64, heading, ordered float64) {
	for deg := 0; deg < 360; deg += 30 {
		rad := float64(deg) * math.Pi / 180
		inner := radius - 12
		if deg%90 == 0 {
			inner = radius - 22
		}
		x1 := cx + math.Sin(rad)*inner
		y1 := cy - math.Cos(rad)*inner
		x2 := cx + math.Sin(rad)*radius
		y2 := cy - math.Cos(rad)*radius
		render.DrawLine(screen, x1, y1, x2, y2, render.ColorPhosphorDim)
	}
	render.DrawLine(screen, cx-radius, cy, cx+radius, cy, render.ColorGrid)
	render.DrawLine(screen, cx, cy-radius, cx, cy+radius, render.ColorGrid)

	hrad := heading * math.Pi / 180
	render.DrawLine(screen, cx, cy, cx+math.Sin(hrad)*(radius-28), cy-math.Cos(hrad)*(radius-28), render.ColorPhosphor)
	ord := ordered * math.Pi / 180
	render.DrawLine(screen, cx, cy, cx+math.Sin(ord)*(radius-50), cy-math.Cos(ord)*(radius-50), render.ColorAmber)

	labels := map[int]string{0: "N", 90: "E", 180: "S", 270: "W"}
	for deg, lbl := range labels {
		rad := float64(deg) * math.Pi / 180
		lx := cx + math.Sin(rad)*(radius+16)
		ly := cy - math.Cos(rad)*(radius+16)
		render.DrawText(screen, lbl, int(lx)-6, int(ly)+4, render.ColorPhosphor, false)
	}
	render.DrawText(screen, fmt.Sprintf("%.0f°", heading), int(cx)-24, int(cy)+6, render.ColorPhosphor, false)
}
