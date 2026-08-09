package world

import (
	"math"
	"testing"
)

func TestDepthChangeRateFPM_PatrolSpeed(t *testing.T) {
	// At ~8 kts and ~10° hull angle: Vz ≈ 8 * sin(10°) * 101.3 ≈ 141 fpm.
	got := DepthChangeRateFPM(8)
	if got < 120 || got > 200 {
		t.Fatalf("8 kts depth rate = %.1f fpm, want ~120–200 (routine ~10°)", got)
	}
}

func TestDepthChangeRateFPM_Bounded(t *testing.T) {
	if DepthChangeRateFPM(0) < DepthRateMinFPM-1e-9 {
		t.Fatalf("crawl rate too low: %.1f", DepthChangeRateFPM(0))
	}
	if DepthChangeRateFPM(30) > DepthRateMaxFPM+1e-9 {
		t.Fatalf("high-speed rate exceeds cap: %.1f", DepthChangeRateFPM(30))
	}
}

func TestAdvanceDepthRespectsRate(t *testing.T) {
	e := &Entity{
		Kind: KindSubmarine, Status: StatusActive,
		SpeedKts: 8, OrderedSpeed: 8,
		DepthFt: 100, OrderedDepth: 500,
	}
	const dt = 1.0
	before := e.DepthFt
	e.Advance(dt)
	delta := math.Abs(e.DepthFt - before)
	max := DepthChangeRateFPM(8)/60*dt + 1e-6
	if delta > max {
		t.Fatalf("depth step %.3f ft exceeds max %.3f ft/s at 8 kts", delta, max)
	}
	// Old model allowed 30 ft/s (1800 fpm) — ensure we are far below that.
	if delta > 5 {
		t.Fatalf("depth still unrealistically fast: %.2f ft in 1s", delta)
	}
}

func TestHullFireIntensitySoftTail(t *testing.T) {
	const until = 100.0
	if g := HullFireIntensity(until, 70); g != 1 {
		t.Fatalf("full fire: got %v", g)
	}
	mid := HullFireIntensity(until, until-10)
	if mid < 0.35 || mid > 0.45 {
		t.Fatalf("mid intensity: got %v want ~0.4", mid)
	}
	near := HullFireIntensity(until, until-1)
	far := HullFireIntensity(until, until-0.01)
	if near <= far {
		t.Fatalf("should decay toward zero: near=%v far=%v", near, far)
	}
	if HullFireIntensity(until, until) != 0 {
		t.Fatal("expired fire should be 0")
	}
	if HullFireIntensity(until, until+1) != 0 {
		t.Fatal("past fire should be 0")
	}
}
