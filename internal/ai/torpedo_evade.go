package ai

import (
	"math"

	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

const (
	// Awareness envelopes (yards) — sonar "hears" the fish, not god-mode knowledge.
	torpAwareShipActiveYd = 5500.0
	torpAwareShipQuietYd  = 3200.0
	torpAwareSubActiveYd  = 4200.0
	torpAwareSubQuietYd   = 2200.0
	// Aim gate: fish nose within this of line-of-sight to us, or already locked.
	torpAimGateDeg = 48.0
)

// tryEvadeTorpedo steers and accelerates away from the most threatening player fish.
// Returns true when evasion orders were applied (caller should skip other AI).
func tryEvadeTorpedo(e *world.Entity, torps []*weapons.Torpedo) bool {
	if e == nil || !e.Alive() {
		return false
	}
	threat := mostThreateningTorpedo(e, torps)
	if threat == nil {
		return false
	}
	applyTorpedoEvade(e, threat)
	return true
}

func mostThreateningTorpedo(e *world.Entity, torps []*weapons.Torpedo) *weapons.Torpedo {
	aware := torpedoAwarenessYd(e)
	var best *weapons.Torpedo
	bestScore := 0.0
	for _, t := range torps {
		if t == nil || !t.Alive || t.Side != world.SidePlayer {
			continue
		}
		d := math.Hypot(t.X-e.X, t.Y-e.Y)
		if d < 1 || d > aware {
			continue
		}
		// Seeker ping makes the fish much more obvious — stretch awareness a bit.
		lim := aware
		if t.Mode == weapons.ModeSearch && t.LastPingTime >= 0 {
			lim *= 1.25
		}
		if d > lim {
			continue
		}
		brgFishToShip := bearingDeg(t.X, t.Y, e.X, e.Y)
		aimErr := math.Abs(shortestRel(brgFishToShip - t.HeadingDeg))
		locked := t.TargetID == e.ID
		if !locked && aimErr > torpAimGateDeg && d > 900 {
			continue // not coming at us
		}
		if locked {
			aimErr *= 0.35
		}
		// Higher score = closer and better aimed.
		score := (lim - d) * (1.1 - aimErr/90)
		if t.SpeedKts > 40 {
			score *= 1.15
		}
		if score > bestScore {
			bestScore = score
			best = t
		}
	}
	return best
}

func torpedoAwarenessYd(e *world.Entity) float64 {
	active := e.ActiveSonar
	switch e.Kind {
	case world.KindSurfaceShip:
		if active {
			return torpAwareShipActiveYd
		}
		return torpAwareShipQuietYd
	default:
		if active {
			return torpAwareSubActiveYd
		}
		return torpAwareSubQuietYd
	}
}

func applyTorpedoEvade(e *world.Entity, threat *weapons.Torpedo) {
	// "Comb the torpedo": high speed, course ~90° to the fish track to open CPA.
	track := threat.HeadingDeg
	combPort := normalizeHead(track - 90)
	combStbd := normalizeHead(track + 90)
	brgToFish := bearingDeg(e.X, e.Y, threat.X, threat.Y)
	if math.Abs(shortestRel(combPort-brgToFish)) > math.Abs(shortestRel(combStbd-brgToFish)) {
		e.OrderedHead = combPort
	} else {
		e.OrderedHead = combStbd
	}
	d := math.Hypot(threat.X-e.X, threat.Y-e.Y)
	if d < 600 {
		// Very close: bias stern-to-threat so we don't comb into the fish.
		away := normalizeHead(brgToFish + 180)
		e.OrderedHead = blendHeadings(e.OrderedHead, away, 0.55)
	}

	e.ActiveSonar = false
	e.AIState = "TORPEDO_EVADE"

	switch e.Kind {
	case world.KindSurfaceShip:
		e.OrderedSpeed = 28
		e.OrderedDepth = 0
	case world.KindSubmarine:
		e.OrderedSpeed = 20
		if threat.DepthFt <= e.DepthFt {
			e.OrderedDepth = math.Min(520, e.DepthFt+160)
		} else {
			e.OrderedDepth = math.Max(80, e.DepthFt-140)
		}
	default:
		e.OrderedSpeed = math.Max(e.OrderedSpeed, 18)
	}
}

func blendHeadings(a, b, towardB float64) float64 {
	d := shortestRel(b - a)
	return normalizeHead(a + d*towardB)
}

func bearingDeg(x0, y0, x1, y1 float64) float64 {
	deg := math.Atan2(x1-x0, y1-y0) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}

func normalizeHead(h float64) float64 {
	for h < 0 {
		h += 360
	}
	for h >= 360 {
		h -= 360
	}
	return h
}

func shortestRel(d float64) float64 {
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}
