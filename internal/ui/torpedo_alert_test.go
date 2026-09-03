package ui

import (
	"strings"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/audio"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/config"
	"github.com/bubnov-mikhail/ssn688/internal/sim"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestOwnTorpedoContactRecognizedAfterDetonation(t *testing.T) {
	app := NewApp(config.DefaultSettings(), nil)
	eng := sim.NewEngine(campaign.DemoRuntime())
	app.Engine = eng
	app.markOwnTorpedo("MK48-1")

	c := &acoustics.Contact{ID: "S1", SourceEntityID: "MK48-1", Kind: world.KindTorpedo, ConfirmedClass: "TORP"}
	if !app.isOwnTorpedoContact(c) {
		t.Fatal("own torpedo contact should still be recognized after fish is gone")
	}
	if !app.reportedTorpedoIDs["MK48-1"] {
		t.Fatal("own fish should be marked so hostile alert is suppressed")
	}
}

func TestGroundingEventIsWeaponImpact(t *testing.T) {
	if !isWeaponImpactEvent("Torpedo struck bottom — warhead detonation") {
		t.Fatal("grounding should count as weapon impact")
	}
}

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
	if _, ok := clips["weps/torpedo_heading_ownship"]; !ok {
		t.Fatal("missing weps/torpedo_heading_ownship voice asset")
	}

	mgr := audio.NewManager(44100)
	app := NewApp(config.DefaultSettings(), mgr)
	eng := sim.NewEngine(campaign.DemoRuntime())
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
	if !strings.Contains(app.StatusMessage, "Incoming torpedo") {
		t.Fatalf("status=%q", app.StatusMessage)
	}
	sub, ok := mgr.Subtitle()
	if !ok || !strings.Contains(sub, "Incoming torpedo") {
		t.Fatalf("expected WEPS incoming subtitle, got %q ok=%v", sub, ok)
	}
}
