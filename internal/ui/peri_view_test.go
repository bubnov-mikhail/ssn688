package ui

import (
	"math"
	"testing"
)

func TestPeriIRAspectMatchesOptic(t *testing.T) {
	ow, oh := periOpticInnerSize()
	irAspect := float64(periIRW) / float64(periIRH)
	opticAspect := float64(ow) / float64(oh)
	if math.Abs(irAspect-opticAspect)/opticAspect > 0.01 {
		t.Fatalf("IR aspect %.3f != optic %.3f (%dx%d vs %dx%d)",
			irAspect, opticAspect, periIRW, periIRH, ow, oh)
	}
}

func TestPeriOpticLetterboxPreservesAspect(t *testing.T) {
	ow, oh := periOpticInnerSize()
	lb := periOpticLetterbox(10, 20, ow, oh)
	irAspect := float64(periIRW) / float64(periIRH)
	outAspect := float64(lb.IW) / float64(lb.IH)
	if math.Abs(irAspect-outAspect)/irAspect > 0.01 {
		t.Fatalf("display aspect %.3f != IR aspect %.3f", outAspect, irAspect)
	}
	if lb.Scale <= 0 {
		t.Fatal("scale must be positive")
	}
}

func TestPeriOpticLetterboxFillsMastPanel(t *testing.T) {
	ow, oh := periOpticInnerSize()
	lb := periOpticLetterbox(0, 0, ow, oh)
	// Aspect-fit may leave a 1–2px gap on one axis after integer rounding.
	if absInt(lb.IW-ow) > 2 && absInt(lb.IH-oh) > 2 {
		t.Fatalf("expected near-full fill of %dx%d, got %dx%d at %d,%d",
			ow, oh, lb.IW, lb.IH, lb.IX, lb.IY)
	}
	if absInt(lb.IX) > 2 || absInt(lb.IY) > 2 {
		t.Fatalf("unexpected letterbox offset %d,%d", lb.IX, lb.IY)
	}
}
