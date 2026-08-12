package ui

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/world"
)

const (
	mastPanelX = 20
	mastPanelY = 50
	mastPanelW = 1260
	mastPanelH = 700

	mastMainX = mastPanelX + 16
	mastMainY = mastPanelY + 66 // below title + sea-state subtitle
	mastMainW = 820
	mastMainH = 602

	mastSideX = mastMainX + mastMainW + 16
	mastSideY = mastMainY
	mastSideW = 380
	mastSideH = 602

	mastStripY = mastMainY + 56
	mastStripH = 72
	mastCtrlY  = mastMainY + 150
	// Illumination sits below RAISE/LOWER (h=36): leave room for the caption above the bar.
	mastIllumY = mastCtrlY + 62
	// SAFE label is drawn at mastIllumY+h+14 — keep peri below that.
	mastPeriY = mastIllumY + 58
)

func (a *App) updateMastUI() {
	if a.Engine == nil || a.Engine.Scenario == nil || a.Engine.Scenario.Player == nil {
		return
	}
	player := a.Engine.Scenario.Player
	player.EnsureDamage()
	mx, my := ebiten.CursorPosition()

	msgX, msgY, msgW, msgH := a.mastCommMessageRect()
	lines := a.mastCommMessageLines()
	vis := msgH / 14
	if vis < 1 {
		vis = 1
	}
	scrollContactTableWheel(mx, my, msgX, msgY, msgW, msgH, len(lines), vis, &a.mastCommScroll)

	btns := a.mastButtons()
	a.updateButtonTooltips(btns, mx, my)
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	for _, m := range a.periMarkerHits {
		if mx >= m.X && mx < m.X+m.W && my >= m.Y && my < m.Y+m.H {
			sonar := &a.Engine.Sonar
			for i := range sonar.Contacts {
				if sonar.Contacts[i].SourceEntityID == m.SourceID {
					a.selectContact(sonar, &sonar.Contacts[i])
					return
				}
			}
			return
		}
	}
	if a.trySelectESMRFContact(mx, my) {
		return
	}
	for _, b := range btns {
		if !b.contains(mx, my) {
			continue
		}
		a.uiPressedID = b.ID
		a.uiPressedAt = time.Now()
		a.handleMastButton(b.ID)
		return
	}
}

const mastESMRFRowH = 22

func (a *App) mastESMRFTableRect() (x, y, w, h, maxRows int) {
	tableY := mastSideY + (mastSideH*3)/5
	x = mastSideX + 8
	y = tableY + 48
	w = mastSideW - 16
	maxRows = (mastSideY + mastSideH - 20 - y) / mastESMRFRowH
	if maxRows < 1 {
		maxRows = 1
	}
	h = maxRows * mastESMRFRowH
	return x, y, w, h, maxRows
}

// mastESMRFContacts lists sonar contacts with a recent RF paint (ESM intercept log order).
func (a *App) mastESMRFContacts() []*acoustics.Contact {
	if a.Engine == nil {
		return nil
	}
	sonar := &a.Engine.Sonar
	esm := &a.Engine.ESM
	gt := a.Engine.Clock.GameTime
	out := make([]*acoustics.Contact, 0, 8)
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if esm.HasRecentRF(c.SourceEntityID, gt) {
			out = append(out, c)
		}
	}
	return out
}

func (a *App) trySelectESMRFContact(mx, my int) bool {
	if a.Engine == nil || !a.Engine.ESM.MastUp() {
		return false
	}
	x, y, w, _, maxRows := a.mastESMRFTableRect()
	contacts := a.mastESMRFContacts()
	if len(contacts) > maxRows {
		contacts = contacts[:maxRows]
	}
	if mx < x || mx >= x+w || my < y || my >= y+len(contacts)*mastESMRFRowH {
		return false
	}
	row := (my - y) / mastESMRFRowH
	if row < 0 || row >= len(contacts) {
		return false
	}
	a.selectContact(&a.Engine.Sonar, contacts[row])
	return true
}

func (a *App) mastButtons() []uiButton {
	y := mastCtrlY
	cx := mastSideX + 14
	cy := mastSideY + 56
	py := mastPeriY + 28
	px := mastMainX + 20
	step := 5.0
	if a.Engine != nil {
		step = a.Engine.Periscope.TrainStepDeg()
	}
	return []uiButton{
		{ID: "esm_raise", Label: "RAISE ESM", Tooltip: "Raise ESM mast", X: mastMainX + 20, Y: y, W: 140, H: 36},
		{ID: "esm_lower", Label: "LOWER ESM", Tooltip: "Stow ESM mast", X: mastMainX + 172, Y: y, W: 140, H: 36},
		{ID: "comm_raise", Label: "RAISE", Tooltip: "Raise COMM mast", X: cx, Y: cy, W: 100, H: 32},
		{ID: "comm_lower", Label: "LOWER", Tooltip: "Stow COMM mast", X: cx + 108, Y: cy, W: 100, H: 32},
		{ID: "comm_report", Label: "REPORT", Tooltip: "Transmit mission status (COMM mast must be raised)", X: cx + 216, Y: cy, W: 110, H: 32},
		{ID: "peri_raise", Label: "RAISE", Tooltip: "Raise periscope", X: px, Y: py, W: 90, H: 32},
		{ID: "peri_lower", Label: "LOWER", Tooltip: "Stow periscope", X: px + 98, Y: py, W: 90, H: 32},
		{ID: "peri_left", Label: "< LEFT", Tooltip: fmt.Sprintf("Train periscope left %.0f°", step), X: px + 210, Y: py, W: 90, H: 32},
		{ID: "peri_right", Label: "RIGHT >", Tooltip: fmt.Sprintf("Train periscope right %.0f°", step), X: px + 308, Y: py, W: 100, H: 32},
		{ID: "peri_zoom_out", Label: "ZOOM -", Tooltip: "Wider field of view", X: px + 430, Y: py, W: 90, H: 32},
		{ID: "peri_zoom_in", Label: "ZOOM +", Tooltip: "Narrower field of view", X: px + 528, Y: py, W: 90, H: 32},
	}
}

func (a *App) handleMastButton(id string) {
	esm := &a.Engine.ESM
	comm := &a.Engine.COMM
	peri := &a.Engine.Periscope
	player := a.Engine.Scenario.Player
	switch id {
	case "esm_raise":
		ok, msg := esm.OrderRaiseESM(player)
		a.StatusMessage = msg
		if ok {
			a.playMastHydraulicFX()
		} else if a.Audio != nil {
			a.Audio.PlayClip(audio.ClipDiveUnableDeeper, msg)
		}
	case "esm_lower":
		a.StatusMessage = esm.OrderLowerESM()
		a.playMastHydraulicFX()
	case "comm_raise":
		ok, msg := comm.OrderRaiseCOMM(player)
		a.StatusMessage = msg
		if ok {
			a.playMastHydraulicFX()
		} else if a.Audio != nil {
			a.Audio.PlayClip(audio.ClipDiveUnableDeeper, msg)
		}
	case "comm_lower":
		a.StatusMessage = comm.OrderLowerCOMM()
		a.playMastHydraulicFX()
	case "comm_report":
		if !comm.MastUp() {
			a.StatusMessage = "Unable to transmit — raise COMM mast first."
			if a.Audio != nil {
				a.Audio.PlayClip(audio.ClipDiveUnableDeeper, a.StatusMessage)
			}
			break
		}
		gt := a.Engine.Clock.GameTime
		report := a.Engine.Scenario.MissionStatusReport()
		comm.AppendLocalTraffic(gt, report)
		a.StatusMessage = "Mission status transmitted on COMM."
		a.mastCommScroll = 1 << 20
	case "peri_raise":
		ok, msg := peri.OrderRaise(player)
		a.StatusMessage = msg
		if ok {
			a.playMastHydraulicFX()
		} else if a.Audio != nil {
			a.Audio.PlayClip(audio.ClipDiveUnableDeeper, msg)
		}
	case "peri_lower":
		a.StatusMessage = peri.OrderLower()
		a.playMastHydraulicFX()
	case "peri_left":
		peri.TrainLeft()
		a.StatusMessage = fmt.Sprintf("Periscope train %+.0f° rel / %03.0f°T", peri.TrainRelDeg, peri.TrueBearingDeg(player.HeadingDeg))
	case "peri_right":
		peri.TrainRight()
		a.StatusMessage = fmt.Sprintf("Periscope train %+.0f° rel / %03.0f°T", peri.TrainRelDeg, peri.TrueBearingDeg(player.HeadingDeg))
	case "peri_zoom_in":
		peri.ZoomIn()
		a.StatusMessage = "Periscope zoom " + peri.ZoomLabel()
	case "peri_zoom_out":
		peri.ZoomOut()
		a.StatusMessage = "Periscope zoom " + peri.ZoomLabel()
	}
}

func (a *App) playMastHydraulicFX() {
	if a.Audio != nil {
		a.Audio.PlayMastHydraulic()
	}
}

func (a *App) drawMast(screen *ebiten.Image) {
	render.DrawConsolePanel(screen, mastPanelX, mastPanelY, mastPanelW, mastPanelH)
	render.DrawScreenTitle(screen, "MAST — ESM / COMM / SCOPE", mastPanelX+20, mastPanelY+28)

	weather := world.WeatherLight
	if a.Engine != nil && a.Engine.Scenario != nil {
		weather = a.Engine.Scenario.Weather
	}
	render.DrawText(screen, "SEA STATE: "+weather.String(), mastPanelX+20, mastPanelY+48, render.ColorPlateLabel, true)

	a.drawMastMainPlate(screen)
	a.drawMastSidePlate(screen)

	mx, my := ebiten.CursorPosition()
	if a.uiTooltip != "" {
		render.DrawTooltip(screen, mx, my, a.uiTooltip)
	}
}

func (a *App) drawMastPlate(screen *ebiten.Image, x, y, w, h int, title string) {
	render.FillRect(screen, x, y, w, h, color.RGBA{18, 18, 20, 255})
	border := color.RGBA{78, 78, 84, 255}
	render.FillRect(screen, x, y, w, 1, border)
	render.FillRect(screen, x, y+h-1, w, 1, border)
	render.FillRect(screen, x, y, 1, h, border)
	render.FillRect(screen, x+w-1, y, 1, h, border)
	render.DrawText(screen, title, x+12, y+20, render.ColorPlateLabel, true)
}

func (a *App) drawMastMainPlate(screen *ebiten.Image) {
	a.drawMastPlate(screen, mastMainX, mastMainY, mastMainW, mastMainH, "ESM SUITE")

	if a.Engine == nil {
		return
	}
	player := a.Engine.Scenario.Player
	esm := &a.Engine.ESM
	player.EnsureDamage()

	status := "STOWED"
	statusClr := render.ColorDim
	switch {
	case player.Damage.Destroyed(world.SysESM) || esm.Sheared:
		status = "DESTROYED"
		statusClr = render.ColorDanger
	case esm.MastUp():
		status = "RAISED — RECEIVING"
		statusClr = render.ColorPhosphor
	case esm.MastMoving():
		if esm.Order == acoustics.ESMMastRaise {
			status = fmt.Sprintf("RAISING %.0f%%", esm.Extension*100)
		} else {
			status = fmt.Sprintf("LOWERING %.0f%%", esm.Extension*100)
		}
		statusClr = render.ColorAmber
	}
	render.DrawText(screen, "MAST: "+status, mastMainX+20, mastMainY+40, statusClr, true)

	// Bearing heat strip (000 center).
	stripX := mastMainX + 20
	stripW := mastMainW - 40
	a.drawESMBearingStrip(screen, stripX, mastStripY, stripW, mastStripH)

	// Controls (ESM only — COMM lives on the side plate).
	mx, my := ebiten.CursorPosition()
	for _, b := range a.mastButtons() {
		if b.ID != "esm_raise" && b.ID != "esm_lower" {
			continue
		}
		hover := b.contains(mx, my)
		pressed := a.uiPressedID == b.ID && time.Since(a.uiPressedAt) < 120*time.Millisecond
		latched := false
		switch b.ID {
		case "esm_raise":
			latched = esm.Order == acoustics.ESMMastRaise && !esm.Sheared
		case "esm_lower":
			latched = esm.Order == acoustics.ESMMastStow
		}
		if latched {
			render.FillRect(screen, b.X-2, b.Y-2, b.W+4, b.H+4, render.ColorAmber)
		}
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	}

	a.drawESMIlluminationBar(screen, mastMainX+20, mastIllumY, mastMainW-40, 28, esm.MaxIllumination)
	a.drawPeriscopeBlock(screen)
}

func (a *App) drawPeriscopeBlock(screen *ebiten.Image) {
	if a.Engine == nil || a.Engine.Scenario == nil || a.Engine.Scenario.Player == nil {
		return
	}
	player := a.Engine.Scenario.Player
	peri := &a.Engine.Periscope
	player.EnsureDamage()

	render.DrawText(screen, "PERISCOPE", mastMainX+20, mastPeriY+14, render.ColorPlateLabel, true)

	status := "STOWED"
	statusClr := render.ColorDim
	switch {
	case player.Damage.Destroyed(world.SysPeriscope) || peri.Sheared:
		status = "DESTROYED"
		statusClr = render.ColorDanger
	case peri.MastUp():
		status = "RAISED — IR SENSOR"
		statusClr = render.ColorPhosphor
	case peri.MastMoving():
		if peri.Order == acoustics.PeriMastRaise {
			status = fmt.Sprintf("RAISING %.0f%%", peri.Extension*100)
		} else {
			status = fmt.Sprintf("LOWERING %.0f%%", peri.Extension*100)
		}
		statusClr = render.ColorAmber
	}
	render.DrawText(screen, "SCOPE: "+status, mastMainX+140, mastPeriY+14, statusClr, true)
	lockTxt := ""
	if peri.Locked() {
		lockTxt = "   LOCK"
	}
	render.DrawText(screen, fmt.Sprintf("ZOOM %s   TRAIN %+.0f°   BRG %03.0f°T%s",
		peri.ZoomLabel(), peri.TrainRelDeg, peri.TrueBearingDeg(player.HeadingDeg), lockTxt),
		mastMainX+420, mastPeriY+14, render.ColorPhosphorDim, true)

	mx, my := ebiten.CursorPosition()
	for _, b := range a.mastButtons() {
		switch b.ID {
		case "peri_raise", "peri_lower", "peri_left", "peri_right", "peri_zoom_in", "peri_zoom_out":
		default:
			continue
		}
		hover := b.contains(mx, my)
		pressed := a.uiPressedID == b.ID && time.Since(a.uiPressedAt) < 120*time.Millisecond
		latched := false
		switch b.ID {
		case "peri_raise":
			latched = peri.Order == acoustics.PeriMastRaise && !peri.Sheared
		case "peri_lower":
			latched = peri.Order == acoustics.PeriMastStow
		case "peri_zoom_in":
			latched = peri.Zoom == acoustics.PeriZoomHigh
		case "peri_zoom_out":
			latched = peri.Zoom == acoustics.PeriZoomLow
		}
		if latched {
			render.FillRect(screen, b.X-2, b.Y-2, b.W+4, b.H+4, render.ColorAmber)
		}
		render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
	}

	vx, vy, vw, vh := a.mastPeriscopeViewRect()
	a.drawPeriscopeOptic(screen, vx, vy, vw, vh, peri, player)
}

func (a *App) mastPeriscopeViewRect() (x, y, w, h int) {
	x = mastMainX + 20
	y = mastPeriY + 68
	w = mastMainW - 40
	h = (mastMainY + mastMainH) - y - 12
	if h < 120 {
		h = 120
	}
	return x, y, w, h
}

func (a *App) drawESMBearingStrip(screen *ebiten.Image, x, y, w, h int) {
	render.FillRect(screen, x, y, w, h, render.ColorPanelInset)
	render.DrawText(screen, "BEARING", x+4, y-4, render.ColorPhosphorDim, true)

	esm := &a.Engine.ESM
	up := esm.MastUp()
	for px := 0; px < w; px++ {
		brg := waterfallDisplayBearingDeg(px, w) // relative, 000 center
		bin := int(brg) % 360
		if bin < 0 {
			bin += 360
		}
		v := 0.0
		if up {
			v = esm.BearingHeat[bin]
		}
		if v < 0.02 {
			continue
		}
		clr := esmHeatColor(v)
		render.FillRect(screen, x+px, y+14, 1, h-18, clr)
	}

	// Ruler ticks / labels (reuse ACTIVE orientation).
	a.drawESMBearingRuler(screen, x, y, w, h)

	// Contact chips only while the mast is up; bearings frozen between RF paints.
	if !up || a.Engine == nil {
		return
	}
	player := a.Engine.Scenario.Player
	sonar := &a.Engine.Sonar
	esmState := &a.Engine.ESM
	gt := 0.0
	if a.Engine != nil {
		gt = a.Engine.Clock.GameTime
	}
	drawn := map[string]bool{}
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if !esmState.HasRecentRF(c.SourceEntityID, gt) {
			continue
		}
		rf := 0.0
		if esmState.RFConfidence != nil {
			rf = esmState.RFConfidence[c.SourceEntityID]
		}
		brg := esmState.FrozenRFBearing(c.SourceEntityID, c.BearingDeg)
		rel := normalizeGyroDeg(brg - player.HeadingDeg)
		sx := x + waterfallBearingDisplayX(rel, w)
		if sx < x+4 || sx > x+w-4 {
			continue
		}
		if drawn[c.ID] {
			continue
		}
		drawn[c.ID] = true
		label := c.ID
		clr := render.ColorPhosphor
		if rf >= 0.60 && esmState.RFEquipmentClass(c.SourceEntityID) != "" {
			clr = render.ColorAmber
		}
		render.DrawText(screen, label, sx-len(label)*3, y+12, clr, true)
		render.FillRect(screen, sx-1, y+14, 2, 4, clr)
	}
}

func (a *App) drawESMBearingRuler(screen *ebiten.Image, x, y, w, h int) {
	baseY := y + h - 2
	for brg := 0; brg < 360; brg += 10 {
		px := x + waterfallBearingDisplayX(float64(brg), w)
		tick := 4
		if brg%30 == 0 {
			tick = 8
		}
		if brg%90 == 0 {
			tick = 12
		}
		render.DrawLine(screen, float64(px), float64(baseY), float64(px), float64(baseY-tick), render.ColorPhosphorDim)
	}
	labels := []struct {
		brg float64
		txt string
	}{
		{180, "180"}, {270, "270"}, {0, "000"}, {90, "090"}, {180, "180"},
	}
	for i, L := range labels {
		px := x + waterfallBearingDisplayX(L.brg, w)
		tx := px - 10
		if i == 0 {
			tx = x + 2
		}
		if i == len(labels)-1 {
			tx = x + w - 24
		}
		render.DrawText(screen, L.txt, tx, baseY-14, render.ColorPhosphorDim, true)
	}
	// Ownship heading marker at center.
	cx := x + waterfallBearingDisplayX(0, w)
	render.DrawLine(screen, float64(cx), float64(y+14), float64(cx), float64(y+h-4), color.RGBA{0, 220, 160, 120})
}

func esmHeatColor(v float64) color.Color {
	if v < 0.25 {
		return color.RGBA{0, uint8(80 + v*400), uint8(60 + v*200), 255}
	}
	if v < 0.55 {
		return color.RGBA{uint8(40 + v*200), uint8(160), 40, 255}
	}
	return color.RGBA{255, uint8(200 - v*80), 40, 255}
}

func (a *App) drawESMIlluminationBar(screen *ebiten.Image, x, y, w, h int, illum float64) {
	render.DrawText(screen, "OWN MAST ILLUMINATION (hostile/neutral search radars)", x, y-6, render.ColorPhosphorDim, true)
	const segs = 12
	segW := w / segs
	band := acoustics.IlluminationBand(illum)
	filled := int(math.Ceil(illum * float64(segs)))
	if illum > 0 && filled < 1 {
		filled = 1
	}
	for i := 0; i < segs; i++ {
		sx := x + i*segW
		face := color.RGBA{28, 28, 30, 255}
		if i < filled {
			switch {
			case float64(i+1)/float64(segs) >= 0.55 || band == 2:
				face = color.RGBA{200, 50, 40, 255}
			case float64(i+1)/float64(segs) >= 0.28 || band == 1:
				face = color.RGBA{210, 170, 40, 255}
			default:
				face = color.RGBA{40, 160, 80, 255}
			}
			// Color by segment threshold, not max band, so left stays green.
			frac := float64(i+1) / float64(segs)
			switch {
			case frac >= 0.55:
				face = color.RGBA{200, 50, 40, 255}
			case frac >= 0.28:
				face = color.RGBA{210, 170, 40, 255}
			default:
				face = color.RGBA{40, 160, 80, 255}
			}
		}
		render.FillRect(screen, sx+1, y+4, segW-2, h-8, face)
	}
	label := "SAFE"
	clr := color.RGBA{40, 160, 80, 255}
	switch band {
	case 1:
		label = "DETECTABLE"
		clr = color.RGBA{210, 170, 40, 255}
	case 2:
		label = "PAINTED"
		clr = color.RGBA{200, 50, 40, 255}
	}
	if !a.Engine.ESM.MastUp() {
		label = "MAST DOWN"
		clr = render.ColorDim
	}
	render.DrawText(screen, label, x+w-len(label)*7, y+h+14, clr, true)
}

func (a *App) mastCommMessageRect() (x, y, w, h int) {
	tableY := mastSideY + (mastSideH*3)/5
	x = mastSideX + 12
	// Below COMM buttons (y=56, h=32) with a small gap — no TRAFFIC label.
	y = mastSideY + 56 + 32 + 10
	w = mastSideW - 24
	// Leave clear air above "RF INTERCEPT LOG" (drawn at tableY-28).
	h = tableY - y - 52
	if h < 48 {
		h = 48
	}
	return x, y, w, h
}

func (a *App) mastCommMessageLines() []string {
	if a.Engine == nil {
		return nil
	}
	var lines []string
	for _, msg := range a.Engine.COMM.Inbox {
		min := int(msg.TimeSec) / 60
		sec := int(msg.TimeSec) % 60
		lines = append(lines, fmt.Sprintf("[T+%02d:%02d]", min, sec))
		for _, part := range strings.Split(msg.Text, "\n") {
			wrapped := wrapTextWidth(part, 42)
			if len(wrapped) == 0 {
				lines = append(lines, "")
				continue
			}
			lines = append(lines, wrapped...)
		}
		lines = append(lines, "")
	}
	return lines
}

func wrapTextWidth(s string, width int) []string {
	if width < 8 {
		width = 8
	}
	s = strings.TrimRight(s, "\r")
	if s == "" {
		return []string{""}
	}
	var out []string
	for len(s) > width {
		cut := width
		if sp := strings.LastIndex(s[:width], " "); sp > width/3 {
			cut = sp
		}
		out = append(out, strings.TrimRight(s[:cut], " "))
		s = strings.TrimLeft(s[cut:], " ")
	}
	if s != "" {
		out = append(out, s)
	}
	return out
}

func (a *App) drawMastSidePlate(screen *ebiten.Image) {
	a.drawMastPlate(screen, mastSideX, mastSideY, mastSideW, mastSideH, "COMM / CONTACTS")

	if a.Engine == nil {
		return
	}
	esm := &a.Engine.ESM
	comm := &a.Engine.COMM
	gt := a.Engine.Clock.GameTime
	player := a.Engine.Scenario.Player
	player.EnsureDamage()
	wx := mastSideX + 14

	// —— COMM mast header ——
	status := "STOWED"
	statusClr := render.ColorDim
	switch {
	case player.Damage.Destroyed(world.SysCOMM) || comm.Sheared:
		status = "DESTROYED"
		statusClr = render.ColorDanger
	case comm.MastUp():
		status = "RAISED — RECEIVING"
		statusClr = render.ColorPhosphor
	case comm.MastMoving():
		if comm.Order == acoustics.COMMMastRaise {
			status = fmt.Sprintf("RAISING %.0f%%", comm.Extension*100)
		} else {
			status = fmt.Sprintf("LOWERING %.0f%%", comm.Extension*100)
		}
		statusClr = render.ColorAmber
	}
	render.DrawText(screen, "COMM MAST: "+status, wx, mastSideY+40, statusClr, true)

	mx, my := ebiten.CursorPosition()
	for _, b := range a.mastButtons() {
		if b.ID != "comm_raise" && b.ID != "comm_lower" && b.ID != "comm_report" {
			continue
		}
		hover := b.contains(mx, my)
		pressed := a.uiPressedID == b.ID && time.Since(a.uiPressedAt) < 120*time.Millisecond
		latched := false
		disabled := false
		switch b.ID {
		case "comm_raise":
			latched = comm.Order == acoustics.COMMMastRaise && !comm.Sheared
		case "comm_lower":
			latched = comm.Order == acoustics.COMMMastStow
		case "comm_report":
			disabled = !comm.MastUp()
		}
		if latched {
			render.FillRect(screen, b.X-2, b.Y-2, b.W+4, b.H+4, render.ColorAmber)
		}
		if disabled {
			render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, false, false)
			render.FillRect(screen, b.X, b.Y, b.W, b.H, color.RGBA{0, 0, 0, 110})
		} else {
			render.DrawBevelButton(screen, b.X, b.Y, b.W, b.H, b.Label, hover, pressed)
		}
	}

	// —— Scrollable message traffic ——
	msgX, msgY, msgW, msgH := a.mastCommMessageRect()
	render.FillRect(screen, msgX, msgY, msgW, msgH, render.ColorPanelInset)
	lines := a.mastCommMessageLines()
	vis := msgH / 14
	if vis < 1 {
		vis = 1
	}
	a.mastCommScroll = clampContactTableScroll(a.mastCommScroll, len(lines), vis)
	start, end := contactTableWindow(len(lines), a.mastCommScroll, vis)
	ty := msgY + 12
	for i := start; i < end; i++ {
		clr := render.ColorPhosphor
		if strings.HasPrefix(lines[i], "[T+") {
			clr = render.ColorAmber
		}
		render.DrawText(screen, trunc(lines[i], 48), msgX+6, ty, clr, true)
		ty += 14
	}
	if len(lines) == 0 {
		render.DrawText(screen, "No traffic", msgX+6, msgY+16, render.ColorDim, true)
	}
	drawContactTableScrollbar(screen, msgX+msgW-6, msgY+8, msgH-12, len(lines), vis, a.mastCommScroll)

	// —— RF intercept table (lower band) ——
	tableY := mastSideY + (mastSideH*3)/5
	render.DrawText(screen, "RF INTERCEPT LOG", wx, tableY-28, render.ColorPhosphorDim, true)
	render.DrawText(screen, "EQIP = radar set, not hull ID", wx, tableY-12, render.ColorDim, true)
	render.DrawLine(screen, float64(mastSideX+8), float64(tableY), float64(mastSideX+mastSideW-8), float64(tableY), color.RGBA{78, 78, 84, 255})
	render.DrawText(screen, "ID", wx, tableY+18, render.ColorPhosphorDim, true)
	render.DrawText(screen, "BRG", wx+36, tableY+18, render.ColorPhosphorDim, true)
	render.DrawText(screen, "SRC", wx+72, tableY+18, render.ColorPhosphorDim, true)
	render.DrawText(screen, "EQIP", wx+118, tableY+18, render.ColorPhosphorDim, true)
	render.DrawText(screen, "RF%", wx+230, tableY+18, render.ColorPhosphorDim, true)
	render.DrawText(screen, "LAST", wx+278, tableY+18, render.ColorPhosphorDim, true)
	render.DrawText(screen, "SGNL", wx+278, tableY+32, render.ColorPhosphorDim, true)

	tx, rowY0, tw, _, maxRows := a.mastESMRFTableRect()
	contacts := a.mastESMRFContacts()
	if len(contacts) > maxRows {
		contacts = contacts[:maxRows]
	}
	clickable := esm.MastUp()
	rowY := rowY0
	for _, c := range contacts {
		rf := 0.0
		if esm.RFConfidence != nil {
			rf = esm.RFConfidence[c.SourceEntityID]
		}
		clr := render.ColorPhosphor
		equip := "—"
		if rf >= 0.60 {
			if name := esm.RFEquipmentClass(c.SourceEntityID); name != "" {
				equip = trunc(name, 16)
				clr = render.ColorAmber
			}
		}
		selected := c.SourceEntityID == a.selectedContactID
		hover := clickable && mx >= tx && mx < tx+tw && my >= rowY && my < rowY+mastESMRFRowH
		if selected {
			render.FillRect(screen, tx, rowY, tw, mastESMRFRowH, color.RGBA{80, 60, 0, 180})
			clr = render.ColorAmber
		} else if hover {
			render.FillRect(screen, tx, rowY, tw, mastESMRFRowH, render.ColorPanelMid)
		}
		brg := esm.FrozenRFBearing(c.SourceEntityID, c.BearingDeg)
		src := trunc(c.DetectedBy, 6)
		age := esm.SecondsSinceRF(c.SourceEntityID, gt)
		// DrawText y is baseline — inset into the row like PASSIVE contact list.
		ty := rowY + 16
		render.DrawText(screen, c.ID, wx, ty, clr, true)
		render.DrawText(screen, fmt.Sprintf("%03.0f", brg), wx+36, ty, clr, true)
		render.DrawText(screen, src, wx+72, ty, clr, true)
		render.DrawText(screen, equip, wx+118, ty, clr, true)
		render.DrawText(screen, fmt.Sprintf("%.0f", rf*100), wx+230, ty, clr, true)
		render.DrawText(screen, fmt.Sprintf("%.0fs", age), wx+278, ty, clr, true)
		rowY += mastESMRFRowH
	}
	if len(contacts) == 0 {
		hint := "No RF intercepts"
		if !clickable {
			hint = "Raise ESM for intercepts"
		}
		render.DrawText(screen, hint, wx, rowY0+16, render.ColorDim, true)
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
