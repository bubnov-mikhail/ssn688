package ui

import (
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
)

// Ship silhouette profiles are sampled continuously (procedural); raster sprites
// cover 0..180° at 1° steps.
// Each sample is a vertical slice height in [0,1] of total ship height (hull+super),
// plus a superstructure fraction of that height. Drawn as white-hot IR with
// hotspots, bloom, waterline reflection and wake — inspired by FLIR/MWIR feeds.

type periShipClass int

const (
	periClassMerchant periShipClass = iota
	periClassTanker
	periClassFishing
	periClassCombatant
)

type periShipCol struct {
	x, hullTop, superTop, hullPx, superPx int
	u                                     float64
	hot                                   int
}

func periShipClassOf(sig string) periShipClass {
	switch sig {
	case "tanker":
		return periClassTanker
	case "fishing":
		return periClassFishing
	case "grisha", "udaloy", "kresta2", "krivak", "gorshkov", "spruance":
		return periClassCombatant
	default:
		return periClassMerchant
	}
}

// periProfileAt interpolates hull/super fractions. aspectDeg may be 0..180;
// the procedural profile is symmetric about beam (stern uses 180−aspect).
// u in [-1,1] where +u is toward the bow.
func periProfileAt(class periShipClass, aspectDeg float64, u float64) (hullFrac, superFrac float64) {
	if aspectDeg > 90 {
		aspectDeg = 180 - aspectDeg
	}
	if aspectDeg < 0 {
		aspectDeg = 0
	}
	if aspectDeg > 90 {
		aspectDeg = 90
	}
	lo := int(math.Floor(aspectDeg))
	hi := lo + 1
	if hi > 90 {
		hi = 90
		lo = 89
	}
	t := aspectDeg - float64(lo)
	if lo == hi {
		t = 0
	}
	h0, s0 := periProfileBin(class, lo, u)
	h1, s1 := periProfileBin(class, hi, u)
	return h0*(1-t) + h1*t, s0*(1-t) + s1*t
}

func periProfileBin(class periShipClass, aspectBin int, u float64) (hullFrac, superFrac float64) {
	if u < -1 {
		u = -1
	}
	if u > 1 {
		u = 1
	}
	au := math.Abs(u)
	endOn := 1.0 - float64(aspectBin)/90.0

	switch class {
	case periClassTanker:
		hull := 0.22 + 0.18*(1-au*au)
		if au > 0.92 {
			hull *= 0.35 + 0.65*endOn
		}
		super := 0.0
		if u > -0.15 && u < 0.55 {
			super = 0.35 * (1 - math.Abs(u-0.2)/0.7)
		}
		if endOn > 0.7 {
			hull = 0.45
			super = 0.55 * (1 - au)
		}
		return hull, super
	case periClassFishing:
		hull := 0.28 + 0.2*(1-au)
		if au > 0.85 {
			hull *= 0.5
		}
		super := 0.0
		if u > -0.4 && u < 0.35 {
			super = 0.4 * (1 - math.Abs(u+0.05)/0.5)
		}
		if endOn > 0.7 {
			hull = 0.5
			super = 0.45 * (1 - au*0.8)
		}
		return hull, super
	case periClassCombatant:
		hull := 0.2 + 0.25*(1-au*au)
		if u > 0.55 {
			hull *= 0.55 + 0.45*(1-u)
		}
		if u < -0.75 {
			hull *= 0.7
		}
		super := 0.0
		if u > -0.55 && u < 0.35 {
			super = 0.55 * (1 - math.Abs(u+0.05)/0.6)
		}
		if u > -0.15 && u < 0.2 {
			super += 0.2
		}
		if endOn > 0.65 {
			hull = 0.4 + 0.15*(1-au)
			super = 0.65 * (1 - au*0.9)
		}
		return hull, math.Min(0.85, super)
	default:
		hull := 0.25 + 0.22*(1-au*au)
		if au > 0.9 {
			hull *= 0.4 + 0.4*endOn
		}
		super := 0.0
		if u > -0.1 && u < 0.45 {
			super = 0.42 * (1 - math.Abs(u-0.15)/0.55)
		}
		if endOn > 0.7 {
			hull = 0.48
			super = 0.5 * (1 - au)
		}
		return hull, super
	}
}

func drawPeriShipSilhouette(pix []byte, depth []float32, w, h int, p acoustics.PeriShipProj) {
	halfW := p.WidthPx / 2
	if halfW < 1 {
		halfW = 1
	}
	totalH := p.HullHPx + p.SuperHPx
	if totalH < 3 {
		totalH = 3
	}
	base := int(112 + p.Brightness*118)
	if p.Sinking {
		base = int(float64(base) * 0.65)
	}
	class := periShipClassOf(p.Signature)
	aspect := p.AspectDeg
	if aspect <= 0 && p.Aspect01 > 0 {
		aspect = math.Asin(math.Min(1, p.Aspect01)) * 180 / math.Pi
	}
	bowSign := 1.0
	if !p.BowRight {
		bowSign = -1
	}

	// Prefer Blender-rendered raster sprites; fall back to column profiles.
	if sp := pickPeriShipSprite(class, aspect); sp != nil {
		// Sprites authored bow-left (−X); flip when bow should face +X.
		flipX := p.BowRight
		// Lift IR hulls above the cold sea plate so silhouettes read clearly.
		br := p.Brightness*1.22 + 0.10
		if p.Sinking {
			br *= 0.65
		}
		if br > 1.35 {
			br = 1.35
		}
		// Angular width × air-draft height (stable vs aspect-bin PNG crop).
		x0, y0, x1, y1, ok := blitPeriShipSprite(pix, depth, w, h, sp, p.CenterX, p.WaterY, p.WidthPx, totalH, flipX, br, p.SinkFrac, p.RangeYd)
		if ok {
			drawPeriShipBloomRect(pix, w, h, x0, y0, x1, y1, p.WaterY, base)
			if p.Fire01 > 0.05 {
				mask := func(xx, yy int) bool {
					return periShipSpriteOpaqueAt(sp, p.CenterX, p.WaterY, p.WidthPx, totalH, flipX, p.SinkFrac, xx, yy)
				}
				drawPeriShipHullFire(pix, w, h, x0, y0, x1, y1, p.WaterY, p.Fire01, int(p.CenterX), p.FirePhase, mask)
			}
			return
		}
	}

	drawPeriShipSilhouetteProcedural(pix, depth, w, h, p, class, aspect, bowSign, base, halfW, totalH)
}

// drawPeriShipHullFire paints a few concentrated MWIR fire foci on hull pixels.
// mask rejects sky/sea gaps inside the ship AABB (sprite alpha or procedural profile).
func drawPeriShipHullFire(pix []byte, w, h, x0, y0, x1, y1, waterY int, fire01 float64, seedX int, phase float64, mask func(xx, yy int) bool) {
	if fire01 <= 0 || x1 <= x0+2 || waterY <= y0+1 {
		return
	}
	if y1 > waterY {
		y1 = waterY
	}
	shipH := waterY - y0
	if shipH < 2 {
		return
	}
	// A handful of stable outbreaks — not a scattered peppering of the hull.
	n := 2 + int(2.5*fire01)
	if n > 4 {
		n = 4
	}
	phaseBin := int(phase * 14)
	bw := x1 - x0
	for i := 0; i < n; i++ {
		// Prefer spaced foci along the length (bow / mid / stern-ish).
		uSlot := (float64(i) + 0.5) / float64(n)
		var px, py int
		found := false
		for try := 0; try < 36; try++ {
			jitter := periBlastNoise01(uint32(seedX*997+i*7919+try*17), i, 21+try)
			v := periBlastNoise01(uint32(seedX*1931+i*104729+try*31), i, 22+try)
			cx := x0 + int((uSlot*0.70+0.15+(jitter-0.5)*0.12)*float64(bw))
			if cx < x0 {
				cx = x0
			}
			if cx >= x1 {
				cx = x1 - 1
			}
			// Mid-deck / superstructure band.
			cy := y0 + int((0.18+0.55*v)*float64(shipH))
			if cy >= waterY {
				cy = waterY - 1
			}
			if mask != nil && !mask(cx, cy) {
				continue
			}
			px, py = cx, cy
			found = true
			break
		}
		if !found {
			continue
		}
		flick := periBlastNoise01(uint32(seedX+i*13), i, phaseBin+3)
		u := periBlastNoise01(uint32(seedX*997+i*7919), i, 21)
		pw := 3 + int(5.5*fire01*(0.55+0.45*flick))
		ph := 3 + int(6.5*fire01*(0.55+0.45*u))
		core := int(240 + 15*fire01*(0.5+0.5*flick))
		for dy := -ph; dy <= ph; dy++ {
			for dx := -pw; dx <= pw; dx++ {
				if dx*dx*2+dy*dy > pw*pw+ph*ph {
					continue
				}
				xx, yy := px+dx, py+dy
				if xx < 0 || xx >= w || yy < 0 || yy >= h || yy >= waterY {
					continue
				}
				if mask != nil && !mask(xx, yy) {
					continue
				}
				fall := 1 - math.Hypot(float64(dx)/float64(pw+1), float64(dy)/float64(ph+1))
				if fall < 0 {
					continue
				}
				g := int(float64(core) * (0.55 + 0.45*fall) * (0.70 + 0.30*fire01))
				periBrighten(pix, w, xx, yy, uint8(min255(g)))
			}
		}
	}
}

func drawPeriShipSilhouetteProcedural(pix []byte, depth []float32, w, h int, p acoustics.PeriShipProj, class periShipClass, aspect, bowSign float64, base, halfW, totalH int) {
	cols := make([]periShipCol, 0, halfW*2+1)
	sinkPx := int(float64(totalH)*p.SinkFrac + 0.5)
	if sinkPx < 0 {
		sinkPx = 0
	}
	if sinkPx >= totalH {
		return
	}
	rangeYd := float32(p.RangeYd)
	if rangeYd < 1 {
		rangeYd = 1
	}
	cx := int(math.Round(p.CenterX))

	for dx := -halfW; dx <= halfW; dx++ {
		x := cx + dx
		if x < 0 || x >= w {
			continue
		}
		uScreen := 0.0
		if halfW > 0 {
			uScreen = float64(dx) / float64(halfW)
		}
		u := uScreen * bowSign // +u = bow
		hf, sf := periProfileAt(class, aspect, u)
		hullPx := int(float64(totalH) * hf)
		superPx := int(float64(totalH) * sf)
		if hullPx < 1 && hf > 0.05 {
			hullPx = 1
		}
		hullTop := p.WaterY - hullPx + sinkPx
		if hullTop < 0 {
			hullTop = 0
		}
		hotBoost := periShipHotBoost(class, u, aspect)
		cols = append(cols, periShipCol{
			x: x, hullTop: hullTop, hullPx: hullPx, superPx: superPx, u: u, hot: hotBoost,
		})

		for y := hullTop; y < p.WaterY && y < h; y++ {
			if !periDepthTry(depth, w, x, y, rangeYd) {
				continue
			}
			frac := float64(y-hullTop) / float64(hullPx+1)
			tone := base - 18 + int(frac*12) + hotBoost/3
			if (dx+halfW)%3 == 0 {
				tone -= 8
			}
			if y >= p.WaterY-2 {
				tone += 18 + hotBoost/4
			}
			periSetGray(pix, w, x, y, uint8(min255(tone)))
		}

		if superPx <= 0 {
			continue
		}
		superTop := hullTop - superPx
		if superTop < 0 {
			superTop = 0
		}
		cols[len(cols)-1].superTop = superTop

		for y := superTop; y < hullTop && y < h && y < p.WaterY; y++ {
			if !periDepthTry(depth, w, x, y, rangeYd) {
				continue
			}
			vFrac := float64(y-superTop) / float64(superPx+1)
			tone := base + 22 + hotBoost
			if class != periClassFishing && vFrac > 0.35 && vFrac < 0.55 {
				tone -= 35
			}
			if vFrac < 0.18 {
				tone += 20 + hotBoost/2
			}
			if periShipFunnelZone(class, u) && vFrac < 0.45 {
				tone = min255(tone + 40)
			}
			periSetGray(pix, w, x, y, uint8(min255(tone)))
		}

		if class == periClassCombatant && aspect > 35 {
			if (u > 0.45 && u < 0.75) || (u > -0.7 && u < -0.45) {
				gunH := 1 + superPx/5
				for y := hullTop - gunH; y < hullTop && y >= 0 && y < p.WaterY; y++ {
					if !periDepthTry(depth, w, x, y, rangeYd) {
						continue
					}
					periSetGray(pix, w, x, y, uint8(min255(base+35)))
				}
			}
		}
	}

	drawPeriShipMasts(pix, w, h, p, class, aspect, bowSign, base, cols)
	drawPeriShipBloom(pix, w, h, p, cols, base)
	if p.Fire01 > 0.05 {
		half := p.WidthPx / 2
		if half < 1 {
			half = 1
		}
		mask := periProceduralShipMask(p, class, aspect, bowSign, half, totalH)
		drawPeriShipHullFire(pix, w, h, cx-half, p.WaterY-(p.HullHPx+p.SuperHPx), cx+half, p.WaterY, p.WaterY, p.Fire01, cx, p.FirePhase, mask)
	}
}

// periProceduralShipMask returns true for pixels that fall on the procedural hull/super profile.
func periProceduralShipMask(p acoustics.PeriShipProj, class periShipClass, aspect, bowSign float64, halfW, totalH int) func(xx, yy int) bool {
	sinkPx := int(float64(totalH)*p.SinkFrac + 0.5)
	if sinkPx < 0 {
		sinkPx = 0
	}
	return func(xx, yy int) bool {
		if yy < 0 || yy >= p.WaterY {
			return false
		}
		dx := xx - int(math.Round(p.CenterX))
		if dx < -halfW || dx > halfW || halfW < 1 {
			return false
		}
		uScreen := float64(dx) / float64(halfW)
		u := uScreen * bowSign
		hf, sf := periProfileAt(class, aspect, u)
		hullPx := int(float64(totalH) * hf)
		superPx := int(float64(totalH) * sf)
		if hullPx < 1 && hf > 0.05 {
			hullPx = 1
		}
		hullTop := p.WaterY - hullPx + sinkPx
		top := hullTop - superPx
		return yy >= top && yy < p.WaterY
	}
}

func drawPeriShipBloomRect(pix []byte, w, h, x0, y0, x1, y1, waterY, base int) {
	thresh := base + 20
	glow := 18
	if y0 < 0 {
		y0 = 0
	}
	if y1 > waterY {
		y1 = waterY
	}
	if y1 > h {
		y1 = h
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x < 0 || x >= w {
				continue
			}
			i := (y*w + x) * 4
			if i+3 >= len(pix) || int(pix[i]) < thresh {
				continue
			}
			for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				xx, yy := x+d[0], y+d[1]
				if xx < 0 || xx >= w || yy < 0 || yy >= h {
					continue
				}
				j := (yy*w + xx) * 4
				if int(pix[j])+glow/2 < int(pix[i]) {
					periBrighten(pix, w, xx, yy, uint8(min255(int(pix[j])+glow)))
				}
			}
		}
	}
}

// periShipHotBoost returns extra IR brightness for engine / island hotspots.
func periShipHotBoost(class periShipClass, u, aspectDeg float64) int {
	boost := 0
	switch class {
	case periClassTanker:
		// Aft island / machinery.
		if u > -0.05 && u < 0.5 {
			boost = int(28 * (1 - math.Abs(u-0.22)/0.55))
		}
	case periClassFishing:
		if u > -0.35 && u < 0.25 {
			boost = int(32 * (1 - math.Abs(u+0.05)/0.45))
		}
	case periClassCombatant:
		// Midships engineering + stack.
		if u > -0.45 && u < 0.25 {
			boost = int(36 * (1 - math.Abs(u+0.1)/0.55))
		}
		if u > -0.2 && u < 0.05 {
			boost += 12
		}
	default:
		if u > -0.05 && u < 0.4 {
			boost = int(26 * (1 - math.Abs(u-0.15)/0.5))
		}
	}
	// End-on (bow or stern): concentrate heat in the narrow silhouette.
	endOn := aspectDeg
	if endOn > 90 {
		endOn = 180 - endOn
	}
	if endOn < 25 {
		boost = boost*2/3 + 10
	}
	return boost
}

func periShipFunnelZone(class periShipClass, u float64) bool {
	switch class {
	case periClassTanker:
		return u > 0.05 && u < 0.28
	case periClassCombatant:
		return u > -0.25 && u < -0.02
	case periClassFishing:
		return u > -0.15 && u < 0.05
	default:
		return u > 0.0 && u < 0.22
	}
}

func drawPeriShipMasts(pix []byte, w, h int, p acoustics.PeriShipProj, class periShipClass, aspect, bowSign float64, base int, cols []periShipCol) {
	if len(cols) == 0 || totalShipVisibleH(cols) < 4 {
		return
	}
	mastTone := uint8(min255(base + 48))
	antennaTone := uint8(min255(base + 55))

	// Primary mast near island peak.
	mastU := 0.08
	switch class {
	case periClassCombatant:
		mastU = -0.05
	case periClassFishing:
		mastU = -0.1
	case periClassTanker:
		mastU = 0.18
	}
	drawThinMastAtU(pix, w, h, p, cols, mastU, bowSign, 0.55+0.25*math.Min(1, aspect/90), mastTone, true)

	// Secondary mast / jackstaff on combatants & merchants when beam-ish.
	if class != periClassFishing && aspect > 40 {
		drawThinMastAtU(pix, w, h, p, cols, 0.55, bowSign, 0.28, antennaTone, false)
		if class == periClassCombatant {
			drawThinMastAtU(pix, w, h, p, cols, -0.55, bowSign, 0.22, antennaTone, false)
		}
	}
}

func totalShipVisibleH(cols []periShipCol) int {
	maxH := 0
	for _, c := range cols {
		h := c.hullPx + c.superPx
		if h > maxH {
			maxH = h
		}
	}
	return maxH
}

func drawThinMastAtU(pix []byte, w, h int, p acoustics.PeriShipProj, cols []periShipCol, uTarget, bowSign, heightFrac float64, tone uint8, withYard bool) {
	// Find closest column to desired bow-relative u.
	best := -1
	bestDU := 99.0
	for i, c := range cols {
		du := math.Abs(c.u - uTarget)
		if du < bestDU {
			bestDU = du
			best = i
		}
	}
	if best < 0 || bestDU > 0.2 {
		return
	}
	c := cols[best]
	top := c.hullTop
	if c.superPx > 0 {
		top = c.superTop
	}
	mastH := int(float64(p.HullHPx+p.SuperHPx) * heightFrac)
	if mastH < 2 {
		mastH = 2
	}
	for y := top - mastH; y < top && y >= 0; y++ {
		periBrighten(pix, w, c.x, y, tone)
	}
	if withYard && mastH >= 4 && p.WidthPx >= 8 {
		yardY := top - mastH*2/3
		arm := 1 + p.WidthPx/14
		for dx := -arm; dx <= arm; dx++ {
			xx := c.x + dx
			if xx < 0 || xx >= w || yardY < 0 || yardY >= h {
				continue
			}
			periBrighten(pix, w, xx, yardY, tone)
		}
		// Short VHF whip above yard.
		periBrighten(pix, w, c.x, top-mastH-1, tone)
	}
	_ = bowSign
}

func drawPeriShipBloom(pix []byte, w, h int, p acoustics.PeriShipProj, cols []periShipCol, base int) {
	if p.Brightness < 0.25 || len(cols) == 0 {
		return
	}
	thresh := base + 30
	glow := uint8(min255(int(40 * p.Brightness)))
	if glow < 12 {
		glow = 12
	}
	for _, c := range cols {
		top := c.hullTop
		if c.superPx > 0 {
			top = c.superTop
		}
		for y := top; y < p.WaterY && y < h; y++ {
			i := (y*w + c.x) * 4
			if i+3 >= len(pix) || int(pix[i]) < thresh {
				continue
			}
			// 1px halo into cooler background (IR bloom).
			for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				xx, yy := c.x+d[0], y+d[1]
				if xx < 0 || xx >= w || yy < 0 || yy >= h {
					continue
				}
				j := (yy*w + xx) * 4
				if int(pix[j])+int(glow)/2 < int(pix[i]) {
					periBrighten(pix, w, xx, yy, uint8(min255(int(pix[j])+int(glow))))
				}
			}
		}
	}
}
