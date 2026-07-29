package acoustics

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestListenBandFavorsTorpedoOnHF(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := testEntity("player", "los_angeles", world.KindSubmarine, 250, 5)
	torp := &world.Entity{
		ID: "mk48-1", SignatureID: "mk48", Kind: world.KindTorpedo, Status: world.StatusActive,
		Y: 2500, DepthFt: 200, SpeedKts: 50, HeadingDeg: 0,
	}
	ship := testEntity("dd", "spruance", world.KindSurfaceShip, 0, 14)
	ship.Y = 2500

	rTorp := model.Detect(listener, torp, ModePassive, 0)
	rShip := model.Detect(listener, ship, ModePassive, 0)

	bbTorp, hfTorp := rTorp, rTorp
	bbShip, hfShip := rShip, rShip
	ApplyListenBand(&bbTorp, ListenBroadband)
	ApplyListenBand(&hfTorp, ListenHF)
	ApplyListenBand(&bbShip, ListenBroadband)
	ApplyListenBand(&hfShip, ListenHF)

	if hfTorp.PeakSNR <= bbTorp.PeakSNR {
		t.Fatalf("HF band should favor torpedo: hf=%.1f bb=%.1f", hfTorp.PeakSNR, bbTorp.PeakSNR)
	}
	if bbShip.PeakSNR <= hfShip.PeakSNR {
		t.Fatalf("broadband should favor surface ship: bb=%.1f hf=%.1f", bbShip.PeakSNR, hfShip.PeakSNR)
	}
}

func TestTorpedoActiveTargetStrength(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := testEntity("player", "los_angeles", world.KindSubmarine, 200, 5)
	torp := &world.Entity{
		ID: "t1", SignatureID: "mk48", Kind: world.KindTorpedo, Status: world.StatusActive,
		Y: 800, DepthFt: 180, SpeedKts: 50, HeadingDeg: 90, LengthFt: 19,
	}
	r := model.Detect(listener, torp, ModeActive, 1.0)
	if r.PeakSNR < 3 {
		t.Fatalf("close torpedo should return active echo, peak=%.1f", r.PeakSNR)
	}
}
