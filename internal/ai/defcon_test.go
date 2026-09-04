package ai

import (
	"math"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
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

func TestExerciseTorpedoLaunchSkipsDefcon(t *testing.T) {
	spr := &world.Entity{
		ID: "spr", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, X: -495, Y: 166,
	}
	hulk := &world.Entity{
		ID: "hulk", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 1105, Y: 166,
	}
	shadow := &world.Entity{
		ID: "shadow", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: -3995, Y: -2234, Defcon: world.DefconHostile,
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: -295, Y: 816, DepthFt: 60,
	}
	fc := weapons.NewFireControl()
	torp := fc.LaunchExerciseShipTube(spr, hulk)
	if torp == nil {
		t.Fatal("exercise launch failed")
	}
	torp.Age = 0.05
	UpdateDefcon(DefconContext{
		Entities: []*world.Entity{shadow, hulk},
		Player:   player,
		Torps:    fc.ActiveTorpedoes,
		GameTime: 180,
		Dt:       0.1,
	})
	if shadow.Defcon >= world.DefconWeaponsFree {
		t.Fatalf("exercise fish must not raise enemy to weapons free: %d", shadow.Defcon)
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
		X: 0, Y: 0, Defcon: world.DefconAware, Kind: world.KindSurfaceShip,
		Damage: world.NewFullHealth(),
	}
	neutral := &world.Entity{ID: "civ", Side: world.SideNeutral, Kind: world.KindSurfaceShip}
	det := &weapons.Detonation{X: 500, Y: 0, DepthFt: 0, Hit: neutral}
	NotifyDefconDetonation([]*world.Entity{enemy}, nil, acoustics.DefaultEnvironment(), det, 20)
	if enemy.Defcon < world.DefconHostile {
		t.Fatalf("expected DEFCON 2 after neutral blast, got %d", enemy.Defcon)
	}
}

func TestBlastDatumSteersEnemyAndAlly(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 0, DepthFt: 200,
	}
	yasen := &world.Entity{
		ID: "rf_yasen", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "yasen_m",
		X: 0, Y: 0, DepthFt: 180, Defcon: world.DefconAware,
		CrewSkill: 90, Damage: world.NewFullHealth(), AIState: "PATROL",
	}
	ally := &world.Entity{
		ID: "ally_spruance", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "spruance",
		X: 8000, Y: 8000, Defcon: world.DefconAware,
		CrewSkill: 60, Damage: world.NewFullHealth(), AIState: "PATROL",
	}
	// Blast ~18 kyd NE of Yasen — old linear model could not hear this.
	det := &weapons.Detonation{
		X: 12000, Y: 12000, DepthFt: 40,
		Hit: &world.Entity{ID: "rf_gorshkov", Side: world.SideEnemy, Kind: world.KindSurfaceShip},
	}
	NotifyDefconDetonation([]*world.Entity{yasen, ally, player}, player, acoustics.DefaultEnvironment(), det, 30)

	if !heardExplosion(acoustics.DefaultEnvironment(), yasen, det.X, det.Y, det.DepthFt) {
		t.Fatal("Yasen should hear 18 kyd blast with new falloff")
	}
	if yasen.Defcon < world.DefconWeaponsFree {
		t.Fatalf("Yasen DEFCON %d after friendly-side hull blast", yasen.Defcon)
	}
	if !yasen.AIProsecuting || !yasen.Track.Valid || yasen.AIState != "DATUM" {
		t.Fatalf("Yasen should prosecute blast datum, state=%s pros=%v track=%v",
			yasen.AIState, yasen.AIProsecuting, yasen.Track.Valid)
	}
	brg := yasen.Track.BearingDegFrom(yasen.X, yasen.Y)
	if math.Abs(shortestRel(brg-45)) > 35 {
		t.Fatalf("Yasen track bearing %.0f should aim near NE blast", brg)
	}
	if ally.Defcon < world.DefconWeaponsFree {
		t.Fatalf("ally DEFCON %d after hostile hit", ally.Defcon)
	}
	if !ally.AIProsecuting || ally.AIState != "DATUM" {
		t.Fatalf("ally should investigate blast, state=%s pros=%v", ally.AIState, ally.AIProsecuting)
	}
	if player.AIProsecuting || player.AIState == "DATUM" {
		t.Fatal("ownship must not be steered by blast AI")
	}
}

func TestHeardExplosionRangeBand(t *testing.T) {
	env := acoustics.DefaultEnvironment()
	listener := &world.Entity{X: 0, Y: 0, DepthFt: 200}
	if !heardExplosion(env, listener, 0, 15000, 50) {
		t.Fatal("expected hear at 15 kyd")
	}
	if heardExplosion(env, listener, 0, 40000, 50) {
		t.Fatal("should not hear beyond blastHearYd")
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
