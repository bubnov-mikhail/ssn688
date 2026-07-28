package acoustics

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestEchoRangeUsesTwoWayTravel(t *testing.T) {
	// After 2 s, one-way reach for returned echo is c*t/2.
	got := EchoRangeYd(2)
	want := SoundSpeedYdPerSec
	if got < want-1 || got > want+1 {
		t.Fatalf("EchoRangeYd(2)=%.1f want ~%.1f", got, want)
	}
	if TwoWayTravelSec(SoundSpeedYdPerSec) < 1.9 || TwoWayTravelSec(SoundSpeedYdPerSec) > 2.1 {
		t.Fatalf("TwoWayTravelSec mismatch: %.3f", TwoWayTravelSec(SoundSpeedYdPerSec))
	}
}

func TestProcessActiveEchoesCloseRange(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := testEntity("player", "los_angeles", world.KindSubmarine, 180, 8)
	listener.HeadingDeg = 90
	target := testEntity("dd", "spruance", world.KindSurfaceShip, 0, 14)
	target.Y = 1200
	target.HeadingDeg = 90 // abeam aspect — previously failed the broken aspect gate
	sonar := NewSonarState()
	sonar.ActiveEnabled = true
	sonar.ActivePower = 1
	emitters := []*world.Entity{listener, target}

	FireActivePingNow(model, listener, emitters, &sonar, 10)
	returnAt := 10 + TwoWayTravelSec(1200) + 0.05
	ProcessActiveEchoes(model, listener, emitters, &sonar, returnAt)
	found := false
	for _, c := range sonar.Contacts {
		if c.SourceEntityID == target.ID && c.DetectedBy == "active" && c.EstimatedRangeYd > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("expected close active contact after echo return time")
	}
}

func TestProcessActiveEchoesWaitsForReturn(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := testEntity("player", "los_angeles", world.KindSubmarine, 200, 5)
	target := testEntity("dd", "spruance", world.KindSurfaceShip, 0, 12)
	target.Y = 4000
	sonar := NewSonarState()
	sonar.ActiveEnabled = true
	sonar.ActivePower = 1
	emitters := []*world.Entity{listener, target}

	FireActivePingNow(model, listener, emitters, &sonar, 10)
	ProcessActiveEchoes(model, listener, emitters, &sonar, 10.1)
	for _, c := range sonar.Contacts {
		if c.SourceEntityID == target.ID && c.DetectedBy == "active" {
			t.Fatalf("echo should not return yet at 0.1s for 4 kyd target")
		}
	}

	// Round-trip for 4000 yd ≈ 4.94 s.
	ProcessActiveEchoes(model, listener, emitters, &sonar, 10+TwoWayTravelSec(4000)+0.05)
	found := false
	for _, c := range sonar.Contacts {
		if c.SourceEntityID == target.ID && c.DetectedBy == "active" && c.EstimatedRangeYd > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("expected active contact after echo return time")
	}
}
