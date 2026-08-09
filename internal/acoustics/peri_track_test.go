package acoustics

import (
	"math"
	"strings"
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestUpdateContactsFromPeriscopeRefinesRange(t *testing.T) {
	player := &world.Entity{
		ID: "p", Kind: world.KindSubmarine, Status: world.StatusActive,
		X: 0, Y: 0, DepthFt: 60, HeadingDeg: 0,
	}
	ship := &world.Entity{
		ID: "civ_tanker", Name: "MT", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 0, Y: 900, HeadingDeg: 90, LengthFt: 900, SignatureID: "tanker", SpeedKts: 9,
	}
	var sonar SonarState
	sonar.Contacts = []Contact{{
		ID: "C01", SourceEntityID: ship.ID, Kind: world.KindSurfaceShip,
		BearingDeg: 20, EstimatedRangeYd: 4000, UncBearingDeg: 20, UncRangeYd: 1800,
		DetectedBy: "passive", FirstSeen: 0, LastUpdate: 0, Confidence: 0.3,
	}}
	peri := PeriscopeState{Extension: 1, Order: PeriMastRaise, Zoom: PeriZoomLow, TrainRelDeg: 0}

	for i := 0; i < 12; i++ {
		UpdateContactsFromPeriscope(&sonar, &peri, player, []*world.Entity{ship}, world.WeatherLight, float64(i+1)*2)
	}
	c := &sonar.Contacts[0]
	if !strings.Contains(c.DetectedBy, "visual") {
		t.Fatalf("DetectedBy=%q want visual tag", c.DetectedBy)
	}
	trueR := player.RangeYardsTo(ship)
	if math.Abs(c.EstimatedRangeYd-trueR) > 250 {
		t.Fatalf("range=%.0f want near %.0f", c.EstimatedRangeYd, trueR)
	}
	if c.UncRangeYd > 800 {
		t.Fatalf("unc range still large: %.0f", c.UncRangeYd)
	}
}

func TestUpdateContactsFromPeriscopeCreatesVisualTrack(t *testing.T) {
	player := &world.Entity{
		ID: "p", Kind: world.KindSubmarine, Status: world.StatusActive,
		X: 0, Y: 0, DepthFt: 60, HeadingDeg: 0,
	}
	ship := &world.Entity{
		ID: "civ_merchant", Name: "MV", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 200, Y: 700, HeadingDeg: 180, LengthFt: 520, SignatureID: "merchant",
	}
	var sonar SonarState
	peri := PeriscopeState{Extension: 1, Order: PeriMastRaise}
	UpdateContactsFromPeriscope(&sonar, &peri, player, []*world.Entity{ship}, world.WeatherLight, 5)
	if len(sonar.Contacts) != 1 {
		t.Fatalf("want 1 visual contact, got %d", len(sonar.Contacts))
	}
	c := sonar.Contacts[0]
	if c.SourceEntityID != ship.ID || c.DetectedBy != "visual" {
		t.Fatalf("contact %#v", c)
	}
}
