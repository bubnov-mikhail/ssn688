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
	depthGaugeX   = 880
	depthGaugeW   = 100
	depthTop      = 190
	depthH        = 380
	depthMaxFt    = 800.0
	depthMinOrder = 60.0
)

var colorDepthActual = color.RGBA{160, 160, 160, 255}

var cachedManeuverButtons struct {
	once sync.Once
	btns []uiButton
}

func maneuverButtons() []uiButton {
	cachedManeuverButtons.once.Do(func() {
		buttons := []uiButton{
			{ID: "spd_down", Label: "−", Tooltip: "Reduce ordered speed by 1 knot", X: 80, Y: 280, H: 44},
			{ID: "spd_up", Label: "+", Tooltip: "Increase ordered speed by 1 knot", X: 80, Y: 220, H: 44},
			{ID: "spd_stop", Label: "STOP", Tooltip: "All stop — ordered speed zero", X: 80, Y: 340, H: 44},
			{ID: "hdg_port10", Label: "◄◄", Tooltip: "Come left 10 degrees", Y: 520, H: 40},
			{ID: "hdg_port", Label: "◄ PORT", Tooltip: "Come left 5 degrees", Y: 520, H: 40},
			{ID: "hdg_stbd", Label: "STBD ►", Tooltip: "Come right 5 degrees", Y: 520, H: 40},
			{ID: "hdg_stbd10", Label: "►►", Tooltip: "Come right 10 degrees", Y: 520, H: 40},
			{ID: "dep_shallow", Label: "▲", Tooltip: "Rise 20 feet (shallower)", X: 980, Y: 200, H: 44},
			{ID: "dep_deep", Label: "▼", Tooltip: "Dive 20 feet (deeper)", X: 980, Y: 520, H: 44},
			{ID: "dep_hold", Label: "HOLD", Tooltip: "Hold present depth", X: 980, Y: 360, H: 44},
			{ID: "bt_cast", Label: "BT CAST", Tooltip: "Launch SSXBT — survey thermocline (~15 s sim time)", X: 780, Y: 665, H: 36},
		}
		for i := range buttons {
			buttons[i].W = render.ButtonWidth(buttons[i].Label, 12)
		}
		x := 330
		for i := range buttons {
			switch buttons[i].ID {
			case "hdg_port10", "hdg_port", "hdg_stbd", "hdg_stbd10":
				buttons[i].X = x
				x += buttons[i].W + 6
			}
		}
		cachedManeuverButtons.btns = buttons
	})
	return cachedManeuverButtons.btns
}

const (
	compassCX = 570.0
	compassCY = 340.0
	compassR  = 150.0
)

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

	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		player.OrderedSpeed = math.Min(30, player.OrderedSpeed+0.4)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		player.OrderedSpeed = math.Max(0, player.OrderedSpeed-0.4)
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

func (a *App) maneuverButtonAction(id string, player *world.Entity) {
	switch id {
	case "spd_up":
		next := math.Min(30, player.OrderedSpeed+1)
		sonar := &a.Engine.Sonar
		if !sonar.TowedDamaged && sonar.TowedCablePct >= 0.20 {
			warnAt := acoustics.TowedWarnSpeedKts(sonar.TowedCablePct)
			shearAt := acoustics.TowedShearSpeedKts(sonar.TowedCablePct)
			if next >= shearAt || player.SpeedKts >= warnAt {
				a.StatusMessage = fmt.Sprintf(
					"WARNING: towed cable stress — shear risk above %.0f kn (now %.0f ordered).",
					shearAt, next)
			}
		}
		player.OrderedSpeed = next
	case "spd_down":
		player.OrderedSpeed = math.Max(0, player.OrderedSpeed-1)
	case "spd_stop":
		player.OrderedSpeed = 0
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

func (a *App) drawManeuver(screen *ebiten.Image) {
	p := a.Engine.Scenario.Player
	env := a.Engine.Acoustics.Env
	gt := a.Engine.Clock.GameTime
	px, py, pw, ph := 20, 50, 1100, 700
	render.DrawPanel(screen, px, py, pw, ph)

	render.DrawTextLarge(screen, "MANEUVERING ROOM — HELM", 48, 88, render.ColorPhosphor)
	render.DrawText(screen, "ENGINE ORDER TELEGRAPH", 60, 170, render.ColorPhosphorDim, true)

	render.FillRect(screen, 160, 200, 180, 120, render.ColorPanelInset)
	render.DrawTextLarge(screen, fmt.Sprintf("%.0f", p.OrderedSpeed), 200, 280, render.ColorAmber)
	render.DrawText(screen, "KTS ORDERED", 188, 300, render.ColorPhosphorDim, true)
	render.DrawText(screen, fmt.Sprintf("ACT %.1f", p.SpeedKts), 188, 318, render.ColorPhosphor, true)

	buttons := maneuverButtons()
	mx, my := ebiten.CursorPosition()
	for _, b := range buttons {
		hover := b.contains(mx, my)
		pressed := a.uiPressedID == b.ID
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	}

	drawCompassRose(screen, compassCX, compassCY, compassR, p.HeadingDeg, p.OrderedHead)
	if hdg, ok := compassHeadingAt(mx, my); ok {
		rad := hdg * math.Pi / 180
		ix := compassCX + math.Sin(rad)*(compassR-6)
		iy := compassCY - math.Cos(rad)*(compassR-6)
		render.FillRect(screen, int(ix)-4, int(iy)-4, 8, 8, render.ColorHighlight)
	}
	render.DrawText(screen, "CLICK OR DRAG TO SET COURSE", int(compassCX)-95, int(compassCY+compassR)+28, render.ColorPhosphorDim, true)

	render.DrawText(screen, "CLICK SCALE TO SET DEPTH", depthGaugeX-10, 160, render.ColorPhosphorDim, true)
	render.DrawText(screen, "DEPTH GAUGE", depthGaugeX, 170, render.ColorPhosphorDim, true)
	render.FillRect(screen, depthGaugeX, depthTop, depthGaugeW, depthH+20, render.ColorPanelInset)
	for _, mark := range []float64{0, 200, 400, 600, 800} {
		y := depthToGaugeY(mark)
		render.DrawLine(screen, float64(depthGaugeX), float64(y), float64(depthGaugeX+15), float64(y), render.ColorGrid)
		render.DrawText(screen, fmt.Sprintf("%.0f", mark), depthGaugeX-32, y+4, render.ColorPhosphorDim, true)
	}

	for _, bd := range env.KnownBoundaryDepthsFt() {
		if bd > depthMaxFt {
			continue
		}
		y := float64(depthToGaugeY(bd))
		drawDashedHLine(screen, float64(depthGaugeX+4), float64(depthGaugeX+depthGaugeW-4), y, color.RGBA{0, 200, 160, 200}, 6, 4)
		render.DrawText(screen, fmt.Sprintf("%.0f", bd), depthGaugeX+depthGaugeW+4, int(y)+4, render.ColorPhosphorDim, true)
	}

	oy := float64(depthToGaugeY(p.OrderedDepth))
	render.DrawLine(screen, float64(depthGaugeX+20), oy, float64(depthGaugeX+depthGaugeW-4), oy, render.ColorAmber)
	drawDepthSubIcon(screen, depthGaugeX+depthGaugeW/2, depthToGaugeY(p.DepthFt))
	if depthGaugeContains(mx, my) {
		hy := float64(depthToGaugeY(depthFromGaugeY(my)))
		render.DrawLine(screen, float64(depthGaugeX+4), hy, float64(depthGaugeX+depthGaugeW-4), hy, color.RGBA{255, 255, 100, 120})
	}

	render.DrawText(screen, fmt.Sprintf("%.0f FT", p.DepthFt), depthGaugeX, 600, render.ColorPhosphor, false)
	render.DrawText(screen, fmt.Sprintf("ORD %.0f", p.OrderedDepth), depthGaugeX, 618, render.ColorAmber, true)
	clearance := env.KeelClearanceFt(p.DepthFt)
	render.DrawText(screen, fmt.Sprintf("KEEL %.0f FT", clearance), depthGaugeX, 636, render.ColorPhosphorDim, true)

	layer := env.LayerNameKnown(p.DepthFt)
	render.DrawText(screen, "WATER LAYER: "+layer, 60, 600, render.ColorPhosphorDim, false)
	if env.LayerSurveyActive(gt) {
		render.DrawText(screen, fmt.Sprintf("BT CAST: %.0fs remaining", env.LayerSurveyRemainingSec(gt)), 60, 622, render.ColorAmber, true)
	} else if env.LayerSurveyKnown {
		render.DrawText(screen, "LAYER PROFILE: ON FILE", 60, 622, render.ColorPhosphorDim, true)
	} else {
		render.DrawText(screen, "LAYER PROFILE: UNKNOWN — launch BT CAST", 60, 622, render.ColorWarn, true)
	}
	cav := acoustics.CavitationSeverity(p.DepthFt, p.SpeedKts)
	if cav > 0.15 {
		render.DrawText(screen, fmt.Sprintf("CAVITATION RISK %.0f%%", cav*100), 60, 644, render.ColorWarn, false)
	}

	p.EnsureDamage()
	warnY := 666
	if p.Damage.Destroyed(world.SysSteering) {
		render.DrawText(screen, "STEERING DAMAGED — RUDDER JAMMED", 60, warnY, render.ColorDanger, true)
		warnY += 16
	}
	if p.Damage.Destroyed(world.SysDepth) {
		render.DrawText(screen, "DEPTH CONTROL DAMAGED — UNCONTROLLED TRIM", 60, warnY, render.ColorDanger, true)
		warnY += 16
	}
	if p.Damage.Destroyed(world.SysPropulsion) {
		render.DrawText(screen, "PROPULSION DESTROYED — NO THRUST", 60, warnY, render.ColorDanger, true)
	} else if !p.Damage.Operational(world.SysPropulsion) {
		render.DrawText(screen, fmt.Sprintf("PROPULSION DEGRADED — MAX %.0f KTS", p.MaxSpeedKts()), 60, warnY, render.ColorWarn, true)
	}

	a.drawBTProgress(screen, env, gt)

	if a.uiTooltip != "" {
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
}

func (a *App) drawBTProgress(screen *ebiten.Image, env acoustics.Environment, gt float64) {
	const (
		barX = 880
		barY = 668
		barW = 200
		barH = 28
	)
	// Progress bar only while a cast is running; hide after completion.
	if !env.LayerSurveyActive(gt) {
		return
	}
	prog := env.LayerSurveyProgress(gt)
	render.DrawText(screen, "BT / LAYERS", barX, barY-4, render.ColorPhosphorDim, true)
	render.FillRect(screen, barX, barY, barW, barH, render.ColorPanelInset)
	fill := int(float64(barW-4) * prog)
	if fill > 0 {
		render.FillRect(screen, barX+2, barY+2, fill, barH-4, render.ColorAmber)
	}
	render.DrawText(screen, fmt.Sprintf("%.0f%%", prog*100), barX+barW/2-20, barY+18, render.ColorPhosphor, true)
	render.DrawText(screen, fmt.Sprintf("BOTTOM %.0f FT", env.BottomDepthFt), barX, barY+barH+14, render.ColorPhosphorDim, true)
}

func drawDepthSubIcon(screen *ebiten.Image, cx, cy int) {
	// Cold Waters CONDITIONS-panel style: long low profile, bow right,
	// cylindrical top-to-bottom phosphor shading.
	hi := color.RGBA{90, 255, 170, 255}
	mid := color.RGBA{0, 210, 110, 255}
	lo := color.RGBA{0, 130, 75, 255}
	dim := color.RGBA{0, 95, 55, 255}

	// Hull scanlines relative to (cx, cy). Matches CW ~60×7 cigar.
	type span struct{ x0, x1 int }
	rows := []struct {
		s span
		c color.RGBA
	}{
		{span{-22, 22}, lo},  // thin top rim
		{span{-26, 26}, hi},  // specular band
		{span{-28, 28}, mid}, // widest mid
		{span{-27, 27}, mid},
		{span{-25, 25}, lo}, // bottom shade
		{span{-18, 20}, dim},
	}
	for i, r := range rows {
		y := cy - 2 + i
		render.FillRect(screen, cx+r.s.x0, y, r.s.x1-r.s.x0+1, 1, r.c)
	}
	// Rounded bow (right).
	render.FillRect(screen, cx+28, cy-1, 3, 3, mid)
	render.FillRect(screen, cx+31, cy, 2, 2, hi)
	render.FillRect(screen, cx+33, cy, 1, 1, lo)
	// Stern taper into prop hub (left).
	render.FillRect(screen, cx-30, cy, 4, 2, mid)
	render.FillRect(screen, cx-32, cy+1, 3, 1, lo)

	// Sail ~1/3 back from bow, with fairwater planes + mast stub.
	sx := cx + 10
	render.FillRect(screen, sx, cy-8, 6, 6, mid)
	render.FillRect(screen, sx, cy-8, 6, 2, hi)
	render.FillRect(screen, sx, cy-4, 6, 2, lo)
	render.FillRect(screen, sx-1, cy-6, 8, 1, mid) // dive planes
	render.FillRect(screen, sx+4, cy-9, 1, 1, hi)  // mast

	// Cruciform stern fins.
	render.FillRect(screen, cx-28, cy-4, 2, 9, mid) // vertical
	render.FillRect(screen, cx-31, cy+1, 7, 1, lo)  // horizontal
}

func drawCompassRose(screen *ebiten.Image, cx, cy, radius float64, heading, ordered float64) {
	render.DrawText(screen, "GYRO COMPASS", int(cx)-50, int(cy-radius)-30, render.ColorPhosphorDim, true)
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
