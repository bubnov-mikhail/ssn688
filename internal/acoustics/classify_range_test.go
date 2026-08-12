package acoustics

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

// Open-source anchors used for the clarity curve:
// - Harbor DEMON ship signatures measured out to ~7 km (~3.8 nm / ~7.7 kyd).
// - Hull-mounted passive often limited to a few nm for useful machinery lines.
// - Towed LF arrays extend detection well beyond hull; harmonic classification
//   is shorter than bare detection but still clearly favors TAS.
func TestClassifyClarityRangeHullVsTowed(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "los_angeles",
		X: 0, Y: 0, DepthFt: 200, SpeedKts: 5, HeadingDeg: 0,
	}

	// Beam contact — TAS sweet spot.
	emBeam := &world.Entity{
		ID: "enemy_grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "grisha",
		X: 12000, Y: 0, DepthFt: 0, SpeedKts: 14, HeadingDeg: 270,
	}
	emitters := []*world.Entity{listener, emBeam}
	hull := &SonarState{PassiveArray: PassiveArrayHull}
	towed := &SonarState{PassiveArray: PassiveArrayTowed, TowedCablePct: 1}

	hullBins := SpectrumAtBearing(model, listener, emitters, hull, 90, 10)
	towedBins := SpectrumAtBearing(model, listener, emitters, towed, 90, 10)
	hullFilter := AnalyzeClassifyFilter(hullBins, 1)
	towedFilter := AnalyzeClassifyFilter(towedBins, 1)

	if towedFilter == ClassifyIndistinct {
		t.Fatalf("towed abeam at 12 kyd should classify Grisha tonals, filter=%v", towedFilter)
	}
	hullPeak, towedPeak := peakOf(hullBins), peakOf(towedBins)
	if towedPeak <= hullPeak+1.5 {
		t.Fatalf("towed peak should clearly beat hull abeam: hull=%.1f towed=%.1f", hullPeak, towedPeak)
	}
	if hullFilter != ClassifyIndistinct {
		t.Fatalf("hull at 12 kyd abeam should still be muddy for classify, got %v", hullFilter)
	}

	// Closer ahead: hull spherical should clear the floor for a noisy corvette.
	emAhead := &world.Entity{
		ID: "enemy_grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "grisha",
		X: 0, Y: 6000, DepthFt: 0, SpeedKts: 14, HeadingDeg: 180,
	}
	emitters = []*world.Entity{listener, emAhead}
	hullNear := AnalyzeClassifyFilter(SpectrumAtBearing(model, listener, emitters, hull, 0, 10), 1)
	if hullNear == ClassifyIndistinct {
		t.Fatalf("hull ahead at 6 kyd on Grisha should classify, got %v", hullNear)
	}
	towedNear := AnalyzeClassifyFilter(SpectrumAtBearing(model, listener, emitters, towed, 0, 10), 1)
	if towedNear == ClassifyIndistinct {
		t.Fatalf("towed ahead at 6 kyd should still classify (endfire softened), got %v", towedNear)
	}
}

func peakOf(bins []float64) float64 {
	p := 0.0
	for _, v := range bins {
		if v > p {
			p = v
		}
	}
	return p
}
