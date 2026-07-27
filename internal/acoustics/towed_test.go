package acoustics

import "testing"

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
	if s.passiveSNRBonusDB() != 5 {
		t.Fatalf("expected +5 dB bonus, got %.1f", s.passiveSNRBonusDB())
	}
	s.PassiveArray = PassiveArrayHull
	if s.passiveSNRBonusDB() != 0 {
		t.Fatalf("hull array should have no towed bonus")
	}
}
