package ai

import (
	"math"
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestSubShadowOpensWhenTooClose(t *testing.T) {
	sub := &world.Entity{
		ID: "enemy_ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 8,
		Damage: world.NewFullHealth(),
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 1200, HeadingDeg: 90, SpeedKts: 8,
	}
	applySubShadowTactics(sub, player, 10, sub.RangeYardsTo(player), sub.BearingDegTo(player))
	if sub.AIState != "OPENING" {
		t.Fatalf("expected OPENING, got %s", sub.AIState)
	}
	// Must not steer straight at the player (bearing ~0).
	brg := sub.BearingDegTo(player)
	diff := math.Abs(shortestRel(sub.OrderedHead - brg))
	if diff < 90 {
		t.Fatalf("opening head %.0f too close to player bearing %.0f", sub.OrderedHead, brg)
	}
}

func TestSubShadowHoldsFlankInBand(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 8,
	}
	// Sit on player's starboard beam at ideal standoff.
	sub := &world.Entity{
		ID: "enemy_ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: subStandoffIdealYd, Y: 0, HeadingDeg: 0, SpeedKts: 8,
		Damage: world.NewFullHealth(),
	}
	applySubShadowTactics(sub, player, 50, sub.RangeYardsTo(player), sub.BearingDegTo(player))
	if sub.AIState != "SHADOW" && sub.AIState != "ATTACK" && sub.AIState != "FIRING" {
		t.Fatalf("expected SHADOW/ATTACK, got %s", sub.AIState)
	}
	if sub.OrderedSpeed > 12 {
		t.Fatalf("shadow speed too high: %.1f", sub.OrderedSpeed)
	}
}

func TestSubShadowClosesFromFar(t *testing.T) {
	player := &world.Entity{
		ID: "player", Status: world.StatusActive, X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 6,
	}
	sub := &world.Entity{
		ID: "enemy_ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 8000, HeadingDeg: 180, SpeedKts: 6,
		Damage: world.NewFullHealth(),
	}
	applySubShadowTactics(sub, player, 20, sub.RangeYardsTo(player), sub.BearingDegTo(player))
	if sub.AIState != "CLOSING" {
		t.Fatalf("expected CLOSING, got %s", sub.AIState)
	}
	if sub.OrderedSpeed > 10 {
		t.Fatalf("closing should stay quiet: %.1f", sub.OrderedSpeed)
	}
}
