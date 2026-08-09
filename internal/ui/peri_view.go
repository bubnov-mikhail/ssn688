package ui

import (
	"fmt"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

const (
	periIRW          = 360
	periIRH          = 200
	periLandStepYd   = 100.0
	periLandInlandYd = 400.0
)

// periBGCacheEnabled reuses the sky/sea/land plate. Benches may disable it.
var periBGCacheEnabled = true

type periShipDraw struct {
	proj   acoustics.PeriShipProj
	waterY int
}

func (a *App) disposePeriscopeImage() {
	disposeImage(&a.periImg)
	a.periPix = nil
	a.periBgPix = nil
	a.periBgCacheKey = 0
}

func (a *App) ensurePeriscopeImage() {
	need := periIRW * periIRH * 4
	if a.periImg != nil && a.periImg.Bounds().Dx() == periIRW && a.periImg.Bounds().Dy() == periIRH {
		if len(a.periPix) == need && len(a.periBgPix) == need {
			return
		}
	}
	disposeImage(&a.periImg)
	a.periImg = ebiten.NewImage(periIRW, periIRH)
	a.periPix = make([]byte, need)
	a.periBgPix = make([]byte, need)
	a.periBgCacheKey = 0
}

// periViewBackgroundCacheKey covers only sky/sea/land inputs (not ships/FX).
func (a *App) periViewBackgroundCacheKey(player *world.Entity, peri *acoustics.PeriscopeState, weather world.Weather, _ float64) uint64 {
	// Quantize pose so tiny drift does not thrash land rays every frame.
	// Sea plate is static (no time ripple) — matches cold, uniform IR water.
	px := int64(player.X / 20)
	py := int64(player.Y / 20)
	hd := int64(player.HeadingDeg * 2)
	tr := int64(peri.TrainRelDeg * 2)
	zm := int64(peri.Zoom)
	ext := int64(peri.Extension * 20)
	dep := int64(player.DepthFt)
	wt := int64(weather)
	k := uint64(px)<<48 ^ uint64(py)<<32 ^ uint64(hd&0x1ff)<<23 ^ uint64(tr&0x1ff)<<14
	k ^= uint64(zm&7)<<11 ^ uint64(ext&0x1f)<<6 ^ uint64(dep&0x3f) ^ uint64(wt)<<3
	return k
}

// drawPeriscopeOptic draws the IR exterior into the reserved optic rect and
// overlays bearing/FOV chrome + reticule.
func (a *App) drawPeriscopeOptic(screen *ebiten.Image, x, y, w, h int, peri *acoustics.PeriscopeState, player *world.Entity) {
	render.FillRect(screen, x, y, w, h, color.RGBA{8, 10, 12, 255})
	border := color.RGBA{70, 78, 72, 255}
	render.FillRect(screen, x, y, w, 1, border)
	render.FillRect(screen, x, y+h-1, w, 1, border)
	render.FillRect(screen, x, y, 1, h, border)
	render.FillRect(screen, x+w-1, y, 1, h, border)

	pad := 10
	ox, oy := x+pad, y+pad+18
	ow, oh := w-2*pad, h-2*pad-18
	if ow < 40 || oh < 40 {
		return
	}

	trueBrg := peri.TrueBearingDeg(player.HeadingDeg)
	fov := peri.FOVDeg()
	up := peri.MastUp() && !peri.Sheared && !player.Damage.Destroyed(world.SysPeriscope)

	cx := ox + ow/2
	cy := oy + oh/2
	hdrY := y + 16
	render.DrawText(screen, "IR SENSOR", x+pad, hdrY, render.ColorPhosphorDim, true)
	brgTxt := fmt.Sprintf("BRG %03.0f°T", trueBrg)
	render.DrawText(screen, brgTxt, cx-len(brgTxt)*3, hdrY, render.ColorPhosphor, true)
	render.DrawText(screen, fmt.Sprintf("FOV %.0f°  %s", fov, peri.ZoomLabel()), x+w-110, hdrY, render.ColorPhosphorDim, true)

	if !up {
		a.periMarkerHits = a.periMarkerHits[:0]
		msg := "SCOPE STOWED"
		if peri.Sheared || player.Damage.Destroyed(world.SysPeriscope) {
			msg = "NO OPTIC"
		} else if peri.MastMoving() {
			msg = "OPTIC MOTION"
		}
		render.FillRect(screen, ox, oy, ow, oh, color.RGBA{12, 16, 14, 255})
		render.DrawText(screen, msg, cx-len(msg)*3, cy-4, render.ColorDim, true)
		return
	}

	weather := world.WeatherLight
	gt := 0.0
	if a.Engine != nil {
		if a.Engine.Scenario != nil {
			weather = a.Engine.Scenario.Weather
		}
		gt = a.Engine.Clock.GameTime
		if a.Mode != ModePaused {
			gt = a.Engine.VisualGameTime()
		}
	}

	a.ensurePeriscopeImage()
	a.renderPeriscopeIRFrame(player, peri, weather, gt)
	a.periImg.WritePixels(a.periPix)

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	sx := float64(ow) / float64(periIRW)
	sy := float64(oh) / float64(periIRH)
	op.GeoM.Scale(sx, sy)
	op.GeoM.Translate(float64(ox), float64(oy))
	screen.DrawImage(a.periImg, op)

	// Reticule + FOV scale over the live picture.
	render.DrawLine(screen, float64(ox+8), float64(cy), float64(ox+ow-8), float64(cy), color.RGBA{40, 70, 55, 120})
	render.DrawLine(screen, float64(cx), float64(oy+8), float64(cx), float64(oy+oh-8), color.RGBA{50, 90, 70, 100})
	render.FillRect(screen, cx-10, cy-1, 20, 2, render.ColorPhosphor)
	render.FillRect(screen, cx-1, cy-10, 2, 20, render.ColorPhosphor)

	half := fov / 2
	scaleHalf := ow/2 - 16
	if scaleHalf < 20 {
		scaleHalf = 20
	}
	for d := -int(half); d <= int(half); d++ {
		frac := float64(d) / half
		px := cx + int(frac*float64(scaleHalf))
		tick := 4
		absD := d
		if absD < 0 {
			absD = -absD
		}
		if absD%5 == 0 {
			tick = 9
		}
		render.DrawLine(screen, float64(px), float64(cy+4), float64(px), float64(cy+4+tick), render.ColorPhosphorDim)
		// Skip center label — BRG is already in the header above the optic.
		if absD%5 == 0 && d != 0 {
			mark := normalizeGyroDeg(trueBrg + float64(d))
			label := fmt.Sprintf("%03.0f", mark)
			render.DrawText(screen, label, px-10, cy+16, render.ColorPhosphorDim, true)
		}
	}

	a.drawPeriTargetMarkers(screen, ox, oy, ow, oh, peri, player, trueBrg, fov)
}

// drawPeriTargetMarkers places waterfall-style contact chips under contacts
// visible in the current FOV (bottom of the optic picture).
func (a *App) drawPeriTargetMarkers(screen *ebiten.Image, ox, oy, ow, oh int, peri *acoustics.PeriscopeState, player *world.Entity, lookBrg, fov float64) {
	a.periMarkerHits = a.periMarkerHits[:0]
	if a.Engine == nil || peri == nil || player == nil {
		return
	}
	sonar := &a.Engine.Sonar
	const (
		chipH = 16
		gap   = 4
	)
	chipY := oy + oh - chipH - 2
	if chipY < oy {
		chipY = oy
	}

	type pending struct {
		srcID string
		label string
		cx    int
		w     int
	}
	var items []pending
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if c.SourceEntityID == "" || c.ID == "" {
			continue
		}
		brg := c.BearingDeg
		for _, e := range a.Engine.Scenario.Entities {
			if e != nil && e.ID == c.SourceEntityID {
				brg = player.BearingDegTo(e)
				break
			}
		}
		px, ok := acoustics.BearingToViewX(brg, lookBrg, fov, ow)
		if !ok {
			continue
		}
		cw := len(c.ID)*7 + 10
		if cw < 22 {
			cw = 22
		}
		items = append(items, pending{srcID: c.SourceEntityID, label: c.ID, cx: ox + px, w: cw})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].cx < items[j].cx })

	mx, my := ebiten.CursorPosition()
	rowEnd := ox - gap
	for _, it := range items {
		chipX := it.cx - it.w/2
		if chipX < ox {
			chipX = ox
		}
		if chipX+it.w > ox+ow {
			chipX = ox + ow - it.w
		}
		if chipX < rowEnd+gap {
			chipX = rowEnd + gap
			if chipX+it.w > ox+ow {
				chipX = ox + ow - it.w
			}
		}
		if chipX < ox {
			chipX = ox
		}

		selected := it.srcID == a.selectedContactID || (peri.Locked() && peri.LockEntityID == it.srcID)
		hover := mx >= chipX && mx < chipX+it.w && my >= chipY && my < chipY+chipH
		bg := render.ColorPanelInset
		if selected {
			bg = color.RGBA{80, 60, 0, 255}
		} else if hover {
			bg = render.ColorPanelMid
		}
		render.FillRect(screen, chipX, chipY, it.w, chipH, bg)
		if selected {
			render.DrawLine(screen, float64(chipX), float64(chipY), float64(chipX+it.w), float64(chipY), render.ColorAmber)
		}
		clr := render.ColorPhosphor
		if selected {
			clr = render.ColorAmber
		}
		render.DrawText(screen, it.label, chipX+5, chipY+12, clr, true)

		a.periMarkerHits = append(a.periMarkerHits, contactChip{
			SourceID: it.srcID,
			X:        chipX,
			Y:        chipY,
			W:        it.w,
			H:        chipH,
		})
		if end := chipX + it.w; end > rowEnd {
			rowEnd = end
		}
	}
}

// renderPeriscopeIRFrame refreshes the cached sky/sea/land plate when needed,
// then always composites ships/FX/grain for continuous optic motion.
func (a *App) renderPeriscopeIRFrame(player *world.Entity, peri *acoustics.PeriscopeState, weather world.Weather, gameTime float64) {
	bgKey := a.periViewBackgroundCacheKey(player, peri, weather, gameTime)
	if !periBGCacheEnabled || bgKey != a.periBgCacheKey {
		a.buildPeriscopeIRBackground(player, peri, weather, gameTime)
		a.periBgCacheKey = bgKey
	}
	copy(a.periPix, a.periBgPix)
	a.composePeriscopeIRForeground(player, peri, weather, gameTime)
	a.applyPeriOpticAccommodation(player, peri, weather, gameTime)
}

func (a *App) buildPeriscopeIRBackground(player *world.Entity, peri *acoustics.PeriscopeState, weather world.Weather, _ float64) {
	pix := a.periBgPix
	w, h := periIRW, periIRH
	look := peri.TrueBearingDeg(player.HeadingDeg)
	fov := peri.FOVDeg()
	eyeH := acoustics.EyeAboveWaterFt(player.DepthFt, peri.Extension)
	maxR := acoustics.OpticalMaxRangeYd(eyeH, weather)
	horizonY := h * 45 / 100 // slightly above mid — flat seas at PD

	// --- Sky ---
	for y := 0; y < horizonY; y++ {
		t := float64(y) / float64(horizonY)
		base := uint8(28 + t*55) // darker zenith → brighter near horizon (IR sky)
		for x := 0; x < w; x++ {
			brg := acoustics.ViewXToBearing(x, w, look, fov)
			cloud := periCloud01(brg, float64(y), weather)
			v := int(base) + int(cloud*40)
			if weather == world.WeatherStorm {
				v -= 12
			}
			if v < 10 {
				v = 10
			}
			if v > 200 {
				v = 200
			}
			periSetGray(pix, w, x, y, uint8(v))
		}
	}

	// --- Sea (cold MWIR water: dark plate with a readable near→far gradient) ---
	// Horizon reads slightly warmer/brighter; near field stays colder/darker.
	rippleAmp := 0.55
	if weather == world.WeatherStorm {
		rippleAmp = 1.3
	} else if weather == world.WeatherCalm {
		rippleAmp = 0.22
	}
	for y := horizonY; y < h; y++ {
		t := float64(y-horizonY) / float64(h-horizonY+1)
		// Ease so the falloff is visible without looking painted.
		ease := t * t * (3 - 2*t)
		base := 34.0 - ease*18.0 // ~34 at horizon → ~16 near ownship
		// Soft 3-row blend into the sky band.
		if y-horizonY < 3 {
			lift := (1.0 - float64(y-horizonY)/3.0) * 5.0
			base += lift
		}
		for x := 0; x < w; x++ {
			// Spatial micro-texture only (no time) — keeps bg cache stable.
			n := float64(periHash8(x, y, 17)&7)*0.15 - 0.5
			ripple := math.Sin(float64(x)*0.11+float64(y)*0.19)*rippleAmp*0.35 + n
			v := int(base + ripple)
			if v < 12 {
				v = 12
			}
			if v > 42 {
				v = 42
			}
			periSetGray(pix, w, x, y, uint8(v))
		}
	}

	// --- Land columns (ray + horizontal smoothing pass) ---
	var bathy world.Bathymetry
	if a.Engine != nil && a.Engine.Scenario != nil && a.Engine.Scenario.Bathy != nil {
		bathy = *a.Engine.Scenario.Bathy
	} else {
		bathy = world.DefaultBathy
	}
	if bathy.Valid() {
		if cap(a.periLandHit) < w {
			a.periLandHit = make([]float64, w)
			a.periLandElev = make([]float64, w)
			a.periLandOK = make([]bool, w)
		} else {
			a.periLandHit = a.periLandHit[:w]
			a.periLandElev = a.periLandElev[:w]
			a.periLandOK = a.periLandOK[:w]
		}
		for x := 0; x < w; x++ {
			a.periLandOK[x] = false
			brg := acoustics.ViewXToBearing(x, w, look, fov)
			rad := brg * math.Pi / 180
			sx := math.Sin(rad)
			sy := math.Cos(rad)
			hitR := 0.0
			found := false
			for r := periLandStepYd; r <= maxR; r += periLandStepYd {
				if bathy.IsLand(player.X+sx*r, player.Y+sy*r) {
					hitR = r
					found = true
					break
				}
			}
			if !found {
				continue
			}
			// Refine hit with a short binary search for less stair-stepping.
			lo, hi := hitR-periLandStepYd, hitR
			if lo < periLandStepYd {
				lo = periLandStepYd
			}
			for i := 0; i < 5; i++ {
				mid := 0.5 * (lo + hi)
				if bathy.IsLand(player.X+sx*mid, player.Y+sy*mid) {
					hi = mid
				} else {
					lo = mid
				}
			}
			hitR = hi
			inland := 0.0
			for s := periLandStepYd; s <= periLandInlandYd; s += periLandStepYd {
				if !bathy.IsLand(player.X+sx*(hitR+s), player.Y+sy*(hitR+s)) {
					break
				}
				inland = s
			}
			elevFt := 35 + inland*0.10 + periLandHashElev(player.X, player.Y, brg)*25
			if elevFt > 700 {
				elevFt = 700
			}
			a.periLandHit[x] = hitR
			a.periLandElev[x] = elevFt
			a.periLandOK[x] = true
		}
		// Extra approximation layer: smooth range/elevation across neighbors
		// so coastline silhouettes do not jump column-to-column.
		a.smoothLandColumns(5)

		vFov := fov * float64(h) / float64(w)
		for x := 0; x < w; x++ {
			if !a.periLandOK[x] {
				continue
			}
			hitR := a.periLandHit[x]
			elevFt := a.periLandElev[x]
			angH := (elevFt / world.FeetPerYard) / hitR * (180 / math.Pi)
			pxH := int(angH / vFov * float64(h))
			if pxH < 2 {
				pxH = 2
			}
			if pxH > horizonY {
				pxH = horizonY
			}
			hazeT := hitR / maxR
			shade := uint8(55 + (1-hazeT)*70)
			if weather == world.WeatherStorm {
				shade = uint8(float64(shade) * 0.75)
			}
			top := horizonY - pxH
			if top < 0 {
				top = 0
			}
			for y := top; y < horizonY; y++ {
				g := shade
				frac := float64(y-top) / float64(pxH+1)
				g = uint8(min255(int(g) + int((1-frac)*16)))
				periSetGray(pix, w, x, y, g)
			}
			if horizonY < h {
				periSetGray(pix, w, x, horizonY, uint8(max0(int(shade)-25)))
			}
		}
	}
}

func (a *App) composePeriscopeIRForeground(player *world.Entity, peri *acoustics.PeriscopeState, weather world.Weather, gameTime float64) {
	pix := a.periPix
	w, h := periIRW, periIRH
	look := peri.TrueBearingDeg(player.HeadingDeg)
	fov := peri.FOVDeg()
	eyeH := acoustics.EyeAboveWaterFt(player.DepthFt, peri.Extension)
	maxR := acoustics.OpticalMaxRangeYd(eyeH, weather)
	horizonY := h * 45 / 100

	// Sea foam / swell streaks (parallax with look + time). Drawn before ships
	// so hulls sit on top of the water plate without thrashing the land BG cache.
	drawPeriSeaFoam(pix, w, h, horizonY, look, gameTime, weather)

	// --- Ships (far → near) ---
	a.periShipScratch = a.periShipScratch[:0]
	if a.Engine != nil && a.Engine.Scenario != nil {
		for _, e := range a.Engine.Scenario.Entities {
			proj, ok := acoustics.ProjectSurfaceShip(player.X, player.Y, look, fov, w, h, horizonY, maxR, eyeH, e)
			if !ok {
				continue
			}
			proj.Fire01 = world.HullFireIntensity(e.HullFireUntil, gameTime)
			proj.FirePhase = gameTime
			a.periShipScratch = append(a.periShipScratch, periShipDraw{proj: proj, waterY: proj.WaterY})
		}
	}
	sort.Slice(a.periShipScratch, func(i, j int) bool {
		return a.periShipScratch[i].proj.RangeYd > a.periShipScratch[j].proj.RangeYd
	})
	for i := range a.periShipScratch {
		drawPeriShipSilhouette(pix, w, h, a.periShipScratch[i].proj)
	}

	// --- Harpoon missiles (airborne) + surface blast flashes ---
	a.drawPeriTransientFX(pix, w, h, horizonY, player, look, fov, maxR, gameTime)

	// Soft film grain (IR noise).
	for y := 0; y < h; y++ {
		for x := 0; x < w; x += 2 {
			n := periHash8(x, y, int(gameTime*7))
			if n < 20 {
				i := (y*w + x) * 4
				v := int(pix[i]) + int(n%5) - 2
				if v < 0 {
					v = 0
				}
				if v > 255 {
					v = 255
				}
				periSetGray(pix, w, x, y, uint8(v))
			}
		}
	}
}

// periOpticAccommodation models photonics-mast AGC after a bright blast:
// brief saturate lift → gain crash (scene goes dark) → slow recovery.
// prox is 0..1 (1 = close / severe). Returns gain and additive white veil.
func periOpticAccommodation(age, prox float64) (gain, veil float64) {
	gain = 1
	if age < 0 || age > 5.0 || prox <= 0.02 {
		return 1, 0
	}
	if prox > 1 {
		prox = 1
	}
	// Soft clip veil while the FPA is still flooding.
	if age < 0.35 {
		veil = (1 - age/0.35) * (0.55 + 0.45*prox) * 210
	}
	floor := 0.10 + 0.22*(1-prox) // closer blast → darker AGC floor
	switch {
	case age < 0.18:
		// Slight exposure lift into the white-out.
		gain = 1.0 + 0.20*prox*(1-age/0.18)
	case age < 0.70:
		// AGC winds gain down hard (accommodation).
		u := (age - 0.18) / 0.52
		u = u * u * (3 - 2*u) // smoothstep
		gain = 1.0*(1-u) + floor*u
	default:
		// Recover toward unity; slight undershoot then settle.
		u := (age - 0.70) / 3.8
		if u > 1 {
			u = 1
		}
		ease := 1 - (1-u)*(1-u)
		gain = floor + (1-floor)*ease
		// Tiny overshoot dip mid-recovery (sensor hunting).
		if u > 0.15 && u < 0.55 {
			hunt := math.Sin((u-0.15)/0.4*math.Pi) * 0.06 * prox
			gain -= hunt
		}
	}
	if gain < 0.06 {
		gain = 0.06
	}
	return gain, veil
}

// applyPeriOpticAccommodation remaps the finished IR frame for AGC after a blast.
func (a *App) applyPeriOpticAccommodation(player *world.Entity, peri *acoustics.PeriscopeState, weather world.Weather, gameTime float64) {
	if a.Engine == nil || player == nil || peri == nil {
		return
	}
	sonar := &a.Engine.Sonar
	if sonar.LastBlastAt <= 0 {
		return
	}
	age := gameTime - sonar.LastBlastAt
	if age < 0 || age > 5.0 {
		return
	}
	bx, by := sonar.LastBlastX, sonar.LastBlastY
	if sonar.LastBlastEntityID != "" && a.Engine.Scenario != nil {
		for _, e := range a.Engine.Scenario.Entities {
			if e != nil && e.ID == sonar.LastBlastEntityID {
				bx, by = e.X, e.Y
				break
			}
		}
		if a.Engine.Scenario.Player != nil && a.Engine.Scenario.Player.ID == sonar.LastBlastEntityID {
			bx, by = a.Engine.Scenario.Player.X, a.Engine.Scenario.Player.Y
		}
	}
	// FPA only accommodates to light that hits the focal plane — off-FOV blasts are invisible.
	look := peri.TrueBearingDeg(player.HeadingDeg)
	fov := peri.FOVDeg()
	brg := math.Atan2(bx-player.X, by-player.Y) * 180 / math.Pi
	if brg < 0 {
		brg += 360
	}
	if _, inView := acoustics.BearingToViewX(brg, look, fov, periIRW); !inView {
		return
	}
	dist := math.Hypot(bx-player.X, by-player.Y)
	eyeH := acoustics.EyeAboveWaterFt(player.DepthFt, peri.Extension)
	maxR := acoustics.OpticalMaxRangeYd(eyeH, weather)
	if maxR < 1 {
		maxR = 1
	}
	// Stronger when the blast is optically near; still twitch a bit out to wash range.
	prox := 1 - dist/(maxR*1.15)
	if wash := sonar.LastBlastRangeYd; wash > 0 && dist < wash {
		p2 := 1 - dist/wash
		if p2 > prox {
			prox = 0.35 + 0.65*p2
		}
	}
	if prox < 0.05 {
		return
	}
	if prox > 1 {
		prox = 1
	}
	gain, veil := periOpticAccommodation(age, prox)
	if math.Abs(gain-1) < 0.02 && veil < 4 {
		return
	}
	pix := a.periPix
	n := periIRW * periIRH
	for i := 0; i < n; i++ {
		o := i * 4
		v := float64(pix[o])*gain + veil
		// Extra sensor noise while AGC is hunting (dark recovery phase).
		if gain < 0.85 {
			v += float64(periHash8(i&0xff, i>>8, int(age*20))&7) - 3
		}
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		g := uint8(v)
		pix[o] = g
		pix[o+1] = g
		pix[o+2] = g
	}
}

// buildPeriscopeIRFrame rebuilds the full optic buffer (bg + foreground).
// Kept for benches / callers that want a single-shot compose.
func (a *App) buildPeriscopeIRFrame(player *world.Entity, peri *acoustics.PeriscopeState, weather world.Weather, gameTime float64) {
	a.buildPeriscopeIRBackground(player, peri, weather, gameTime)
	a.periBgCacheKey = a.periViewBackgroundCacheKey(player, peri, weather, gameTime)
	copy(a.periPix, a.periBgPix)
	a.composePeriscopeIRForeground(player, peri, weather, gameTime)
}

func (a *App) smoothLandColumns(radius int) {
	hit, elev, ok := a.periLandHit, a.periLandElev, a.periLandOK
	n := len(hit)
	if n == 0 || radius < 1 {
		return
	}
	if cap(a.periLandHitTmp) < n {
		a.periLandHitTmp = make([]float64, n)
		a.periLandElevTmp = make([]float64, n)
		a.periLandOKTmp = make([]bool, n)
	} else {
		a.periLandHitTmp = a.periLandHitTmp[:n]
		a.periLandElevTmp = a.periLandElevTmp[:n]
		a.periLandOKTmp = a.periLandOKTmp[:n]
	}
	copy(a.periLandHitTmp, hit)
	copy(a.periLandElevTmp, elev)
	copy(a.periLandOKTmp, ok)
	tmpH, tmpE, tmpOK := a.periLandHitTmp, a.periLandElevTmp, a.periLandOKTmp
	for x := 0; x < n; x++ {
		sumH, sumE, wsum := 0.0, 0.0, 0.0
		for d := -radius; d <= radius; d++ {
			xi := x + d
			if xi < 0 || xi >= n || !tmpOK[xi] {
				continue
			}
			wt := float64(radius + 1 - absInt(d))
			sumH += tmpH[xi] * wt
			sumE += tmpE[xi] * wt
			wsum += wt
		}
		if wsum <= 0 {
			ok[x] = false
			continue
		}
		hit[x] = sumH / wsum
		elev[x] = sumE / wsum
		ok[x] = true
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func periLandHashElev(x, y, brg float64) float64 {
	// Stable soft variation so ridges are not perfectly flat after smoothing.
	n := math.Sin(x*0.0011+brg*0.07) * 0.5 + math.Sin(y*0.0009-brg*0.03)*0.5
	return (n + 1) * 0.5
}

func (a *App) drawPeriTransientFX(pix []byte, w, h, horizonY int, player *world.Entity, look, fov, maxR, gameTime float64) {
	if a.Engine == nil {
		return
	}
	peri := &a.Engine.Periscope
	eyeH := acoustics.EyeAboveWaterFt(player.DepthFt, peri.Extension)
	// Airborne Harpoons: bright IR streak + bloom.
	for _, hm := range a.Engine.FireControl.ActiveHarpoons {
		if hm == nil || !hm.Alive || hm.Phase == weapons.HarpoonUnderwater {
			continue
		}
		dx := hm.X - player.X
		dy := hm.Y - player.Y
		rangeYd := math.Hypot(dx, dy)
		if rangeYd < 80 || rangeYd > maxR*1.1 {
			continue
		}
		brg := math.Atan2(dx, dy) * 180 / math.Pi
		if brg < 0 {
			brg += 360
		}
		cx, ok := acoustics.BearingToViewX(brg, look, fov, w)
		if !ok {
			continue
		}
		t := rangeYd / maxR
		if t > 1 {
			t = 1
		}
		waterY := acoustics.SeaSurfacePixelY(eyeH, rangeYd, w, h, horizonY, fov)
		// Sea-skimmer: just above waterline.
		my := waterY - 2 - int((1-t)*4)
		if my < 1 {
			my = 1
		}
		bloom := 3 + int((1-t)*4)
		for by := -bloom; by <= bloom; by++ {
			for bx := -bloom; bx <= bloom; bx++ {
				if bx*bx+by*by > bloom*bloom {
					continue
				}
				xx, yy := cx+bx, my+by
				if xx < 0 || xx >= w || yy < 0 || yy >= h {
					continue
				}
				periSetGray(pix, w, xx, yy, uint8(min255(180+int((1-t)*60))))
			}
		}
		// Exhaust streak opposite heading.
		hrad := (hm.HeadingDeg + 180) * math.Pi / 180
		for s := 1; s <= 8; s++ {
			// Offset in world → bearing shift approximated in pixels.
			sx := cx - int(math.Sin(hrad)*float64(s)*0.8)
			sy := my + s/3
			if sx < 0 || sx >= w || sy < 0 || sy >= h {
				continue
			}
			periSetGray(pix, w, sx, sy, uint8(min255(140-s*8)))
		}
	}

	// Surface / near-surface detonations (torpedo, Harpoon hit, cook-off).
	sonar := &a.Engine.Sonar
	if sonar.LastBlastAt <= 0 {
		return
	}
	age := gameTime - sonar.LastBlastAt
	// IR water column outlives acoustic washout — gate on column lifetime below.
	if age < 0 {
		return
	}

	bx, by := sonar.LastBlastX, sonar.LastBlastY
	var hit *world.Entity
	if sonar.LastBlastEntityID != "" && a.Engine.Scenario != nil {
		if a.Engine.Scenario.Player != nil && a.Engine.Scenario.Player.ID == sonar.LastBlastEntityID {
			hit = a.Engine.Scenario.Player
		}
		for _, e := range a.Engine.Scenario.Entities {
			if e != nil && e.ID == sonar.LastBlastEntityID {
				hit = e
				break
			}
		}
	}
	if hit != nil {
		bx, by = hit.X, hit.Y
	}

	dx := bx - player.X
	dy := by - player.Y
	rangeYd := math.Hypot(dx, dy)
	if rangeYd > maxR*1.2 {
		return
	}
	brg := math.Atan2(dx, dy) * 180 / math.Pi
	if brg < 0 {
		brg += 360
	}
	cx, ok := acoustics.BearingToViewX(brg, look, fov, w)
	if !ok {
		return
	}
	t := rangeYd / maxR
	if t > 1 {
		t = 1
	}
	waterY := acoustics.SeaSurfacePixelY(eyeH, rangeYd, w, h, horizonY, fov)
	shipW := 0
	if hit != nil && hit.Kind == world.KindSurfaceShip {
		if proj, pok := acoustics.ProjectSurfaceShip(player.X, player.Y, look, fov, w, h, horizonY, maxR*1.2, eyeH, hit); pok {
			cx = proj.CenterX
			waterY = proj.WaterY
			shipW = proj.WidthPx
		}
	}
	// Visual IR: short punchy flash + slow water-column rise/hang/fall (~12.3s).
	// Do not cap by acoustic flash — tonnes of spray outlive the sensor washout cue.
	const (
		colRise = 4.0
		colHang = 0.3
		colFall = 8.0
		softTail = 1.5
	)
	visDur := colRise + colHang + colFall + softTail
	if age > visDur {
		return
	}
	rem := visDur - age
	life := 1.0
	if rem < softTail {
		tLife := rem / softTail
		life = tLife * tLife * (3 - 2*tLife)
	}
	grow := age / 0.14
	if grow > 1 {
		grow = 1
	}
	// Scale from range/grow only — column mass must not shrink with the soft tail.
	scale := (0.55 + 0.45*grow) * (1 - t*0.25)
	if scale < 0.05 || life < 0.02 {
		return
	}
	seed := uint32(sonar.LastBlastAt*1009) ^ uint32(int(bx)*733) ^ uint32(int(by)*997)
	drawPeriBlastIR(pix, w, h, cx, waterY, shipW, scale, life, age, seed)
}

// drawPeriBlastIR paints a photonics-mast IR kill flash:
// bright splash/bloom, rising/falling water column, polygonal debris.
// Lingering hull fires live on the ship sprite via Entity.HullFireUntil.
func drawPeriBlastIR(pix []byte, w, h, cx, waterY, shipW int, scale, life, age float64, seed uint32) {
	rx := 8.0 + 28.0*scale
	ry := 9.0 + 30.0*scale
	if shipW > 4 {
		hullHalf := float64(shipW) * 0.48
		if hullHalf > rx {
			rx = hullHalf
		}
	}

	sprayPhase := 1.0
	if age < 0.10 {
		sprayPhase = age / 0.10
	}
	afterglow := 0.0
	if age > 0.55 {
		afterglow = math.Min(1, (age-0.55)/1.2) * life
	}

	// Splash/bloom luminance only (size stays full): clip-white → mid-gray → gone.
	// Independent of the longer water-column life curve.
	flashLum := 0.0
	switch {
	case age < 0.07:
		flashLum = 1.0
	case age < 0.28:
		u := (age - 0.07) / 0.21
		u = u * u * (3 - 2*u)
		flashLum = 1.0*(1-u) + 0.48*u // ~255 → ~122
	case age < 0.70:
		u := (age - 0.28) / 0.42
		u = u * u // ease-in fade
		flashLum = 0.48 * (1 - u)
	}

	// --- 1) Bright white-out bloom (geometry unchanged; luminance from flashLum) ---
	if flashLum > 0.02 {
		satR := rx * (0.95 + 0.75*sprayPhase)
		satH := ry * (0.70 + 0.65*sprayPhase)
		srx := int(satR) + 3
		sry := int(satH) + 3
		for dy := -sry; dy <= sry; dy++ {
			for dx := -srx; dx <= srx; dx++ {
				nx := float64(dx) / (satR + 0.01)
				ny := float64(dy) / (satH + 0.01)
				if ny > 0.35 {
					ny *= 1.55
				}
				r2 := nx*nx + ny*ny
				if r2 > 1.0 {
					continue
				}
				xx, yy := cx+dx, waterY+dy-int(satH*0.28)
				if xx < 0 || xx >= w || yy < 0 || yy >= h {
					continue
				}
				edge := 1.0
				if r2 > 0.55 {
					edge = 1.0 - (r2-0.55)/0.45
				}
				g := int(255 * flashLum * edge)
				if g < 16 {
					continue
				}
				periBrighten(pix, w, xx, yy, uint8(min255(g)))
			}
		}

		// --- 2) Waterline splash sheet ---
		haloW := rx * (1.55 + 0.55*sprayPhase)
		haloH := math.Max(2.5, 4.0+9.0*scale*(0.45+0.55*sprayPhase))
		hw := int(haloW) + 2
		hh := int(haloH) + 2
		for dy := -hh; dy <= hh; dy++ {
			for dx := -hw; dx <= hw; dx++ {
				nx := float64(dx) / (haloW + 0.01)
				ny := float64(dy) / (haloH + 0.01)
				r2 := nx*nx + ny*ny
				if r2 > 1.05 {
					continue
				}
				xx, yy := cx+dx, waterY+dy
				if xx < 0 || xx >= w || yy < 0 || yy >= h {
					continue
				}
				edge := 1.05 - r2*0.78
				if edge < 0 {
					continue
				}
				g := int(255 * flashLum * edge)
				if g < 16 {
					continue
				}
				periBrighten(pix, w, xx, yy, uint8(min255(g)))
			}
		}
	}

	// --- 2b) Water column: rise → hang → fall (Dena / DoW-style spray plume) ---
	drawPeriWaterColumn(pix, w, h, cx, waterY, rx, scale, life, age, seed)

	// --- 3) Small white-hot metal fragments (3–7-gons) on parabolic arcs ---
	nDebris := 22 + int(30*scale)
	if nDebris > 52 {
		nDebris = 52
	}
	const debrisG = 28.0 // slower fall so arcs reach the waterline
	for i := 0; i < nDebris; i++ {
		u1 := periBlastNoise01(seed+uint32(i)*2654435761, i, 1)
		u2 := periBlastNoise01(seed+uint32(i)*2246822519, i, 2)
		u3 := periBlastNoise01(seed+uint32(i)*3266489917, i, 3)
		u4 := periBlastNoise01(seed+uint32(i)*668265263, i, 4)
		birth := u1 * 0.10
		if age < birth {
			continue
		}
		tFrag := age - birth
		// Long enough that most shards complete the up→down arc into the sea.
		lifeDeb := 1.6 + 2.0*u2
		if tFrag > lifeDeb {
			continue
		}
		ox := float64(cx) + (u3-0.5)*rx*1.0
		oy := float64(waterY) - 2 - u1*5
		side := 1.0
		if u1 < 0.5 {
			side = -1
		}
		// Modest kick — readable arcs, not screen-edge disappearances.
		vx := side * (5.0 + 14.0*u2) * (0.75 + 0.45*u3)
		vy := -(8.0 + 18.0*u2) // screen up is negative
		xx := ox + vx*tFrag
		yy := oy + vy*tFrag + 0.5*debrisG*tFrag*tFrag
		// Allow a brief splash into the water plate before cutting.
		if yy > float64(waterY)+6 {
			continue
		}
		fade := 1 - tFrag/lifeDeb
		// Dim only after the shard has returned near/into the water.
		if yy >= float64(waterY)-1 {
			into := math.Min(1, (yy-float64(waterY)+1)/7)
			fade *= 1 - 0.55*into
		}
		fade *= 0.70 + 0.30*life
		if fade < 0.08 {
			continue
		}
		sides := 3 + int(u4*5) // 3..7
		if sides > 7 {
			sides = 7
		}
		rad := 0.65 + 1.15*u2*(0.7+0.4*scale)
		ang0 := u3 * 6.283185307179586
		spin := (u1 - 0.5) * 10.0 * tFrag
		g := uint8(min255(int(220 + 35*fade)))
		periFillPolyIRForce(pix, w, h, xx, yy, rad, sides, ang0+spin, g, waterY+6)
	}

	// --- 4) Brief waterline afterglow ---
	if afterglow > 0.05 {
		nPatch := 2 + int(3*afterglow)
		for i := 0; i < nPatch; i++ {
			u1 := periBlastNoise01(seed^0x51eb, i, 11)
			u2 := periBlastNoise01(seed^0xc0ff, i, 12)
			px := cx + int((u1-0.5)*rx*1.3)
			py := waterY - 1 - int(u2*2)
			pw := 1 + int(2.5*afterglow*(0.5+u2))
			ph := 1 + int(1.5*afterglow)
			for dy := -ph; dy <= ph; dy++ {
				for dx := -pw; dx <= pw; dx++ {
					if dx*dx+dy*dy*4 > pw*pw {
						continue
					}
					xx, yy := px+dx, py+dy
					if xx < 0 || xx >= w || yy < 0 || yy >= h {
						continue
					}
					periBrighten(pix, w, xx, yy, uint8(min255(int((130+80*afterglow)*life))))
				}
			}
		}
	}
}

// drawPeriWaterColumn paints a cool-white spray plume: heavy water rises slowly
// (decelerating into the peak), hangs briefly, then settles over many seconds —
// buoyed by residual heat from the blast (lift), not free-fall.
func drawPeriWaterColumn(pix []byte, w, h, cx, waterY int, rx, scale, life, age float64, seed uint32) {
	const riseT = 4.0
	const hangT = 0.3
	const fallT = 8.0
	if age > riseT+hangT+fallT || life < 0.04 {
		return
	}
	peakH := 20.0 + 52.0*scale
	stemW := math.Max(3.0, rx*0.22)
	bulbW := math.Max(stemW*1.6, 7.0+18.0*scale)

	var heightFrac, bulbFrac, opacity float64
	switch {
	case age < riseT:
		u := age / riseT
		// Cubic ease-out: tonnes of water climb fast at first, brake near the apex.
		s := 1 - u
		heightFrac = 1 - s*s*s
		bulbFrac = 0.20 + 0.80*heightFrac
		opacity = 0.45 + 0.55*heightFrac
	case age < riseT+hangT:
		heightFrac = 1
		bulbFrac = 1
		opacity = 1
	default:
		u := (age - riseT - hangT) / fallT
		if u > 1 {
			u = 1
		}
		// Cubic ease-in settle: hot-air lift keeps the mass aloft, then it sinks.
		heightFrac = 1 - u*u*u
		bulbFrac = 1 - 0.50*u
		opacity = (1 - 0.75*u) * (0.70 + 0.30*life)
	}
	colH := peakH * heightFrac
	if colH < 2.5 || opacity < 0.06 {
		return
	}

	// Extra headroom so the crown can feather above the nominal height
	// instead of printing a hard horizontal cut.
	softPad := int(6.0 + 10.0*scale)
	maxHalf := int(bulbW*1.15) + 3
	for dx := -maxHalf; dx <= maxHalf; dx++ {
		xx := cx + dx
		if xx < 0 || xx >= w {
			continue
		}
		// Per-column crown height — broken cauliflower silhouette, not a flat lid.
		jag := periBlastNoise01(seed^0xbabe, dx, int(age*3))
		jag2 := periBlastNoise01(seed^0x0f0f, dx/2, 7)
		localH := colH * (0.72 + 0.38*jag + 0.12*jag2)
		if localH < 2 {
			continue
		}
		// Width envelope along height (stem → bulb); evaluated per row below.
		for dy := 0; dy <= int(localH)+softPad; dy++ {
			yy := waterY - dy
			if yy < 0 || yy >= h {
				continue
			}
			along := float64(dy) / (localH + 0.01) // 0=water … 1=local crown
			// Soft crown: fade and break up well before the tip.
			topFade := 1.0
			if along > 0.55 {
				tTop := (along - 0.55) / 0.55 // continues into softPad (along>1)
				if tTop > 1 {
					tTop = 1
				}
				topFade = (1 - tTop) * (1 - tTop)
				// Sparse mist above the dense body.
				if along > 0.85 && periBlastNoise01(seed, dx+dy*3, dy) < 0.45+0.35*tTop {
					continue
				}
			}
			if topFade < 0.04 {
				continue
			}
			aClamp := along
			if aClamp > 1 {
				aClamp = 1
			}
			half := stemW*(1-aClamp) + bulbW*aClamp*aClamp*bulbFrac
			half *= 0.80 + 0.30*periBlastNoise01(seed, dx, dy)
			nx := math.Abs(float64(dx)) / (half + 0.01)
			edgeN := 0.68 + 0.32*periBlastNoise01(seed^0x0f0f, dx, dy)
			if nx > edgeN {
				continue
			}
			core := 1 - nx*nx
			vert := 0.55 + 0.45*aClamp
			if age > riseT+hangT {
				vert = 0.45 + 0.40*(1-aClamp)
			}
			g := int((150 + 90*core) * opacity * vert * topFade * life)
			if g < 22 {
				continue
			}
			periBrighten(pix, w, xx, yy, uint8(min255(g)))
		}
	}

	// Falling sheets / droplets — slow, long-lived (buoyant spray return).
	if age < riseT*0.45 {
		return
	}
	nDrop := 10 + int(18*scale)
	if nDrop > 28 {
		nDrop = 28
	}
	const dropG = 10.0 // weak gravity vs hot-air lift
	for i := 0; i < nDrop; i++ {
		u1 := periBlastNoise01(seed+uint32(i)*2246822519, i, 41)
		u2 := periBlastNoise01(seed+uint32(i)*3266489917, i, 42)
		u3 := periBlastNoise01(seed+uint32(i)*668265263, i, 43)
		release := riseT*0.55 + u1*(hangT+fallT*0.85)
		if age < release {
			continue
		}
		tFall := age - release
		lifeDrop := 3.5 + 5.5*u2
		if tFall > lifeDrop {
			continue
		}
		ox := float64(cx) + (u3-0.5)*bulbW*1.3
		startH := peakH * (0.55 + 0.40*u2) * math.Min(1, heightFrac+0.15)
		oy := float64(waterY) - startH
		vx := (u3 - 0.5) * (1.5 + 4.0*u1)
		vy := 0.4 + 2.2*u2
		xx := int(ox + vx*tFall)
		yy := int(oy + vy*tFall + 0.5*dropG*tFall*tFall)
		if xx < 0 || xx >= w || yy < 0 || yy >= h {
			continue
		}
		if yy > waterY+4 {
			continue
		}
		fade := (1 - tFall/lifeDrop) * opacity * life
		if fade < 0.08 {
			continue
		}
		g := uint8(min255(int((170 + 70*u2) * fade)))
		periBrighten(pix, w, xx, yy, g)
		if u2 > 0.4 && xx+1 < w {
			periBrighten(pix, w, xx+1, yy, uint8(min255(int(float64(g)*0.75))))
		}
	}
}

// drawPeriSeaFoam paints sparse IR-bright swell / foam streaks on the sea plate.
// Scroll uses look-bearing + time with near-field parallax (far streaks crawl,
// near streaks race) — matches drifting surface texture in mast IR footage.
func drawPeriSeaFoam(pix []byte, w, h, horizonY int, lookDeg, gameTime float64, weather world.Weather) {
	if horizonY >= h-1 {
		return
	}
	amp := 1.0
	switch weather {
	case world.WeatherCalm:
		amp = 0.40
	case world.WeatherStorm:
		amp = 1.55
	}
	seaH := float64(h - horizonY)
	if seaH < 1 {
		seaH = 1
	}
	for y := horizonY + 1; y < h; y++ {
		depth := float64(y-horizonY) / seaH // 0 = horizon (far), 1 = near
		// Parallax: near water shifts more with train/time than far bands.
		scroll := lookDeg*(0.16+0.62*depth) + gameTime*(0.9+5.2*depth)
		for x := 0; x < w; x++ {
			u := float64(x) + scroll
			swell := math.Sin(u*0.036 + float64(y)*0.065)
			foam := math.Sin(u*0.155 + float64(y)*0.21 + scroll*0.35)
			cap := math.Sin(u*0.29 - float64(y)*0.11 + scroll*0.55)
			add := 0.0
			if swell > 0.58 {
				add += (swell - 0.58) / 0.42 * (3.5 + 4.0*depth) * amp
			}
			if foam > 0.80 {
				add += (foam - 0.80) / 0.20 * (6.0 + 10.0*depth) * amp
			}
			if cap > 0.90 {
				add += (cap - 0.90) / 0.10 * (4.0 + 6.0*depth) * amp
			}
			if add < 1.2 {
				continue
			}
			// Break continuous ridges into streaky foam patches.
			if periHash8(x+int(scroll*0.25), y, 11) > 200 && foam < 0.93 {
				continue
			}
			i := (y*w + x) * 4
			if i+3 >= len(pix) {
				continue
			}
			periBrighten(pix, w, x, y, uint8(min255(int(pix[i])+int(add))))
		}
	}
}

// periFillPolyIRForce fills a small irregular polygon with an opaque IR gray write
// (not brighten-only), so debris stay visible over a saturated splash.
func periFillPolyIRForce(pix []byte, w, h int, cx, cy, rad float64, sides int, ang0 float64, g uint8, maxY int) {
	if sides < 3 {
		sides = 3
	}
	if sides > 7 {
		sides = 7
	}
	if rad < 0.45 {
		rad = 0.45
	}
	var xs, ys [7]float64
	minX, maxX := cx, cx
	minY, maxYf := cy, cy
	for i := 0; i < sides; i++ {
		a := ang0 + float64(i)*2*math.Pi/float64(sides)
		r := rad * (0.70 + 0.45*periBlastNoise01(uint32(i*97+int(cx*3)), i, int(cy)))
		xs[i] = cx + math.Cos(a)*r
		ys[i] = cy + math.Sin(a)*r
		if xs[i] < minX {
			minX = xs[i]
		}
		if xs[i] > maxX {
			maxX = xs[i]
		}
		if ys[i] < minY {
			minY = ys[i]
		}
		if ys[i] > maxYf {
			maxYf = ys[i]
		}
	}
	x0 := int(math.Floor(minX))
	x1 := int(math.Ceil(maxX))
	y0 := int(math.Floor(minY))
	y1 := int(math.Ceil(maxYf))
	if y1 > maxY {
		y1 = maxY
	}
	for yy := y0; yy <= y1; yy++ {
		if yy < 0 || yy >= h {
			continue
		}
		for xx := x0; xx <= x1; xx++ {
			if xx < 0 || xx >= w {
				continue
			}
			if periPointInPolyN(float64(xx)+0.5, float64(yy)+0.5, xs[:sides], ys[:sides]) {
				periSetGray(pix, w, xx, yy, g)
			}
		}
	}
}

func periPointInPolyN(x, y float64, xs, ys []float64) bool {
	n := len(xs)
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := xs[i], ys[i]
		xj, yj := xs[j], ys[j]
		if ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi+1e-9)+xi) {
			inside = !inside
		}
		j = i
	}
	return inside
}

func periBrighten(pix []byte, w, x, y int, g uint8) {
	i := (y*w + x) * 4
	if i < 0 || i+3 >= len(pix) {
		return
	}
	if g > pix[i] {
		periSetGray(pix, w, x, y, g)
	}
}

func periBlastNoise01(seed uint32, x, y int) float64 {
	h := seed
	h ^= uint32(x) * 374761393
	h ^= uint32(y) * 668265263
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xffff) / 65535.0
}

func periSetGray(pix []byte, w, x, y int, g uint8) {
	i := (y*w + x) * 4
	if i < 0 || i+3 >= len(pix) {
		return
	}
	pix[i] = g
	pix[i+1] = g
	pix[i+2] = g
	pix[i+3] = 255
}

func periCloud01(brgDeg, y float64, weather world.Weather) float64 {
	// Cheap value-noise bands drifting with bearing.
	u := brgDeg*0.07 + y*0.04
	n := math.Sin(u)*0.5 + math.Sin(u*2.7+1.3)*0.3 + math.Sin(u*5.1+0.4)*0.2
	n = (n + 1) * 0.5
	if weather == world.WeatherCalm {
		n *= 0.35
	} else if weather == world.WeatherStorm {
		n = 0.4 + n*0.6
	}
	if n < 0 {
		return 0
	}
	if n > 1 {
		return 1
	}
	return n
}

func periHash8(x, y, t int) int {
	n := x*374761393 + y*668265263 + t*1274126177
	n = (n ^ (n >> 13)) * 1274126177
	return (n ^ (n >> 16)) & 255
}

func min255(v int) int {
	if v > 255 {
		return 255
	}
	if v < 0 {
		return 0
	}
	return v
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
