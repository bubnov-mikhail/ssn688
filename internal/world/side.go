package world

// IsOwnship reports the human-controlled Scenario.Player entity.
func IsOwnship(e, player *Entity) bool {
	return e != nil && player != nil && e.ID == player.ID
}

// IsFriendly is true for the player's side (ownship and AI allies).
func IsFriendly(e *Entity) bool {
	return e != nil && e.Side == SidePlayer
}

// IsAllyAI is a friendly combatant that is not the human ownship.
func IsAllyAI(e, player *Entity) bool {
	if e == nil || !IsFriendly(e) || IsOwnship(e, player) {
		return false
	}
	return e.Kind == KindSubmarine || e.Kind == KindSurfaceShip
}

// IsHostile is true for opposing combatants.
func IsHostile(e *Entity) bool {
	return e != nil && e.Side == SideEnemy
}

// IsEnemyQuarryTarget is a friendly platform enemy AI may prosecute (ownship or ally).
func IsEnemyQuarryTarget(e, player *Entity) bool {
	if e == nil || !e.Alive() || !IsFriendly(e) {
		return false
	}
	if e.Kind != KindSubmarine && e.Kind != KindSurfaceShip {
		return false
	}
	return IsOwnship(e, player) || IsAllyAI(e, player)
}

// HostileTorpedoSide is the side whose fish threaten this platform.
func HostileTorpedoSide(e *Entity) Side {
	if e != nil && e.Side == SidePlayer {
		return SideEnemy
	}
	return SidePlayer
}
