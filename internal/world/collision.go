package world

import "math"

// CollisionThreat2D reports whether two horizontal tracks are on a dangerous
// closing CPA within the lookahead window. It ignores depth and assumes
// constant current course/speed for both movers.
func CollisionThreat2D(ownX, ownY, ownHeadDeg, ownSpeedKts, tgtX, tgtY, tgtHeadDeg, tgtSpeedKts, lookaheadSec, missYd float64) bool {
	if tgtSpeedKts < 0.5 {
		return false
	}
	or := ownHeadDeg * math.Pi / 180
	tr := tgtHeadDeg * math.Pi / 180
	ovx := math.Sin(or) * ownSpeedKts * KnotsToYPS
	ovy := math.Cos(or) * ownSpeedKts * KnotsToYPS
	tvx := math.Sin(tr) * tgtSpeedKts * KnotsToYPS
	tvy := math.Cos(tr) * tgtSpeedKts * KnotsToYPS
	rx := tgtX - ownX
	ry := tgtY - ownY
	rvx := tvx - ovx
	rvy := tvy - ovy
	rv2 := rvx*rvx + rvy*rvy
	if rv2 < 1e-6 {
		return false
	}
	closing := -((rx * rvx) + (ry * rvy))
	if closing <= 0 {
		return false
	}
	tCPA := closing / rv2
	if tCPA < 0 || tCPA > lookaheadSec {
		return false
	}
	cx := rx + rvx*tCPA
	cy := ry + rvy*tCPA
	return math.Hypot(cx, cy) <= missYd
}
