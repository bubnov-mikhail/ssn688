package ai

import (
	"math"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

// UpdateEnemyAI drives hostile unit behavior using the unified acoustic model.
func UpdateEnemyAI(entities []*world.Entity, player *world.Entity, gameTime float64, model acoustics.Model, torps []*weapons.Torpedo, evade EvadeContext) {
	for _, e := range entities {
		if !e.Alive() || e.Side != world.SideEnemy {
			continue
		}
		switch e.Kind {
		case world.KindSurfaceShip:
			updateSurfaceAI(e, player, gameTime, model, torps, evade)
		case world.KindSubmarine:
			updateSubAI(e, player, gameTime, model, torps, evade)
		}
	}
}

func updateSurfaceAI(ship, player *world.Entity, gameTime float64, model acoustics.Model, torps []*weapons.Torpedo, evade EvadeContext) {
	if tryEvadeTorpedo(ship, torps, evade) {
		return
	}
	ship.EnsureDamage()

	rangeYd := ship.RangeYardsTo(player)
	bearing := ship.BearingDegTo(player)
	heardPing := acoustics.HeardPlayerPing(model.Env, ship, player, gameTime)

	canActive := ship.Damage.Operational(world.SysActive)
	if canActive && gameTime-ship.LastPingTime > 8 {
		ship.ActiveSonar = true
		ship.LastPingTime = gameTime
		ship.AIState = "PINGING"
	} else if gameTime-ship.LastPingTime > 1 {
		ship.ActiveSonar = false
	}

	detected := false
	if ship.ActiveSonar && canActive {
		detected = model.CanDetectActive(ship, player, 0.75)
	} else if ship.Damage.Operational(world.SysPassiveHull) {
		detected = model.CanDetectPlayerPassive(ship, player, gameTime)
	}

	if heardPing || detected || rangeYd < 5000 {
		ship.AIState = "INTERCEPT"
		if heardPing && !detected {
			ship.AIState = "PING_ALERT"
		}
		if !ship.Damage.Destroyed(world.SysSteering) {
			ship.OrderedHead = bearing
		}
		maxSpd := ship.MaxSpeedKts()
		if rangeYd > 4000 {
			ship.OrderedSpeed = math.Min(22, maxSpd)
			if heardPing {
				ship.OrderedSpeed = math.Min(24, maxSpd)
			}
		} else {
			ship.OrderedSpeed = math.Min(12, maxSpd)
		}
		return
	}

	leg := int(gameTime/60) % 4
	if !ship.Damage.Destroyed(world.SysSteering) {
		switch leg {
		case 0:
			ship.OrderedHead = 45
		case 1:
			ship.OrderedHead = 135
		case 2:
			ship.OrderedHead = 225
		default:
			ship.OrderedHead = 315
		}
	}
	ship.OrderedSpeed = math.Min(14, ship.MaxSpeedKts())
	ship.AIState = "SEARCH"
}

func updateSubAI(sub, player *world.Entity, gameTime float64, model acoustics.Model, torps []*weapons.Torpedo, evade EvadeContext) {
	if tryEvadeTorpedo(sub, torps, evade) {
		return
	}
	sub.EnsureDamage()

	rangeYd := sub.RangeYardsTo(player)
	bearing := sub.BearingDegTo(player)
	heardPing := acoustics.HeardPlayerPing(model.Env, sub, player, gameTime)

	passiveDetected := false
	if sub.Damage.Operational(world.SysPassiveHull) || sub.Damage.Operational(world.SysTowed) {
		passiveDetected = model.CanDetectPlayerPassive(sub, player, gameTime)
	}
	active := acoustics.DetectionResult{}
	if sub.Damage.Operational(world.SysActive) {
		active = model.Detect(sub, player, acoustics.ModeActive, 0.6)
	}

	if passiveDetected || active.Detected {
		applySubShadowTactics(sub, player, gameTime, rangeYd, bearing)
		return
	}

	if heardPing {
		sub.AIState = "EVADE"
		if !sub.Damage.Destroyed(world.SysSteering) {
			sub.OrderedHead = normalizeHead(bearing + 180)
		}
		sub.OrderedSpeed = math.Min(10, sub.MaxSpeedKts())
		if !sub.Damage.Destroyed(world.SysDepth) {
			sub.OrderedDepth = 300
			if rangeYd < 6000 {
				sub.OrderedDepth = 340
			}
		}
		return
	}

	if sub.Damage.Operational(world.SysActive) && int(gameTime/90)%3 == 0 && rangeYd > 12000 {
		sub.ActiveSonar = true
		sub.LastPingTime = gameTime
		sub.AIState = "ACTIVE_SEARCH"
	} else {
		sub.ActiveSonar = false
	}

	sub.OrderedSpeed = math.Min(6, sub.MaxSpeedKts())
	if !sub.Damage.Destroyed(world.SysDepth) {
		sub.OrderedDepth = 160
	}
	if !sub.Damage.Destroyed(world.SysSteering) {
		leg := int(gameTime/45) % 3
		sub.OrderedHead = float64(leg * 120)
	}
	sub.AIState = "PATROL"
}

// Hostile SSK standoff / shadow geometry (yards, degrees).
const (
	subStandoffMinYd   = 2200.0 // inside → open range (no ramming)
	subStandoffIdealYd = 3200.0
	subStandoffMaxYd   = 5000.0 // outside → close toward flank station
	subFireMinYd       = 1600.0
	subFireMaxYd       = 3400.0
)

// applySubShadowTactics keeps a safe trail, holds a flank station, and only marks
// fire when geometry is favorable — never charges down the player's bearing.
func applySubShadowTactics(sub, player *world.Entity, gameTime, rangeYd, bearingToPlayer float64) {
	maxSpd := sub.MaxSpeedKts()
	side := subAttackSide(sub, player, gameTime)

	if !sub.Damage.Destroyed(world.SysDepth) {
		d := player.DepthFt + 40
		if d < 140 {
			d = 140
		}
		if d > 380 {
			d = 380
		}
		sub.OrderedDepth = d
	}

	// Desired station: slightly forward of the player's beam on alternating flanks.
	stationBrgFromPlayer := normalizeHead(player.HeadingDeg + 75*side)
	sx := player.X + math.Sin(stationBrgFromPlayer*math.Pi/180)*subStandoffIdealYd
	sy := player.Y + math.Cos(stationBrgFromPlayer*math.Pi/180)*subStandoffIdealYd
	brgToStation := bearingDeg(sub.X, sub.Y, sx, sy)

	switch {
	case rangeYd < subStandoffMinYd:
		// Too close — peel away onto the far flank, do not aim at the target.
		sub.AIState = "OPENING"
		if !sub.Damage.Destroyed(world.SysSteering) {
			sub.OrderedHead = normalizeHead(bearingToPlayer + 180 + 40*side)
		}
		sub.OrderedSpeed = math.Min(14, maxSpd)
		return

	case rangeYd > subStandoffMaxYd:
		sub.AIState = "CLOSING"
		if !sub.Damage.Destroyed(world.SysSteering) {
			sub.OrderedHead = brgToStation
		}
		// Quiet approach — don't sprint in.
		sub.OrderedSpeed = math.Min(9, maxSpd)
		return
	}

	// In the standoff band: hold/trail the station.
	sub.AIState = "SHADOW"
	actualBrgFromPlayer := player.BearingDegTo(sub)
	stationErr := math.Abs(shortestRel(stationBrgFromPlayer - actualBrgFromPlayer))
	if !sub.Damage.Destroyed(world.SysSteering) {
		if stationErr > 30 || math.Abs(rangeYd-subStandoffIdealYd) > 700 {
			sub.OrderedHead = brgToStation
		} else {
			// Parallel trail — match ownship course with a slight lead.
			sub.OrderedHead = normalizeHead(player.HeadingDeg + 8*side)
		}
	}
	trail := player.SpeedKts + 1.5
	if trail < 5 {
		trail = 5
	}
	if trail > 11 {
		trail = 11
	}
	sub.OrderedSpeed = math.Min(trail, maxSpd)

	// Torpedo opportunity: standoff fire envelope + not pointed at a collision course.
	rel := math.Abs(sub.RelativeBearingDeg(player))
	goodGeom := rangeYd >= subFireMinYd && rangeYd <= subFireMaxYd && rel >= 20 && rel <= 160
	if goodGeom {
		sub.AIState = "ATTACK"
		if int(gameTime*10)%90 == 0 {
			sub.AIState = "FIRING"
		}
	}
}

func subAttackSide(sub, player *world.Entity, gameTime float64) float64 {
	// Prefer the side we already occupy; otherwise flip on a slow cadence.
	brg := player.BearingDegTo(sub)
	rel := shortestRel(brg - player.HeadingDeg) // +stbd / −port of target
	if math.Abs(rel) > 20 {
		if rel > 0 {
			return 1
		}
		return -1
	}
	if int(gameTime/100)%2 == 0 {
		return 1
	}
	return -1
}

// PlayerDetectedByEnemy returns true if any enemy has localized the player.
func PlayerDetectedByEnemy(entities []*world.Entity, player *world.Entity, model acoustics.Model, gameTime float64) bool {
	for _, e := range entities {
		if !e.Alive() || e.Side != world.SideEnemy {
			continue
		}
		if acoustics.HeardPlayerPing(model.Env, e, player, gameTime) {
			return true
		}
		if e.ActiveSonar && model.CanDetectActive(e, player, 0.7) {
			return true
		}
		if model.CanDetectPlayerPassive(e, player, gameTime) {
			return true
		}
	}
	return false
}
