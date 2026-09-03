package ui

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestPeriOpticAccommodationPhases(t *testing.T) {
	// Far / weak: almost no effect.
	g, v := periOpticAccommodation(0.1, 0.01)
	if g != 1 || v != 0 {
		t.Fatalf("weak prox: gain=%v veil=%v", g, v)
	}

	// Flash: slight lift + veil.
	g, v = periOpticAccommodation(0.05, 1)
	if g <= 1 || v < 100 {
		t.Fatalf("flash phase: gain=%v veil=%v want lift+veil", g, v)
	}

	// AGC crash: dark floor.
	g, v = periOpticAccommodation(0.55, 1)
	if g > 0.45 || v > 80 {
		t.Fatalf("accommodation: gain=%v veil=%v want dark", g, v)
	}

	// Late recovery toward unity.
	g, _ = periOpticAccommodation(4.2, 1)
	if g < 0.85 || g > 1.05 {
		t.Fatalf("recovery: gain=%v", g)
	}

	g, _ = periOpticAccommodation(5.5, 1)
	if g != 1 {
		t.Fatalf("done: gain=%v", g)
	}
}

func TestPeriOpticAccommodationSkipsOffFOVBlast(t *testing.T) {
	player := &world.Entity{X: 0, Y: 0, HeadingDeg: 0, DepthFt: 40}
	peri := &acoustics.PeriscopeState{Extension: 1, Zoom: acoustics.PeriZoomLow}
	// Look north; blast due east — outside typical peri FOV.
	look := peri.TrueBearingDeg(player.HeadingDeg)
	fov := peri.FOVDeg()
	brg := 90.0
	if _, ok := acoustics.BearingToViewX(brg, look, fov, periIRW); ok {
		t.Fatalf("expected blast at 90° outside FOV look=%v fov=%v", look, fov)
	}
	brgOn := look
	if _, ok := acoustics.BearingToViewX(brgOn, look, fov, periIRW); !ok {
		t.Fatalf("expected on-boresight blast inside FOV")
	}
}
