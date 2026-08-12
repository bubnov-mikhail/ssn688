package acoustics

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestCanDetectPlayerPassiveCachesQuietCase(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := &world.Entity{
		ID: "e1", SignatureID: "grisha", Kind: world.KindSurfaceShip,
		Status: world.StatusActive, DepthFt: 20, SpeedKts: 12, X: 0, Y: 0,
	}
	player := &world.Entity{
		ID: "p1", SignatureID: "los_angeles", Kind: world.KindSubmarine,
		Status: world.StatusActive, DepthFt: 200, SpeedKts: 5, X: 8000, Y: 0,
	}

	a := model.CanDetectPlayerPassive(listener, player, 1.0)
	if listener.PassiveDetectCacheAt != 1.0 {
		t.Fatalf("cache stamp=%v", listener.PassiveDetectCacheAt)
	}
	if listener.PassiveDetectCached != a {
		t.Fatal("cached flag mismatch")
	}
	b := model.CanDetectPlayerPassive(listener, player, 1.2)
	if a != b {
		t.Fatal("cache should return same result within TTL")
	}
	_ = model.CanDetectPlayerPassive(listener, player, 1.0+passiveDetectCacheTTLSec+0.01)
	if listener.PassiveDetectCacheAt < 1.3 {
		t.Fatalf("expected cache refresh, at=%v", listener.PassiveDetectCacheAt)
	}
}
