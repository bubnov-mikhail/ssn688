package ai

import (
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

func TestRaiseDefconMonotonic(t *testing.T) {
	e := &world.Entity{Defcon: world.DefconAware}
	e.RaiseDefcon(world.DefconHostile)
	if e.Defcon != world.DefconHostile {
		t.Fatalf("expected 2, got %d", e.Defcon)
	}
	e.RaiseDefcon(world.DefconAware)
	if e.Defcon != world.DefconHostile {
		t.Fatalf("defcon must not decrease: %d", e.Defcon)
	}
}

func TestProximityRaisesToHostile(t *testing.T) {
	enemy := &world.Entity{
		ID: "dd", Side: world.SideEnemy, Status: world.StatusActive,
		X: 0, Y: 0, Defcon: world.DefconAware,
	}
	player := &world.Entity{
		ID: "player", Side: world.SidePlayer, Status: world.StatusActive,
		X: 0, Y: 3000,
	}
	UpdateDefcon(DefconContext{
		Entities: []*world.Entity{enemy},
		Player:   player,
		GameTime: 10,
		Dt:       0.1,
	})
	if enemy.Defcon < world.DefconHostile {
		t.Fatalf("expected DEFCON >= 2, got %d", enemy.Defcon)
	}
}

func TestSubmergedTorpedoAlertsSurfaceAndSubs(t *testing.T) {
	dd := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 5000, Y: 0, Defcon: world.DefconAware,
	}
	ss := &world.Entity{
		ID: "ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 5000, Y: 200, Defcon: world.DefconAware,
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 0, DepthFt: 200,
	}
	torp := &weapons.Torpedo{
		ID: "MK48-1", Side: world.SidePlayer, Alive: true, Age: 0.05,
		GyroCourseDeg: 90,
	}
	UpdateDefcon(DefconContext{
		Entities: []*world.Entity{dd, ss},
		Player:   player,
		Torps:    []*weapons.Torpedo{torp},
		GameTime: 5,
		Dt:       0.1,
	})
	if dd.Defcon < world.DefconWeaponsFree {
		t.Fatalf("surface under aimed Mk48 should be weapons free: %d", dd.Defcon)
	}
	if ss.Defcon < world.DefconWeaponsFree {
		t.Fatalf("hostile sub should be weapons free: %d", ss.Defcon)
	}
}

func TestTorpedoThreatRaisesSurfaceDefcon(t *testing.T) {
	dd := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, Defcon: world.DefconHostile,
	}
	player := &world.Entity{
		ID: "player", Side: world.SidePlayer, Status: world.StatusActive,
	}
	fish := &weapons.Torpedo{
		ID: "MK48-9", Side: world.SidePlayer, Alive: true, Age: 45,
		X: 2000, Y: 0, HeadingDeg: 270, SpeedKts: 50,
	}
	UpdateDefcon(DefconContext{
		Entities: []*world.Entity{dd},
		Player:   player,
		Torps:    []*weapons.Torpedo{fish},
		GameTime: 60,
		Dt:       0.1,
	})
	if dd.Defcon < world.DefconWeaponsFree {
		t.Fatalf("surface under torpedo threat should be weapons free: %d", dd.Defcon)
	}
}

func TestNeutralDetonationHeardRaisesHostile(t *testing.T) {
	enemy := &world.Entity{
		ID: "dd", Side: world.SideEnemy, Status: world.StatusActive,
		X: 0, Y: 0, Defcon: world.DefconAware,
	}
	neutral := &world.Entity{ID: "civ", Side: world.SideNeutral, Kind: world.KindSurfaceShip}
	det := &weapons.Detonation{X: 500, Y: 0, DepthFt: 0, Hit: neutral}
	NotifyDefconDetonation([]*world.Entity{enemy}, acoustics.DefaultEnvironment(), det, 20)
	if enemy.Defcon < world.DefconHostile {
		t.Fatalf("expected DEFCON 2 after neutral blast, got %d", enemy.Defcon)
	}
}

func TestRestrictedZoneRaisesWeaponsFree(t *testing.T) {
	enemy := &world.Entity{
		ID: "dd", Side: world.SideEnemy, Status: world.StatusActive, Defcon: 0,
	}
	player := &world.Entity{
		ID: "player", Side: world.SidePlayer, Status: world.StatusActive,
		X: 100, Y: 100,
	}
	UpdateDefcon(DefconContext{
		Entities: []*world.Entity{enemy},
		Player:   player,
		Zones:    []world.RestrictedZone{{ID: "z1", CenterX: 0, CenterY: 0, RadiusYd: 500}},
		GameTime: 1,
		Dt:       0.1,
	})
	if enemy.Defcon < world.DefconWeaponsFree {
		t.Fatalf("restricted zone should raise to 3, got %d", enemy.Defcon)
	}
}

func TestHarpoonUnderwaterLaunchRaisesSurfaceDefcon(t *testing.T) {
	dd := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 4000, Defcon: world.DefconAware,
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 0, DepthFt: 180,
	}
	h := &weapons.HarpoonMissile{
		ID: "HSM-1", Side: world.SidePlayer, Alive: true, Age: 0.05,
		Phase: weapons.HarpoonUnderwater, X: 50, Y: 50, LaunchX: 0, LaunchY: 0,
		HeadingDeg: 0, DestructRangeYd: weapons.HarpoonMaxRangeYd,
	}
	UpdateDefcon(DefconContext{
		Entities: []*world.Entity{dd},
		Player:   player,
		Harpoons: []*weapons.HarpoonMissile{h},
		Model:    acoustics.Model{Env: acoustics.DefaultEnvironment()},
		GameTime: 10,
		Dt:       0.1,
	})
	if dd.Defcon < world.DefconWeaponsFree {
		t.Fatalf("surface should hear Harpoon underwater launch: DEFCON=%d", dd.Defcon)
	}
}

func TestNotifyHarpoonLaunchAcoustic(t *testing.T) {
	dd := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 5000, Defcon: world.DefconAware,
	}
	launcher := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 0, DepthFt: 160,
	}
	NotifyHarpoonLaunchAcoustic([]*world.Entity{dd}, acoustics.DefaultEnvironment(), launcher, 20)
	if dd.Defcon < world.DefconWeaponsFree {
		t.Fatalf("expected weapons free after heard launch, got %d", dd.Defcon)
	}
}
