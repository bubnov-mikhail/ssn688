package weapons

import (
	"math"
	"testing"
)

func TestSeekAcquireLimitsSameLayerFullRange(t *testing.T) {
	atten := func(src, dst float64) float64 { return 0 }
	r, cone := seekAcquireLimits(180, 200, atten)
	if math.Abs(r-SeekAcquireRangeYd) > 1 {
		t.Fatalf("same-layer range=%v want %v", r, SeekAcquireRangeYd)
	}
	if math.Abs(cone-SeekConeHalfAngleDeg) > 0.1 {
		t.Fatalf("same-layer cone=%v want %v", cone, SeekConeHalfAngleDeg)
	}
}

func TestSeekAcquireLimitsThermoclineShrinksButNotZero(t *testing.T) {
	// Typical mixed↔thermocline crossing (~16 dB) plus some column.
	atten := func(src, dst float64) float64 { return 16 + 4 }
	r, cone := seekAcquireLimits(100, 400, atten)
	if r >= SeekAcquireRangeYd*0.85 {
		t.Fatalf("cross-layer should shrink range, got %v", r)
	}
	if r < SeekAcquireRangeYd*SeekLayerMinRangeFactor-1 {
		t.Fatalf("range below floor: %v", r)
	}
	if cone >= SeekConeHalfAngleDeg {
		t.Fatalf("cone should narrow across layers, got %v", cone)
	}
	if cone < SeekConeHalfAngleDeg*0.55-0.1 {
		t.Fatalf("cone narrower than floor: %v", cone)
	}
}

func TestSeekAcquireLimitsHeavyLossHitsFloor(t *testing.T) {
	atten := func(src, dst float64) float64 { return 40 }
	r, _ := seekAcquireLimits(50, 900, atten)
	want := SeekAcquireRangeYd * SeekLayerMinRangeFactor
	if math.Abs(r-want) > 1 {
		t.Fatalf("heavy loss should floor at %v, got %v", want, r)
	}
}
