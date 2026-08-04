package ui

import (
	"strings"
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/config"
	"github.com/ssn688/sim/internal/sim"
	"github.com/ssn688/sim/internal/world"
)

func TestTorpedoThreatensOwnshipDetectsCrossing(t *testing.T) {
	if !torpedoThreatensOwnship(0, 0, 0, 0, 0, 4000, 180, 45) {
		t.Fatal("expected crossing torpedo to threaten ownship")
	}
}

func TestTorpedoThreatensOwnshipRejectsOpeningTrack(t *testing.T) {
	if torpedoThreatensOwnship(0, 0, 0, 10, 4000, 0, 90, 45) {
		t.Fatal("opening torpedo track must not alert")
	}
}

func TestIncomingTorpedoAlertPlaysVoice(t *testing.T) {
	clips, err := audio.LoadVoiceClips(44100)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := clips[audio.ClipWepsTorpedoHeadingOwnship]; !ok {
		t.Fatal("missing weps/torpedo_heading_ownship voice asset")
	}

	mgr := audio.NewManager(44100)
	app := NewApp(config.DefaultSettings(), mgr)
	eng := sim.NewEngine(world.NewTrainingScenario())
	app.Engine = eng
	player := eng.Scenario.Player
	player.X, player.Y = 0, 0
	player.HeadingDeg, player.SpeedKts = 0, 0

	eng.Sonar.Contacts = []acoustics.Contact{{
		ID: "S1", SourceEntityID: "ETORP-1", Kind: world.KindTorpedo,
		ConfirmedClass: "TORP", BearingDeg: 0, EstimatedRangeYd: 4000,
		TMACourseDeg: 180, TMASpeedKts: 45, TMAAccuracy: 0.9,
	}}
	app.tactical.smoothedPos = map[string]smoothedContactPos{
		"ETORP-1": {RelX: 0, RelY: 4000, LastAt: eng.Clock.GameTime},
	}

	app.pollTorpedoCollisionAlerts()
	if !app.torpedoThreatActive["ETORP-1"] {
		t.Fatal("expected threat to be marked active")
	}
	if !strings.Contains(app.StatusMessage, "Incomming torpedo") {
		t.Fatalf("status=%q", app.StatusMessage)
	}
	sub, ok := mgr.Subtitle()
	if !ok || !strings.Contains(sub, "Incomming torpedo") {
		t.Fatalf("expected WEPS incoming subtitle, got %q ok=%v", sub, ok)
	}
}
