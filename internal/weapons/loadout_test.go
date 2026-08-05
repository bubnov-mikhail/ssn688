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
	if LightweightTorpedoSignature("grisha") != "set40" {
		t.Fatal("Grisha tubes fire SET-40")
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
