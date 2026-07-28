package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/world"
)

const (
	tacticalPanelX = 20
	tacticalPanelY = 50
	tacticalPanelW = 1260
	tacticalPanelH = 700

	tacticalZoomMin  = 0.012
	tacticalZoomMax  = 0.12
	tacticalZoomStep = 1.25
	tacticalTrailMax = 48
	tacticalTrailSec = 2.5
	tacticalOuterYd  = 12000.0 // bearing-only contacts sit on this ring
	tacticalCourseDragPx = 10
)

type trailPoint struct {
	X, Y float64
}

type coastSegment struct {
	X0, Y0 float64
	X1, Y1 float64
}

type tacticalState struct {
	zoom            float64
	panX, panY      float64
	courseDragging  bool
	courseArmed     bool // LMB pressed, waiting to distinguish click vs course drag
	courseDeg       float64
	coursePressMX   int
	coursePressMY   int
	panDragging     bool
	panLastMX       int
	panLastMY       int
	trails          map[string][]trailPoint
	lastTrailSample float64
	fitPending      bool
	coastBathy      *world.Bathymetry
	coastSegments   []coastSegment
}

func (a *App) ensureTactical() {
	if a.tactical.zoom == 0 {
		a.tactical.zoom = 0.035
		a.tactical.trails = map[string][]trailPoint{}
		a.tactical.fitPending = true
	}
}

func (a *App) updateTacticalUI() {
	a.ensureTactical()
	if a.Engine == nil {
		return
	}
	a.updateContactTrails()

	mx, my := ebiten.CursorPosition()
	buttons := a.tacticalButtons()
	a.updateButtonTooltips(buttons, mx, my)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for _, b := range buttons {
			if b.contains(mx, my) {
				a.handleTacticalButton(b.ID)
				return
			}
		}
	}

	inMap := inRect(mx, my, tacticalPanelX+8, tacticalPanelY+36, tacticalPanelW-16, tacticalPanelH-48)
	player := a.Engine.Scenario.Player
	sonar := &a.Engine.Sonar

	// Middle mouse (wheel click) — pan. RMB also pans.
	panHeld := (ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)) && inMap
	if panHeld {
		if !a.tactical.panDragging {
			a.tactical.panDragging = true
			a.tactical.panLastMX, a.tactical.panLastMY = mx, my
			a.tactical.courseArmed = false
			a.tactical.courseDragging = false
		} else {
			dx := mx - a.tactical.panLastMX
			dy := my - a.tactical.panLastMY
			a.tactical.panX -= float64(dx) / a.tactical.zoom
			a.tactical.panY += float64(dy) / a.tactical.zoom
			a.tactical.panLastMX, a.tactical.panLastMY = mx, my
		}
	} else {
		a.tactical.panDragging = false
	}

	// LMB: click selects contact; drag orders course.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && inMap && !a.tactical.panDragging {
		a.tactical.courseArmed = true
		a.tactical.courseDragging = false
		a.tactical.coursePressMX, a.tactical.coursePressMY = mx, my
		wx, wy := a.tacticalScreenToWorld(mx, my)
		a.tactical.courseDeg = bearingDeg(player.X, player.Y, wx, wy)
	}

	if a.tactical.courseArmed && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		dx := mx - a.tactical.coursePressMX
		dy := my - a.tactical.coursePressMY
		if dx*dx+dy*dy >= tacticalCourseDragPx*tacticalCourseDragPx {
			a.tactical.courseDragging = true
		}
		if a.tactical.courseDragging && inMap {
			wx, wy := a.tacticalScreenToWorld(mx, my)
			a.tactical.courseDeg = bearingDeg(player.X, player.Y, wx, wy)
		}
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && (a.tactical.courseArmed || a.tactical.courseDragging) {
		if a.tactical.courseDragging {
			player.OrderedHead = a.tactical.courseDeg
			msg := fmt.Sprintf("Ordered course %.0f degrees.", a.tactical.courseDeg)
			a.StatusMessage = msg
			diff := acoustics.AngleDiffDeg(a.tactical.courseDeg, player.HeadingDeg)
			if math.Abs(diff) > 2 {
				if diff < 0 {
					a.Audio.PlayClip(audio.ClipDiveComeLeft, msg)
				} else {
					a.Audio.PlayClip(audio.ClipDiveComeRight, msg)
				}
			}
		} else if a.tactical.courseArmed {
			// Click without drag — select nearest contact under cursor.
			if c := a.tacticalContactAt(mx, my); c != nil {
				a.selectContact(sonar, c)
				a.StatusMessage = fmt.Sprintf("Selected %s", contactLongLabel(c))
			}
		}
		a.tactical.courseArmed = false
		a.tactical.courseDragging = false
	}

	_, wheelY := ebiten.Wheel()
	if wheelY != 0 && inMap {
		wx, wy := a.tacticalScreenToWorld(mx, my)
		if wheelY > 0 {
			a.tactical.zoom = math.Min(tacticalZoomMax, a.tactical.zoom*tacticalZoomStep)
		} else {
			a.tactical.zoom = math.Max(tacticalZoomMin, a.tactical.zoom/tacticalZoomStep)
		}
		a.tactical.panX = wx - player.X - (float64(mx)-float64(tacticalPanelX+tacticalPanelW/2))/a.tactical.zoom
		a.tactical.panY = wy - player.Y + (float64(my)-float64(tacticalPanelY+tacticalPanelH/2))/a.tactical.zoom
	}

	if a.tactical.fitPending {
		a.tacticalFitAll()
		a.tactical.fitPending = false
	}
}

func (a *App) tacticalContactAt(mx, my int) *acoustics.Contact {
	player := a.Engine.Scenario.Player
	sonar := &a.Engine.Sonar
	const hitR2 = 14 * 14
	var best *acoustics.Contact
	bestD := 1e18
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		wx, wy := contactPlotWorld(player, c)
		sx, sy := a.tacticalWorldToScreen(wx, wy)
		dx := float64(mx) - sx
		dy := float64(my) - sy
		d2 := dx*dx + dy*dy
		if d2 <= hitR2 && d2 < bestD {
			bestD = d2
			best = c
		}
	}
	return best
}

func bearingDeg(x0, y0, x1, y1 float64) float64 {
	deg := math.Atan2(x1-x0, y1-y0) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}

func (a *App) tacticalButtons() []uiButton {
	y := tacticalPanelY + 8
	btns := []uiButton{
		{ID: "tac_zoom_in", Label: "+", Tooltip: "Zoom in", X: tacticalPanelX + tacticalPanelW - 150, Y: y, H: 28},
		{ID: "tac_zoom_out", Label: "-", Tooltip: "Zoom out", X: tacticalPanelX + tacticalPanelW - 115, Y: y, H: 28},
		{ID: "tac_fit", Label: "FIT", Tooltip: "Center on ownship and fit known contacts", X: tacticalPanelX + tacticalPanelW - 80, Y: y, H: 28},
	}
	for i := range btns {
		btns[i].W = render.ButtonWidth(btns[i].Label, 10)
	}
	return btns
}

func (a *App) handleTacticalButton(id string) {
	switch id {
	case "tac_zoom_in":
		a.tactical.zoom = math.Min(tacticalZoomMax, a.tactical.zoom*tacticalZoomStep)
	case "tac_zoom_out":
		a.tactical.zoom = math.Max(tacticalZoomMin, a.tactical.zoom/tacticalZoomStep)
	case "tac_fit":
		a.tacticalFitAll()
	}
}

func (a *App) tacticalViewCenter() (cx, cy float64) {
	p := a.Engine.Scenario.Player
	return p.X + a.tactical.panX, p.Y + a.tactical.panY
}

func (a *App) tacticalWorldToScreen(wx, wy float64) (sx, sy float64) {
	cx, cy := a.tacticalViewCenter()
	sx = float64(tacticalPanelX+tacticalPanelW/2) + (wx-cx)*a.tactical.zoom
	sy = float64(tacticalPanelY+tacticalPanelH/2) - (wy-cy)*a.tactical.zoom
	return
}

func (a *App) tacticalScreenToWorld(sx, sy int) (wx, wy float64) {
	cx, cy := a.tacticalViewCenter()
	wx = cx + (float64(sx)-float64(tacticalPanelX+tacticalPanelW/2))/a.tactical.zoom
	wy = cy - (float64(sy)-float64(tacticalPanelY+tacticalPanelH/2))/a.tactical.zoom
	return
}

func (a *App) tacticalFitAll() {
	player := a.Engine.Scenario.Player
	a.tactical.panX, a.tactical.panY = 0, 0
	maxR := 2500.0
	for _, c := range a.Engine.Sonar.Contacts {
		wx, wy := contactPlotWorld(player, &c)
		r := math.Hypot(wx-player.X, wy-player.Y)
		if r > maxR {
			maxR = r
		}
	}
	usable := math.Min(float64(tacticalPanelW), float64(tacticalPanelH)) - 80
	zoom := usable / (maxR * 2.2)
	if zoom < tacticalZoomMin {
		zoom = tacticalZoomMin
	}
	if zoom > tacticalZoomMax {
		zoom = tacticalZoomMax
	}
	a.tactical.zoom = zoom
}

func contactPlotWorld(player *world.Entity, c *acoustics.Contact) (x, y float64) {
	rad := c.BearingDeg * math.Pi / 180
	r := c.EstimatedRangeYd
	if !contactRangeAccurate(c) {
		// Bearing-only fix: park on the outer range ring to avoid range jitter.
		r = tacticalOuterYd
	} else if r < 100 {
		r = 100
	}
	return player.X + math.Sin(rad)*r, player.Y + math.Cos(rad)*r
}

// contactRangeAccurate is true when range uncertainty is within ~10% (90% accuracy).
func contactRangeAccurate(c *acoustics.Contact) bool {
	if c.EstimatedRangeYd < 200 {
		return false
	}
	unc := c.UncRangeYd
	if unc <= 0 {
		unc = c.EstimatedRangeYd * 0.45
	}
	return unc/c.EstimatedRangeYd <= 0.10
}

func contactIsClassified(c *acoustics.Contact) bool {
	return c.ConfirmedClass != "" || c.Confidence >= 0.55
}

func contactTrackAccurate(c *acoustics.Contact) bool {
	return c.Confidence >= 0.80 || c.ConfirmedClass != ""
}

func contactDisplaySide(c *acoustics.Contact) world.Side {
	id := c.ConfirmedID
	if id == "" {
		id = c.BestMatchID
	}
	switch id {
	case "merchant", "tanker", "fishing":
		return world.SideNeutral
	default:
		return world.SideEnemy
	}
}

func (a *App) updateContactTrails() {
	gt := a.Engine.Clock.GameTime
	if gt-a.tactical.lastTrailSample < tacticalTrailSec {
		return
	}
	a.tactical.lastTrailSample = gt
	if a.tactical.trails == nil {
		a.tactical.trails = map[string][]trailPoint{}
	}
	player := a.Engine.Scenario.Player
	alive := map[string]bool{}
	for i := range a.Engine.Sonar.Contacts {
		c := &a.Engine.Sonar.Contacts[i]
		alive[c.SourceEntityID] = true
		if !contactTrackAccurate(c) || !contactRangeAccurate(c) {
			continue
		}
		wx, wy := contactPlotWorld(player, c)
		tr := a.tactical.trails[c.SourceEntityID]
		if len(tr) > 0 {
			last := tr[len(tr)-1]
			if math.Hypot(wx-last.X, wy-last.Y) < 80 {
				continue
			}
		}
		tr = append(tr, trailPoint{X: wx, Y: wy})
		if len(tr) > tacticalTrailMax {
			tr = tr[len(tr)-tacticalTrailMax:]
		}
		a.tactical.trails[c.SourceEntityID] = tr
	}
	for id := range a.tactical.trails {
		if !alive[id] {
			delete(a.tactical.trails, id)
		}
	}
}

func (a *App) drawTactical(screen *ebiten.Image) {
	a.ensureTactical()
	render.FillRect(screen, tacticalPanelX, tacticalPanelY, tacticalPanelW, tacticalPanelH, render.ColorPanel)
	render.DrawTextLarge(screen, "TACTICAL PLOT", tacticalPanelX+20, tacticalPanelY+28, render.ColorText)
	render.DrawText(screen, "LMB click: select   LMB drag: course   MMB/RMB drag: pan   wheel: zoom", tacticalPanelX+280, tacticalPanelY+26, render.ColorPhosphorDim, true)

	for _, b := range a.tacticalButtons() {
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, a.uiHoverID == b.ID, a.uiPressedID == b.ID)
	}

	mapX := tacticalPanelX + 8
	mapY := tacticalPanelY + 40
	mapW := tacticalPanelW - 16
	mapH := tacticalPanelH - 52
	a.drawTacticalMap(screen, mapX, mapY, mapW, mapH)

	if a.uiTooltip != "" {
		mx, my := ebiten.CursorPosition()
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
}

func (a *App) drawTacticalMap(screen *ebiten.Image, mapX, mapY, mapW, mapH int) {
	player := a.Engine.Scenario.Player
	sonar := &a.Engine.Sonar
	bathy := a.Engine.Scenario.Bathy

	render.FillRect(screen, mapX, mapY, mapW, mapH, color.RGBA{4, 18, 28, 255})
	a.drawTacticalBathymetry(screen, mapX, mapY, mapW, mapH, bathy)
	a.drawTacticalCoastline(screen, bathy)

	px, py := a.tacticalWorldToScreen(player.X, player.Y)
	for _, rYd := range []float64{2000, 4000, 8000, 12000} {
		rad := rYd * a.tactical.zoom
		if rad < 12 || rad > float64(mapW) {
			continue
		}
		drawCircle(screen, px, py, rad, color.RGBA{0, 70, 55, 160})
	}

	for _, tr := range a.tactical.trails {
		drawDashedTrail(screen, tr, a)
	}

	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		wx, wy := contactPlotWorld(player, c)
		sx, sy := a.tacticalWorldToScreen(wx, wy)
		lineClr := color.RGBA{0, 180, 120, 200}
		if c.SourceEntityID == a.selectedContactID {
			lineClr = color.RGBA{255, 200, 60, 230}
		}
		render.DrawLine(screen, px, py, sx, sy, lineClr)
		a.drawTacticalContactIcon(screen, c, sx, sy)
	}

	drawOwnshipSymbol(screen, px, py, player.HeadingDeg, render.ColorHighlight)

	for _, t := range a.Engine.FireControl.ActiveTorpedoes {
		if !t.Alive {
			continue
		}
		sx, sy := a.tacticalWorldToScreen(t.X, t.Y)
		render.FillRect(screen, int(sx)-2, int(sy)-2, 5, 5, render.ColorActive)
	}

	if a.tactical.courseDragging {
		rad := a.tactical.courseDeg * math.Pi / 180
		lenPx := 120.0
		render.DrawLine(screen, px, py, px+math.Sin(rad)*lenPx, py-math.Cos(rad)*lenPx, render.ColorAmber)
		render.DrawText(screen, fmt.Sprintf("CSE %.0f", a.tactical.courseDeg), int(px)+14, int(py)+18, render.ColorAmber, true)
	}

	kyd := 2000.0
	bar := kyd * a.tactical.zoom
	bx := mapX + 16
	by := mapY + mapH - 18
	render.DrawLine(screen, float64(bx), float64(by), float64(bx)+bar, float64(by), render.ColorPhosphor)
	render.DrawText(screen, "2 KYD", bx, by-4, render.ColorPhosphorDim, true)
}

func (a *App) drawTacticalBathymetry(screen *ebiten.Image, mapX, mapY, mapW, mapH int, bathy *world.Bathymetry) {
	if bathy == nil || !bathy.Valid() {
		return
	}
	step := 4
	for sy := mapY; sy < mapY+mapH; sy += step {
		for sx := mapX; sx < mapX+mapW; sx += step {
			wx, wy := a.tacticalScreenToWorld(sx, sy)
			render.FillRect(screen, sx, sy, step, step, bathyColor(bathy.DepthAtFt(wx, wy)))
		}
	}
}

func (a *App) drawTacticalCoastline(screen *ebiten.Image, bathy *world.Bathymetry) {
	if bathy == nil || !bathy.Valid() {
		return
	}
	if a.tactical.coastBathy != bathy || a.tactical.coastSegments == nil {
		a.tactical.coastSegments = buildCoastSegments(bathy)
		a.tactical.coastBathy = bathy
	}
	shadow := color.RGBA{5, 10, 12, 220}
	shore := color.RGBA{226, 232, 214, 235}
	for _, seg := range a.tactical.coastSegments {
		x0, y0 := a.tacticalWorldToScreen(seg.X0, seg.Y0)
		x1, y1 := a.tacticalWorldToScreen(seg.X1, seg.Y1)
		render.DrawLine(screen, x0+1, y0+1, x1+1, y1+1, shadow)
		render.DrawLine(screen, x0, y0, x1, y1, shore)
	}
}

func bathyColor(depthFt float64) color.RGBA {
	if depthFt <= 0 {
		return color.RGBA{42, 58, 38, 255}
	}
	// Log scale keeps both shelf water and Catalina basin depths readable.
	t := math.Log1p(depthFt) / math.Log1p(6000)
	if t > 1 {
		t = 1
	}
	r := uint8(3 + (1-t)*18)
	g := uint8(22 + (1-t)*52 + t*4)
	b := uint8(34 + t*105)
	return color.RGBA{r, g, b, 255}
}

func buildCoastSegments(bathy *world.Bathymetry) []coastSegment {
	if bathy == nil || !bathy.Valid() || bathy.Width < 2 || bathy.Height < 2 {
		return nil
	}
	segments := make([]coastSegment, 0, bathy.Width*bathy.Height/4)
	for j := 0; j < bathy.Height-1; j++ {
		y0 := bathy.OriginY + (float64(j)+0.5)*bathy.CellSize
		y1 := bathy.OriginY + (float64(j+1)+0.5)*bathy.CellSize
		for i := 0; i < bathy.Width-1; i++ {
			x0 := bathy.OriginX + (float64(i)+0.5)*bathy.CellSize
			x1 := bathy.OriginX + (float64(i+1)+0.5)*bathy.CellSize
			dBL := float64(bathy.Depths[j*bathy.Width+i])
			dBR := float64(bathy.Depths[j*bathy.Width+i+1])
			dTL := float64(bathy.Depths[(j+1)*bathy.Width+i])
			dTR := float64(bathy.Depths[(j+1)*bathy.Width+i+1])
			mask := 0
			if dBL > 0 {
				mask |= 1
			}
			if dBR > 0 {
				mask |= 2
			}
			if dTR > 0 {
				mask |= 4
			}
			if dTL > 0 {
				mask |= 8
			}
			if mask == 0 || mask == 15 {
				continue
			}
			bottomX, bottomY := interpZero(x0, y0, dBL, x1, y0, dBR)
			rightX, rightY := interpZero(x1, y0, dBR, x1, y1, dTR)
			topX, topY := interpZero(x0, y1, dTL, x1, y1, dTR)
			leftX, leftY := interpZero(x0, y0, dBL, x0, y1, dTL)
			switch mask {
			case 1, 14:
				segments = append(segments, coastSegment{leftX, leftY, bottomX, bottomY})
			case 2, 13:
				segments = append(segments, coastSegment{bottomX, bottomY, rightX, rightY})
			case 3, 12:
				segments = append(segments, coastSegment{leftX, leftY, rightX, rightY})
			case 4, 11:
				segments = append(segments, coastSegment{rightX, rightY, topX, topY})
			case 5:
				segments = append(segments,
					coastSegment{leftX, leftY, topX, topY},
					coastSegment{bottomX, bottomY, rightX, rightY},
				)
			case 6, 9:
				segments = append(segments, coastSegment{bottomX, bottomY, topX, topY})
			case 7, 8:
				segments = append(segments, coastSegment{leftX, leftY, topX, topY})
			case 10:
				segments = append(segments,
					coastSegment{leftX, leftY, bottomX, bottomY},
					coastSegment{rightX, rightY, topX, topY},
				)
			}
		}
	}
	return segments
}

func interpZero(x0, y0, d0, x1, y1, d1 float64) (float64, float64) {
	den := d0 - d1
	if math.Abs(den) < 1e-6 {
		return (x0 + x1) * 0.5, (y0 + y1) * 0.5
	}
	t := d0 / den
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return x0 + (x1-x0)*t, y0 + (y1-y0)*t
}

func drawDashedTrail(screen *ebiten.Image, tr []trailPoint, a *App) {
	if len(tr) < 2 {
		return
	}
	clr := color.RGBA{200, 220, 210, 160}
	on := true
	for i := 1; i < len(tr); i++ {
		x0, y0 := a.tacticalWorldToScreen(tr[i-1].X, tr[i-1].Y)
		x1, y1 := a.tacticalWorldToScreen(tr[i].X, tr[i].Y)
		if on {
			render.DrawLine(screen, x0, y0, x1, y1, clr)
		}
		on = !on
	}
}

func (a *App) drawTacticalContactIcon(screen *ebiten.Image, c *acoustics.Contact, sx, sy float64) {
	classified := contactIsClassified(c)
	clr := color.RGBA{140, 145, 150, 255}
	kind := world.EntityKind(-1)
	if classified {
		switch contactDisplaySide(c) {
		case world.SideNeutral:
			clr = color.RGBA{210, 210, 120, 255}
		default:
			clr = color.RGBA{220, 60, 50, 255}
		}
		kind = c.Kind
		id := c.ConfirmedID
		if id == "" {
			id = c.BestMatchID
		}
		if p, ok := world.ProfileByID(id); ok {
			kind = p.Kind
		}
	}

	drawContactPictogram(screen, int(sx), int(sy), kind, clr)
	label := c.ID
	if classified {
		label = contactLongLabel(c)
	}
	if c.SourceEntityID == a.selectedContactID {
		half := 12
		x, y := int(sx), int(sy)
		arm := 5
		drawCornerBracket(screen, x, y, half, arm, color.RGBA{0, 0, 0, 255}, 1)
		drawCornerBracket(screen, x, y, half, arm, color.RGBA{255, 255, 255, 255}, 0)
		clr = color.RGBA{255, 255, 255, 255}
	}
	render.DrawText(screen, label, int(sx)+10, int(sy)+4, clr, true)
}

func drawContactPictogram(screen *ebiten.Image, cx, cy int, kind world.EntityKind, clr color.RGBA) {
	switch kind {
	case world.KindSurfaceShip:
		render.FillRect(screen, cx-7, cy-2, 14, 4, clr)
		render.FillRect(screen, cx-3, cy-6, 6, 4, clr)
		render.FillRect(screen, cx+2, cy-9, 2, 3, clr)
	case world.KindSubmarine:
		render.FillRect(screen, cx-8, cy-1, 16, 3, clr)
		render.FillRect(screen, cx-1, cy-5, 4, 4, clr)
	default:
		render.DrawLine(screen, float64(cx), float64(cy-6), float64(cx+6), float64(cy), clr)
		render.DrawLine(screen, float64(cx+6), float64(cy), float64(cx), float64(cy+6), clr)
		render.DrawLine(screen, float64(cx), float64(cy+6), float64(cx-6), float64(cy), clr)
		render.DrawLine(screen, float64(cx-6), float64(cy), float64(cx), float64(cy-6), clr)
	}
}

func drawOwnshipSymbol(screen *ebiten.Image, sx, sy, heading float64, clr color.Color) {
	rad := heading * math.Pi / 180
	tipX := sx + math.Sin(rad)*10
	tipY := sy - math.Cos(rad)*10
	lx := sx + math.Sin(rad+2.5)*7
	ly := sy - math.Cos(rad+2.5)*7
	rx := sx + math.Sin(rad-2.5)*7
	ry := sy - math.Cos(rad-2.5)*7
	render.DrawLine(screen, tipX, tipY, lx, ly, clr)
	render.DrawLine(screen, tipX, tipY, rx, ry, clr)
	render.FillRect(screen, int(sx)-2, int(sy)-2, 5, 5, clr)
}
