package ai

import (
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// UpdateEnemyAI drives hostile unit behavior using the unified acoustic model.
func UpdateEnemyAI(entities []*world.Entity, player *world.Entity, gameTime, dt float64, model acoustics.Model, torps []*weapons.Torpedo, evade EvadeContext, bathy *world.Bathymetry, routes []*world.Route) {
	all := trafficUniverse(entities, player)
	for _, e := range entities {
		if !e.Alive() || e.Side != world.SideEnemy {
			continue
		}
		quarry := selectEnemyQuarry(e, entities, player, model, gameTime)
		if quarry == nil && e.AIProsecuting && e.Track.Valid {
			quarry = e.Track.GhostTarget("enemy-datum-"+e.ID, world.SidePlayer)
		}
		switch e.Kind {
		case world.KindSurfaceShip:
			if quarry == nil {
				if e.Defcon < world.DefconAware {
					e.AIProsecuting = false
					e.AILostContactSec = 0
				}
				surfacePatrol(e, routes)
				e.ActiveSonar = false
				applyColregsTraffic(e, all)
				applyShoreAvoidance(e, bathy)
				continue
			}
			updateSurfaceAI(e, quarry, gameTime, dt, model, torps, evade, routes)
			applyColregsTraffic(e, all)
			applyShoreAvoidance(e, bathy)
		case world.KindSubmarine:
			if quarry == nil {
				if e.Defcon < world.DefconAware {
					e.AIProsecuting = false
					e.AILostContactSec = 0
				}
				subPatrol(e, routes)
				e.ActiveSonar = false
				applyShoreAvoidance(e, bathy)
				continue
			}
			updateSubAI(e, quarry, gameTime, dt, model, torps, evade, routes)
			applyShoreAvoidance(e, bathy)
		}
	}
}

func selectEnemyQuarry(hunter *world.Entity, entities []*world.Entity, player *world.Entity, model acoustics.Model, gameTime float64) *world.Entity {
	if hunter == nil {
		return nil
	}
	var best *world.Entity
	bestScore := -1.0
	aswShip := hunter.Kind == world.KindSurfaceShip && weapons.SurfaceHasRastrub(hunter.SignatureID)
	for _, e := range enemyQuarryCandidates(entities, player) {
		rangeYd := hunter.RangeYardsTo(e)
		score := math.Max(0, 25000-rangeYd)
		detected := false
		if hunter.Damage.Operational(world.SysPassiveHull) || hunter.Damage.Operational(world.SysTowed) {
			detected = model.CanDetectPlayerPassive(hunter, e, gameTime)
		}
		if hunter.ActiveSonar && hunter.Damage.Operational(world.SysActive) && model.CanDetectActive(hunter, e, 0.65) {
			detected = true
		}
		if e.Kind == world.KindSurfaceShip {
			if acoustics.EnemyRadarDetectsSurface(hunter, e, gameTime, 0.1, model.Bathy) {
				detected = true
				score += 4000
			}
		}
		if detected {
			score += 12000
		} else if !hunter.AIProsecuting {
			if rangeYd > 14000 {
				continue
			}
			score *= 0.15
		}
		if hunter.Track.Valid {
			d := math.Hypot(e.X-hunter.Track.X, e.Y-hunter.Track.Y)
			if d < 2000 {
				score += 8000
			}
		}
		if aswShip && e.Kind == world.KindSubmarine {
			score += 3500
		}
		if world.IsOwnship(e, player) {
			score += 1500
		}
		if score > bestScore {
			bestScore = score
			best = e
		}
	}
	return best
}

// SelectEnemyQuarry picks the best friendly combatant for a hostile hunter to prosecute.
func SelectEnemyQuarry(hunter *world.Entity, entities []*world.Entity, player *world.Entity, model acoustics.Model, gameTime float64) *world.Entity {
	return selectEnemyQuarry(hunter, entities, player, model, gameTime)
}

func enemyQuarryCandidates(entities []*world.Entity, player *world.Entity) []*world.Entity {
	out := make([]*world.Entity, 0, len(entities)+1)
	if player != nil && world.IsEnemyQuarryTarget(player, player) {
		out = append(out, player)
	}
	for _, e := range entities {
		if e == nil || !world.IsEnemyQuarryTarget(e, player) {
			continue
		}
		if player != nil && e.ID == player.ID {
			continue
		}
		out = append(out, e)
	}
	return out
}

func updateSurfaceAI(ship, player *world.Entity, gameTime, dt float64, model acoustics.Model, torps []*weapons.Torpedo, evade EvadeContext, routes []*world.Route) {
	if tryEvadeTorpedo(ship, torps, evade) {
		markRouteInterrupted(ship)
		return
	}
	ship.EnsureDamage()

	// Practice hulks are passive targets — no ASW prosecution geometry.
	if world.IsExerciseTarget(ship) {
		ship.AIProsecuting = false
		ship.AILostContactSec = 0
		surfacePatrol(ship, routes)
		ship.ActiveSonar = false
		return
	}

	// DEFCON 0 — patrol route only; ignore player.
	if ship.Defcon < world.DefconAware {
		ship.AIProsecuting = false
		ship.AILostContactSec = 0
		surfacePatrol(ship, routes)
		ship.ActiveSonar = false
		return
	}

	rangeYd := ship.RangeYardsTo(player)
	bearing := ship.BearingDegTo(player)
	heardPing := acoustics.HeardPlayerPing(model.Env, ship, player, gameTime)
	// Mast paints only apply when the quarry is ownship (player peri/ESM/COMM).
	radarMast := false
	if evade.Ownship != nil && player == evade.Ownship {
		radarMast = acoustics.EnemyRadarDetectsMast(ship, player, evade.Weather, evade.ESM, evade.COMM, evade.Peri, gameTime, model.Bathy)
	}
	radarSurface := false
	if player.Kind == world.KindSurfaceShip {
		radarSurface = acoustics.EnemyRadarDetectsSurface(ship, player, gameTime, dt, model.Bathy)
	}
	radarCue := radarMast || radarSurface

	canActive := ship.Damage.Operational(world.SysActive)
	pingInterval := 8.0
	pingHold := 1.0
	if ship.CanDefconAttack() {
		// Weapons Free: denser ASW search pings.
		pingInterval = 3.5
		pingHold = 2.5
	}
	if ship.CanDefconPing() && canActive && gameTime-ship.LastPingTime > pingInterval {
		ship.ActiveSonar = true
		ship.LastPingTime = gameTime
		ship.AIState = "PINGING"
	} else if gameTime-ship.LastPingTime > pingHold {
		ship.ActiveSonar = false
	}

	detected := false
	activeHit := false
	if ship.ActiveSonar && canActive {
		activeHit = model.CanDetectActive(ship, player, 0.75)
		detected = activeHit
	} else if ship.Damage.Operational(world.SysPassiveHull) {
		detected = model.CanDetectPlayerPassive(ship, player, gameTime)
	}
	if radarCue {
		detected = true
	}
	snr := PeakSNRForAI(model, ship, player, activeHit || (radarCue && ship.ActiveSonar))
	if radarCue {
		snr = math.Max(snr, 18)
	}
	if heardPing && snr < 12 {
		snr = 12
	}
	UpdateCrewTrack(ship, player, detected || radarCue || heardPing, activeHit || radarCue || heardPing, snr, gameTime, dt)
	if ship.Track.Valid {
		bearing = ship.Track.BearingDegFrom(ship.X, ship.Y)
		rangeYd = ship.Track.RangeYdFrom(ship.X, ship.Y)
	}
	classified := TrackClassified(ship)

	if radarCue && ship.CanDefconManeuver() {
		ship.OrderedHead = bearing
	}

	// DEFCON 1 — may ping / hold contact; no intercept geometry.
	if !ship.CanDefconManeuver() {
		ship.AIProsecuting = false
		ship.AILostContactSec = 0
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

	liveSense := detected || radarCue || heardPing
	solidCue := liveSense || (classified && ship.Track.Valid)
	dwellOK := ship.AIProsecuting || radarCue || heardPing ||
		ship.Track.HoldSec >= 2.0 || ship.Track.ClassConf >= 0.18
	onCooldown := gameTime < ship.AIEngageCooldownUntil

	// --- Sticky prosecute: commit to hunt or cleanly return to patrol ---
	if ship.AIProsecuting {
		if liveSense {
			ship.AILostContactSec = 0
			applySurfaceASWDoctrine(ship, rangeYd, bearing, classified, radarCue, heardPing)
			return
		}
		// Lost contact — hold DATUM until timeout.
		ship.AILostContactSec += dt
		if ship.AILostContactSec >= surfaceDatumHoldSec(ship) {
			breakSurfaceProsecute(ship, gameTime)
			surfacePatrol(ship, routes)
			return
		}
		applySurfaceDatumSearch(ship, bearing)
		return
	}

	// Not prosecuting: enter only on solid, dwelled cue outside cooldown.
	if solidCue && dwellOK && !onCooldown {
		ship.AIProsecuting = true
		ship.AILostContactSec = 0
		applySurfaceASWDoctrine(ship, rangeYd, bearing, classified, radarCue, heardPing)
		return
	}

	// Weapons Free / tasked: close and search toward quarry instead of idling
	// on a missing patrol route (demo allies have no RouteID).
	if ship.CanDefconAttack() && !onCooldown {
		applySurfaceWeaponsFreeSearch(ship, bearing)
		return
	}

	// Cooldown / weak blip: stay on route (no interrupt thrash).
	surfacePatrol(ship, routes)
}

func applySurfaceWeaponsFreeSearch(ship *world.Entity, bearing float64) {
	markRouteInterrupted(ship)
	if ship.AIState != "PINGING" {
		ship.AIState = "CLOSING"
	}
	steerOK := !ship.Damage.Destroyed(world.SysSteering)
	if steerOK {
		ship.OrderedHead = bearing
	}
	ship.OrderedSpeed = math.Min(22, ship.MaxSpeedKts())
}

func surfaceDatumHoldSec(ship *world.Entity) float64 {
	return aiDatumHoldSec(ship)
}

func surfaceEngageCooldownSec(ship *world.Entity) float64 {
	return aiEngageCooldownSec(ship)
}

func aiDatumHoldSec(e *world.Entity) float64 {
	// Green ~45s; veterans ~90s.
	return 45 + 45*e.CrewSkill01()
}

func aiEngageCooldownSec(e *world.Entity) float64 {
	// Green ~60s; veterans ~90s.
	return 60 + 30*e.CrewSkill01()
}

func breakSurfaceProsecute(ship *world.Entity, gameTime float64) {
	breakAIProsecute(ship, gameTime)
}

func breakAIProsecute(e *world.Entity, gameTime float64) {
	e.AIProsecuting = false
	e.AILostContactSec = 0
	e.AIEngageCooldownUntil = gameTime + aiEngageCooldownSec(e)
	e.Track.Valid = false
	e.Track.ClassConf = 0
	e.Track.HoldSec = 0
}

func applySurfaceDatumSearch(ship *world.Entity, bearing float64) {
	markRouteInterrupted(ship)
	ship.AIState = "DATUM"
	steerOK := !ship.Damage.Destroyed(world.SysSteering)
	if steerOK {
		if ship.Track.Valid {
			ship.OrderedHead = ship.Track.BearingDegFrom(ship.X, ship.Y)
		} else {
			ship.OrderedHead = bearing
		}
	}
	ship.OrderedSpeed = math.Min(16, ship.MaxSpeedKts())
}

func applySurfaceASWDoctrine(ship *world.Entity, rangeYd, bearing float64, classified, radarMast, heardPing bool) {
	markRouteInterrupted(ship)
	if radarMast && classified {
		ship.AIState = "RADAR_TRACK"
	} else if heardPing && !classified && rangeYd > weapons.RastrubMaxRangeYd {
		ship.AIState = "PING_ALERT"
	}
	maxSpd := ship.MaxSpeedKts()
	steerOK := !ship.Damage.Destroyed(world.SysSteering)
	hasRastrub := weapons.SurfaceHasRastrub(ship.SignatureID)
	const rastrubIdealYd = 5500.0
	const grishaIdealYd = 1400.0
	weaponOK := TrackWeaponRelease(ship) || radarMast ||
		(ship.CanDefconAttack() && heardPing && ship.Track.Valid && ship.Track.HoldSec >= 2)
	// Track depth drives Grisha RBU vs tubes (rockets only shock shallow subs).
	trackDepth := 0.0
	if ship.Track.Valid {
		trackDepth = ship.Track.DepthFt
	}
	rbuOK := trackDepth > 0 && trackDepth <= weapons.RBUMaxTargetDepthFt

	if !hasRastrub {
		// Grisha: close for RBU / ship tubes — no Metel standoff.
		switch {
		case rangeYd > weapons.RBUMaxRangeYd:
			ship.AIState = "CLOSING"
			if steerOK {
				ship.OrderedHead = bearing
			}
			ship.OrderedSpeed = math.Min(28, maxSpd)
		case rangeYd >= weapons.RBUMinRangeYd && rbuOK:
			if weaponOK {
				ship.AIState = "RBU"
			} else {
				ship.AIState = "TRACKING"
			}
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
			if weaponOK {
				ship.AIState = "SHIP_TUBE"
			} else {
				ship.AIState = "TRACKING"
			}
			if steerOK {
				ship.OrderedHead = bearing
			}
			// Keep below lightweight exit speed so the hull cannot overrun its own fish.
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
		if weaponOK {
			ship.AIState = "RASTRUB"
		} else {
			ship.AIState = "TRACKING"
		}
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
		if weaponOK {
			ship.AIState = "SHIP_TUBE"
		} else {
			ship.AIState = "TRACKING"
		}
		if steerOK {
			ship.OrderedHead = bearing
		}
		ship.OrderedSpeed = math.Min(12, maxSpd)

	default:
		ship.AIState = "INTERCEPT"
		if steerOK {
			ship.OrderedHead = normalizeHead(bearing + 150)
		}
		ship.OrderedSpeed = math.Min(14, maxSpd)
	}
}

func surfacePatrol(ship *world.Entity, routes []*world.Route) {
	if followAssignedRoute(ship, routes, "PATROL", routeCruiseSpeed(ship)) {
		return
	}
	ship.OrderedSpeed = math.Min(14, ship.MaxSpeedKts())
	ship.AIState = "PATROL"
}

func updateSubAI(sub, player *world.Entity, gameTime, dt float64, model acoustics.Model, torps []*weapons.Torpedo, evade EvadeContext, routes []*world.Route) {
	if tryEvadeTorpedo(sub, torps, evade) {
		markRouteInterrupted(sub)
		return
	}
	sub.EnsureDamage()

	if sub.Defcon < world.DefconAware {
		sub.AIProsecuting = false
		sub.AILostContactSec = 0
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
	detected := passiveDetected || active.Detected
	snr := active.PeakSNR
	if !active.Detected {
		snr = PeakSNRForAI(model, sub, player, false)
	}
	UpdateCrewTrack(sub, player, detected, active.Detected, snr, gameTime, dt)
	if sub.Track.Valid {
		bearing = sub.Track.BearingDegFrom(sub.X, sub.Y)
		rangeYd = sub.Track.RangeYdFrom(sub.X, sub.Y)
	}
	classified := TrackClassified(sub)

	if !sub.CanDefconManeuver() {
		sub.AIProsecuting = false
		sub.AILostContactSec = 0
		if sub.CanDefconPing() && sub.Damage.Operational(world.SysActive) && int(gameTime/90)%3 == 0 && rangeYd > 12000 {
			sub.ActiveSonar = true
			sub.LastPingTime = gameTime
			sub.AIState = "ACTIVE_SEARCH"
		} else {
			sub.ActiveSonar = false
		}
		subPatrol(sub, routes)
		return
	}

	liveSense := detected || heardPing
	solidCue := liveSense || (classified && sub.Track.Valid)
	dwellOK := sub.AIProsecuting || heardPing ||
		sub.Track.HoldSec >= 2.0 || sub.Track.ClassConf >= 0.18
	onCooldown := gameTime < sub.AIEngageCooldownUntil

	if sub.AIProsecuting {
		if detected {
			sub.AILostContactSec = 0
			markRouteInterrupted(sub)
			applySubShadowTactics(sub, player, gameTime, rangeYd, bearing)
			return
		}
		if heardPing {
			sub.AILostContactSec = 0
			applySubPingEvade(sub, rangeYd, bearing)
			return
		}
		sub.AILostContactSec += dt
		if sub.AILostContactSec >= aiDatumHoldSec(sub) {
			breakAIProsecute(sub, gameTime)
			subPatrol(sub, routes)
			return
		}
		applySubDatumSearch(sub, bearing)
		return
	}

	if solidCue && dwellOK && !onCooldown {
		sub.AIProsecuting = true
		sub.AILostContactSec = 0
		if detected {
			markRouteInterrupted(sub)
			applySubShadowTactics(sub, player, gameTime, rangeYd, bearing)
			return
		}
		if heardPing {
			applySubPingEvade(sub, rangeYd, bearing)
			return
		}
		applySubDatumSearch(sub, bearing)
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

func applySubPingEvade(sub *world.Entity, rangeYd, bearing float64) {
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
}

func applySubDatumSearch(sub *world.Entity, bearing float64) {
	markRouteInterrupted(sub)
	sub.AIState = "DATUM"
	sub.ActiveSonar = false
	if !sub.Damage.Destroyed(world.SysSteering) {
		if sub.Track.Valid {
			sub.OrderedHead = sub.Track.BearingDegFrom(sub.X, sub.Y)
		} else {
			sub.OrderedHead = bearing
		}
	}
	closeSpd := 6.0
	if sub.SignatureID == "victor_iii" || sub.SignatureID == "yasen_m" {
		closeSpd = 8.0
	}
	sub.OrderedSpeed = math.Min(closeSpd, sub.MaxSpeedKts())
	if !sub.Damage.Destroyed(world.SysDepth) {
		d := 200.0
		if sub.Track.Valid && sub.Track.DepthFt > 80 {
			d = sub.Track.DepthFt
		}
		if d < 140 {
			d = 140
		}
		if d > 380 {
			d = 380
		}
		sub.OrderedDepth = d
	}
}

func subPatrol(sub *world.Entity, routes []*world.Route) {
	spd := routeCruiseSpeed(sub)
	patrolDepth := 160.0
	if sub.SignatureID == "victor_iii" || sub.SignatureID == "yasen_m" {
		patrolDepth = 220
	}
	if followAssignedRoute(sub, routes, "PATROL", spd) {
		if !sub.Damage.Destroyed(world.SysDepth) {
			sub.OrderedDepth = patrolDepth
		}
		return
	}
	sub.OrderedSpeed = math.Min(spd, sub.MaxSpeedKts())
	if !sub.Damage.Destroyed(world.SysDepth) {
		sub.OrderedDepth = patrolDepth
	}
	sub.AIState = "PATROL"
}

// Hostile SSK standoff / shadow geometry (yards, degrees).
const (
	subStandoffMinYd   = 2200.0 // inside → open range (no ramming)
	subStandoffIdealYd = 3200.0
	subStandoffMaxYd   = 5000.0 // outside → close toward flank station
	subFireMinYd       = 1400.0
	subFireMaxYd       = 3600.0
)

// applySubShadowTactics keeps a safe trail, holds a flank station, and only marks
// fire when geometry is favorable — never charges down the player's bearing.
// Geometry uses the crew's Track estimate when available.
func applySubShadowTactics(sub, player *world.Entity, gameTime, rangeYd, bearingToPlayer float64) {
	maxSpd := sub.MaxSpeedKts()
	side := subAttackSide(sub, player, gameTime)

	// Estimated player pose for station-keeping / TMA trail.
	px, py := player.X, player.Y
	pHead := player.HeadingDeg
	pSpd := player.SpeedKts
	pDepth := player.DepthFt
	if sub.Track.Valid {
		px, py = sub.Track.X, sub.Track.Y
		pHead = sub.Track.CourseDeg
		pSpd = sub.Track.SpeedKts
		pDepth = sub.Track.DepthFt
	}

	if !sub.Damage.Destroyed(world.SysDepth) {
		d := pDepth + 40
		if d < 140 {
			d = 140
		}
		if d > 380 {
			d = 380
		}
		// Green depth solutions wander.
		s := sub.CrewSkill01()
		d += (1 - s) * 80 * pseudoNoise(sub.ID, gameTime, 9)
		sub.OrderedDepth = d
	}

	// Desired station: slightly forward of the (estimated) player's beam.
	stationBrgFromPlayer := normalizeHead(pHead + 75*side)
	sx := px + math.Sin(stationBrgFromPlayer*math.Pi/180)*subStandoffIdealYd
	sy := py + math.Cos(stationBrgFromPlayer*math.Pi/180)*subStandoffIdealYd
	brgToStation := bearingDeg(sub.X, sub.Y, sx, sy)

	switch {
	case rangeYd < subStandoffMinYd:
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
		closeSpd := 9.0
		if sub.SignatureID == "victor_iii" || sub.SignatureID == "yasen_m" {
			closeSpd = 12.0
		}
		if sub.SignatureID == "foxtrot" {
			closeSpd = 7.0
		}
		sub.OrderedSpeed = math.Min(closeSpd, maxSpd)
		return
	}

	sub.AIState = "SHADOW"
	actualBrgFromEst := bearingDeg(px, py, sub.X, sub.Y)
	stationErr := math.Abs(shortestRel(stationBrgFromPlayer - actualBrgFromEst))
	if !sub.Damage.Destroyed(world.SysSteering) {
		if stationErr > 30 || math.Abs(rangeYd-subStandoffIdealYd) > 700 {
			sub.OrderedHead = brgToStation
		} else {
			sub.OrderedHead = normalizeHead(pHead + 8*side)
		}
	}
	trail := pSpd + 1.5
	if trail < 5 {
		trail = 5
	}
	if trail > 11 {
		trail = 11
	}
	sub.OrderedSpeed = math.Min(trail, maxSpd)

	// Torpedo opportunity only after classification + favorable geometry.
	rel := math.Abs(shortestRel(bearingToPlayer - sub.HeadingDeg))
	goodGeom := rangeYd >= subFireMinYd && rangeYd <= subFireMaxYd && rel >= 20 && rel <= 160
	if goodGeom && sub.CanDefconAttack() && TrackClassified(sub) {
		sub.AIState = "ATTACK"
		cadence := 90
		if sub.CrewSkill01() >= 0.85 {
			cadence = 40
		}
		if int(gameTime*10)%cadence == 0 {
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
		if acoustics.EnemyRadarDetectsMast(e, player, weather, esm, comm, peri, gameTime, model.Bathy) {
			return true
		}
	}
	return false
}
