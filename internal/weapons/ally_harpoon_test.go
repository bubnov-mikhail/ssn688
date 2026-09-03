package weapons

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestSpawnAIHarpoonConsumesMagazine(t *testing.T) {
	fc := NewFireControl()
	sub := &world.Entity{
		ID: "ally_688", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "los_angeles",
		X: 0, Y: 0, HeadingDeg: 90, DepthFt: 60, CrewSkill: 80,
	}
	target := &world.Entity{
		ID: "plan_krivak", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 12000, Y: 0, SpeedKts: 14,
	}
	fc.AllyHarpoonMag = map[string]int{"ally_688": 2}
	h := fc.SpawnAIHarpoon(sub, target)
	if h == nil {
		t.Fatal("expected harpoon")
	}
	if fc.AllyHarpoonLeft("ally_688") != 1 {
		t.Fatalf("mag=%d want 1", fc.AllyHarpoonLeft("ally_688"))
	}
	if h.Side != world.SidePlayer {
		t.Fatalf("side=%v", h.Side)
	}
}

func TestAllyHarpoonMagazineFor688(t *testing.T) {
	if AllyHarpoonMagazineFor("los_angeles") != 8 {
		t.Fatalf("688 harpoon default")
	}
	if AllyHarpoonMagazineFor("kilo") != 0 {
		t.Fatal("kilo should have no harpoon")
	}
}
