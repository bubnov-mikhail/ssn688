package ai

import (
	"math"
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
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

func TestSubStickyHoldsDatumOnLostContact(t *testing.T) {
	sub := &world.Entity{
		ID: "enemy_ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 6,
		Damage: world.NewFullHealth(), LastPingTime: 0, Defcon: world.DefconHostile,
		SignatureID: "foxtrot", CrewSkill: 40,
		AIProsecuting: true,
		RouteID:       "r1",
		Track: world.AITrack{
			Valid: true, ClassConf: 0.45, X: 0, Y: 3200, DepthFt: 200,
		},
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 40000, DepthFt: 600, SpeedKts: 0,
	}
	route := &world.Route{
		ID: "r1", PingPong: true,
		Waypoints: []world.Waypoint{{X: 8000, Y: 0}, {X: 12000, Y: 0}},
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	updateSubAI(sub, player, 200, 0.1, model, nil, EvadeContext{}, []*world.Route{route})
	if sub.AIState == "PATROL" {
		t.Fatal("lost contact while prosecuting should hold DATUM, not PATROL")
	}
	if sub.AIState != "DATUM" {
		t.Fatalf("expected DATUM, got %s", sub.AIState)
	}
	if !sub.AIProsecuting || !sub.Track.Valid {
		t.Fatal("should keep prosecute + frozen track")
	}
}

func TestSubStickyBreaksToPatrolAfterTimeout(t *testing.T) {
	sub := &world.Entity{
		ID: "enemy_ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, SpeedKts: 5,
		Damage: world.NewFullHealth(), LastPingTime: 0, Defcon: world.DefconHostile,
		SignatureID: "foxtrot", CrewSkill: 30,
		AIProsecuting: true,
		RouteID:       "r1",
		Track:         world.AITrack{Valid: true, ClassConf: 0.4, X: 0, Y: 3000, DepthFt: 180},
	}
	sub.AILostContactSec = aiDatumHoldSec(sub) - 0.05
	player := &world.Entity{
		ID: "player", Status: world.StatusActive, X: 0, Y: 40000, DepthFt: 600, SpeedKts: 0,
	}
	route := &world.Route{
		ID: "r1", PingPong: true,
		Waypoints: []world.Waypoint{{X: 5000, Y: 0}, {X: 9000, Y: 0}},
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	updateSubAI(sub, player, 300, 0.1, model, nil, EvadeContext{}, []*world.Route{route})
	if sub.AIProsecuting || sub.AIState != "PATROL" {
		t.Fatalf("expected break to PATROL, got state=%s prosecuting=%v", sub.AIState, sub.AIProsecuting)
	}
	if sub.AIEngageCooldownUntil <= 300 {
		t.Fatal("expected engage cooldown")
	}
}

func TestSubEngageCooldownBlocksReentry(t *testing.T) {
	sub := &world.Entity{
		ID: "enemy_ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, SpeedKts: 5,
		Damage: world.NewFullHealth(), LastPingTime: -100, Defcon: world.DefconHostile,
		SignatureID: "foxtrot", CrewSkill: 90,
		AIEngageCooldownUntil: 500,
		RouteID:               "r1",
		Track: world.AITrack{
			Valid: true, ClassConf: 0.9, HoldSec: 30,
			X: 0, Y: 3000, DepthFt: 200, CourseDeg: 90, SpeedKts: 6,
		},
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 3000, DepthFt: 200, SpeedKts: 8,
	}
	route := &world.Route{
		ID: "r1", PingPong: true,
		Waypoints: []world.Waypoint{{X: 8000, Y: 0}, {X: 12000, Y: 0}},
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	updateSubAI(sub, player, 400, 0.1, model, nil, EvadeContext{}, []*world.Route{route})
	if sub.AIProsecuting || sub.AIState == "CLOSING" || sub.AIState == "SHADOW" {
		t.Fatalf("cooldown should block re-engage, got state=%s prosecuting=%v", sub.AIState, sub.AIProsecuting)
	}
	if sub.AIState != "PATROL" {
		t.Fatalf("expected PATROL during cooldown, got %s", sub.AIState)
	}
}
