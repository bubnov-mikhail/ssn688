package ui

import (
	"fmt"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

const (
	tacticalPanelX = 20
	tacticalPanelY = 50
	tacticalPanelW = 1260
	tacticalPanelH = 700

	tacticalZoomMin        = 0.012
	tacticalZoomMax        = 0.12
	tacticalZoomStep       = 1.25
	tacticalOuterYd        = 12000.0 // bearing-only contacts sit on this ring
	tacticalCourseDragPx   = 10
	tacticalSmoothStaleSec = 8.0 // reset average if contact absent this long
	tacticalRulerTickYd    = 3000.0
)

var tacticalRulerColor = color.RGBA{255, 255, 255, 230}

var contactTMALineColor = color.RGBA{88, 88, 88, 210}
var torpedoThreatBlinkColor = color.RGBA{255, 80, 60, 255}

// smoothedContactPos is an EMA of tactical plot position relative to ownship.
type smoothedContactPos struct {
	RelX, RelY float64
	LastAt     float64
}

type coastSegment struct {
	X0, Y0 float64
	X1, Y1 float64
}

type bathyViewKey struct {
	zoom                 float64
	centerX, centerY     float64
	mapX, mapY           int
	mapW, mapH           int
}

// tacticalMapView maps world yards to screen pixels for a plot panel.
type tacticalMapView struct {
	mapX, mapY, mapW, mapH int
	centerX, centerY       float64
	zoom                   float64
}

func (v tacticalMapView) worldToScreen(wx, wy float64) (sx, sy float64) {
	sx = float64(v.mapX+v.mapW/2) + (wx-v.centerX)*v.zoom
	sy = float64(v.mapY+v.mapH/2) - (wy-v.centerY)*v.zoom
	return sx, sy
}

func (v tacticalMapView) screenToWorld(sx, sy int) (wx, wy float64) {
	wx = v.centerX + (float64(sx)-float64(v.mapX+v.mapW/2))/v.zoom
	wy = v.centerY - (float64(sy)-float64(v.mapY+v.mapH/2))/v.zoom
	return wx, wy
}

func (v tacticalMapView) containsScreen(sx, sy int) bool {
	return inRect(sx, sy, v.mapX, v.mapY, v.mapW, v.mapH)
}

type tacticalMapOpts struct {
	minimap       bool
	debugOverlay  bool
	showSelection bool
	showChrome    bool
}

type tacticalState struct {
	zoom               float64
	panX, panY         float64
	courseDragging     bool
	courseArmed        bool // LMB pressed, waiting to distinguish click vs course drag
	courseDeg          float64
	coursePressMX      int
	coursePressMY      int
	panDragging        bool
	panLastMX          int
	panLastMY          int
	smoothedPos        map[string]smoothedContactPos
	smoothAliveScratch map[string]bool
	fitPending         bool
	coastBathy         *world.Bathymetry
	coastSegments      []coastSegment
	bathyImg           *ebiten.Image
	bathyPix           []byte
	bathyKey           bathyViewKey
	minimapBathyImg    *ebiten.Image
	minimapBathyPix    []byte
	minimapBathyKey    bathyViewKey
	rulerActive        bool
	rulerX0, rulerY0   float64
}

func (a *App) ensureTactical() {
	if a.tactical.zoom == 0 {
		a.tactical.zoom = 0.035
		a.tactical.smoothedPos = map[string]smoothedContactPos{}
		a.tactical.fitPending = true
	}
}

func (a *App) updateTacticalUI() {
	a.ensureTactical()
	if a.Engine == nil {
		return
	}

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

	mapX := tacticalPanelX + 8
	mapY := tacticalPanelY + 40
	mapW := tacticalPanelW - 16
	mapH := tacticalPanelH - 52
	inMap := inRect(mx, my, mapX, mapY, mapW, mapH)
	player := a.Engine.Scenario.Player
	sonar := &a.Engine.Sonar

	if a.pendingPlotMarker {
		a.pendingPlotMarker = false
		if inMap {
			wx, wy := a.tacticalScreenToWorld(mx, my)
			m := a.Engine.AddPlotMarker(wx, wy)
			a.selectedPlotMarkerID = m.ID
			a.selectedContactID = ""
			a.StatusMessage = fmt.Sprintf("Marker %s placed", m.ID)
			return
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if a.selectedPlotMarkerID != "" && a.Engine.DeletePlotMarker(a.selectedPlotMarkerID) {
			a.StatusMessage = fmt.Sprintf("Marker %s deleted", a.selectedPlotMarkerID)
			a.selectedPlotMarkerID = ""
			return
		}
	}

	// Hold R over map — range/bearing ruler from first press point to cursor.
	if inpututil.IsKeyJustReleased(ebiten.KeyR) {
		a.tactical.rulerActive = false
	}
	if inMap && inpututil.IsKeyJustPressed(ebiten.KeyR) {
		wx, wy := a.tacticalScreenToWorld(mx, my)
		a.tactical.rulerX0, a.tactical.rulerY0 = wx, wy
		a.tactical.rulerActive = true
		a.tactical.courseArmed = false
		a.tactical.courseDragging = false
	}
	rulerHeld := a.tactical.rulerActive && ebiten.IsKeyPressed(ebiten.KeyR)

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
			a.invalidateTacticalBathy()
		}
	} else {
		a.tactical.panDragging = false
	}

	// LMB: click selects contact/marker; drag orders course (not while ruler held).
	if !rulerHeld && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && inMap && !a.tactical.panDragging {
		a.tactical.courseArmed = true
		a.tactical.courseDragging = false
		a.tactical.coursePressMX, a.tactical.coursePressMY = mx, my
		wx, wy := a.tacticalScreenToWorld(mx, my)
		a.tactical.courseDeg = bearingDeg(player.X, player.Y, wx, wy)
	}

	if !rulerHeld && a.tactical.courseArmed && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
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

	if !rulerHeld && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && (a.tactical.courseArmed || a.tactical.courseDragging) {
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
			// Click without drag — prefer chart markers, then contacts.
			if m := a.tacticalMarkerAt(mx, my); m != nil {
				a.selectedPlotMarkerID = m.ID
				a.selectedContactID = ""
				a.StatusMessage = fmt.Sprintf("Selected marker %s", m.ID)
			} else if c := a.tacticalContactAt(mx, my); c != nil {
				a.selectContact(sonar, c)
				a.StatusMessage = fmt.Sprintf("Selected %s", contactLongLabel(c))
			} else {
				a.selectedPlotMarkerID = ""
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
		a.invalidateTacticalBathy()
	}

	if a.tactical.fitPending {
		a.tacticalFitAll()
		a.tactical.fitPending = false
	}
}

func (a *App) tacticalContactAt(mx, my int) *acoustics.Contact {
	player := a.Engine.Scenario.Player
	sonar := &a.Engine.Sonar
	gt := a.Engine.Clock.GameTime
	const hitR2 = 14 * 14
	var best *acoustics.Contact
	bestD := 1e18
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if a.isOwnTorpedoContact(c) {
			continue
		}
		wx, wy := a.contactPlotWorld(player, c, gt)
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

func (a *App) tacticalMarkerAt(mx, my int) *world.PlotMarker {
	var best *world.PlotMarker
	bestD := 1e18
	halfYd := world.PlotMarkerSizeYd * 0.5
	const minHalfPx = 8.0
	for i := range a.Engine.PlotMarkers {
		m := &a.Engine.PlotMarkers[i]
		sx, sy := a.tacticalWorldToScreen(m.X, m.Y)
		halfPx := halfYd * a.tactical.zoom
		if halfPx < minHalfPx {
			halfPx = minHalfPx
		}
		dx := math.Abs(float64(mx) - sx)
		dy := math.Abs(float64(my) - sy)
		if dx <= halfPx && dy <= halfPx {
			d2 := dx*dx + dy*dy
			if d2 < bestD {
				bestD = d2
				best = m
			}
		}
	}
	return best
}

// isOwnTorpedoContact is true when the track is one of our wire/in-water Mk48s
// (shown as a blue telemetry square, not a hostile/unknown contact glyph).
func (a *App) isOwnTorpedoContact(c *acoustics.Contact) bool {
	if a == nil || c == nil || a.Engine == nil {
		return false
	}
	id := c.SourceEntityID
	if id == "" {
		return false
	}
	if a.ownTorpedoIDs != nil && a.ownTorpedoIDs[id] {
		return true
	}
	for _, t := range a.Engine.FireControl.ActiveTorpedoes {
		if t != nil && t.Alive && t.Side == world.SidePlayer && t.ID == id {
			return true
		}
	}
	return false
}

func bearingDeg(x0, y0, x1, y1 float64) float64 {
	deg := math.Atan2(x1-x0, y1-y0) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}

func (a *App) tacticalButtons() []uiButton {
	cachedTacticalButtonsOnce.Do(initTacticalButtons)
	return cachedTacticalButtons
}

var (
	cachedTacticalButtonsOnce sync.Once
	cachedTacticalButtons     []uiButton
)

func initTacticalButtons() {
	const inset = 8
	const gap = 6
	y := tacticalPanelY + inset
	right := tacticalPanelX + tacticalPanelW - inset
	btns := []uiButton{
		{ID: "tac_zoom_in", Label: "+", Tooltip: "Zoom in", Y: y, H: 28},
		{ID: "tac_zoom_out", Label: "-", Tooltip: "Zoom out", Y: y, H: 28},
		{ID: "tac_fit", Label: "FIT", Tooltip: "Center on ownship and fit known contacts", Y: y, H: 28},
	}
	for i := range btns {
		btns[i].W = render.ButtonWidth(btns[i].Label, 10)
	}
	btns[2].X = right - btns[2].W
	btns[1].X = btns[2].X - gap - btns[1].W
	btns[0].X = btns[1].X - gap - btns[0].W
	cachedTacticalButtons = btns
}

func (a *App) handleTacticalButton(id string) {
	switch id {
	case "tac_zoom_in":
		a.tactical.zoom = math.Min(tacticalZoomMax, a.tactical.zoom*tacticalZoomStep)
		a.invalidateTacticalBathy()
	case "tac_zoom_out":
		a.tactical.zoom = math.Max(tacticalZoomMin, a.tactical.zoom/tacticalZoomStep)
		a.invalidateTacticalBathy()
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
	gt := a.Engine.Clock.GameTime
	a.tactical.panX, a.tactical.panY = 0, 0
	maxR := 2500.0
	for _, c := range a.Engine.Sonar.Contacts {
		wx, wy := a.contactPlotWorld(player, &c, gt)
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
	a.invalidateTacticalBathy()
}

// contactPlotRaw returns the instantaneous polar estimate (no display averaging).
func contactPlotRaw(player *world.Entity, c *acoustics.Contact, gameTime float64) (x, y float64) {
	bearing := c.BearingDeg
	r := c.EstimatedRangeYd
	switch {
	case acoustics.ContactActiveFixValid(c, gameTime):
		// Freeze last active range-bearing while the ACTIVE marker would still be fading.
		bearing = c.LastActiveBearingDeg
		r = c.LastActiveRangeYd
	case contactRangeAccurate(c):
		// High-probability passive (or fresh) range fix — use live track.
		if r < 100 {
			r = 100
		}
	default:
		// Bearing-only: park on the outer range ring to avoid range jitter.
		r = tacticalOuterYd
	}
	rad := bearing * math.Pi / 180
	return player.X + math.Sin(rad)*r, player.Y + math.Cos(rad)*r
}

// contactPlotWorld returns the display position (EMA-smoothed to damp marker jump).
func (a *App) contactPlotWorld(player *world.Entity, c *acoustics.Contact, gameTime float64) (x, y float64) {
	if c == nil || player == nil {
		return 0, 0
	}
	id := c.SourceEntityID
	if id == "" {
		id = c.ID
	}
	if pos, ok := a.tactical.smoothedPos[id]; ok {
		return player.X + pos.RelX, player.Y + pos.RelY
	}
	return contactPlotRaw(player, c, gameTime)
}

// contactSmoothAlpha is the EMA blend weight toward the raw fix (lower = steadier marker).
func contactSmoothAlpha(c *acoustics.Contact, gameTime float64) float64 {
	switch {
	case acoustics.ContactActiveFixValid(c, gameTime):
		// Active snapshot is already stable; blend lightly when a new ping updates it.
		return 0.45
	case contactRangeAccurate(c):
		relUnc := c.UncRangeYd / math.Max(c.EstimatedRangeYd, 1)
		bearQ := math.Min(1, c.UncBearingDeg/20)
		q := math.Max(relUnc, bearQ*0.35)
		// ~5% unc → α≈0.22; ~10% → α≈0.12; noisier → α≈0.05
		return 0.05 + 0.20*(1-math.Min(1, q/0.14))
	default:
		// Bearing-only on the outer ring — heavy average against LOB jitter.
		bearQ := math.Min(1, c.UncBearingDeg/28)
		return 0.04 + 0.10*(1-bearQ)
	}
}

func (a *App) updateSmoothedContactPositions() {
	if a.Engine == nil || a.Engine.Scenario == nil || a.Engine.Scenario.Player == nil {
		return
	}
	player := a.Engine.Scenario.Player
	gt := a.Engine.Clock.GameTime
	if a.tactical.smoothedPos == nil {
		a.tactical.smoothedPos = map[string]smoothedContactPos{}
	}
	if a.tactical.smoothAliveScratch == nil {
		a.tactical.smoothAliveScratch = map[string]bool{}
	} else {
		clear(a.tactical.smoothAliveScratch)
	}
	alive := a.tactical.smoothAliveScratch

	for i := range a.Engine.Sonar.Contacts {
		c := &a.Engine.Sonar.Contacts[i]
		if a.isOwnTorpedoContact(c) {
			continue
		}
		id := c.SourceEntityID
		if id == "" {
			id = c.ID
		}
		alive[id] = true
		rawX, rawY := contactPlotRaw(player, c, gt)
		rawRelX := rawX - player.X
		rawRelY := rawY - player.Y
		prev, ok := a.tactical.smoothedPos[id]
		if !ok || gt-prev.LastAt > tacticalSmoothStaleSec {
			a.tactical.smoothedPos[id] = smoothedContactPos{RelX: rawRelX, RelY: rawRelY, LastAt: gt}
			continue
		}
		alpha := contactSmoothAlpha(c, gt)
		a.tactical.smoothedPos[id] = smoothedContactPos{
			RelX:   prev.RelX*(1-alpha) + rawRelX*alpha,
			RelY:   prev.RelY*(1-alpha) + rawRelY*alpha,
			LastAt: gt,
		}
	}
	for id := range a.tactical.smoothedPos {
		if !alive[id] {
			delete(a.tactical.smoothedPos, id)
		}
	}
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

// contactHasKnownRange is false for bearing-only tracks parked on the outer ring.
func contactHasKnownRange(c *acoustics.Contact, gameTime float64) bool {
	if c == nil {
		return false
	}
	return acoustics.ContactActiveFixValid(c, gameTime) || contactRangeAccurate(c)
}

func contactIsClassified(c *acoustics.Contact) bool {
	return c.ConfirmedClass != "" || c.Confidence >= 0.55
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

func (a *App) drawTactical(screen *ebiten.Image) {
	a.ensureTactical()
	render.DrawConsolePanel(screen, tacticalPanelX, tacticalPanelY, tacticalPanelW, tacticalPanelH)
	title := "TACTICAL PLOT"
	if a.Settings.Debug {
		title = "TACTICAL PLOT · DEBUG"
	}
	render.DrawScreenTitle(screen, title, tacticalPanelX+20, tacticalPanelY+28)
	render.DrawText(screen, "LMB: select   LMB drag: course   Hold R: ruler   M: marker   Del: delete marker   MMB/RMB: pan   wheel: zoom", tacticalPanelX+280, tacticalPanelY+26, render.ColorPlateLabel, true)

	for _, b := range a.tacticalButtons() {
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, a.uiHoverID == b.ID, a.uiPressedID == b.ID)
	}

	mapX := tacticalPanelX + 8
	mapY := tacticalPanelY + 40
	mapW := tacticalPanelW - 16
	mapH := tacticalPanelH - 52
	render.DrawMonitor(screen, mapX, mapY, mapW, mapH)
	a.drawTacticalMap(screen, mapX, mapY, mapW, mapH, tacticalMapOpts{
		showSelection: true,
		showChrome:    true,
		debugOverlay:  a.Settings.Debug,
	})

	if a.uiTooltip != "" {
		mx, my := ebiten.CursorPosition()
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
}

func (a *App) drawTacticalMap(screen *ebiten.Image, mapX, mapY, mapW, mapH int, opts tacticalMapOpts) {
	player := a.Engine.Scenario.Player
	sonar := &a.Engine.Sonar
	bathy := a.Engine.Scenario.Bathy

	var view tacticalMapView
	if opts.minimap {
		view = tacticalMapView{mapX, mapY, mapW, mapH, player.X, player.Y, a.tactical.zoom}
	} else {
		cx, cy := a.tacticalViewCenter()
		view = tacticalMapView{mapX, mapY, mapW, mapH, cx, cy, a.tactical.zoom}
	}

	render.FillRect(screen, mapX, mapY, mapW, mapH, color.RGBA{4, 18, 28, 255})
	a.drawTacticalBathymetry(screen, view, bathy, opts.minimap)
	a.drawTacticalCoastline(screen, view, bathy)

	px, py := view.worldToScreen(player.X, player.Y)
	ringLabelClr := color.RGBA{0, 150, 120, 210}
	for _, rYd := range []float64{2000, 4000, 8000, 12000} {
		rad := rYd * view.zoom
		if rad < 12 || rad > float64(mapW) {
			continue
		}
		drawCircle(screen, px, py, rad, color.RGBA{0, 70, 55, 160})
		if !opts.minimap {
			drawMapRangeRingLabel(screen, px, py, rad, rYd, ringLabelClr)
		}
	}

	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if c.Kind == world.KindCountermeasure {
			continue
		}
		if a.isOwnTorpedoContact(c) {
			continue
		}
		wx, wy := a.contactPlotWorld(player, c, a.Engine.Clock.GameTime)
		sx, sy := view.worldToScreen(wx, wy)
		if x1, y1, ok := contactTMAWorldLineEnd(c, wx, wy); ok {
			sx1, sy1 := view.worldToScreen(x1, y1)
			render.DrawLine(screen, sx, sy, sx1, sy1, contactTMALineColor)
		}
		if !contactHasKnownRange(c, a.Engine.Clock.GameTime) {
			lineClr := color.RGBA{0, 180, 120, 200}
			if opts.showSelection && c.SourceEntityID == a.selectedContactID {
				lineClr = color.RGBA{255, 200, 60, 230}
			}
			render.DrawLine(screen, px, py, sx, sy, lineClr)
		}
		a.drawTacticalContactIcon(screen, c, sx, sy, opts)
		if a.torpedoThreatMarkerActive(c.SourceEntityID) {
			drawThreatBlinkMarker(screen, sx, sy)
		}
	}

	drawOwnshipSymbol(screen, px, py, player.HeadingDeg, render.ColorHighlight)

	a.drawTacticalPlotMarkers(screen, view, opts)

	for _, aroc := range a.Engine.FireControl.ActiveRastrub {
		if aroc == nil || !aroc.Alive {
			continue
		}
		ax, ay := aroc.Pos(a.Engine.Clock.GameTime)
		sx, sy := view.worldToScreen(ax, ay)
		render.FillRect(screen, int(sx)-2, int(sy)-2, 5, 5, render.ColorAmber)
		sx1, sy1 := view.worldToScreen(aroc.X1, aroc.Y1)
		render.DrawLine(screen, sx, sy, sx1, sy1, color.RGBA{255, 180, 40, 90})
	}

	for _, t := range a.Engine.FireControl.ActiveTorpedoes {
		if t == nil || !t.Alive || t.Side != world.SidePlayer {
			continue
		}
		sx, sy := view.worldToScreen(t.X, t.Y)
		clr := render.ColorActive
		if t.CMLockID != "" && t.Mode == weapons.ModeSearch {
			clr = color.RGBA{180, 80, 255, 255}
		}
		render.FillRect(screen, int(sx)-2, int(sy)-2, 5, 5, clr)
		if a.torpedoThreatMarkerActive(t.ID) {
			drawThreatBlinkMarker(screen, sx, sy)
		}
	}

	if opts.debugOverlay {
		a.drawTacticalDebugOverlay(screen, view)
	}

	if opts.showChrome && a.tactical.courseDragging {
		rad := a.tactical.courseDeg * math.Pi / 180
		lenPx := 120.0
		render.DrawLine(screen, px, py, px+math.Sin(rad)*lenPx, py-math.Cos(rad)*lenPx, render.ColorAmber)
		render.DrawText(screen, fmt.Sprintf("CSE %.0f", a.tactical.courseDeg), int(px)+14, int(py)+18, render.ColorAmber, true)
	}

	if opts.showChrome {
		a.drawTacticalRuler(screen)
		kyd := 2000.0
		bar := kyd * view.zoom
		bx := mapX + 16
		by := mapY + mapH - 34
		render.DrawLine(screen, float64(bx), float64(by), float64(bx)+bar, float64(by), render.ColorPhosphor)
		render.DrawText(screen, "2 KYD", bx, by-4, render.ColorPhosphorDim, true)
		a.drawTacticalNavCoords(screen, bx, by+14, mapX, mapY, mapW, mapH, player)
	}
}

func (a *App) drawTacticalRuler(screen *ebiten.Image) {
	if !a.tactical.rulerActive || !ebiten.IsKeyPressed(ebiten.KeyR) {
		return
	}
	mx, my := ebiten.CursorPosition()
	wx1, wy1 := a.tacticalScreenToWorld(mx, my)
	x0, y0 := a.tactical.rulerX0, a.tactical.rulerY0
	distYd := math.Hypot(wx1-x0, wy1-y0)
	brg := bearingDeg(x0, y0, wx1, wy1)

	sx0, sy0 := a.tacticalWorldToScreen(x0, y0)
	sx1, sy1 := a.tacticalWorldToScreen(wx1, wy1)

	radPx := distYd * a.tactical.zoom
	if radPx > 2 {
		drawCircle(screen, sx0, sy0, radPx, tacticalRulerColor)
	}
	render.DrawLine(screen, sx0, sy0, sx1, sy1, tacticalRulerColor)

	if distYd > 1 {
		dx := sx1 - sx0
		dy := sy1 - sy0
		segLen := math.Hypot(dx, dy)
		if segLen > 1 {
			px := -dy / segLen
			py := dx / segLen
			const tickHalf = 5.0
			for d := tacticalRulerTickYd; d < distYd; d += tacticalRulerTickYd {
				f := d / distYd
				tx, ty := sx0+dx*f, sy0+dy*f
				render.DrawLine(screen, tx-px*tickHalf, ty-py*tickHalf, tx+px*tickHalf, ty+py*tickHalf, tacticalRulerColor)
			}
		}
	}

	label := fmt.Sprintf("%.0f yd / %.0f°", distYd, brg)
	render.DrawText(screen, label, mx+10, my-6, tacticalRulerColor, true)
}

func (a *App) drawTacticalNavCoords(screen *ebiten.Image, x, y, mapX, mapY, mapW, mapH int, player *world.Entity) {
	if player == nil {
		return
	}
	lat, lon := world.WorldToLatLon(player.X, player.Y)
	line := world.FormatNavLatLon(lat, lon)
	mx, my := ebiten.CursorPosition()
	if inRect(mx, my, mapX, mapY, mapW, mapH) {
		wx, wy := a.tacticalScreenToWorld(mx, my)
		clat, clon := world.WorldToLatLon(wx, wy)
		line += " | " + world.FormatNavLatLon(clat, clon)
	}
	render.DrawText(screen, line, x, y, render.ColorPhosphorDim, true)
}

func (a *App) drawTacticalPlotMarkers(screen *ebiten.Image, view tacticalMapView, opts tacticalMapOpts) {
	halfYd := world.PlotMarkerSizeYd * 0.5
	for i := range a.Engine.PlotMarkers {
		m := &a.Engine.PlotMarkers[i]
		sx, sy := view.worldToScreen(m.X, m.Y)
		halfPx := halfYd * view.zoom
		if halfPx < 4 {
			halfPx = 4
		}
		clr := color.RGBA{220, 200, 80, 255}
		if opts.showSelection && m.ID == a.selectedPlotMarkerID {
			clr = render.ColorAmber
		}
		x0, y0 := sx-halfPx, sy-halfPx
		x1, y1 := sx+halfPx, sy+halfPx
		render.DrawLine(screen, x0, y0, x1, y0, clr)
		render.DrawLine(screen, x1, y0, x1, y1, clr)
		render.DrawLine(screen, x1, y1, x0, y1, clr)
		render.DrawLine(screen, x0, y1, x0, y0, clr)
		render.DrawLine(screen, x0, y0, x1, y1, clr)
		render.DrawLine(screen, x1, y0, x0, y1, clr)
		if !opts.minimap {
			render.DrawText(screen, m.ID, int(sx)+int(halfPx)+4, int(sy)-2, clr, true)
		}
	}
}

func contactTMAWorldLineEnd(c *acoustics.Contact, x, y float64) (x1, y1 float64, ok bool) {
	if !acoustics.ContactTMAAccurate(c) {
		return 0, 0, false
	}
	travelYd := 600.0 * c.TMASpeedKts * world.KnotsToYPS
	if travelYd < 50 {
		return 0, 0, false
	}
	rad := c.TMACourseDeg * math.Pi / 180
	return x + math.Sin(rad)*travelYd, y + math.Cos(rad)*travelYd, true
}

func drawThreatBlinkMarker(screen *ebiten.Image, sx, sy float64) {
	const r = 10.0
	render.DrawLine(screen, sx-r, sy-r, sx+r, sy-r, torpedoThreatBlinkColor)
	render.DrawLine(screen, sx+r, sy-r, sx+r, sy+r, torpedoThreatBlinkColor)
	render.DrawLine(screen, sx+r, sy+r, sx-r, sy+r, torpedoThreatBlinkColor)
	render.DrawLine(screen, sx-r, sy+r, sx-r, sy-r, torpedoThreatBlinkColor)
}

func (a *App) drawTacticalBathymetry(screen *ebiten.Image, view tacticalMapView, bathy *world.Bathymetry, minimap bool) {
	if bathy == nil || !bathy.Valid() {
		return
	}
	img := a.ensureTacticalBathyImage(view, bathy, minimap)
	if img == nil {
		return
	}
	const step = 4
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(step, step)
	op.GeoM.Translate(float64(view.mapX), float64(view.mapY))
	screen.DrawImage(img, op)
}

func (a *App) invalidateTacticalBathy() {
	a.tactical.bathyKey = bathyViewKey{}
	a.tactical.minimapBathyKey = bathyViewKey{}
}

func (a *App) ensureTacticalBathyImage(view tacticalMapView, bathy *world.Bathymetry, minimap bool) *ebiten.Image {
	key := bathyViewKey{
		zoom: view.zoom, centerX: view.centerX, centerY: view.centerY,
		mapX: view.mapX, mapY: view.mapY, mapW: view.mapW, mapH: view.mapH,
	}
	var img **ebiten.Image
	var pix *[]byte
	var cachedKey *bathyViewKey
	if minimap {
		img = &a.tactical.minimapBathyImg
		pix = &a.tactical.minimapBathyPix
		cachedKey = &a.tactical.minimapBathyKey
	} else {
		img = &a.tactical.bathyImg
		pix = &a.tactical.bathyPix
		cachedKey = &a.tactical.bathyKey
	}
	if *img != nil && *cachedKey == key {
		return *img
	}

	const step = 4
	w := (view.mapW + step - 1) / step
	h := (view.mapH + step - 1) / step
	if w < 1 || h < 1 {
		return nil
	}
	need := w * h * 4
	if *pix == nil || len(*pix) != need {
		*pix = make([]byte, need)
		if *img == nil {
			*img = ebiten.NewImage(w, h)
		} else if (*img).Bounds().Dx() != w {
			*img = ebiten.NewImage(w, h)
		}
	} else if *img == nil || (*img).Bounds().Dx() != w {
		*img = ebiten.NewImage(w, h)
	}

	buf := *pix
	for py := 0; py < h; py++ {
		sy := view.mapY + py*step
		for px := 0; px < w; px++ {
			sx := view.mapX + px*step
			wx, wy := view.screenToWorld(sx, sy)
			clr := bathyColor(bathy.DepthAtFt(wx, wy))
			off := (py*w + px) * 4
			buf[off] = clr.R
			buf[off+1] = clr.G
			buf[off+2] = clr.B
			buf[off+3] = clr.A
		}
	}
	(*img).WritePixels(buf)
	*cachedKey = key
	return *img
}

func (a *App) drawTacticalCoastline(screen *ebiten.Image, view tacticalMapView, bathy *world.Bathymetry) {
	if bathy == nil || !bathy.Valid() {
		return
	}
	if a.tactical.coastBathy != bathy || a.tactical.coastSegments == nil {
		a.tactical.coastSegments = buildCoastSegments(bathy)
		a.tactical.coastBathy = bathy
	}
	left := float64(view.mapX)
	top := float64(view.mapY)
	right := float64(view.mapX + view.mapW - 1)
	bottom := float64(view.mapY + view.mapH - 1)
	shadow := color.RGBA{5, 10, 12, 220}
	shore := color.RGBA{226, 232, 214, 235}
	for _, seg := range a.tactical.coastSegments {
		x0, y0 := view.worldToScreen(seg.X0, seg.Y0)
		x1, y1 := view.worldToScreen(seg.X1, seg.Y1)
		x0, y0, x1, y1, ok := clipLineToRect(x0, y0, x1, y1, left, top, right, bottom)
		if !ok {
			continue
		}
		render.DrawLine(screen, x0+1, y0+1, x1+1, y1+1, shadow)
		render.DrawLine(screen, x0, y0, x1, y1, shore)
	}
}

// clipLineToRect clips a line segment to an axis-aligned rectangle (Cohen–Sutherland).
func clipLineToRect(x0, y0, x1, y1, left, top, right, bottom float64) (float64, float64, float64, float64, bool) {
	const (
		clipLeft   = 1
		clipRight  = 2
		clipBottom = 4
		clipTop    = 8
	)
	encode := func(x, y float64) int {
		c := 0
		if x < left {
			c |= clipLeft
		} else if x > right {
			c |= clipRight
		}
		if y < top {
			c |= clipTop
		} else if y > bottom {
			c |= clipBottom
		}
		return c
	}
	c0 := encode(x0, y0)
	c1 := encode(x1, y1)
	for {
		if c0 == 0 && c1 == 0 {
			return x0, y0, x1, y1, true
		}
		if c0&c1 != 0 {
			return 0, 0, 0, 0, false
		}
		out := c0
		if out == 0 {
			out = c1
		}
		var x, y float64
		switch {
		case out&clipTop != 0:
			x = x0 + (x1-x0)*(top-y0)/(y1-y0)
			y = top
		case out&clipBottom != 0:
			x = x0 + (x1-x0)*(bottom-y0)/(y1-y0)
			y = bottom
		case out&clipRight != 0:
			y = y0 + (y1-y0)*(right-x0)/(x1-x0)
			x = right
		case out&clipLeft != 0:
			y = y0 + (y1-y0)*(left-x0)/(x1-x0)
			x = left
		}
		if out == c0 {
			x0, y0 = x, y
			c0 = encode(x0, y0)
		} else {
			x1, y1 = x, y
			c1 = encode(x1, y1)
		}
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

func (a *App) drawTacticalContactIcon(screen *ebiten.Image, c *acoustics.Contact, sx, sy float64, opts tacticalMapOpts) {
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
	if opts.minimap {
		return
	}
	label := c.ID
	if classified {
		label = contactLongLabel(c)
	}
	if opts.showSelection && c.SourceEntityID == a.selectedContactID {
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
	case world.KindTorpedo:
		// Cigar body, pointed nose to the right, cruciform fins aft.
		render.FillRect(screen, cx-7, cy-1, 12, 3, clr)
		render.FillRect(screen, cx+5, cy-1, 2, 3, clr)
		render.FillRect(screen, cx+7, cy, 2, 1, clr)
		render.DrawLine(screen, float64(cx-7), float64(cy), float64(cx-10), float64(cy-4), clr)
		render.DrawLine(screen, float64(cx-7), float64(cy), float64(cx-10), float64(cy+4), clr)
		render.DrawLine(screen, float64(cx-7), float64(cy-1), float64(cx-9), float64(cy), clr)
		render.DrawLine(screen, float64(cx-7), float64(cy+1), float64(cx-9), float64(cy), clr)
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
