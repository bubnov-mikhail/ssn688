package weapons

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestCloseDoorAfterWireCutKeepsFish(t *testing.T) {
	fc := NewFireControl()
	sub := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 0, HeadingDeg: 0, DepthFt: 200,
	}
	fc.OpenOuterDoor(1)
	fish := fc.Shoot(sub, 1)
	if fish == nil {
		t.Fatal("shoot failed")
	}
	// Run past tube clear + SearchArmMinDist
	for i := 0; i < 400; i++ {
		fish.Advance(0.1, 10+float64(i)*0.1, []*world.Entity{sub}, nil, nil)
	}
	fc.EnableSeeker(fish)
	for i := 0; i < 50; i++ {
		fish.Advance(0.1, 50+float64(i)*0.1, []*world.Entity{sub}, nil, nil)
	}
	fc.CutWire(fish)
	if !fish.WireCut || !fish.Alive {
		t.Fatalf("after cut: wire=%v alive=%v", fish.WireCut, fish.Alive)
	}
	// Arm search after cut
	for i := 0; i < 30; i++ {
		fish.Advance(0.1, 60+float64(i)*0.1, []*world.Entity{sub}, nil, nil)
	}
	if fish.Mode != ModeSearch {
		t.Fatalf("expected ModeSearch after cut+arm, got %d", fish.Mode)
	}
	id := fish.ID
	modeBeforeClose := fish.Mode
	headBeforeClose := fish.OrderedHead
	if !fc.CloseOuterDoor(1, 100) {
		t.Fatal("close failed")
	}
	found := fc.TorpedoByID(id)
	if found == nil {
		t.Fatal("torpedo removed from ActiveTorpedoes after close")
	}
	if !found.Alive {
		t.Fatal("torpedo dead after close door")
	}
	if found.Mode != modeBeforeClose {
		t.Fatalf("close door reset mode %d → %d (should stay autonomous)", modeBeforeClose, found.Mode)
	}
	if found.OrderedHead != headBeforeClose {
		t.Fatalf("close door reset OrderedHead %.1f → %.1f", headBeforeClose, found.OrderedHead)
	}
	// Continue running
	for i := 0; i < 100; i++ {
		found.Advance(0.1, 200+float64(i)*0.1, []*world.Entity{sub}, nil, nil)
	}
	if !found.Alive {
		t.Fatal("torpedo died shortly after close")
	}
}
