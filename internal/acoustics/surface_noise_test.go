package acoustics

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestSurfaceCavitationIsSpeedDriven(t *testing.T) {
	if CavitationSeverity(0, 7) > 0.05 {
		t.Fatalf("slow surface should not cavitate hard: %.2f", CavitationSeverity(0, 7))
	}
	cruise := CavitationSeverity(0, 11)
	if cruise < 0.15 || cruise > 0.4 {
		t.Fatalf("merchant cruise cavitation unexpected: %.2f", cruise)
	}
	sprint := CavitationSeverity(0, 22)
	if sprint < 0.95 {
		t.Fatalf("surface sprint should fully cavitate: %.2f", sprint)
	}
}

func TestCivilianQuieterThanWarshipAtRange(t *testing.T) {
	m := NewModel(DefaultEnvironment())
	listener := testEntity("p", "los_angeles", world.KindSubmarine, 180, 8)
	merchant := testEntity("m", "merchant", world.KindSurfaceShip, 0, 11)
	dd := testEntity("d", "udaloy", world.KindSurfaceShip, 0, 14)
	merchant.Y = 6000
	dd.Y = 6000
	mSNR := m.Detect(listener, merchant, ModePassive, 0).PeakSNR
	dSNR := m.Detect(listener, dd, ModePassive, 0).PeakSNR
	if mSNR >= dSNR {
		t.Fatalf("merchant should be quieter than DD at 6 kyd: merchant=%.1f dd=%.1f", mSNR, dSNR)
	}
	if mSNR > 22 {
		t.Fatalf("merchant still too loud at 6 kyd: %.1f", mSNR)
	}
}
