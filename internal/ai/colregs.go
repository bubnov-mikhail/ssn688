package ai

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

const (
	colregsDetectYd   = 2800.0
	colregsUrgentYd   = 900.0
	colregsHeadOnDeg  = 15.0
	colregsOvertakeDeg = 112.5 // abaft the beam
)

// applyColregsTraffic applies simplified COLREGS give-way rules for surface units
// that are following a route (PATROL/CRUISE). Temporary AVOID interrupts the route.
func applyColregsTraffic(ship *world.Entity, all []*world.Entity) {
	if ship == nil || !ship.Alive() || ship.Kind != world.KindSurfaceShip {
		return
	}
	switch ship.AIState {
	case "PATROL", "CRUISE", "AVOID", "TRANSIT", "SEARCH":
	default:
		// Combat / torpedo evade / shore avoid own the helm.
		return
	}
	head, speed, avoid := colregsManeuver(ship, all)
	if !avoid {
		return
	}
	if ship.AIState != "AVOID" {
		markRouteInterrupted(ship)
	}
	ship.OrderedHead = head
	if speed > 0 {
		ship.OrderedSpeed = speed
	}
	ship.AIState = "AVOID"
}

func colregsManeuver(self *world.Entity, all []*world.Entity) (heading, speed float64, avoid bool) {
	heading = self.OrderedHead
	speed = self.OrderedSpeed
	type threat struct {
		e    *world.Entity
		dist float64
		kind string // headon | giveway | overtake | emergency
	}
	var best *threat
	for _, o := range all {
		if o == nil || o.ID == self.ID || !o.Alive() {
			continue
		}
		if o.Kind == world.KindSubmarine && o.DepthFt > 40 {
			continue
		}
		if o.Kind != world.KindSurfaceShip && !(o.Kind == world.KindSubmarine && o.DepthFt <= 40) {
			continue
		}
		r := self.RangeYardsTo(o)
		if r > colregsDetectYd || r < 1 {
			continue
		}
		// Relative bearing of other from self (−180..180, +stbd).
		relOther := normalizeRel(self.BearingDegTo(o) - self.HeadingDeg)
		// Relative bearing of self from other.
		relSelf := normalizeRel(o.BearingDegTo(self) - o.HeadingDeg)

		closing := world.CollisionThreat2D(
			self.X, self.Y, self.HeadingDeg, math.Max(self.SpeedKts, 1),
			o.X, o.Y, o.HeadingDeg, math.Max(o.SpeedKts, 1),
			20*60, 400,
		)
		urgent := r < colregsUrgentYd || (closing && r < colregsDetectYd*0.55)

		kind := ""
		switch {
		case math.Abs(relOther) <= colregsHeadOnDeg && math.Abs(relSelf) <= colregsHeadOnDeg:
			// Head-on / nearly reciprocal: both alter to starboard (Rule 14).
			kind = "headon"
		case math.Abs(relSelf) > colregsOvertakeDeg:
			// We are abaft the other's beam → overtaking vessel keeps clear (Rule 13).
			kind = "overtake"
		case relOther > 0 && relOther < colregsOvertakeDeg && (closing || r < 1800):
			// Crossing: other on our starboard → we are give-way (Rule 15/16).
			kind = "giveway"
		case urgent && math.Abs(relOther) < 90:
			kind = "emergency"
		default:
			// Stand-on or no risk — do nothing for this contact.
			continue
		}
		if best == nil || r < best.dist {
			best = &threat{e: o, dist: r, kind: kind}
		}
	}
	if best == nil {
		return heading, speed, false
	}

	// Starboard bias for give-way / head-on; overtaking may pass either side —
	// prefer starboard unless the target already bears to port.
	rel := normalizeRel(self.BearingDegTo(best.e) - self.HeadingDeg)
	turn := 35.0
	switch best.kind {
	case "headon":
		turn = 40
	case "giveway":
		turn = 45
	case "overtake":
		if rel >= 0 {
			turn = 30 // already to stbd — open further stbd
		} else {
			turn = -30 // target to port — alter to port to clear
		}
	case "emergency":
		if rel >= 0 {
			turn = -55 // hard away
		} else {
			turn = 55
		}
	}
	if best.kind == "headon" || best.kind == "giveway" {
		turn = math.Abs(turn) // always starboard
	}
	heading = normalizeHead(self.HeadingDeg + turn)
	speed = math.Max(4, self.OrderedSpeed*0.75)
	if best.kind == "emergency" {
		speed = math.Max(4, self.OrderedSpeed*0.55)
	}
	return heading, speed, true
}
