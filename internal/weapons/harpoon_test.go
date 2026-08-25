package weapons

import (
	"math"
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestEnsureHarpoonDestructValid(t *testing.T) {
	// SRCH presets are ≤8 nm; medium DSTR (40 nm) covers all of them.
	got := EnsureHarpoonDestructValid(HarpoonSRCHXLong, HarpoonDSTRMedium)
	if got != HarpoonDSTRMedium {
		t.Fatalf("8 nm SRCH should allow medium DSTR, got %s", got)
	}
	got = EnsureHarpoonDestructValid(HarpoonSRCHShort, HarpoonDSTRMax)
	if got != HarpoonDSTRMax {
		t.Fatalf("max DSTR ok for short SRCH, got %s", got)
	}
	got = EnsureHarpoonDestructValid(HarpoonSRCHMin, HarpoonDSTRMedium)
	if got != HarpoonDSTRMedium {
		t.Fatalf("1 nm SRCH should allow medium DSTR, got %s", got)
	}
	if HarpoonRadarRangeYd(HarpoonSRCHMin) != HarpoonRadarMinYd {
		t.Fatalf("min SRCH yards: got %.0f want %.0f", HarpoonRadarRangeYd(HarpoonSRCHMin), HarpoonRadarMinYd)
	}
	want := []float64{HarpoonRadarMinYd, HarpoonRadarShortYd, HarpoonRadarMediumYd, HarpoonRadarLongYd, HarpoonRadarXLongYd}
	for i, set := range []string{HarpoonSRCHMin, HarpoonSRCHShort, HarpoonSRCHMedium, HarpoonSRCHLong, HarpoonSRCHXLong} {
		if got := HarpoonRadarRangeYd(set); got != want[i] {
			t.Fatalf("%s: got %.0f want %.0f", set, got, want[i])
		}
	}
}

func TestRequestOrdnanceReload(t *testing.T) {
	fc := NewFireControl()
	fc.HarpoonMagLeft = 2
	tube := &fc.Tubes[0]
	tube.State = TubeLoaded
	tube.TorpedoType = OrdnanceMk48
	if !fc.RequestOrdnanceReload(1, OrdnanceHarpoon, 10) {
		t.Fatal("expected harpoon reload to start")
	}
	if tube.State != TubeReloading || tube.ReloadOrdnance != OrdnanceHarpoon {
		t.Fatalf("state=%d ord=%s", tube.State, tube.ReloadOrdnance)
	}
	if fc.MagazineLeft != PlayerMagazineCapacity-2+1 {
		t.Fatalf("mk48 returned to mag: %d", fc.MagazineLeft)
	}
	if fc.HarpoonMagLeft != 1 {
		t.Fatalf("harpoon consumed: %d", fc.HarpoonMagLeft)
	}
	// Same type while reloading — no-op
	if fc.RequestOrdnanceReload(1, OrdnanceHarpoon, 11) {
		t.Fatal("same reload type should be no-op")
	}
	// Switch reload target
	if !fc.RequestOrdnanceReload(1, OrdnanceMk48, 12) {
		t.Fatal("expected reload switch")
	}
	if tube.ReloadOrdnance != OrdnanceMk48 {
		t.Fatalf("switched to %s", tube.ReloadOrdnance)
	}
}

func TestHarpoonTargetPriority(t *testing.T) {
	h := &HarpoonMissile{
		Alive: true, RadarOn: true, HeadingDeg: 0, BeamHalfDeg: 45,
		X: 0, Y: 0,
	}
	dd := &world.Entity{ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy, X: 0, Y: 500, DepthFt: 0, Status: world.StatusActive}
	civ := &world.Entity{ID: "civ", Kind: world.KindSurfaceShip, Side: world.SideNeutral, SignatureID: "merchant", X: 0, Y: 400, DepthFt: 0, Status: world.StatusActive}
	hit := h.acquireTarget([]*world.Entity{dd, civ})
	if hit == nil || hit.ID != "dd" {
		t.Fatalf("expected combatant priority, got %v", hit)
	}
}

func TestHarpoonSeekerSteersTowardLock(t *testing.T) {
	h := &HarpoonMissile{
		Alive: true, Phase: HarpoonCruise, RadarOn: true,
		HeadingDeg: 0, ProgrammedHead: 0, BeamHalfDeg: 45, SpeedKts: HarpoonCruiseKts,
		X: 0, Y: 0, LaunchX: 0, LaunchY: 0, DestructRangeYd: HarpoonMaxRangeYd,
	}
	tgt := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		X: 2000, Y: 8000, DepthFt: 0, Status: world.StatusActive,
	}
	for i := 0; i < 40; i++ {
		h.Advance(0.5, []*world.Entity{tgt})
		if !h.Alive {
			break
		}
	}
	if h.LockedTargetID != "dd" {
		t.Fatalf("expected lock on dd, got %q", h.LockedTargetID)
	}
	if h.HeadingDeg < 5 || h.HeadingDeg > 30 {
		t.Fatalf("expected turn toward target, heading=%.1f", h.HeadingDeg)
	}
	ax, ay := h.AssumedXY()
	// Assumed track stays on programmed north; real X drifts east toward target.
	if math.Abs(ax) > 50 {
		t.Fatalf("assumed track should stay on programmed course, ax=%.0f", ax)
	}
	if h.X <= ax+100 {
		t.Fatalf("real missile should have steered off assumed track: realX=%.0f assumedX=%.0f", h.X, ax)
	}
	_ = ay
}

func TestHarpoonAssumedTrackSurvivesIntercept(t *testing.T) {
	fc := NewFireControl()
	h := &HarpoonMissile{
		ID: "H-g", Alive: true, VisibleOnWEPS: true, Phase: HarpoonCruise,
		HeadingDeg: 0, ProgrammedHead: 0, SpeedKts: HarpoonCruiseKts,
		LaunchX: 0, LaunchY: 0, X: 0, Y: 0,
		DestructRangeYd: 20000, AssumedDistanceYd: 1000,
	}
	fc.ActiveHarpoons = []*HarpoonMissile{h}
	h.Alive = false // SAM soft-kill
	dets := fc.AdvanceHarpoons(1.0, 50, nil, nil)
	if len(dets) != 0 {
		t.Fatalf("ghost advance should not detonate: %#v", dets)
	}
	if !h.VisibleOnWEPS {
		t.Fatal("WEPS assumed track should remain after intercept")
	}
	if len(fc.ActiveHarpoons) != 1 {
		t.Fatal("ghost harpoon should stay in ActiveHarpoons")
	}
	if h.AssumedDistanceYd <= 1000 {
		t.Fatalf("assumed distance should advance, got %.0f", h.AssumedDistanceYd)
	}
}

func TestNewFireControlHarpoonTubes(t *testing.T) {
	fc := NewFireControl()
	if fc.Tubes[0].TorpedoType != OrdnanceMk48 || fc.Tubes[1].TorpedoType != OrdnanceMk48 {
		t.Fatal("tubes 1–2 should be Mk48")
	}
	if fc.Tubes[2].TorpedoType != OrdnanceHarpoon || fc.Tubes[3].TorpedoType != OrdnanceHarpoon {
		t.Fatal("tubes 3–4 should be Harpoon")
	}
}

func TestBeginReloadUsesLastOrdnance(t *testing.T) {
	fc := NewFireControl()
	fc.HarpoonMagLeft = 3
	tube := &fc.Tubes[1]
	tube.State = TubeFired
	tube.TorpedoType = OrdnanceHarpoon
	fc.beginReload(tube, 100)
	if tube.LastOrdnance != OrdnanceHarpoon || tube.ReloadOrdnance != OrdnanceHarpoon {
		t.Fatalf("last=%s reload=%s", tube.LastOrdnance, tube.ReloadOrdnance)
	}
}
