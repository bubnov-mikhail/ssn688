package acoustics

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestPlayerPingHeardAfterOneWayDelay(t *testing.T) {
	player := &world.Entity{ID: "player", LastPingTime: 100, LastPingPower: 1}
	enemy := &world.Entity{ID: "dd", X: 8000, Y: 0}

	travel := 8000.0 / SoundSpeedYdPerSec
	if HeardPlayerPing(DefaultEnvironment(), enemy, player, 100+travel*0.5) {
		t.Fatal("ping should not be heard before one-way travel time")
	}
	if !HeardPlayerPing(DefaultEnvironment(), enemy, player, 100+travel+0.2) {
		t.Fatal("ping should be heard after one-way travel time")
	}
}

func TestPlayerPingPassiveBonusDecays(t *testing.T) {
	b0 := PlayerPingPassiveBonusDB(0, 1)
	b10 := PlayerPingPassiveBonusDB(10, 1)
	if b0 <= b10 {
		t.Fatalf("bonus should decay: start=%.1f later=%.1f", b0, b10)
	}
	if PlayerPingPassiveBonusDB(-1, 1) != 0 {
		t.Fatal("no bonus without heard ping")
	}
}

func TestCanDetectPlayerPassiveUsesPingBonus(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	player := testEntity("player", "los_angeles", world.KindSubmarine, 180, 22)
	player.LastPingTime = 50
	player.LastPingPower = 1
	enemy := testEntity("dd", "udaloy", world.KindSurfaceShip, 0, 12)
	enemy.X = 9000

	travel := enemy.RangeYardsTo(player) / SoundSpeedYdPerSec
	tHear := 50 + travel + 0.5

	if !model.CanDetectPlayerPassive(enemy, player, tHear) {
		t.Fatalf("expected ping reveal to help passive detect at t=%.2f", tHear)
	}
}
