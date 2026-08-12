package acoustics

import (
	"math"
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

	hullBins := SpectrumAtBearing(model, listener, emitters, hull, listener.BearingDegTo(emBeam), 10)
	var towListen world.Entity
	if !PlaceTowedListener(&towListen, listener, 1) {
		t.Fatal("towed listener")
	}
	towedBins := SpectrumAtBearing(model, listener, emitters, towed, towListen.BearingDegTo(emBeam), 10)
	hullFilter := AnalyzeClassifyFilter(hullBins, 1)
	towedFilter := AnalyzeClassifyFilter(towedBins, 1)

	if towedFilter == ClassifyIndistinct {
		t.Fatalf("towed abeam at 12 kyd should classify Grisha tonals, filter=%v peak=%.1f", towedFilter, peakOf(towedBins))
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

// Near-abeam contacts produce large hull↔towed bearing parallax. SPECTRUM look
// locks to the contact bearing (from the active aperture); beam weights must
// use that same origin or TOWED looks empty while HULL still shows tonals.
func TestSpectrumTowedUsesApertureBearingNotHull(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "los_angeles",
		X: 0, Y: 0, DepthFt: 200, SpeedKts: 5, HeadingDeg: 0,
	}
	em := &world.Entity{
		ID: "enemy_grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "grisha",
		X: 2800, Y: 0, DepthFt: 0, SpeedKts: 14, HeadingDeg: 270,
	}
	emitters := []*world.Entity{listener, em}
	towed := &SonarState{PassiveArray: PassiveArrayTowed, TowedCablePct: 1}

	var aperture world.Entity
	if !PlaceTowedListener(&aperture, listener, 1) {
		t.Fatal("towed listener")
	}
	hullBrg := listener.BearingDegTo(em)
	towBrg := aperture.BearingDegTo(em)
	if math.Abs(AngleDiffDeg(hullBrg, towBrg)) < 8 {
		t.Fatalf("expected large parallax at 2.8 kyd abeam, hull=%.1f tow=%.1f", hullBrg, towBrg)
	}

	// Analyzer locked on the towed contact bearing (as the UI does).
	bins := SpectrumAtBearing(model, listener, emitters, towed, towBrg, 10)
	if AnalyzeClassifyFilter(bins, 1) == ClassifyIndistinct {
		t.Fatalf("towed spectrum at aperture bearing %.1f should classify (hull brg was %.1f), peak=%.1f",
			towBrg, hullBrg, peakOf(bins))
	}
	// Looking at the hull bearing while on TOWED must not be required for ID.
	if peakOf(bins) < 8 {
		t.Fatalf("expected strong towed peak at aperture brg, got %.1f", peakOf(bins))
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
