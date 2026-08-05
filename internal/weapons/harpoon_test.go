package weapons

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestEnsureHarpoonDestructValid(t *testing.T) {
	got := EnsureHarpoonDestructValid(HarpoonSRCHLong, HarpoonDSTRMedium)
	if got != HarpoonDSTRLong {
		t.Fatalf("long SRCH should force long DSTR, got %s", got)
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
	if fc.MagazineLeft != PlayerMagazineCapacity-4+1 {
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
		HeadingDeg: 0, BeamHalfDeg: 45, SpeedKts: HarpoonCruiseKts,
		X: 0, Y: 0, DestructRangeYd: HarpoonMaxRangeYd,
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
