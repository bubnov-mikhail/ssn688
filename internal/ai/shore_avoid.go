package ai

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

const (
	shoreClearanceYd = 800.0
	shoreLookAheadYd = 2400.0
)

// applyShoreAvoidance steers surface units away from coast when look-ahead
// shows the ordered course would violate the 800 yd clearance band.
func applyShoreAvoidance(ship *world.Entity, bathy *world.Bathymetry) {
	if ship == nil || bathy == nil || !bathy.Valid() {
		return
	}
	if !ship.Alive() || ship.Kind != world.KindSurfaceShip || ship.DepthFt > 1 {
		return
	}
	head, avoid := shoreAvoidHeading(bathy, ship)
	if !avoid {
		return
	}
	ship.OrderedHead = head
	if ship.AIState != "SHORE_AVOID" {
		world.InterruptRoute(ship)
	}
	if ship.AIState == "TRANSIT" || ship.AIState == "CRUISE" || ship.AIState == "SEARCH" || ship.AIState == "PATROL" || ship.AIState == "SHORE_AVOID" {
		ship.AIState = "SHORE_AVOID"
	}
}

func shoreAvoidHeading(b *world.Bathymetry, ship *world.Entity) (float64, bool) {
	if courseThreatensShore(b, ship.X, ship.Y, ship.OrderedHead) {
		return headingToOpenWater(b, ship), true
	}
	return ship.OrderedHead, false
}

func courseThreatensShore(b *world.Bathymetry, x, y, headingDeg float64) bool {
	cur := b.DistanceToShoreYd(x, y)
	rad := headingDeg * math.Pi / 180
	sinH := math.Sin(rad)
	cosH := math.Cos(rad)
	minAhead := cur
	for d := 200.0; d <= shoreLookAheadYd; d += 200 {
		px := x + sinH*d
		py := y + cosH*d
		if b.IsShoreBlocked(px, py) {
			return true
		}
		if !b.OnChart(px, py) {
			continue
		}
		if dist := b.DistanceToShoreYd(px, py); dist < minAhead {
			minAhead = dist
		}
	}
	// Closing on shore inside the 800 yd band (parallel coast-keeping is OK).
	if minAhead < shoreClearanceYd && minAhead < cur*0.85 {
		return true
	}
	return false
}

func headingToOpenWater(b *world.Bathymetry, ship *world.Entity) float64 {
	cur := b.DistanceToShoreYd(ship.X, ship.Y)
	base := ship.OrderedHead
	best := base
	bestScore := -1e9
	for delta := -180; delta <= 180; delta += 10 {
		h := normalizeHead(base + float64(delta))
		score := shoreHeadingScore(b, ship.X, ship.Y, h)
		if score > cur {
			score += (score - cur) * 0.5
		}
		score -= math.Abs(float64(delta)) * 0.4
		if score > bestScore {
			bestScore = score
			best = h
		}
	}
	if bestScore < -500 {
		shoreBrg := b.NearestShoreBearingDeg(ship.X, ship.Y)
		rel := normalizeRel(shoreBrg - ship.HeadingDeg)
		turn := 90.0
		if rel >= 0 {
			turn = -90
		}
		return normalizeHead(ship.HeadingDeg + turn)
	}
	return best
}

func normalizeRel(d float64) float64 {
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}

func shoreHeadingScore(b *world.Bathymetry, x, y, headingDeg float64) float64 {
	rad := headingDeg * math.Pi / 180
	sinH := math.Sin(rad)
	cosH := math.Cos(rad)
	worst := 1e9
	for _, d := range []float64{shoreClearanceYd, shoreClearanceYd * 1.5, shoreClearanceYd * 2} {
		px := x + sinH*d
		py := y + cosH*d
		if b.IsShoreBlocked(px, py) || !b.OnChart(px, py) {
			return -1000
		}
		dist := b.DistanceToShoreYd(px, py)
		if dist < worst {
			worst = dist
		}
	}
	return worst
}
