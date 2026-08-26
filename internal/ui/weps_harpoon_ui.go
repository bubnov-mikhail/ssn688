package ui

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/i18n"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

const (
	wepsOrdnanceBtnX  = wepsLeftX + 82
	wepsOrdnanceMenuH = 44
)

func contactTMAUsableForLead(c *acoustics.Contact) bool {
	return c != nil && c.TMAAccuracy >= 0.5 && c.TMASpeedKts >= 0.5
}

func (a *App) wepsTubeRowStatusExtra(t weapons.Tube, reloadRemainSec float64) string {
	switch t.State {
	case weapons.TubeEmpty:
		return a.L(i18n.UIEmpty)
	case weapons.TubeReloading:
		if reloadRemainSec > 0 {
			return a.Lf(i18n.UIReloadSecs, int(reloadRemainSec+0.5))
		}
		return a.L(i18n.UIReloading)
	case weapons.TubeFired:
		if t.WireIntact {
			return "wired"
		}
	}
	return ""
}

func (a *App) wepsOrdnancePickButton(tube int, y int) sonarUIButton {
	label := "Mk48 ▼"
	fc := &a.Engine.FireControl
	if fc != nil && tube >= 1 && tube <= 4 {
		t := fc.Tubes[tube-1]
		switch t.State {
		case weapons.TubeReloading:
			if t.ReloadOrdnance != "" {
				label = weapons.NormalizeOrdnance(t.ReloadOrdnance) + " ▼"
			} else {
				label = "? ▼"
			}
		case weapons.TubeEmpty:
			label = "— ▼"
		default:
			if t.TorpedoType != "" {
				label = weapons.NormalizeOrdnance(t.TorpedoType) + " ▼"
			}
		}
	}
	w := render.ButtonWidth(label, 10)
	if w < 56 {
		w = 56
	}
	const btnH = 24
	btnY := y + 4 - btnH/2 // same vertical center as OPEN/FIRE
	return sonarUIButton{
		ID:      fmt.Sprintf("ordnance_%d", tube),
		Label:   label,
		Tooltip: a.L(i18n.UITipOrdnance),
		X:       wepsOrdnanceBtnX,
		Y:       btnY,
		W:       w,
		H:       btnH,
	}
}

func (a *App) wepsOrdnanceMenuButtons(tube int, y int) []sonarUIButton {
	if a.wepsOrdnanceMenuTube != tube {
		return nil
	}
	ordBtn := a.wepsOrdnancePickButton(tube, y)
	menuY := ordBtn.Y + ordBtn.H + 2
	var btns []sonarUIButton
	for i, ord := range weapons.AllTubeOrdnance() {
		btns = append(btns, sonarUIButton{
			ID:      fmt.Sprintf("ordnance_%d_pick_%s", tube, ord),
			Label:   ord,
			Tooltip: fmt.Sprintf("Load %s into tube %d", ord, tube),
			X:       ordBtn.X,
			Y:       menuY + i*wepsOrdnanceMenuH/2,
			W:       ordBtn.W,
			H:       wepsOrdnanceMenuH/2 - 1,
		})
	}
	return btns
}

func (a *App) wepsHarpoonCtrlButtons(fc *weapons.FireControl) []sonarUIButton {
	spinY, spinH, btnW, g0, _, inner, _ := wepsCtrlSpinLayout(0)

	beamLabel := a.L(i18n.UIBeamWide)
	if fc.HarpoonRadarBeam == weapons.HarpoonBeamNarrow {
		beamLabel = a.L(i18n.UIBeamNarrow)
	}

	gyroBtns := []sonarUIButton{
		{ID: "gyro_m", Label: "-", Tooltip: "Decrease course 5°", X: g0, Y: spinY, W: btnW, H: spinH},
		{ID: "gyro_p", Label: "+", Tooltip: "Increase course 5°", X: g0 + btnW + inner, Y: spinY, W: btnW, H: spinH},
	}
	rowX := g0 + btnW + inner + btnW + 10
	row := layoutButtonRow(rowX, spinY, spinH, 4, []buttonSpec{
		{"harp_beam", beamLabel, "Radar search beam width (±30° wide / ±15° narrow)"},
		{"harp_srch", weapons.HarpoonRadarRangeLabel(fc.HarpoonRadarRange), "Distance before active radar search activates"},
		{"harp_dstr", weapons.HarpoonDestructRangeLabel(fc.HarpoonDestructRange), "Self-destruct range if no target (must exceed SRCH)"},
	})
	return append(gyroBtns, row...)
}

func (a *App) wepsCollectOrdnanceButtons(fc *weapons.FireControl) []sonarUIButton {
	var btns []sonarUIButton
	for i := 1; i <= 4; i++ {
		y := wepsTubeY0 + (i-1)*wepsTubeRowH
		btns = append(btns, a.wepsOrdnancePickButton(i, y))
		btns = append(btns, a.wepsOrdnanceMenuButtons(i, y)...)
	}
	return btns
}

func (a *App) wepsOrdnanceAction(id string, fc *weapons.FireControl, gameTime float64) bool {
	if len(id) < 9 || id[:9] != "ordnance_" {
		return false
	}
	rest := id[9:]
	if parts := strings.SplitN(rest, "_pick_", 2); len(parts) == 2 {
		tube, err := strconv.Atoi(parts[0])
		if err == nil && tube >= 1 && tube <= 4 {
			a.wepsOrdnanceMenuTube = 0
			if fc.RequestOrdnanceReload(tube, parts[1], gameTime) {
				a.Statusf(i18n.StatusTubeReloading, tube, parts[1])
			}
			return true
		}
	}
	tube, err := strconv.Atoi(rest)
	if err != nil || tube < 1 || tube > 4 {
		return true
	}
	t := fc.TubeByNumber(tube)
	if t != nil && (t.State == weapons.TubeDoorOpen || t.State == weapons.TubeFired) {
		a.Status(i18n.StatusCannotChangeOrdnance)
		return true
	}
	if a.wepsOrdnanceMenuTube == tube {
		a.wepsOrdnanceMenuTube = 0
	} else {
		a.wepsOrdnanceMenuTube = tube
	}
	return true
}

func (a *App) wepsFireTube(fc *weapons.FireControl, player *world.Entity, tube int, gameTime float64) {
	fc.SelectTube(tube)
	player.EnsureDamage()
	sys := world.TubeSys(tube)
	if sys != world.SysNone && !player.Damage.Operational(sys) {
		a.Statusf(i18n.StatusTubeDamagedNoFire, tube)
		return
	}
	t := fc.TubeByNumber(tube)
	if t == nil || t.State != weapons.TubeDoorOpen {
		a.Status(i18n.StatusOpenDoorFirst)
		return
	}
	ord := weapons.NormalizeOrdnance(t.TorpedoType)
	if ord == weapons.OrdnanceHarpoon {
		if h := fc.ShootHarpoon(player, tube); h != nil {
			a.Engine.EmitHarpoonLaunch(player, gameTime)
			if a.Audio != nil {
				a.Audio.PlayTorpedoLaunch()
				a.Audio.PlayClip(audio.TubeClip("torpedo_away", tube),
					a.Lf(i18n.StatusVoiceHarpoonAway, tube))
			}
		}
		return
	}
	if torp := fc.Shoot(player, tube); torp != nil {
		a.markOwnTorpedo(torp.ID)
		if a.Audio != nil {
			a.Audio.PlayTorpedoLaunch()
			a.Audio.PlayClip(audio.TubeClip("torpedo_away", tube),
				a.Lf(i18n.StatusVoiceTorpedoAway, tube))
		}
	} else {
		a.Status(i18n.StatusOpenDoorFirst)
	}
}

func (a *App) wepsApplyContactToHarpoonPrep(fc *weapons.FireControl, player *world.Entity, c *acoustics.Contact) {
	if fc == nil || c == nil {
		return
	}
	fc.GyroAngleDeg = normalizeGyroDeg(c.BearingDeg)
	gt := 0.0
	if a.Engine != nil {
		gt = a.Engine.Clock.GameTime
	}
	if player != nil && contactHasKnownRange(c, gt) && contactTMAUsableForLead(c) {
		tx, ty := contactPlotRaw(player, c, gt)
		if course, ok := weapons.InterceptCourseDeg(
			tx-player.X, ty-player.Y,
			c.TMACourseDeg, c.TMASpeedKts, weapons.HarpoonCruiseKts,
		); ok {
			fc.GyroAngleDeg = normalizeGyroDeg(course)
		}
	}
}

func (a *App) drawWepsOrdnanceMenus(screen *ebiten.Image, fc *weapons.FireControl, mx, my int) {
	for i := 1; i <= 4; i++ {
		if a.wepsOrdnanceMenuTube != i {
			continue
		}
		y := wepsTubeY0 + (i-1)*wepsTubeRowH
		ordBtn := a.wepsOrdnancePickButton(i, y)
		menuY := ordBtn.Y + ordBtn.H + 2
		h := len(weapons.AllTubeOrdnance()) * wepsOrdnanceMenuH / 2
		render.FillRect(screen, ordBtn.X-1, menuY-1, ordBtn.W+2, h+2, color.RGBA{24, 24, 28, 250})
		for _, b := range a.wepsOrdnanceMenuButtons(i, y) {
			a.drawWepsButton(screen, b, mx, my, fc)
		}
	}
}

func (a *App) drawWepsHarpoonGeometry(screen *ebiten.Image, px, py float64, player *world.Entity, h *weapons.HarpoonMissile) {
	if h == nil || !h.VisibleOnWEPS {
		return
	}
	ax, ay := h.AssumedXY()
	tx := px + (ax-player.X)*a.wepsMapZoom
	ty := py - (ay-player.Y)*a.wepsMapZoom
	if !wepsMapMarkerInside(tx, ty) {
		return
	}
	head := h.ProgrammedHead
	if head == 0 && h.HeadingDeg != 0 && h.AssumedDistanceYd < 1 {
		head = h.HeadingDeg
	}
	// Compact markers (same order of magnitude as torpedo cone/line on WEPS).
	const (
		cruiseSec = 600.0
		refSpd    = weapons.HarpoonCruiseKts
	)
	maxRangeYd := cruiseSec * refSpd * world.KnotsToYPS
	coneYd := maxRangeYd / 80 // short seek cone
	lineYd := maxRangeYd / 60 // short course stub

	rad := head * math.Pi / 180
	lx := tx + math.Sin(rad)*lineYd*a.wepsMapZoom
	ly := ty - math.Cos(rad)*lineYd*a.wepsMapZoom
	render.DrawLine(screen, tx, ty, lx, ly, color.RGBA{255, 140, 60, 255})
	render.FillRect(screen, int(tx)-2, int(ty)-2, 5, 5, color.RGBA{255, 120, 40, 255})

	beam := h.BeamHalfDeg
	if beam <= 0 {
		beam = weapons.HarpoonWideBeamDeg
	}
	coneR := coneYd * a.wepsMapZoom
	for _, ang := range []float64{head - beam, head + beam} {
		ar := ang * math.Pi / 180
		render.DrawLine(screen, tx, ty, tx+math.Sin(ar)*coneR, ty-math.Cos(ar)*coneR, color.RGBA{255, 180, 80, 160})
	}
}
