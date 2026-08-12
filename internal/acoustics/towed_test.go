package acoustics

import (
	"math"
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestTowedCableMotion(t *testing.T) {
	s := NewSonarState()
	s.StartDeploy()
	if s.TowedCableRate <= 0 {
		t.Fatal("expected positive deploy rate")
	}
	for i := 0; i < 200; i++ {
		s.UpdateTowed(1.0)
	}
	if !s.TowedDeployed() {
		t.Fatalf("expected deployed, pct=%.2f rate=%.3f", s.TowedCablePct, s.TowedCableRate)
	}

	s.StartRetract()
	for i := 0; i < 150; i++ {
		s.UpdateTowed(1.0)
	}
	if !s.TowedStowed() {
		t.Fatalf("expected stowed, pct=%.2f rate=%.3f", s.TowedCablePct, s.TowedCableRate)
	}
}

func TestStopTowed(t *testing.T) {
	s := NewSonarState()
	s.StartDeploy()
	s.UpdateTowed(2.0)
	if !s.TowedInMotion() {
		t.Fatal("expected motion before stop")
	}
	s.StopTowed()
	if s.TowedInMotion() {
		t.Fatal("expected stopped")
	}
	if s.TowedCablePct <= 0 {
		t.Fatal("cable should remain partially deployed")
	}
}

func TestTowedArraySNRBonus(t *testing.T) {
	s := NewSonarState()
	s.PassiveArray = PassiveArrayTowed
	s.TowedCablePct = 1
	if s.passiveSNRBonusDB() != 11 {
		t.Fatalf("expected +11 dB bonus, got %.1f", s.passiveSNRBonusDB())
	}
	s.PassiveArray = PassiveArrayHull
	if s.passiveSNRBonusDB() != 0 {
		t.Fatalf("hull array should have no towed bonus")
	}
}

func TestTowedLeverArmBearingOffset(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	player := &world.Entity{
		ID: "player", SignatureID: "los_angeles", Kind: world.KindSubmarine, Status: world.StatusActive,
		X: 0, Y: 0, DepthFt: 200, SpeedKts: 6, HeadingDeg: 0,
	}
	// Target abeam to starboard — max parallax from aft array.
	target := &world.Entity{
		ID: "dd", SignatureID: "udaloy", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 4000, Y: 0, DepthFt: 0, SpeedKts: 12,
	}
	emitters := []*world.Entity{player, target}
	sonar := NewSonarState()
	sonar.TowedCablePct = 1

	hull := BearingWaterfallSlice(model, player, emitters, &sonar, PassiveArrayHull, 10)
	towed := BearingWaterfallSlice(model, player, emitters, &sonar, PassiveArrayTowed, 10)

	peakBin := func(row BearingWaterfallRow) int {
		best, bi := -1.0, 0
		for i, v := range row.Bearings {
			if v > best {
				best, bi = v, i
			}
		}
		return bi
	}
	hb, tb := peakBin(hull), peakBin(towed)
	if hb == tb {
		t.Fatalf("expected towed lever-arm bearing shift; hullBin=%d towedBin=%d", hb, tb)
	}
	hDeg := BearingBinToDeg(hb)
	tDeg := BearingBinToDeg(tb)
	diff := math.Abs(AngleDiffDeg(hDeg, tDeg))
	if diff < 1.5 || diff > 25 {
		t.Fatalf("unexpected parallax %.1f° (hull=%.0f towed=%.0f)", diff, hDeg, tDeg)
	}
}

func TestTriangulationBonusShrinksUncertainty(t *testing.T) {
	c := &Contact{UncBearingDeg: 20, UncRangeYd: 2000, EstimatedRangeYd: 5000}
	ApplyTriangulationBonus(c, TowedCableFullYd, 5000, 90)
	if c.UncRangeYd >= 2000*0.85 {
		t.Fatalf("expected strong range shrink abeam full cable, unc=%.0f", c.UncRangeYd)
	}
	if c.UncBearingDeg >= 20*0.95 {
		t.Fatalf("expected bearing shrink, unc=%.1f", c.UncBearingDeg)
	}
	ahead := &Contact{UncBearingDeg: 20, UncRangeYd: 2000, EstimatedRangeYd: 5000}
	ApplyTriangulationBonus(ahead, TowedCableFullYd, 5000, 5)
	if ahead.UncRangeYd < 1990 {
		t.Fatalf("ahead geometry should not triangulate well, unc=%.0f", ahead.UncRangeYd)
	}
}

func TestTowedShearAtHighSpeed(t *testing.T) {
	s := NewSonarState()
	s.TowedCablePct = 1
	if _, warn := s.CheckTowedSpeed(21); !warn {
		t.Fatal("expected warn near 20 kn full cable")
	}
	sheared, _ := s.CheckTowedSpeed(24)
	if !sheared || !s.TowedDamaged || s.TowedCablePct != 0 {
		t.Fatalf("expected shear: damaged=%v pct=%.2f sheared=%v", s.TowedDamaged, s.TowedCablePct, sheared)
	}
	s.StartDeploy()
	if s.TowedInMotion() {
		t.Fatal("damaged array must not deploy")
	}
}

func TestSpectrumAtBearingRespectsStowedTowedArray(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := &world.Entity{
		ID: "player", SignatureID: "los_angeles", Kind: world.KindSubmarine, Status: world.StatusActive,
		DepthFt: 240, SpeedKts: 5, HeadingDeg: 0, OrderedHead: 0,
	}
	emitter := &world.Entity{
		ID: "target", SignatureID: "udaloy", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		Y: 5000, SpeedKts: 14,
	}
	emitters := []*world.Entity{listener, emitter}

	hull := NewSonarState()
	hull.PassiveArray = PassiveArrayHull
	hullBins := SpectrumAtBearing(model, listener, emitters, &hull, 0, 0)

	towed := NewSonarState()
	towed.PassiveArray = PassiveArrayTowed
	towed.TowedCablePct = 0
	towedBins := SpectrumAtBearing(model, listener, emitters, &towed, 0, 0)

	hullPeak, towedPeak := 0.0, 0.0
	for i := range hullBins {
		if hullBins[i] > hullPeak {
			hullPeak = hullBins[i]
		}
		if towedBins[i] > towedPeak {
			towedPeak = towedBins[i]
		}
	}
	if towedPeak >= hullPeak-4 {
		t.Fatalf("expected stowed towed spectrum weaker than hull: hull=%.1f towed=%.1f", hullPeak, towedPeak)
	}
}
