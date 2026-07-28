package acoustics

import (
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
	if s.passiveSNRBonusDB() != 8 {
		t.Fatalf("expected +8 dB bonus, got %.1f", s.passiveSNRBonusDB())
	}
	s.PassiveArray = PassiveArrayHull
	if s.passiveSNRBonusDB() != 0 {
		t.Fatalf("hull array should have no towed bonus")
	}
}

func TestSpectrumAtBearingRespectsStowedTowedArray(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := &world.Entity{
		ID: "player", SignatureID: "los_angeles", Kind: world.KindSubmarine, Status: world.StatusActive,
		DepthFt: 240, SpeedKts: 5, HeadingDeg: 0, OrderedHead: 0,
	}
	emitter := &world.Entity{
		ID: "target", SignatureID: "spruance", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		Y: 5000, SpeedKts: 14,
	}
	emitters := []*world.Entity{listener, emitter}

	hull := NewSonarState()
	hull.PassiveArray = PassiveArrayHull
	hullBins := SpectrumAtBearing(model, listener, emitters, &hull, 0)

	towed := NewSonarState()
	towed.PassiveArray = PassiveArrayTowed
	towed.TowedCablePct = 0
	towedBins := SpectrumAtBearing(model, listener, emitters, &towed, 0)

	hullPeak, towedPeak := 0.0, 0.0
	for i := range hullBins {
		if hullBins[i] > hullPeak {
			hullPeak = hullBins[i]
		}
		if towedBins[i] > towedPeak {
			towedPeak = towedBins[i]
		}
	}
	if towedPeak >= hullPeak-12 {
		t.Fatalf("expected stowed towed spectrum much weaker than hull: hull=%.1f towed=%.1f", hullPeak, towedPeak)
	}
}
