package ui

import (
	"testing"
	"time"
)

func TestIsOwnshipDamageFXEvent(t *testing.T) {
	cases := []struct {
		ev   string
		want bool
	}{
		{"OWN SHIP HIT — systems damaged", true},
		{"OWN SHIP CRITICAL — Propulsion", true},
		{"PLAYER SUBMARINE FATAL DAMAGE — SINKING", true},
		{"PLAYER SUBMARINE LOST — hull breach", true},
		{"WARNING — ESM mast near crush depth", false},
		{"AUTO-RETRACT — masts lowering to prevent damage", false},
		{"Target hit: Foxtrot — damaged", false},
		{"Torpedo in the water.", false},
	}
	for _, tc := range cases {
		if got := isOwnshipDamageFXEvent(tc.ev); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.ev, got, tc.want)
		}
	}
}

func TestTriggerOwnshipHitFXSetsAlertUnlessOnDC(t *testing.T) {
	a := &App{CurrentScreen: ScreenPassive}
	a.triggerOwnshipHitFX()
	if !a.dcTabAlert {
		t.Fatal("expected DC tab alert after hit off DC screen")
	}
	if a.hitVignetteAt.IsZero() || a.hitShakeAt.IsZero() {
		t.Fatal("expected vignette/shake timestamps")
	}

	a2 := &App{CurrentScreen: ScreenDamage}
	a2.triggerOwnshipHitFX()
	if a2.dcTabAlert {
		t.Fatal("DC alert should stay off when already on DC")
	}
}

func TestClearDCTabAlertIfOnDamage(t *testing.T) {
	a := &App{CurrentScreen: ScreenPassive, dcTabAlert: true}
	a.clearDCTabAlertIfOnDamage()
	if !a.dcTabAlert {
		t.Fatal("should keep alert when not on DC")
	}
	a.CurrentScreen = ScreenDamage
	a.clearDCTabAlertIfOnDamage()
	if a.dcTabAlert {
		t.Fatal("should clear alert when on DC")
	}
}

func TestHitShakeOffsetDecays(t *testing.T) {
	a := &App{}
	if dx, dy := a.hitShakeOffset(); dx != 0 || dy != 0 {
		t.Fatalf("idle shake want 0,0 got %d,%d", dx, dy)
	}
	a.hitShakeAt = time.Now().Add(-ownshipHitShakeDur / time.Duration(ownshipHitShakeKicks*2))
	dx, dy := a.hitShakeOffset()
	if dx != 0 {
		t.Fatalf("shake must be vertical-only, got dx=%d", dx)
	}
	if dy == 0 {
		t.Fatal("expected non-zero vertical shake mid-kick")
	}
	a.hitShakeAt = time.Now().Add(-ownshipHitShakeDur - time.Millisecond)
	if dx, dy := a.hitShakeOffset(); dx != 0 || dy != 0 {
		t.Fatalf("expired shake want 0,0 got %d,%d", dx, dy)
	}
}

func TestHitShakeHasMultipleKicks(t *testing.T) {
	a := &App{}
	kickDur := ownshipHitShakeDur / time.Duration(ownshipHitShakeKicks)
	var peaks int
	for k := 0; k < ownshipHitShakeKicks; k++ {
		a.hitShakeAt = time.Now().Add(-time.Duration(k)*kickDur - kickDur/2)
		dx, dy := a.hitShakeOffset()
		if dx != 0 {
			t.Fatalf("kick %d: expected dx=0, got %d", k, dx)
		}
		if dy != 0 {
			peaks++
		}
	}
	if peaks < 7 {
		t.Fatalf("expected many vertical kicks, got %d", peaks)
	}
}

func TestHitVignetteAlphaEnvelope(t *testing.T) {
	a := &App{}
	if a.hitVignetteAlpha() != 0 {
		t.Fatal("idle vignette alpha should be 0")
	}
	a.hitVignetteAt = time.Now()
	if a.hitVignetteAlpha() == 0 {
		t.Fatal("fresh vignette should be visible")
	}
	a.hitVignetteAt = time.Now().Add(-ownshipHitVignetteDur - time.Millisecond)
	if a.hitVignetteAlpha() != 0 {
		t.Fatal("expired vignette alpha should be 0")
	}
}
