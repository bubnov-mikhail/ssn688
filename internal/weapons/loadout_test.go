package weapons

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestLoadoutByClass(t *testing.T) {
	if SurfaceHasRastrub("grisha") {
		t.Fatal("Grisha should not have Rastrub")
	}
	if !SurfaceHasRBU("grisha") {
		t.Fatal("Grisha should have RBU")
	}
	if !SurfaceHasRastrub("kresta2") || SurfaceHasRBU("kresta2") {
		t.Fatal("Kresta II: Rastrub yes, RBU no")
	}
	if EnemySubMagazineFor("victor_iii") <= EnemySubMagazineFor("foxtrot") {
		t.Fatal("Victor magazine should exceed Foxtrot")
	}
	if HostileTorpedoCruiseKts("victor_iii") <= HostileTorpedoCruiseKts("foxtrot") {
		t.Fatal("Victor fish should be faster")
	}
	if EnemySubMagazineFor("yasen_m") <= EnemySubMagazineFor("victor_iii") {
		t.Fatal("Yasen-M magazine should exceed Victor")
	}
	if !SurfaceHasRastrub("gorshkov") || SurfaceHasRBU("gorshkov") {
		t.Fatal("Gorshkov: Otvet yes, RBU no")
	}
	if SurfaceASWRocketLabel("gorshkov") != "Otvet" {
		t.Fatal("Gorshkov rocket label should be Otvet")
	}
	if ShipTubeMagazineFor("gorshkov") != 8 || SAMMagazineFor("gorshkov") < 20 {
		t.Fatal("Gorshkov Paket/Redut magazines")
	}
	if LightweightTorpedoSignature("grisha") != "set40" {
		t.Fatal("Grisha tubes fire SET-40")
	}
}

func TestSpawnHostileTorpedoGreenSmearsSolution(t *testing.T) {
	fc := NewFireControl()
	sub := &world.Entity{
		ID: "enemy_ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "foxtrot",
		X: 0, Y: 0, HeadingDeg: 0, DepthFt: 150, CrewSkill: 0,
		Track: world.AITrack{
			Valid: true, ClassConf: 0.9, X: 500, Y: 2500, DepthFt: 400,
			CourseDeg: 180, SpeedKts: 12,
		},
	}
	tgt := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 2500, DepthFt: 200, HeadingDeg: 90, SpeedKts: 6,
	}
	fish := fc.SpawnHostileTorpedo(sub, tgt)
	if fish == nil {
		t.Fatal("expected hostile fish")
	}
	// Perfect gyro toward track ghost (~brg 11°) would be far from smeared green shot.
	brgTruth := sub.BearingDegTo(tgt)
	diff := fish.GyroCourseDeg - brgTruth
	for diff > 180 {
		diff -= 360
	}
	for diff < -180 {
		diff += 360
	}
	if diff > -2 && diff < 2 && fish.RunDepthFt > 180 && fish.RunDepthFt < 220 {
		t.Fatalf("green crew should smear gyro/depth; gyro=%.1f depth=%.0f", fish.GyroCourseDeg, fish.RunDepthFt)
	}
}

func TestLaunchRBUSplash(t *testing.T) {
	fc := NewFireControl()
	ship := &world.Entity{
		ID: "cv", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "grisha", X: 0, Y: 0,
	}
	tgt := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 1200, DepthFt: 150,
	}
	if fc.LaunchRastrub(ship, tgt, 10) != nil {
		t.Fatal("Grisha must not launch Rastrub")
	}
	salvo := fc.LaunchRBU(ship, tgt, 10)
	if salvo == nil {
		t.Fatal("expected RBU")
	}
	dets := fc.AdvanceRBU(10+salvo.FlightSec+0.1, []*world.Entity{tgt})
	if len(dets) != 1 || !dets[0].RBU {
		t.Fatalf("expected RBU detonation, got %#v", dets)
	}
	if dets[0].Hit == nil || dets[0].Hit.ID != "player" {
		t.Fatal("expected player in blast")
	}
}
