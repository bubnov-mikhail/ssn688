package ai

import (
	"math"

	"github.com/ssn688/sim/internal/acoustics"
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
	// Zigzag amplitude around the comb course while evading.
	evadeZigzagDeg = 18.0
	evadeZigzagSec = 9.0
)

// EvadeContext supplies CM batteries and ocean layers for enriched evasion.
type EvadeContext struct {
	CM       *weapons.CountermeasureSystem
	Env      acoustics.Environment
	GameTime float64
}

// tryEvadeTorpedo steers, deploys CM, and accelerates away from the most threatening fish.
// Returns true when evasion orders were applied (caller should skip other AI).
func tryEvadeTorpedo(e *world.Entity, torps []*weapons.Torpedo, ctx EvadeContext) bool {
	if e == nil || !e.Alive() {
		return false
	}
	threat := mostThreateningTorpedo(e, torps)
	if threat == nil {
		if ctx.CM != nil && e.Kind == world.KindSurfaceShip {
			// Drop Nixie once the immediate torpedo threat is gone.
			ctx.CM.SetNixie(e.ID, false)
		}
		return false
	}
	applyTorpedoEvade(e, threat, ctx)
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

func applyTorpedoEvade(e *world.Entity, threat *weapons.Torpedo, ctx EvadeContext) {
	track := threat.HeadingDeg
	combPort := normalizeHead(track - 90)
	combStbd := normalizeHead(track + 90)
	brgToFish := bearingDeg(e.X, e.Y, threat.X, threat.Y)
	baseComb := combStbd
	if math.Abs(shortestRel(combPort-brgToFish)) > math.Abs(shortestRel(combStbd-brgToFish)) {
		baseComb = combPort
	}
	d := math.Hypot(threat.X-e.X, threat.Y-e.Y)
	if d < 600 {
		away := normalizeHead(brgToFish + 180)
		baseComb = blendHeadings(baseComb, away, 0.55)
	}

	// Zigzag around comb course — harder for a re-acquiring seeker to hold CPA.
	zig := evadeZigzagDeg * math.Sin(ctx.GameTime*(2*math.Pi/evadeZigzagSec)+float64(hashID(e.ID)%7))
	e.OrderedHead = normalizeHead(baseComb + zig)

	e.ActiveSonar = false
	e.AIState = "TORPEDO_EVADE"
	collisionThreat := torpedoCollisionThreat(e, threat)

	switch e.Kind {
	case world.KindSurfaceShip:
		e.OrderedSpeed = 28
		e.OrderedDepth = 0
		if ctx.CM != nil && collisionThreat {
			ctx.CM.EnsureMagazine(e.ID)
			ctx.CM.SetNixie(e.ID, true)
			ctx.CM.DeployADC(e, threat.X, threat.Y, ctx.GameTime)
			ctx.CM.DeployJitter(e, threat.X, threat.Y, ctx.GameTime)
		} else if ctx.CM != nil {
			ctx.CM.SetNixie(e.ID, false)
		}
	case world.KindSubmarine:
		e.OrderedSpeed = 20
		e.OrderedDepth = evadeDepthForSub(e, threat, ctx.Env)
		if ctx.CM != nil && collisionThreat {
			ctx.CM.EnsureMagazine(e.ID)
			ctx.CM.DeployADC(e, threat.X, threat.Y, ctx.GameTime)
			ctx.CM.DeployJitter(e, threat.X, threat.Y, ctx.GameTime)
		}
	default:
		e.OrderedSpeed = math.Max(e.OrderedSpeed, 18)
	}
}

func torpedoCollisionThreat(e *world.Entity, threat *weapons.Torpedo) bool {
	if e == nil || threat == nil {
		return false
	}
	return world.CollisionThreat2D(
		e.X, e.Y, e.HeadingDeg, e.SpeedKts,
		threat.X, threat.Y, threat.HeadingDeg, threat.SpeedKts,
		14*60.0, 260.0,
	)
}

func evadeDepthForSub(e *world.Entity, threat *weapons.Torpedo, env acoustics.Environment) float64 {
	// Prefer crossing a known layer boundary relative to the fish.
	fishZ := threat.DepthFt
	ownZ := e.DepthFt
	bound := 240.0 // default mixed/thermocline from DefaultEnvironment
	if len(env.Layers) > 1 {
		bound = env.Layers[1].TopDepthFt
	}
	if fishZ <= bound+40 {
		// Fish in/near mixed layer — go deep under the thermocline.
		return math.Min(520, math.Max(bound+120, ownZ+140))
	}
	if ownZ > fishZ {
		return math.Min(520, ownZ+120)
	}
	return math.Max(80, ownZ-140)
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
