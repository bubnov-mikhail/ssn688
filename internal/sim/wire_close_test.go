package sim

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestCloseDoorAfterWireCutKeepsFishInEngine(t *testing.T) {
	eng := NewEngine(world.NewTrainingScenario())
	player := eng.Scenario.Player
	fc := &eng.FireControl

	if !fc.OpenOuterDoor(1) {
		t.Fatal("open door")
	}
	fish := fc.Shoot(player, 1)
	if fish == nil {
		t.Fatal("shoot")
	}
	id := fish.ID

	// Run past tube-clear and SearchArmMinDist.
	for i := 0; i < 800; i++ {
		eng.Update(0.1)
	}
	fish = fc.TorpedoByID(id)
	if fish == nil || !fish.Alive {
		t.Fatalf("fish lost before cut: %+v", fish)
	}

	fc.CutWire(fish)
	for i := 0; i < 20; i++ {
		eng.Update(0.1)
	}
	fish = fc.TorpedoByID(id)
	if fish == nil || !fish.Alive || !fish.WireCut {
		t.Fatalf("after cut: fish=%v", fish)
	}

	if !fc.CloseOuterDoor(1, eng.Clock.GameTime) {
		t.Fatal("close door")
	}
	if fc.Tubes[0].TorpedoID != "" {
		t.Fatalf("tube still linked: %q", fc.Tubes[0].TorpedoID)
	}

	for i := 0; i < 100; i++ {
		eng.Update(0.1)
	}
	fish = fc.TorpedoByID(id)
	if fish == nil {
		t.Fatal("torpedo removed from ActiveTorpedoes after close + sim ticks")
	}
	if !fish.Alive {
		t.Fatalf("torpedo dead after close: age=%.1f mode=%d", fish.Age, fish.Mode)
	}
}
