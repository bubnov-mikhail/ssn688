package weapons

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestSeekerPrefersDecoyThenRejects(t *testing.T) {
	fish := &Torpedo{
		ID: "MK48-1", Side: world.SidePlayer, Alive: true, Mode: ModeSearch,
		X: 0, Y: 0, HeadingDeg: 0, DepthFt: 200, SpeedKts: 50, CruiseKts: 50,
		LastPingTime: -1,
	}
	ship := &world.Entity{
		ID: "enemy_dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 900, DepthFt: 0, SpeedKts: 14,
	}
	decoy := &world.Entity{
		ID: "ADC-1", Kind: world.KindCountermeasure, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "adc",
		X: 40, Y: 500, DepthFt: 60, SpeedKts: 0.3,
	}
	targets := []*world.Entity{ship, decoy}

	got := fish.acquireInCone(targets, nil, 10)
	if got == nil || got.ID != "ADC-1" {
		t.Fatalf("expected decoy lock, got %#v", got)
	}
	fish.CMLockID = got.ID
	fish.CMLockSince = 10
	fish.TargetID = got.ID

	fish.maybeRejectCM(targets, 10+AntiCMVerifySec+0.5)
	if fish.RejectedUntil["ADC-1"] <= 0 {
		t.Fatal("expected decoy rejection")
	}
	got2 := fish.acquireInCone(targets, nil, 10+AntiCMVerifySec+1)
	if got2 == nil || got2.ID != "enemy_dd" {
		t.Fatalf("expected ship after reject, got %#v", got2)
	}
}

func TestEnemySeekerStaysSeducedWithShipInCone(t *testing.T) {
	fish := &Torpedo{
		ID: "ETORP-1", Side: world.SideEnemy, Alive: true, Mode: ModeSearch,
		X: 0, Y: 0, HeadingDeg: 0, DepthFt: 200, SpeedKts: 48, CruiseKts: 48,
		LastPingTime: -1, WireCut: true,
	}
	sub := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 80, Y: 1100, DepthFt: 200, SpeedKts: 8,
	}
	decoy := &world.Entity{
		ID: "ADC-1", Kind: world.KindCountermeasure, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "adc",
		X: 20, Y: 450, DepthFt: 180, SpeedKts: 0.4,
	}
	targets := []*world.Entity{sub, decoy}

	got := fish.acquireInCone(targets, nil, 5)
	if got == nil || got.ID != "ADC-1" {
		t.Fatalf("hostile seeker should prefer decoy, got %#v", got)
	}
	fish.CMLockID = got.ID
	fish.CMLockSince = 5
	fish.TargetID = got.ID

	// After a normal verify window, player fish would reject — enemy should hold.
	fish.maybeRejectCM(targets, 5+AntiCMVerifySec+1)
	if fish.RejectedUntil["ADC-1"] > 0 {
		t.Fatal("hostile seeker rejected decoy too early while still seduced")
	}
	got2 := fish.acquireInCone(targets, nil, 5+AntiCMVerifySec+1)
	if got2 == nil || got2.ID != "ADC-1" {
		t.Fatalf("expected continued decoy lock, got %#v", got2)
	}
}

func TestWireModeIgnoresDecoy(t *testing.T) {
	fish := &Torpedo{
		ID: "MK48-1", Alive: true, Mode: ModeWire, HeadingDeg: 0, X: 0, Y: 0, DepthFt: 200,
	}
	decoy := &world.Entity{
		ID: "ADC-1", Kind: world.KindCountermeasure, Status: world.StatusActive,
		X: 0, Y: 400, DepthFt: 60,
	}
	fish.Mode = ModeWire
	if got := fish.acquireInCone([]*world.Entity{decoy}, nil, 1); got != nil {
		t.Fatalf("wire mode should not acquire decoy, got %s", got.ID)
	}
}

func TestDeployDecoyAndJitterMagazines(t *testing.T) {
	cs := NewCountermeasureSystem()
	plat := &world.Entity{
		ID: "enemy_ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, DepthFt: 200, HeadingDeg: 90,
	}
	cm := cs.DeployADC(plat, 1000, 0, 5)
	if cm == nil {
		t.Fatal("decoy deploy failed")
	}
	if cs.DecoyLeft(plat.ID) != CMMagazineDefault-1 {
		t.Fatalf("decoy mag=%d", cs.DecoyLeft(plat.ID))
	}
	if cs.DeployADC(plat, 1000, 0, 6) != nil {
		t.Fatal("expected decoy cooldown")
	}
	jit := cs.DeployJitter(plat, 1000, 0, 5)
	if jit == nil {
		t.Fatal("jitter deploy failed")
	}
	if jit.Kind != CMExpendableJitter {
		t.Fatalf("kind=%v", jit.Kind)
	}
	if cs.JitterLeft(plat.ID) != CMJitterMagazineDefault-1 {
		t.Fatalf("jitter mag=%d", cs.JitterLeft(plat.ID))
	}
}

func TestJitterConfusesShipLock(t *testing.T) {
	fish := &Torpedo{
		ID: "MK48-1", Alive: true, Mode: ModeSearch,
		X: 0, Y: 0, HeadingDeg: 0, DepthFt: 200, SpeedKts: 50,
		LastPingTime: -1,
	}
	ship := &world.Entity{
		ID: "enemy_dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 800, DepthFt: 0, SpeedKts: 14,
	}
	jitter := &world.Entity{
		ID: "JIT-1", Kind: world.KindCountermeasure, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "jitter",
		X: 30, Y: 400, DepthFt: 50, SpeedKts: 1,
	}
	// With jitter present, decoy-less acquire may still pick ship, but jam factor < 1.
	f := JitterConfuseFactor([]*world.Entity{ship, jitter}, fish.X, fish.Y, fish.HeadingDeg, SeekConeHalfAngleDeg, SeekAcquireRangeYd)
	if f >= 0.99 {
		t.Fatalf("expected jam factor < 1, got %.2f", f)
	}
}
