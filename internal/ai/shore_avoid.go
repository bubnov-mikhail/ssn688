package ai

import (
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const (
	shoreClearanceYd = 800.0
	shoreLookAheadYd = 2400.0
)

// applyShoreAvoidance steers units away from coast/shallow water when look-ahead
// shows the ordered course would ground or violate the surface clearance band.
func applyShoreAvoidance(e *world.Entity, bathy *world.Bathymetry) {
	if e == nil || bathy == nil || !bathy.Valid() || !e.Alive() {
		return
	}
	// Scripted civilian routes are already shore-stitched; helm avoidance fights the track.
	if e.Side == world.SideNeutral {
		return
	}
	var head float64
	var avoid bool
	switch e.Kind {
	case world.KindSurfaceShip:
		if e.DepthFt > 1 {
			return
		}
		head, avoid = surfaceShoreAvoidHeading(bathy, e)
	case world.KindSubmarine:
		head, avoid = subTerrainAvoidHeading(bathy, e)
	default:
		return
	}
	if !avoid {
		return
	}
	e.OrderedHead = head
	if e.Side == world.SideNeutral {
		if routeCruiseAIState(e.AIState) {
			e.AIState = "SHORE_AVOID"
		}
		return
	}
	if e.AIState != "SHORE_AVOID" {
		world.InterruptRoute(e)
	}
	if routeCruiseAIState(e.AIState) {
		e.AIState = "SHORE_AVOID"
	}
}

func routeCruiseAIState(state string) bool {
	switch state {
	case "TRANSIT", "CRUISE", "SEARCH", "PATROL", "SHORE_AVOID":
		return true
	default:
		return false
	}
}

func surfaceShoreAvoidHeading(b *world.Bathymetry, ship *world.Entity) (float64, bool) {
	if courseThreatensShore(b, ship.X, ship.Y, ship.OrderedHead) {
		return headingToOpenWater(b, ship), true
	}
	return ship.OrderedHead, false
}

func subTerrainAvoidHeading(b *world.Bathymetry, sub *world.Entity) (float64, bool) {
	if courseThreatensTerrain(b, sub, sub.OrderedHead) {
		return headingToOpenTerrain(b, sub), true
	}
	return sub.OrderedHead, false
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
	if minAhead < shoreClearanceYd && minAhead < cur*0.85 {
		return true
	}
	return false
}

func courseThreatensTerrain(b *world.Bathymetry, e *world.Entity, headingDeg float64) bool {
	depth := world.NavDepthFt(e)
	rad := headingDeg * math.Pi / 180
	sinH := math.Sin(rad)
	cosH := math.Cos(rad)
	for d := 200.0; d <= shoreLookAheadYd; d += 200 {
		px := e.X + sinH*d
		py := e.Y + cosH*d
		if !b.NavigableFor(px, py, e.Kind, depth) {
			return true
		}
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

func headingToOpenTerrain(b *world.Bathymetry, e *world.Entity) float64 {
	base := e.OrderedHead
	best := base
	bestScore := -1e9
	for delta := -180; delta <= 180; delta += 10 {
		h := normalizeHead(base + float64(delta))
		score := terrainHeadingScore(b, e, h)
		score -= math.Abs(float64(delta)) * 0.4
		if score > bestScore {
			bestScore = score
			best = h
		}
	}
	if bestScore < -500 {
		shoreBrg := b.NearestShoreBearingDeg(e.X, e.Y)
		rel := normalizeRel(shoreBrg - e.HeadingDeg)
		turn := 90.0
		if rel >= 0 {
			turn = -90
		}
		return normalizeHead(e.HeadingDeg + turn)
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

func terrainHeadingScore(b *world.Bathymetry, e *world.Entity, headingDeg float64) float64 {
	depth := world.NavDepthFt(e)
	rad := headingDeg * math.Pi / 180
	sinH := math.Sin(rad)
	cosH := math.Cos(rad)
	worst := 1e9
	for _, d := range []float64{400, 800, 1200, 2000} {
		px := e.X + sinH*d
		py := e.Y + cosH*d
		if !b.NavigableFor(px, py, e.Kind, depth) {
			return -1000
		}
		bottom := b.DepthAtFt(px, py)
		margin := bottom - depth - 40
		if margin < worst {
			worst = margin
		}
	}
	return worst
}
