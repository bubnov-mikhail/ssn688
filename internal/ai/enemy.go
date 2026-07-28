package ai

import (
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/world"
)

// UpdateEnemyAI drives hostile unit behavior using the unified acoustic model.
func UpdateEnemyAI(entities []*world.Entity, player *world.Entity, gameTime float64, model acoustics.Model) {
	for _, e := range entities {
		if !e.Alive() || e.Side != world.SideEnemy {
			continue
		}
		switch e.Kind {
		case world.KindSurfaceShip:
			updateSurfaceAI(e, player, gameTime, model)
		case world.KindSubmarine:
			updateSubAI(e, player, gameTime, model)
		}
	}
}

func updateSurfaceAI(ship, player *world.Entity, gameTime float64, model acoustics.Model) {
	rangeYd := ship.RangeYardsTo(player)
	bearing := ship.BearingDegTo(player)
	heardPing := acoustics.HeardPlayerPing(ship, player, gameTime)

	if gameTime-ship.LastPingTime > 8 {
		ship.ActiveSonar = true
		ship.LastPingTime = gameTime
		ship.AIState = "PINGING"
	} else if gameTime-ship.LastPingTime > 1 {
		ship.ActiveSonar = false
	}

	detected := false
	if ship.ActiveSonar {
		detected = model.CanDetectActive(ship, player, 0.75)
	} else {
		detected = model.CanDetectPlayerPassive(ship, player, gameTime)
	}

	if heardPing || detected || rangeYd < 5000 {
		ship.AIState = "INTERCEPT"
		if heardPing && !detected {
			ship.AIState = "PING_ALERT"
		}
		ship.OrderedHead = bearing
		if rangeYd > 4000 {
			ship.OrderedSpeed = 22
			if heardPing {
				ship.OrderedSpeed = 24
			}
		} else {
			ship.OrderedSpeed = 12
		}
		return
	}

	leg := int(gameTime/60) % 4
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
	ship.OrderedSpeed = 14
	ship.AIState = "SEARCH"
}

func updateSubAI(sub, player *world.Entity, gameTime float64, model acoustics.Model) {
	rangeYd := sub.RangeYardsTo(player)
	bearing := sub.BearingDegTo(player)
	heardPing := acoustics.HeardPlayerPing(sub, player, gameTime)

	passiveDetected := model.CanDetectPlayerPassive(sub, player, gameTime)
	active := model.Detect(sub, player, acoustics.ModeActive, 0.6)

	if passiveDetected || active.Detected {
		sub.AIState = "ATTACK"
		sub.OrderedHead = bearing
		sub.OrderedSpeed = 18
		sub.OrderedDepth = 200
		if rangeYd < 3000 && int(gameTime*10)%80 == 0 {
			sub.AIState = "FIRING"
		}
		return
	}

	if heardPing {
		sub.AIState = "EVADE"
		sub.OrderedHead = bearing + 180
		if sub.OrderedHead >= 360 {
			sub.OrderedHead -= 360
		}
		sub.OrderedSpeed = 10
		sub.OrderedDepth = 300
		if rangeYd < 6000 {
			sub.OrderedDepth = 340
		}
		return
	}

	if int(gameTime/90)%3 == 0 && rangeYd > 12000 {
		sub.ActiveSonar = true
		sub.LastPingTime = gameTime
		sub.AIState = "ACTIVE_SEARCH"
	} else {
		sub.ActiveSonar = false
	}

	sub.OrderedSpeed = 6
	sub.OrderedDepth = 160
	leg := int(gameTime/45) % 3
	sub.OrderedHead = float64(leg * 120)
	sub.AIState = "PATROL"
}

// PlayerDetectedByEnemy returns true if any enemy has localized the player.
func PlayerDetectedByEnemy(entities []*world.Entity, player *world.Entity, model acoustics.Model, gameTime float64) bool {
	for _, e := range entities {
		if !e.Alive() || e.Side != world.SideEnemy {
			continue
		}
		if acoustics.HeardPlayerPing(e, player, gameTime) {
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
