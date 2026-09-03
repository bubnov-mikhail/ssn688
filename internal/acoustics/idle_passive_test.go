package acoustics

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func idlePassivePair(targetSig string, kind world.EntityKind, tgtSpeed, rangeYd, tgtDepth float64) DetectionResult {
	model := NewModel(DefaultEnvironment())
	listener := &world.Entity{
		ID: "player", Name: "ownship", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "los_angeles",
		X: 0, Y: 0, DepthFt: 200, SpeedKts: 5, HeadingDeg: 0,
	}
	target := &world.Entity{
		ID: "tgt", Name: "target", Kind: kind, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: targetSig,
		X: rangeYd, Y: 0, DepthFt: tgtDepth, SpeedKts: tgtSpeed, HeadingDeg: 90,
	}
	return model.Detect(listener, target, ModePassive, 0)
}

func TestIdlePlantHardToHearOnPassive(t *testing.T) {
	cases := []struct {
		sig   string
		kind  world.EntityKind
		depth float64
	}{
		{"los_angeles", world.KindSubmarine, 200},
		{"foxtrot", world.KindSubmarine, 180},
		{"grisha", world.KindSurfaceShip, 0},
		{"tanker", world.KindSurfaceShip, 0},
	}
	for _, tc := range cases {
		near := idlePassivePair(tc.sig, tc.kind, 0, 200, tc.depth)
		if !near.Detected {
			t.Errorf("%s idle at 200 yd should be a confident detect (snr=%.1f bands=%d)",
				tc.sig, near.PeakSNR, near.BandsAbove)
		}
		far := idlePassivePair(tc.sig, tc.kind, 0, 800, tc.depth)
		if far.Detected {
			t.Errorf("%s idle at 800 yd should be below detect (snr=%.1f bands=%d)",
				tc.sig, far.PeakSNR, far.BandsAbove)
		}
		mid := idlePassivePair(tc.sig, tc.kind, 0, 400, tc.depth)
		if mid.Detected && mid.PeakSNR >= PeakDetectSNR {
			t.Errorf("%s idle at 400 yd should not be a confident detect (snr=%.1f bands=%d)",
				tc.sig, mid.PeakSNR, mid.BandsAbove)
		}
	}
}

func TestUnderwayPlantStillHeardBeyondIdleRange(t *testing.T) {
	r := idlePassivePair("foxtrot", world.KindSubmarine, 8, 800, 180)
	if !r.Detected {
		t.Fatalf("foxtrot at 8 kn / 800 yd should still be heard, snr=%.1f bands=%d", r.PeakSNR, r.BandsAbove)
	}
}

func TestIdleTransientStillRadiates(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "los_angeles",
		DepthFt: 200, SpeedKts: 5,
	}
	quiet := &world.Entity{
		ID: "tgt", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "foxtrot",
		X: 2500, DepthFt: 180, SpeedKts: 0,
	}
	quiet.X = 600
	idle := model.Detect(listener, quiet, ModePassive, 0)
	if idle.Detected {
		t.Fatal("idle foxtrot at 600 yd should be below detect without transients")
	}
	quiet.TransientUntil = 10
	quiet.TransientFreqHz = 180
	quiet.TransientLevelDB = 22
	with := model.Detect(listener, quiet, ModePassive, 0)
	if with.PeakSNR <= idle.PeakSNR+5 {
		t.Fatalf("transient should raise idle radiated SNR, idle=%.1f with=%.1f", idle.PeakSNR, with.PeakSNR)
	}
}
