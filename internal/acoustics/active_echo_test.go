package acoustics

import (
	"math"
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

func TestExpireActivePingBeyondDisplay(t *testing.T) {
	sonar := NewSonarState()
	listener := testEntity("player", "los_angeles", world.KindSubmarine, 320, 8)
	sonar.LastPingTime = 10
	sonar.activeEchoAt = 10
	sonar.activeEchoDone = map[string]bool{"target": true}
	listener.LastPingTime = 10
	listener.LastPingPower = 0.9
	listener.ActiveSonar = true

	// Front still inside 12 kyd — keep state.
	inside := 10 + TwoWayTravelSec(ActiveDisplayMaxRangeYd)*0.5
	if ExpireActivePingIfBeyondDisplay(&sonar, listener, inside) {
		t.Fatal("should not expire ping while front is inside display range")
	}
	if sonar.LastPingTime != 10 || sonar.activeEchoDone == nil {
		t.Fatal("ping state should remain while front is inside display range")
	}

	// Front beyond 12 kyd — purge.
	beyond := 10 + TwoWayTravelSec(ActiveDisplayMaxRangeYd) + 1
	if !ExpireActivePingIfBeyondDisplay(&sonar, listener, beyond) {
		t.Fatal("expected ping state to expire beyond display range")
	}
	if sonar.LastPingTime != 0 || sonar.activeEchoAt != 0 || sonar.activeEchoDone != nil {
		t.Fatal("sonar ping state should be cleared")
	}
	if listener.LastPingTime != 0 || listener.LastPingPower != 0 || listener.ActiveSonar {
		t.Fatal("listener ping state should be cleared")
	}
}

func TestFireActivePingNowWorksInStandby(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := testEntity("player", "los_angeles", world.KindSubmarine, 200, 5)
	sonar := NewSonarState()
	sonar.ActiveEnabled = false
	sonar.ActivePower = 0.8
	if !FireActivePingNow(model, listener, []*world.Entity{listener}, &sonar, 42) {
		t.Fatal("PING NOW should transmit while active mode is standby")
	}
	if sonar.LastPingTime != 42 || listener.LastPingTime != 42 {
		t.Fatalf("ping timestamps not set: sonar=%.0f listener=%.0f", sonar.LastPingTime, listener.LastPingTime)
	}
	if !listener.ActiveSonar {
		t.Fatal("listener should flag active transmit for this pulse")
	}
}

func TestProcessActiveEchoesCloseRange(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := testEntity("player", "los_angeles", world.KindSubmarine, 180, 8)
	listener.HeadingDeg = 90
	target := testEntity("dd", "udaloy", world.KindSurfaceShip, 0, 14)
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
	target := testEntity("dd", "udaloy", world.KindSurfaceShip, 0, 12)
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

func TestContactTMAEstimatesCourseAndSpeed(t *testing.T) {
	c := &Contact{}
	origin := &world.Entity{X: 0, Y: 0}
	speedKts := 24.0
	courseDeg := 90.0
	stepSec := 60.0
	for i := 0; i < 4; i++ {
		distYd := float64(i) * stepSec * speedKts * world.KnotsToYPS
		updateContactTMA(c, sampleTMAPosition(origin, courseDeg, 3000+distYd, float64(i)*stepSec+10, 0.98))
	}
	if !ContactTMAAccurate(c) {
		t.Fatalf("expected accurate TMA, got acc=%.2f spd=%.2f cse=%.1f", c.TMAAccuracy, c.TMASpeedKts, c.TMACourseDeg)
	}
	if math.Abs(c.TMASpeedKts-speedKts) > 1.0 {
		t.Fatalf("speed=%.2f want %.2f", c.TMASpeedKts, speedKts)
	}
	if math.Abs(normalizeBearingDiff(c.TMACourseDeg-courseDeg)) > 5 {
		t.Fatalf("course=%.1f want %.1f", c.TMACourseDeg, courseDeg)
	}
}
