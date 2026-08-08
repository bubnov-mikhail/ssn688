package ui

import (
	"math"

	"github.com/ssn688/sim/internal/acoustics"
)

// Ship silhouette profiles are sampled every 5° of aspect (0=bow-on … 90=beam).
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
	case "grisha", "udaloy", "kresta2", "krivak":
		return periClassCombatant
	default:
		return periClassMerchant
	}
}

// periProfileAt returns (hullFrac, superFrac) of total height at horizontal
// position u in [-1,1] where +u is toward the bow.
func periProfileAt(class periShipClass, aspectDeg float64, u float64) (hullFrac, superFrac float64) {
	bin := acoustics.ShipAspectBin5(aspectDeg)
	lo := bin
	if aspectDeg < float64(bin) {
		lo = bin - 5
	}
	if lo < 0 {
		lo = 0
	}
	hi := lo + 5
	if hi > 90 {
		hi = 90
		lo = 85
	}
	t := 0.0
	if hi > lo {
		t = (aspectDeg - float64(lo)) / float64(hi-lo)
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
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

func drawPeriShipSilhouette(pix []byte, w, h int, p acoustics.PeriShipProj) {
	halfW := p.WidthPx / 2
	if halfW < 1 {
		halfW = 1
	}
	totalH := p.HullHPx + p.SuperHPx
	if totalH < 3 {
		totalH = 3
	}
	base := int(95 + p.Brightness*100)
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

	// Track peak column tops for masts / bloom seed.
	cols := make([]periShipCol, 0, halfW*2+1)

	for dx := -halfW; dx <= halfW; dx++ {
		x := p.CenterX + dx
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
		hullTop := p.WaterY - hullPx
		if hullTop < 0 {
			hullTop = 0
		}
		hotBoost := periShipHotBoost(class, u, aspect)
		cols = append(cols, periShipCol{
			x: x, hullTop: hullTop, hullPx: hullPx, superPx: superPx, u: u, hot: hotBoost,
		})

		// Hull: cooler lower, warmer upper plate + waterline friction glow.
		for y := hullTop; y < p.WaterY && y < h; y++ {
			frac := float64(y-hullTop) / float64(hullPx+1) // 0=deck, 1=water
			tone := base - 18 + int(frac*12) + hotBoost/3
			// Panel / rib suggestion along length.
			if (dx+halfW)%3 == 0 {
				tone -= 8
			}
			if y >= p.WaterY-2 {
				tone += 18 + hotBoost/4 // waterline thermal halo
			}
			periBrighten(pix, w, x, y, uint8(min255(tone)))
		}

		if superPx <= 0 {
			continue
		}
		superTop := hullTop - superPx
		if superTop < 0 {
			superTop = 0
		}
		cols[len(cols)-1].superTop = superTop

		for y := superTop; y < hullTop && y < h; y++ {
			vFrac := float64(y-superTop) / float64(superPx+1)
			tone := base + 22 + hotBoost
			// Bridge glass / recess bands (cooler).
			if class != periClassFishing && vFrac > 0.35 && vFrac < 0.55 {
				tone -= 35
			}
			// Roof / stack lip hotter.
			if vFrac < 0.18 {
				tone += 20 + hotBoost/2
			}
			// Exhaust funnel zone (class-specific u).
			if periShipFunnelZone(class, u) && vFrac < 0.45 {
				tone = min255(tone + 40)
			}
			periBrighten(pix, w, x, y, uint8(min255(tone)))
		}

		// Deck gear / gun mount bumps on combatants.
		if class == periClassCombatant && aspect > 35 {
			if (u > 0.45 && u < 0.75) || (u > -0.7 && u < -0.45) {
				gunH := 1 + superPx/5
				for y := hullTop - gunH; y < hullTop && y >= 0; y++ {
					periBrighten(pix, w, x, y, uint8(min255(base+35)))
				}
			}
		}
	}

	// Masts, yards, thin antennas — crisp IR lines.
	drawPeriShipMasts(pix, w, h, p, class, aspect, bowSign, base, cols)

	// Soft bloom around the hottest silhouette pixels.
	drawPeriShipBloom(pix, w, h, p, cols, base)

	// Thermal reflection / shimmer under the hull.
	drawPeriShipReflection(pix, w, h, p, cols, base)

	// Wake / bow spray when making way.
	if p.SpeedKts >= 3 && aspect > 25 && !p.Sinking {
		drawPeriShipWake(pix, w, h, p, bowSign, base)
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
	// End-on: concentrate heat in the narrow silhouette.
	if aspectDeg < 25 {
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

func drawPeriShipReflection(pix []byte, w, h int, p acoustics.PeriShipProj, cols []periShipCol, base int) {
	depth := 2 + (p.HullHPx+p.SuperHPx)/3
	if depth > 14 {
		depth = 14
	}
	if depth < 2 || p.WaterY >= h-1 {
		return
	}
	for _, c := range cols {
		srcTop := c.hullTop
		if c.superPx > 0 {
			srcTop = c.superTop
		}
		shipH := p.WaterY - srcTop
		if shipH < 2 {
			continue
		}
		for d := 1; d <= depth; d++ {
			yy := p.WaterY + d
			if yy >= h {
				break
			}
			// Sample mirrored ship pixel (compressed vertically).
			srcY := p.WaterY - 1 - (d-1)*shipH/depth
			if srcY < srcTop {
				srcY = srcTop
			}
			si := (srcY*w + c.x) * 4
			if si+3 >= len(pix) {
				continue
			}
			fall := 1 - float64(d)/float64(depth+1)
			// Hotter columns leave a stronger shimmer; add speckled noise.
			n := periHash8(c.x, yy, d) % 7
			g := int(float64(pix[si]) * 0.55 * fall * p.Brightness)
			g += int(n) - 3
			if g < 0 {
				g = 0
			}
			// Keep reflection brighter than open sea near waterline.
			floor := base/5 + int(20*fall)
			if g < floor {
				g = floor
			}
			periBrighten(pix, w, c.x, yy, uint8(min255(g)))
			if d <= 2 && c.x+1 < w {
				periBrighten(pix, w, c.x+1, yy, uint8(min255(g*3/4)))
			}
		}
	}
}

func drawPeriShipWake(pix []byte, w, h int, p acoustics.PeriShipProj, bowSign float64, base int) {
	halfW := p.WidthPx / 2
	if halfW < 2 {
		halfW = 2
	}
	// Stern is opposite bow on screen.
	sternDir := -1
	if bowSign < 0 {
		sternDir = 1
	}
	wakeLen := halfW + int(p.SpeedKts*0.9)
	if wakeLen > halfW*3 {
		wakeLen = halfW * 3
	}
	if wakeLen > 40 {
		wakeLen = 40
	}
	spd := math.Min(1, p.SpeedKts/18)
	for s := 1; s <= wakeLen; s++ {
		fall := 1 - float64(s)/float64(wakeLen+1)
		cx := p.CenterX + sternDir*(halfW+s)
		// Slight V-spread.
		spread := 1 + s/5
		for dx := -spread; dx <= spread; dx++ {
			xx := cx + dx
			yy := p.WaterY + 1 + (s % 3)
			if xx < 0 || xx >= w || yy < 0 || yy >= h {
				continue
			}
			edge := 1 - math.Abs(float64(dx))/float64(spread+1)
			g := int(float64(base) * 0.35 * fall * edge * (0.5 + 0.5*spd))
			g += int(periHash8(xx, yy, s)%5) - 2
			if g > 0 {
				periBrighten(pix, w, xx, yy, uint8(min255(g)))
			}
		}
	}
	// Bow spray speckles.
	bowX := p.CenterX + int(bowSign)*halfW
	for i := 0; i < 6+int(spd*8); i++ {
		xx := bowX + int(bowSign)*(i%3) + (int(periHash8(bowX, i, 3)%3) - 1)
		yy := p.WaterY - 1 - int(periHash8(i, bowX, 7)%3)
		if xx < 0 || xx >= w || yy < 0 || yy >= h {
			continue
		}
		periBrighten(pix, w, xx, yy, uint8(min255(base+20)))
	}
}
