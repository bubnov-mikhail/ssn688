package world

// Enemy DEFCON alert levels (0 = lowest). Levels only increase in play.
const (
	DefconPassive     = 0 // ignores player; no active sonar
	DefconAware       = 1 // knows player may be present; may ping; no intercept / weapons
	DefconHostile     = 2 // intercept / attack geometry; no weapons
	DefconWeaponsFree = 3 // weapons free
)

// DefconTorpedoRangeYd is the player-to-enemy range that counts as "inside torpedo run"
// for DEFCON escalation (approx. tactical Mk48 envelope).
const DefconTorpedoRangeYd = 10000.0

// RaiseDefcon sets alert level if higher than current (monotonic).
func (e *Entity) RaiseDefcon(level int) {
	if e == nil || level < 0 || level > DefconWeaponsFree {
		return
	}
	if level > e.Defcon {
		e.Defcon = level
	}
}

// CanDefconPing is true when active sonar employment is allowed.
func (e *Entity) CanDefconPing() bool {
	return e != nil && e.Defcon >= DefconAware
}

// CanDefconManeuver is true when homing / intercept geometry is allowed.
func (e *Entity) CanDefconManeuver() bool {
	return e != nil && e.Defcon >= DefconHostile
}

// CanDefconAttack is true when torpedoes / Rastrub may be fired.
func (e *Entity) CanDefconAttack() bool {
	return e != nil && e.Defcon >= DefconWeaponsFree
}

// IsCombatant reports military subs / surface combatants (not neutrals).
func IsCombatant(e *Entity) bool {
	if e == nil {
		return false
	}
	if e.Side != SideEnemy {
		return false
	}
	return e.Kind == KindSubmarine || e.Kind == KindSurfaceShip
}
