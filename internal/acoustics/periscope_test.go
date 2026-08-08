package acoustics

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestPeriscopeRaiseGates(t *testing.T) {
	player := &world.Entity{ID: "p", Kind: world.KindSubmarine, DepthFt: 55, SpeedKts: 5}
	world.InitCombatantDamage(player)
	ok, _ := CanRaisePeriscope(player)
	if !ok {
		t.Fatal("should raise at periscope depth")
	}
	player.DepthFt = 90
	if ok, _ = CanRaisePeriscope(player); ok {
		t.Fatal("too deep")
	}
	player.DepthFt = 55
	player.SpeedKts = 12
	if ok, _ = CanRaisePeriscope(player); ok {
		t.Fatal("too fast")
	}
}

func TestPeriscopeShearsOnSpeed(t *testing.T) {
	player := &world.Entity{ID: "p", Kind: world.KindSubmarine, DepthFt: 50, SpeedKts: 5}
	world.InitCombatantDamage(player)
	var peri PeriscopeState
	peri.Order = PeriMastRaise
	peri.Extension = 1
	player.SpeedKts = 12
	_, sheared := peri.AdvanceMastMotion(0.1, 10, player)
	if !sheared || !peri.Sheared {
		t.Fatal("expected shear")
	}
	if !player.Damage.Destroyed(world.SysPeriscope) {
		t.Fatal("SysPeriscope should be destroyed")
	}
}

func TestPeriscopeTrainAndZoom(t *testing.T) {
	var peri PeriscopeState
	peri.TrainRight()
	peri.TrainRight()
	if peri.TrainRelDeg != 10 {
		t.Fatalf("train=%v", peri.TrainRelDeg)
	}
	peri.ZoomIn()
	peri.ZoomIn()
	peri.ZoomIn()
	if peri.Zoom != PeriZoomHigh {
		t.Fatalf("zoom=%d", peri.Zoom)
	}
	if peri.TrainStepDeg() != 1 {
		t.Fatalf("high zoom step=%v want 1", peri.TrainStepDeg())
	}
	before := peri.TrainRelDeg
	peri.TrainRight()
	if peri.TrainRelDeg != before+1 {
		t.Fatalf("high zoom train step failed: %v → %v", before, peri.TrainRelDeg)
	}
	if PeriZoomLabel(peri.Zoom) != "12×" {
		t.Fatalf("label=%s", PeriZoomLabel(peri.Zoom))
	}
	brg := peri.TrueBearingDeg(90)
	if mathAbs(brg-101) > 0.01 {
		t.Fatalf("true bearing=%v", brg)
	}
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func TestPeriLockTracksAndClears(t *testing.T) {
	player := &world.Entity{ID: "p", Kind: world.KindSubmarine, X: 0, Y: 0, HeadingDeg: 0, DepthFt: 50}
	tgt := &world.Entity{ID: "ship1", Kind: world.KindSurfaceShip, X: 0, Y: 2000, HeadingDeg: 90, DepthFt: 0, Side: world.SideNeutral}
	world.InitCombatantDamage(player)

	sonar := &SonarState{Contacts: []Contact{{
		ID: "C1", SourceEntityID: "ship1", BearingDeg: 0, LastUpdate: 10,
	}}}
	var peri PeriscopeState
	peri.Extension = 1
	peri.Order = PeriMastRaise
	peri.EngageLock("ship1", player.HeadingDeg, 0)
	if !peri.Locked() || peri.TrainRelDeg != 0 {
		t.Fatalf("engage lock train=%v locked=%v", peri.TrainRelDeg, peri.Locked())
	}

	// Target moves to starboard; lock should slew toward +brg.
	tgt.X = 500
	brg := player.BearingDegTo(tgt)
	peri.UpdateLock(0.5, player, sonar, nil, []*world.Entity{tgt}, world.WeatherLight, 10)
	if !peri.Locked() {
		t.Fatal("lock dropped while contact alive")
	}
	desired := normalizeRel180(AngleDiffSigned(brg, player.HeadingDeg))
	if mathAbs(peri.TrainRelDeg) < 1 {
		t.Fatalf("expected slew toward target, train=%v desired=%v", peri.TrainRelDeg, desired)
	}

	peri.TrainLeft()
	if peri.Locked() {
		t.Fatal("manual train should clear lock")
	}

	peri.EngageLock("ship1", player.HeadingDeg, brg)
	sonar.Contacts = nil
	peri.Extension = 0 // no visual either
	peri.UpdateLock(0.1, player, sonar, nil, []*world.Entity{tgt}, world.WeatherLight, 10)
	if peri.Locked() {
		t.Fatal("lock should clear without acoustic/ESM/visual")
	}
}

func TestPeriLockVisualOnly(t *testing.T) {
	player := &world.Entity{ID: "p", Kind: world.KindSubmarine, X: 0, Y: 0, HeadingDeg: 0, DepthFt: 50}
	tgt := &world.Entity{ID: "ship1", Kind: world.KindSurfaceShip, X: 0, Y: 1500, HeadingDeg: 90, DepthFt: 0}
	var peri PeriscopeState
	peri.Extension = 1
	peri.Order = PeriMastRaise
	peri.EngageLock("ship1", 0, 0)
	brg, ok := PeriLockBearing("ship1", player, &SonarState{}, nil, []*world.Entity{tgt}, &peri, world.WeatherLight, 0)
	if !ok || mathAbs(brg) > 1 {
		t.Fatalf("visual lock brg=%v ok=%v", brg, ok)
	}
}
