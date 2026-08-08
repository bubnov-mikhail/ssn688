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

type periShipDraw struct {
	proj   acoustics.PeriShipProj
	waterY int
}

func (a *App) disposePeriscopeImage() {
	disposeImage(&a.periImg)
	a.periPix = nil
	a.periCacheKey = 0
}

func (a *App) ensurePeriscopeImage() {
	if a.periImg != nil && a.periImg.Bounds().Dx() == periIRW && a.periImg.Bounds().Dy() == periIRH {
		if len(a.periPix) == periIRW*periIRH*4 {
			return
		}
	}
	disposeImage(&a.periImg)
	a.periImg = ebiten.NewImage(periIRW, periIRH)
	a.periPix = make([]byte, periIRW*periIRH*4)
	a.periCacheKey = 0
}

func (a *App) periViewCacheKey(player *world.Entity, peri *acoustics.PeriscopeState, weather world.Weather, gameTime float64) uint64 {
	// Quantize pose so tiny drift does not thrash WritePixels every frame.
	px := int64(player.X / 40)
	py := int64(player.Y / 40)
	hd := int64(player.HeadingDeg)
	tr := int64(peri.TrainRelDeg)
	zm := int64(peri.Zoom)
	ext := int64(peri.Extension * 20)
	dep := int64(player.DepthFt)
	wt := int64(weather)
	tb := int64(gameTime / 0.2) // sea ripple bucket ~5 Hz
	var eh uint64
	if a.Engine != nil {
		for _, e := range a.Engine.Scenario.Entities {
			if e == nil || e.Kind != world.KindSurfaceShip {
				continue
			}
			if !e.Alive() && e.Status != world.StatusSinking {
				continue
			}
			eh ^= uint64(int64(e.X)/80)<<1 ^ uint64(int64(e.Y)/80)<<7 ^ uint64(int64(e.HeadingDeg))<<13
			eh = eh*1315423911 + uint64(len(e.ID))
		}
	}
	k := uint64(px)<<48 ^ uint64(py)<<32 ^ uint64(hd&0xff)<<24 ^ uint64(tr&0xff)<<16
	k ^= uint64(zm&7)<<12 ^ uint64(ext&0x1f)<<6 ^ uint64(dep&0x3f) ^ uint64(wt)<<3 ^ uint64(tb)
	k ^= eh
	if a.Engine != nil {
		sonar := &a.Engine.Sonar
		if sonar.LastBlastAt > 0 {
			age := gameTime - sonar.LastBlastAt
			if age >= 0 && age < 6.5 {
				// ~20 Hz while the IR flash evolves (grow + fade).
				k ^= uint64(int(age*20)+1) * 0x9e3779b97f4a7c15
				k ^= uint64(int(sonar.LastBlastX)/40)<<5 ^ uint64(int(sonar.LastBlastY)/40)<<11
			}
		}
		for _, hm := range a.Engine.FireControl.ActiveHarpoons {
			if hm == nil || !hm.Alive {
				continue
			}
			k ^= uint64(int(hm.X)/60)<<2 ^ uint64(int(hm.Y)/60)<<9
		}
	}
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
	key := a.periViewCacheKey(player, peri, weather, gt)
	if key != a.periCacheKey {
		a.buildPeriscopeIRFrame(player, peri, weather, gt)
		a.periCacheKey = key
		a.periImg.WritePixels(a.periPix)
	}

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
		if absD%5 == 0 {
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

func (a *App) buildPeriscopeIRFrame(player *world.Entity, peri *acoustics.PeriscopeState, weather world.Weather, gameTime float64) {
	pix := a.periPix
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

	// --- Sea ---
	rippleAmp := 6.0
	if weather == world.WeatherStorm {
		rippleAmp = 14
	} else if weather == world.WeatherCalm {
		rippleAmp = 2
	}
	for y := horizonY; y < h; y++ {
		t := float64(y-horizonY) / float64(h-horizonY)
		base := 70 - t*35 // brighter near horizon
		for x := 0; x < w; x++ {
			ripple := math.Sin(float64(x)*0.17+gameTime*2.1+float64(y)*0.31)*rippleAmp +
				math.Sin(float64(x)*0.41-gameTime*1.3)*rippleAmp*0.4
			v := int(base + ripple)
			if v < 18 {
				v = 18
			}
			if v > 160 {
				v = 160
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

	// --- Ships (far → near) ---
	a.periShipScratch = a.periShipScratch[:0]
	if a.Engine != nil && a.Engine.Scenario != nil {
		for _, e := range a.Engine.Scenario.Entities {
			proj, ok := acoustics.ProjectSurfaceShip(player.X, player.Y, look, fov, w, h, horizonY, maxR, eyeH, e)
			if !ok {
				continue
			}
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
	flash := sonar.LastBlastFlashSec
	if flash < 1 {
		flash = 2.5
	}
	if age < 0 || age > flash {
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
	if hit != nil && hit.Kind == world.KindSurfaceShip {
		if proj, pok := acoustics.ProjectSurfaceShip(player.X, player.Y, look, fov, w, h, horizonY, maxR*1.2, eyeH, hit); pok {
			cx = proj.CenterX
			waterY = proj.WaterY
		}
	}
	// Visual IR bloom is much shorter than acoustic washout.
	visDur := 5.5
	if flash < visDur {
		visDur = flash
	}
	if age > visDur {
		return
	}
	life := 1 - age/visDur
	// Expand for the first ~0.2s, then hold a flat blown-out silhouette.
	grow := age / 0.18
	if grow > 1 {
		grow = 1
	}
	scale := (0.35 + 0.65*grow) * (0.45 + 0.55*math.Sqrt(life)) * (1 - t*0.35)
	if scale < 0.08 {
		return
	}
	seed := uint32(sonar.LastBlastAt*1009) ^ uint32(int(bx)*733) ^ uint32(int(by)*997)
	drawPeriBlastIR(pix, w, h, cx, waterY, scale, life, seed)
}

// drawPeriBlastIR paints a flat white IR detonation: clipped plume core, low
// waterline halo, and hot ejecta dots — matching a blown-out thermal eyepiece.
func drawPeriBlastIR(pix []byte, w, h, cx, waterY int, scale, life float64, seed uint32) {
	rx := 6.0 + 28.0*scale // half-width of plume
	ry := 8.0 + 36.0*scale // half-height (rises above water)
	// Plume sits on the waterline and blooms upward (and a little downward spray).
	cy := waterY - int(ry*0.35)
	if cy < 1 {
		cy = 1
	}

	// --- Waterline halo: bright horizontal smear of hot gas / spray ---
	haloW := rx * (1.6 + 0.9*life)
	haloH := math.Max(1.5, 2.2+4.5*scale*life)
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
			// Flat white inside; only the extreme rim softens (IR clip).
			g := 255
			if r2 > 0.82 {
				g = int(255 * life * (1.05 - r2) / 0.23)
			} else {
				g = 255
			}
			periBrighten(pix, w, xx, yy, uint8(min255(g)))
		}
	}

	// --- Flat white plume (no internal shading) ---
	prx := int(rx) + 3
	pry := int(ry) + 3
	for dy := -pry; dy <= pry; dy++ {
		for dx := -prx; dx <= prx; dx++ {
			fx := float64(dx) / (rx + 0.01)
			fy := float64(dy) / (ry + 0.01)
			// Lean slightly "downrange" and bulge asymmetrically like a gas cloud.
			fx -= 0.12 * fy * fy
			n := periBlastNoise01(seed, dx, dy)
			edge := 0.72 + 0.38*n // irregular silhouette radius
			r := math.Hypot(fx, fy)
			if r > edge {
				continue
			}
			xx, yy := cx+dx, cy+dy
			if xx < 0 || xx >= w || yy < 0 || yy >= h {
				continue
			}
			// Pure clip-white core; 1px fringe only — never a soft volumetric falloff.
			if r < edge*0.88 {
				periBrighten(pix, w, xx, yy, 255)
			} else {
				periBrighten(pix, w, xx, yy, uint8(min255(int(200+55*life))))
			}
		}
	}

	// --- Ejecta / spray dots around the plume ---
	nDots := 18 + int(55*scale*life)
	if nDots > 90 {
		nDots = 90
	}
	for i := 0; i < nDots; i++ {
		u1 := periBlastNoise01(seed+uint32(i)*2654435761, i, 1)
		u2 := periBlastNoise01(seed+uint32(i)*2246822519, i, 2)
		u3 := periBlastNoise01(seed+uint32(i)*3266489917, i, 3)
		// Prefer upper hemisphere and sides; a few low skips on the water.
		ang := (u1*1.35 - 0.15) * math.Pi // ~-27°..~227° biased up/out
		dist := (0.55 + 1.35*u2) * ry
		dx := int(math.Cos(ang) * dist * (0.85 + 0.5*rx/ry))
		dy := int(-math.Sin(ang) * dist)
		// Secondary spray sheet near waterline.
		if u3 < 0.22 {
			dy = int((u2 - 0.5) * haloH * 2)
			dx = int((u1 - 0.5) * haloW * 2.2)
		}
		xx, yy := cx+dx, cy+dy
		if u3 < 0.22 {
			yy = waterY + dy
		}
		if xx < 0 || xx >= w || yy < 0 || yy >= h {
			continue
		}
		periBrighten(pix, w, xx, yy, 255)
		// Slightly soft 2×2 / cross blob so dots aren't perfect squares.
		if u3 > 0.45 {
			if xx+1 < w {
				periBrighten(pix, w, xx+1, yy, 255)
			}
			if yy+1 < h {
				periBrighten(pix, w, xx, yy+1, uint8(min255(int(220+35*life))))
			}
		}
	}
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
