package ai

import (
	"math"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

// UpdateAllAI drives enemy combatants and neutral shipping.
func UpdateAllAI(entities []*world.Entity, player *world.Entity, gameTime float64, model acoustics.Model, torps []*weapons.Torpedo) {
	UpdateEnemyAI(entities, player, gameTime, model, torps)
	UpdateCivilianAI(entities, player, gameTime)
}

// UpdateCivilianAI steers neutrals on random transit legs with simple collision avoidance.
func UpdateCivilianAI(entities []*world.Entity, player *world.Entity, gameTime float64) {
	all := make([]*world.Entity, 0, len(entities)+1)
	all = append(all, player)
	all = append(all, entities...)

	for _, e := range entities {
		if !e.Alive() || e.Side != world.SideNeutral {
			continue
		}
		updateCivilian(e, all, gameTime)
	}
}

func updateCivilian(ship *world.Entity, all []*world.Entity, gameTime float64) {
	ship.OrderedDepth = 0
	ship.DepthFt = 0

	// Periodic random course change (~every 70–110 s, staggered by ID hash).
	period := 70.0 + float64(hashID(ship.ID)%40)
	phase := float64(hashID(ship.ID)%17) * 3.7
	leg := int((gameTime + phase) / period)
	baseHead := float64((leg*97 + hashID(ship.ID)*13) % 360)
	ship.OrderedHead = baseHead
	ship.OrderedSpeed = cruiseSpeed(ship)
	ship.AIState = "TRANSIT"

	// Avoid nearby vessels / ownship.
	avoidHead, avoid := collisionAvoidance(ship, all)
	if avoid {
		ship.OrderedHead = avoidHead
		ship.OrderedSpeed = math.Max(4, ship.OrderedSpeed*0.7)
		ship.AIState = "AVOID"
	}
}

func cruiseSpeed(e *world.Entity) float64 {
	switch e.SignatureID {
	case "tanker":
		return 9
	case "fishing":
		return 7
	default:
		return 11
	}
}

func collisionAvoidance(self *world.Entity, all []*world.Entity) (heading float64, avoid bool) {
	const dangerYd = 900.0
	bestThreat := 0.0
	var threat *world.Entity
	for _, o := range all {
		if o == nil || o.ID == self.ID || !o.Alive() {
			continue
		}
		// Only care about surface contacts / ownship for CPA-style avoid.
		if o.Kind == world.KindSubmarine && o.DepthFt > 40 {
			continue
		}
		r := self.RangeYardsTo(o)
		if r > dangerYd || r < 1 {
			continue
		}
		// Relative bearing ahead of own bow?
		rel := normalizeRel(self.BearingDegTo(o) - self.HeadingDeg)
		if math.Abs(rel) > 75 {
			continue
		}
		score := (dangerYd - r) * (1 - math.Abs(rel)/90)
		if score > bestThreat {
			bestThreat = score
			threat = o
		}
	}
	if threat == nil {
		return self.OrderedHead, false
	}
	// Turn away from threat.
	rel := normalizeRel(self.BearingDegTo(threat) - self.HeadingDeg)
	turn := 40.0
	if rel >= 0 {
		turn = -50
	} else {
		turn = 50
	}
	h := self.HeadingDeg + turn
	for h < 0 {
		h += 360
	}
	for h >= 360 {
		h -= 360
	}
	return h, true
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

func hashID(id string) int {
	h := 0
	for _, c := range id {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}
