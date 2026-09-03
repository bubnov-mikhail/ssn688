package simreplay

import (
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func SideLabel(e *world.Entity, playerID string) string {
	if e == nil {
		return ""
	}
	if e.ID == playerID {
		return "PLAYER"
	}
	switch e.Side {
	case world.SideEnemy:
		return "ENEMY"
	case world.SideNeutral:
		return "NEUTRAL"
	default:
		if e.Kind == world.KindSubmarine || e.Kind == world.KindSurfaceShip {
			return "ALLY"
		}
		return "FRIENDLY"
	}
}

func SideLabelFromWorld(s world.Side, playerSide bool) string {
	if playerSide {
		return "PLAYER"
	}
	switch s {
	case world.SideEnemy:
		return "ENEMY"
	case world.SideNeutral:
		return "NEUTRAL"
	default:
		return "ALLY"
	}
}

func StatusLabel(e *world.Entity) string {
	if e == nil {
		return ""
	}
	if !e.Alive() {
		return "SUNK"
	}
	if e.AIState != "" {
		return e.AIState
	}
	return "ACTIVE"
}
