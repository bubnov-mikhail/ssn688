package ai

import (
	"math"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

// UpdateEnemyAI drives hostile unit behavior using the unified acoustic model.
func UpdateEnemyAI(entities []*world.Entity, player *world.Entity, gameTime float64, model acoustics.Model, torps []*weapons.Torpedo, evade EvadeContext, bathy *world.Bathymetry, routes []*world.Route) {
	all := trafficUniverse(entities, player)
	for _, e := range entities {
		if !e.Alive() || e.Side != world.SideEnemy {
			continue
		}
		switch e.Kind {
		case world.KindSurfaceShip:
			updateSurfaceAI(e, player, gameTime, model, torps, evade, routes)
			applyColregsTraffic(e, all)
			applyShoreAvoidance(e, bathy)
		case world.KindSubmarine:
			updateSubAI(e, player, gameTime, model, torps, evade, routes)
		}
	}
}

func updateSurfaceAI(ship, player *world.Entity, gameTime float64, model acoustics.Model, torps []*weapons.Torpedo, evade EvadeContext, routes []*world.Route) {
	if tryEvadeTorpedo(ship, torps, evade) {
		markRouteInterrupted(ship)
		return
	}
	ship.EnsureDamage()

	// DEFCON 0 — patrol route only; ignore player.
	if ship.Defcon < world.DefconAware {
		surfacePatrol(ship, routes)
		ship.ActiveSonar = false
		return
	}

	rangeYd := ship.RangeYardsTo(player)
	bearing := ship.BearingDegTo(player)
	heardPing := acoustics.HeardPlayerPing(model.Env, ship, player, gameTime)
	radarMast := acoustics.EnemyRadarDetectsMast(ship, player, evade.Weather, evade.ESM, evade.COMM, evade.Peri, gameTime)

	canActive := ship.Damage.Operational(world.SysActive)
	if ship.CanDefconPing() && canActive && gameTime-ship.LastPingTime > 8 {
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
	if radarMast {
		detected = true
		// Solid radar paint: accurate bearing + range for intercept geometry.
		if ship.CanDefconManeuver() {
			ship.OrderedHead = bearing
		}
	}

	// DEFCON 1 — may ping / hold contact; no intercept geometry.
	if !ship.CanDefconManeuver() {
		if heardPing && !detected {
			markRouteInterrupted(ship)
			ship.AIState = "PING_ALERT"
		} else if detected {
			markRouteInterrupted(ship)
			ship.AIState = "TRACKING"
		} else {
			surfacePatrol(ship, routes)
		}
		return
	}

	// ASW surface doctrine by class: Rastrub standoff (Udaloy/Kresta) or close RBU/tubes (Grisha).
	hasRastrub := weapons.SurfaceHasRastrub(ship.SignatureID)
	const rastrubIdealYd = 5500.0
	const grishaIdealYd = 1400.0
	engageHorizon := weapons.RastrubMaxRangeYd
	if !hasRastrub {
		engageHorizon = weapons.RBUMaxRangeYd + 800
	}
	if heardPing || detected || radarMast || rangeYd < engageHorizon {
		markRouteInterrupted(ship)
		if radarMast && detected {
			ship.AIState = "RADAR_TRACK"
		} else if heardPing && !detected {
			ship.AIState = "PING_ALERT"
		}
		maxSpd := ship.MaxSpeedKts()
		steerOK := !ship.Damage.Destroyed(world.SysSteering)

		if !hasRastrub {
			// Grisha: close for RBU / ship tubes — no Metel standoff.
			switch {
			case rangeYd > weapons.RBUMaxRangeYd:
				ship.AIState = "CLOSING"
				if steerOK {
					ship.OrderedHead = bearing
				}
				ship.OrderedSpeed = math.Min(28, maxSpd)
			case rangeYd >= weapons.RBUMinRangeYd:
				ship.AIState = "RBU"
				if steerOK {
					switch {
					case rangeYd < grishaIdealYd-400:
						ship.OrderedHead = normalizeHead(bearing + 180)
					case rangeYd > grishaIdealYd+400:
						ship.OrderedHead = bearing
					default:
						ship.OrderedHead = normalizeHead(bearing + 70)
					}
				}
				ship.OrderedSpeed = math.Min(18, maxSpd)
			case rangeYd >= weapons.ShipTubeMinRangeYd:
				ship.AIState = "SHIP_TUBE"
				if steerOK {
					ship.OrderedHead = bearing
				}
				ship.OrderedSpeed = math.Min(14, maxSpd)
			default:
				ship.AIState = "INTERCEPT"
				if steerOK {
					ship.OrderedHead = normalizeHead(bearing + 150)
				}
				ship.OrderedSpeed = math.Min(16, maxSpd)
			}
			return
		}

		switch {
		case rangeYd > weapons.RastrubMaxRangeYd:
			ship.AIState = "CLOSING"
			if steerOK {
				ship.OrderedHead = bearing
			}
			ship.OrderedSpeed = math.Min(24, maxSpd)

		case rangeYd >= weapons.RastrubMinRangeYd:
			ship.AIState = "RASTRUB"
			if steerOK {
				switch {
				case rangeYd < rastrubIdealYd-900:
					ship.OrderedHead = normalizeHead(bearing + 180)
				case rangeYd > rastrubIdealYd+900:
					ship.OrderedHead = bearing
				default:
					ship.OrderedHead = normalizeHead(bearing + 85)
				}
			}
			ship.OrderedSpeed = math.Min(16, maxSpd)
			if heardPing {
				ship.OrderedSpeed = math.Min(18, maxSpd)
			}

		case rangeYd >= weapons.ShipTubeMinRangeYd:
			ship.AIState = "SHIP_TUBE"
			if steerOK {
				ship.OrderedHead = bearing
			}
			ship.OrderedSpeed = math.Min(12, maxSpd)

		default:
			// Inside ship-tube min — open a little while keeping sonar contact.
			ship.AIState = "INTERCEPT"
			if steerOK {
				ship.OrderedHead = normalizeHead(bearing + 150)
			}
			ship.OrderedSpeed = math.Min(14, maxSpd)
		}
		return
	}

	surfacePatrol(ship, routes)
}

func surfacePatrol(ship *world.Entity, routes []*world.Route) {
	if followAssignedRoute(ship, routes, "PATROL", routeCruiseSpeed(ship)) {
		return
	}
	ship.OrderedSpeed = math.Min(14, ship.MaxSpeedKts())
	ship.AIState = "PATROL"
}

func updateSubAI(sub, player *world.Entity, gameTime float64, model acoustics.Model, torps []*weapons.Torpedo, evade EvadeContext, routes []*world.Route) {
	if tryEvadeTorpedo(sub, torps, evade) {
		markRouteInterrupted(sub)
		return
	}
	sub.EnsureDamage()

	if sub.Defcon < world.DefconAware {
		subPatrol(sub, routes)
		sub.ActiveSonar = false
		return
	}

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

	if sub.CanDefconManeuver() && (passiveDetected || active.Detected) {
		markRouteInterrupted(sub)
		applySubShadowTactics(sub, player, gameTime, rangeYd, bearing)
		return
	}

	if sub.CanDefconManeuver() && heardPing {
		markRouteInterrupted(sub)
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

	if sub.CanDefconPing() && sub.Damage.Operational(world.SysActive) && int(gameTime/90)%3 == 0 && rangeYd > 12000 {
		sub.ActiveSonar = true
		sub.LastPingTime = gameTime
		sub.AIState = "ACTIVE_SEARCH"
	} else {
		sub.ActiveSonar = false
	}

	subPatrol(sub, routes)
}

func subPatrol(sub *world.Entity, routes []*world.Route) {
	spd := routeCruiseSpeed(sub)
	if followAssignedRoute(sub, routes, "PATROL", spd) {
		if !sub.Damage.Destroyed(world.SysDepth) {
			sub.OrderedDepth = 160
			if sub.SignatureID == "victor_iii" {
				sub.OrderedDepth = 220
			}
		}
		return
	}
	sub.OrderedSpeed = math.Min(spd, sub.MaxSpeedKts())
	if !sub.Damage.Destroyed(world.SysDepth) {
		sub.OrderedDepth = 160
		if sub.SignatureID == "victor_iii" {
			sub.OrderedDepth = 220
		}
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
	// Quiet approach — don't sprint in (Victor may close a bit hotter).
		closeSpd := 9.0
		if sub.SignatureID == "victor_iii" {
			closeSpd = 12.0
		}
		if sub.SignatureID == "foxtrot" {
			closeSpd = 7.0
		}
		sub.OrderedSpeed = math.Min(closeSpd, maxSpd)
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
	if goodGeom && sub.CanDefconAttack() {
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
func PlayerDetectedByEnemy(entities []*world.Entity, player *world.Entity, model acoustics.Model, gameTime float64, weather world.Weather, esm *acoustics.ESMState, comm *acoustics.COMMState, peri *acoustics.PeriscopeState) bool {
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
		if acoustics.EnemyRadarDetectsMast(e, player, weather, esm, comm, peri, gameTime) {
			return true
		}
	}
	return false
}
