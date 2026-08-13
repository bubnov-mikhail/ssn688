package ai

import (
	"math"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

// UpdateFriendlyAI drives SidePlayer allies (not ownship) against hostile units.
func UpdateFriendlyAI(entities []*world.Entity, player *world.Entity, gameTime, dt float64, model acoustics.Model, torps []*weapons.Torpedo, evade EvadeContext, bathy *world.Bathymetry, routes []*world.Route) {
	all := trafficUniverse(entities, player)
	for _, e := range entities {
		if !world.IsAllyAI(e, player) || !e.Alive() {
			continue
		}
		quarry := selectHostileQuarry(e, entities, model, gameTime)
		if quarry == nil && e.AIProsecuting && e.Track.Valid {
			quarry = e.Track.GhostTarget("ally-datum-"+e.ID, world.SideEnemy)
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
				continue
			}
			updateSubAI(e, quarry, gameTime, dt, model, torps, evade, routes)
		}
	}
}

// UpdateFriendlyDefcon escalates ally alert from hostile contacts / fish.
func UpdateFriendlyDefcon(entities []*world.Entity, player *world.Entity, model acoustics.Model, torps []*weapons.Torpedo, gameTime float64) {
	for _, ally := range entities {
		if !world.IsAllyAI(ally, player) || !ally.Alive() {
			continue
		}
		if mostThreateningTorpedo(ally, torps) != nil {
			ally.RaiseDefcon(world.DefconWeaponsFree)
		}
		for _, foe := range entities {
			if !world.IsHostile(foe) || !foe.Alive() {
				continue
			}
			if ally.RangeYardsTo(foe) < world.DefconTorpedoRangeYd {
				ally.RaiseDefcon(world.DefconHostile)
			}
			if acoustics.HeardPlayerPing(model.Env, ally, foe, gameTime) {
				ally.RaiseDefcon(world.DefconHostile)
			}
			detected := false
			if ally.Damage.Operational(world.SysPassiveHull) || ally.Damage.Operational(world.SysTowed) {
				detected = model.CanDetectPlayerPassive(ally, foe, gameTime)
			}
			if ally.ActiveSonar && ally.Damage.Operational(world.SysActive) {
				if model.CanDetectActive(ally, foe, 0.7) {
					detected = true
				}
			}
			if detected {
				ally.RaiseDefcon(world.DefconAware)
				if ally.RangeYardsTo(foe) < world.DefconTorpedoRangeYd*0.7 {
					ally.RaiseDefcon(world.DefconHostile)
				}
			}
		}
	}
}

func selectHostileQuarry(hunter *world.Entity, entities []*world.Entity, model acoustics.Model, gameTime float64) *world.Entity {
	if hunter == nil {
		return nil
	}
	var best *world.Entity
	bestScore := -1.0
	aswShip := hunter.Kind == world.KindSurfaceShip && weapons.SurfaceHasRastrub(hunter.SignatureID)
	for _, e := range entities {
		if e == nil || !e.Alive() || !world.IsHostile(e) {
			continue
		}
		if e.Kind != world.KindSubmarine && e.Kind != world.KindSurfaceShip {
			continue
		}
		rangeYd := hunter.RangeYardsTo(e)
		score := math.Max(0, 25000-rangeYd)
		detected := false
		if hunter.Damage.Operational(world.SysPassiveHull) || hunter.Damage.Operational(world.SysTowed) {
			detected = model.CanDetectPlayerPassive(hunter, e, gameTime)
		}
		if hunter.ActiveSonar && hunter.Damage.Operational(world.SysActive) && model.CanDetectActive(hunter, e, 0.65) {
			detected = true
		}
		if detected {
			score += 12000
		} else if !hunter.AIProsecuting {
			// Outside prosecute, ignore undetection beyond a wide horizon.
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
		// ASROC/Metel DDGs prefer submerged targets when scores are close.
		if aswShip && e.Kind == world.KindSubmarine {
			score += 3500
		}
		if score > bestScore {
			bestScore = score
			best = e
		}
	}
	return best
}
