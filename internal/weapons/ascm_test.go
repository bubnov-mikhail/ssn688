package weapons

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestEnemyASCMMagazineFor(t *testing.T) {
	if EnemyASCMMagazineFor("kilo") != 4 {
		t.Fatal("kilo")
	}
	if EnemyASCMMagazineFor("yasen_m") != 16 {
		t.Fatal("yasen")
	}
	if EnemyASCMMagazineFor("victor_iii") != 0 {
		t.Fatal("victor")
	}
}

func TestSpawnEnemyASCM(t *testing.T) {
	fc := NewFireControl()
	sub := &world.Entity{
		ID: "enemy_kilo", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "kilo",
		X: 0, Y: 0, DepthFt: 60, HeadingDeg: 90,
	}
	tgt := &world.Entity{
		ID: "ally_dd", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, X: 20000, Y: 0,
	}
	h := fc.SpawnEnemyASCM(sub, tgt)
	if h == nil {
		t.Fatal("nil missile")
	}
	if h.Variant != ASCMKlub {
		t.Fatalf("variant=%d", h.Variant)
	}
	if fc.EnemyASCMLeft(sub.ID) != 3 {
		t.Fatalf("mag=%d", fc.EnemyASCMLeft(sub.ID))
	}
	if ASCMDebugBaseLabel(h.Variant) != "KLUB" {
		t.Fatalf("label=%s", ASCMDebugBaseLabel(h.Variant))
	}
}

func TestSpawnAIHarpoonStillWorks(t *testing.T) {
	fc := NewFireControl()
	sub := &world.Entity{
		ID: "ally_688", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "los_angeles",
		X: 0, Y: 0, DepthFt: 60,
	}
	fc.AllyHarpoonMag[sub.ID] = 2
	tgt := &world.Entity{
		ID: "enemy_dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 15000, Y: 0,
	}
	h := fc.SpawnAIHarpoon(sub, tgt)
	if h == nil || h.Variant != ASCMHarpoon {
		t.Fatalf("harpoon variant=%d", h.Variant)
	}
}

func TestYasenAlternatesASCMVariants(t *testing.T) {
	fc := NewFireControl()
	sub := &world.Entity{
		ID: "yasen", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "yasen_m",
	}
	tgt := &world.Entity{ID: "t", Kind: world.KindSurfaceShip, Status: world.StatusActive, X: 1}
	h1 := fc.SpawnEnemyASCM(sub, tgt)
	h2 := fc.SpawnEnemyASCM(sub, tgt)
	if h1.Variant == h2.Variant {
		t.Fatalf("expected alternate variants got %d %d", h1.Variant, h2.Variant)
	}
}

func TestEnemyASCMCooldown(t *testing.T) {
	fc := NewFireControl()
	if fc.EnemyASCMOnCooldown("yasen", 10) {
		t.Fatal("cold start")
	}
	fc.NoteEnemyASCMLaunch("yasen", 100)
	if !fc.EnemyASCMOnCooldown("yasen", 100+EnemyASCMCooldownSec-1) {
		t.Fatal("should still be cooling down")
	}
	if fc.EnemyASCMOnCooldown("yasen", 100+EnemyASCMCooldownSec) {
		t.Fatal("cooldown elapsed")
	}
}

func TestHasActiveEnemyASCM(t *testing.T) {
	fc := NewFireControl()
	fc.ActiveHarpoons = []*HarpoonMissile{{
		ID: "KLBR-1", ParentSubID: "yasen", Alive: true, Variant: ASCMKalibr,
	}}
	if !fc.HasActiveEnemyASCM("yasen") {
		t.Fatal("expected active Kalibr")
	}
	if fc.HasActiveEnemyASCM("kilo") {
		t.Fatal("wrong parent")
	}
	fc.ActiveHarpoons[0].Alive = false
	if fc.HasActiveEnemyASCM("yasen") {
		t.Fatal("dead missile should not block")
	}
}
