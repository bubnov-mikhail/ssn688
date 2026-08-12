package world

import (
	"math"
	"testing"
)

func TestMaxTurnRateDegPerSec_HullClasses(t *testing.T) {
	tanker := &Entity{Kind: KindSurfaceShip, SignatureID: "tanker", LengthFt: 900}
	fish := &Entity{Kind: KindSurfaceShip, SignatureID: "fishing", LengthFt: 140}
	merchant := &Entity{Kind: KindSurfaceShip, SignatureID: "merchant", LengthFt: 520}
	grisha := &Entity{Kind: KindSurfaceShip, SignatureID: "grisha", LengthFt: 235}
	sub := &Entity{Kind: KindSubmarine, SignatureID: "los_angeles", LengthFt: 360}

	if MaxTurnRateDegPerSec(tanker) >= MaxTurnRateDegPerSec(merchant) {
		t.Fatalf("tanker %.2f should turn slower than merchant %.2f",
			MaxTurnRateDegPerSec(tanker), MaxTurnRateDegPerSec(merchant))
	}
	if MaxTurnRateDegPerSec(merchant) >= MaxTurnRateDegPerSec(fish) {
		t.Fatalf("merchant %.2f should turn slower than fishing %.2f",
			MaxTurnRateDegPerSec(merchant), MaxTurnRateDegPerSec(fish))
	}
	if MaxTurnRateDegPerSec(grisha) <= MaxTurnRateDegPerSec(merchant) {
		t.Fatalf("grisha %.2f should out-turn merchant %.2f",
			MaxTurnRateDegPerSec(grisha), MaxTurnRateDegPerSec(merchant))
	}
	if MaxTurnRateDegPerSec(sub) < 2.0 || MaxTurnRateDegPerSec(sub) > 3.5 {
		t.Fatalf("sub turn rate %.2f out of expected band", MaxTurnRateDegPerSec(sub))
	}
}

func TestAdvance_TurnRateRespectsHull(t *testing.T) {
	tanker := &Entity{
		Kind: KindSurfaceShip, Status: StatusActive, SignatureID: "tanker", LengthFt: 900,
		HeadingDeg: 0, OrderedHead: 90, SpeedKts: 9, OrderedSpeed: 9,
	}
	fish := &Entity{
		Kind: KindSurfaceShip, Status: StatusActive, SignatureID: "fishing", LengthFt: 140,
		HeadingDeg: 0, OrderedHead: 90, SpeedKts: 7, OrderedSpeed: 7,
	}
	const dt = 1.0
	tanker.Advance(dt)
	fish.Advance(dt)
	if tanker.HeadingDeg >= fish.HeadingDeg {
		t.Fatalf("after 1s tanker yawed %.2f°, fishing %.2f° — tanker should lag",
			tanker.HeadingDeg, fish.HeadingDeg)
	}
	maxT := MaxTurnRateDegPerSec(tanker)*RudderEffectiveness(tanker)*dt + 1e-6
	if tanker.HeadingDeg > maxT {
		t.Fatalf("tanker yaw %.3f° exceeds max %.3f°/s", tanker.HeadingDeg, MaxTurnRateDegPerSec(tanker)*RudderEffectiveness(tanker))
	}
	// ~3 min of tanker turn still well short of 90°.
	for i := 0; i < 180; i++ {
		tanker.Advance(dt)
	}
	if tanker.HeadingDeg > 85 {
		t.Fatalf("tanker reached %.1f° in 3 min — still too agile for VLCC", tanker.HeadingDeg)
	}
	if math.Abs(shortestAngleDiff(tanker.HeadingDeg, 90)) < 1 {
		t.Fatal("tanker should not complete 90° in 3 minutes")
	}
}

func TestRudderEffectiveness_SubNeedsWayOn(t *testing.T) {
	sub := &Entity{Kind: KindSubmarine, SignatureID: "los_angeles", LengthFt: 360}
	sub.SpeedKts = 0
	if RudderEffectiveness(sub) != 0 {
		t.Fatal("all-stop sub must have zero rudder authority")
	}
	sub.SpeedKts = 1.0
	if RudderEffectiveness(sub) != 0 {
		t.Fatal("below deadband sub must not yaw")
	}
	sub.SpeedKts = 4.0
	mid := RudderEffectiveness(sub)
	if mid <= 0 || mid >= 1 {
		t.Fatalf("mid-speed effectiveness should be partial, got %.2f", mid)
	}
	sub.SpeedKts = 10
	if RudderEffectiveness(sub) != 1 {
		t.Fatalf("full way-on should be 1, got %.2f", RudderEffectiveness(sub))
	}
	sub.SpeedKts = -6
	if RudderEffectiveness(sub) <= 0 {
		t.Fatal("astern flow should still feed the rudder")
	}
}

func TestAdvance_SubCannotTurnInPlace(t *testing.T) {
	sub := &Entity{
		Kind: KindSubmarine, Status: StatusActive, SignatureID: "los_angeles", LengthFt: 360,
		HeadingDeg: 0, OrderedHead: 90, SpeedKts: 0, OrderedSpeed: 0, DepthFt: 200, OrderedDepth: 200,
	}
	for i := 0; i < 60; i++ {
		sub.Advance(1)
	}
	if math.Abs(sub.HeadingDeg) > 0.01 {
		t.Fatalf("all-stop sub yawed to %.2f° — thrusterless turn forbidden", sub.HeadingDeg)
	}

	slow := &Entity{
		Kind: KindSubmarine, Status: StatusActive, SignatureID: "los_angeles", LengthFt: 360,
		HeadingDeg: 0, OrderedHead: 90, SpeedKts: 4, OrderedSpeed: 4, DepthFt: 200, OrderedDepth: 200,
	}
	fast := &Entity{
		Kind: KindSubmarine, Status: StatusActive, SignatureID: "los_angeles", LengthFt: 360,
		HeadingDeg: 0, OrderedHead: 90, SpeedKts: 12, OrderedSpeed: 12, DepthFt: 200, OrderedDepth: 200,
	}
	slow.Advance(1)
	fast.Advance(1)
	if fast.HeadingDeg <= slow.HeadingDeg {
		t.Fatalf("faster sub should yaw more: slow=%.2f fast=%.2f", slow.HeadingDeg, fast.HeadingDeg)
	}
}
