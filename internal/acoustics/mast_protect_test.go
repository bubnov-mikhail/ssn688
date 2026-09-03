package acoustics

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestAutoProtectMastsOnSpeed(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Status: world.StatusActive,
		DepthFt: 50, SpeedKts: 12, OrderedSpeed: 12, Damage: world.NewFullHealth(),
	}
	var esm ESMState
	esm.Extension = 1
	esm.Order = ESMMastRaise
	var comm COMMState
	comm.Extension = 0.5
	comm.Order = COMMMastRaise
	var peri PeriscopeState
	peri.Extension = 1
	peri.Order = PeriMastRaise

	evs := AutoProtectExtendedGear(player, &esm, &comm, &peri, &SonarState{})
	if len(evs) != 1 || evs[0] != EventAutoRetractMasts {
		t.Fatalf("events=%v", evs)
	}
	if esm.Order != ESMMastStow || comm.Order != COMMMastStow || peri.Order != PeriMastStow {
		t.Fatalf("orders esm=%v comm=%v peri=%v", esm.Order, comm.Order, peri.Order)
	}
	if player.Damage.Destroyed(world.SysESM) || player.Damage.Destroyed(world.SysCOMM) || player.Damage.Destroyed(world.SysPeriscope) {
		t.Fatal("masts must not be damaged by auto-retract")
	}
}

func TestAutoProtectTowedOnSpeed(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Status: world.StatusActive,
		SpeedKts: 21, OrderedSpeed: 21, Damage: world.NewFullHealth(),
	}
	sonar := NewSonarState()
	sonar.TowedCablePct = 1
	sonar.TowedCableRate = 0

	evs := AutoProtectExtendedGear(player, nil, nil, nil, &sonar)
	if len(evs) != 1 || evs[0] != EventAutoRetractTowed {
		t.Fatalf("events=%v", evs)
	}
	if sonar.TowedCableRate >= 0 {
		t.Fatal("expected retract in progress")
	}
	if sonar.TowedDamaged {
		t.Fatal("towed must not be damaged")
	}
}

func TestAutoProtectBothMastsAndTowed(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Status: world.StatusActive,
		DepthFt: 50, SpeedKts: 22, OrderedSpeed: 22, Damage: world.NewFullHealth(),
	}
	var esm ESMState
	esm.Extension = 1
	esm.Order = ESMMastRaise
	sonar := NewSonarState()
	sonar.TowedCablePct = 1

	evs := AutoProtectExtendedGear(player, &esm, nil, nil, &sonar)
	if len(evs) != 1 || evs[0] != EventAutoRetractBoth {
		t.Fatalf("events=%v", evs)
	}
}

func TestAutoProtectMastsOnOrderedDepth(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Status: world.StatusActive,
		DepthFt: 55, OrderedDepth: 90, SpeedKts: 5, Damage: world.NewFullHealth(),
	}
	var esm ESMState
	esm.Extension = 1
	esm.Order = ESMMastRaise

	evs := AutoProtectExtendedGear(player, &esm, nil, nil, nil)
	if len(evs) != 1 {
		t.Fatalf("expected mast retract on ordered depth, events=%v", evs)
	}
}

func TestAutoProtectNoSpamWhileLowering(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Status: world.StatusActive,
		SpeedKts: 12, Damage: world.NewFullHealth(),
	}
	var esm ESMState
	esm.Extension = 0.8
	esm.Order = ESMMastStow

	if evs := AutoProtectExtendedGear(player, &esm, nil, nil, nil); len(evs) != 0 {
		t.Fatalf("already lowering should not re-notify: %v", evs)
	}
}

func TestAutoProtectMastThresholds(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Status: world.StatusActive,
		DepthFt: 50, SpeedKts: 8, OrderedSpeed: 8, Damage: world.NewFullHealth(),
	}
	var esm ESMState
	esm.Extension = 1
	esm.Order = ESMMastRaise

	if evs := AutoProtectExtendedGear(player, &esm, nil, nil, nil); len(evs) != 0 {
		t.Fatalf("8 kn / 50 ft should not retract: %v", evs)
	}
	player.SpeedKts = MastAutoRetractSpeedKts
	if evs := AutoProtectExtendedGear(player, &esm, nil, nil, nil); len(evs) == 0 {
		t.Fatal("8.5 kn should retract")
	}

	esm.Order = ESMMastRaise
	esm.Extension = 1
	player.SpeedKts = 5
	player.DepthFt = 64
	player.OrderedDepth = 64
	if evs := AutoProtectExtendedGear(player, &esm, nil, nil, nil); len(evs) != 0 {
		t.Fatalf("64 ft should not retract: %v", evs)
	}
	player.OrderedDepth = MastAutoRetractDepthFt
	if evs := AutoProtectExtendedGear(player, &esm, nil, nil, nil); len(evs) == 0 {
		t.Fatal("65 ft ordered should retract")
	}
}

func TestAdvanceMastMotionNoShearOnSpeed(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Status: world.StatusActive,
		DepthFt: 50, SpeedKts: 14, Damage: world.NewFullHealth(),
	}
	var esm ESMState
	esm.Extension = 1
	_, sheared := esm.AdvanceMastMotion(0.1, 10, player)
	if sheared || esm.Sheared || player.Damage.Destroyed(world.SysESM) {
		t.Fatal("AdvanceMastMotion must not shear on speed")
	}
}
