package world

import "testing"

func TestSideHelpers(t *testing.T) {
	player := &Entity{ID: "player", Side: SidePlayer, Kind: KindSubmarine}
	ally := &Entity{ID: "ally", Side: SidePlayer, Kind: KindSurfaceShip}
	foe := &Entity{ID: "foe", Side: SideEnemy, Kind: KindSubmarine}
	civ := &Entity{ID: "civ", Side: SideNeutral, Kind: KindSurfaceShip}

	if !IsOwnship(player, player) || IsAllyAI(player, player) {
		t.Fatal("ownship identity")
	}
	if !IsAllyAI(ally, player) || !IsFriendly(ally) {
		t.Fatal("ally identity")
	}
	if !IsHostile(foe) || IsFriendly(foe) {
		t.Fatal("hostile identity")
	}
	if IsHostile(civ) || IsAllyAI(civ, player) {
		t.Fatal("neutral identity")
	}
	if HostileTorpedoSide(ally) != SideEnemy || HostileTorpedoSide(foe) != SidePlayer {
		t.Fatal("hostile torpedo side")
	}
	if !IsEnemyQuarryTarget(player, player) || !IsEnemyQuarryTarget(ally, player) {
		t.Fatal("enemy quarry targets")
	}
	if IsEnemyQuarryTarget(foe, player) || IsEnemyQuarryTarget(civ, player) {
		t.Fatal("enemy quarry excludes hostile/neutral")
	}
}
