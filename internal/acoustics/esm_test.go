package acoustics_test

import (
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/world"
)

func TestRadarScanPeriodsByClass(t *testing.T) {
	cases := map[string]float64{
		"udaloy":   6.0,  // ~6–12 rpm air/surface
		"grisha":   4.0,  // ~15 rpm surface search
		"gorshkov": 4.5,  // Poliment / Furke
		"spruance": 5.0,  // SPS-40 / SPS-55
		"merchant": 2.5,  // ~24 rpm nav
		"tanker":   3.0,
		"fishing":  2.5,
		"krivak":   5.0,
	}
	for sig, want := range cases {
		p, ok := world.RadarBySignature(sig)
		if !ok {
			t.Fatalf("%s: missing radar profile", sig)
		}
		if p.ScanPeriodSec != want {
			t.Fatalf("%s: period=%.1f want %.1f", sig, p.ScanPeriodSec, want)
		}
	}
}

func TestESMMastRaiseGates(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, DepthFt: 180, SpeedKts: 5, Damage: world.NewFullHealth(),
	}
	var esm acoustics.ESMState
	if ok, _ := esm.OrderRaiseESM(player); ok {
		t.Fatal("should refuse raise at 180 ft")
	}
	player.DepthFt = 60
	player.SpeedKts = 12
	if ok, _ := esm.OrderRaiseESM(player); ok {
		t.Fatal("should refuse raise at 12 kn")
	}
	player.SpeedKts = 6
	if ok, msg := esm.OrderRaiseESM(player); !ok {
		t.Fatalf("expected raise ok: %s", msg)
	}
}

func TestESMMastShearsOnSpeed(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, DepthFt: 60, SpeedKts: 6, Damage: world.NewFullHealth(),
	}
	var esm acoustics.ESMState
	esm.OrderRaiseESM(player)
	esm.Extension = 1
	player.SpeedKts = 14 // well past 8+1.5
	evs, sheared := esm.AdvanceMastMotion(0.1, 10, player)
	if !sheared {
		t.Fatalf("expected shear, events=%v", evs)
	}
	if player.Damage.EffOf(world.SysESM) > world.RepairThresholdPct {
		t.Fatal("SysESM should be destroyed")
	}
}

func TestStormReducesMastDetect(t *testing.T) {
	calm := world.WeatherCalm.MastDetectFactor()
	storm := world.WeatherStorm.MastDetectFactor()
	if storm >= calm {
		t.Fatalf("storm factor %.2f should be < calm %.2f", storm, calm)
	}
}

func TestMerchantESMDetectableNear(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, DepthFt: 60, HeadingDeg: 0, SpeedKts: 0,
		Damage: world.NewFullHealth(),
	}
	mv := &world.Entity{
		ID: "civ_merchant", Name: "MV", Kind: world.KindSurfaceShip, Side: world.SideNeutral,
		Status: world.StatusActive, SignatureID: "merchant",
		X: 0, Y: 500, // 500 yd north
	}
	var esm acoustics.ESMState
	esm.Order = acoustics.ESMMastRaise
	esm.Extension = 1
	sonar := &acoustics.SonarState{}

	hit := false
	for step := 0; step < 80; step++ {
		gt := float64(step) * 0.1
		acoustics.UpdateESM(sonar, &esm, player, []*world.Entity{mv}, world.WeatherLight, gt, 0.1)
		if esm.HasRecentRF(mv.ID, gt) {
			hit = true
			break
		}
	}
	if !hit {
		t.Fatal("merchant nav radar should paint ESM within one scan at 500 yd")
	}
}

func TestESMBearingFrozenBetweenPaints(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, DepthFt: 60, HeadingDeg: 0,
		Damage: world.NewFullHealth(),
	}
	mv := &world.Entity{
		ID: "civ_merchant", Kind: world.KindSurfaceShip, Side: world.SideNeutral,
		Status: world.StatusActive, SignatureID: "merchant",
		X: 0, Y: 400,
	}
	var esm acoustics.ESMState
	esm.Order = acoustics.ESMMastRaise
	esm.Extension = 1
	sonar := &acoustics.SonarState{}

	var firstBrg float64
	var firstAt float64
	got := false
	for step := 0; step < 100; step++ {
		gt := float64(step) * 0.05
		acoustics.UpdateESM(sonar, &esm, player, []*world.Entity{mv}, world.WeatherLight, gt, 0.1)
		if esm.HasRecentRF(mv.ID, gt) {
			firstBrg = esm.FrozenRFBearing(mv.ID, -1)
			firstAt = gt
			got = true
			break
		}
	}
	if !got {
		t.Fatal("expected initial RF paint")
	}

	// Move emitter; sidelobes may heat but must not move frozen bearing until next main-beam paint.
	mv.X = 300
	for step := 0; step < 30; step++ {
		gt := firstAt + 0.05*float64(step+1)
		if gt > firstAt+2.0 {
			break // next merchant scan ~2.5 s
		}
		if world.RadarBeamPassed(mv, gt, 0.05, mv.BearingDegTo(player)) {
			continue
		}
		acoustics.UpdateESM(sonar, &esm, player, []*world.Entity{mv}, world.WeatherLight, gt, 0.05)
		brg := esm.FrozenRFBearing(mv.ID, -1)
		if brg != firstBrg {
			t.Fatalf("bearing moved between paints: %.1f → %.1f", firstBrg, brg)
		}
	}
}

func TestESMRFClassIsEquipmentNotHull(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, DepthFt: 60, HeadingDeg: 0,
		Damage: world.NewFullHealth(),
	}
	mv := &world.Entity{
		ID: "civ_merchant", Kind: world.KindSurfaceShip, Side: world.SideNeutral,
		Status: world.StatusActive, SignatureID: "merchant",
		X: 0, Y: 400,
	}
	var esm acoustics.ESMState
	esm.Order = acoustics.ESMMastRaise
	esm.Extension = 1
	sonar := &acoustics.SonarState{}

	locked := false
	for step := 0; step < 200; step++ {
		gt := float64(step) * 0.1
		acoustics.UpdateESM(sonar, &esm, player, []*world.Entity{mv}, world.WeatherLight, gt, 0.1)
		if esm.RFEquipmentClass(mv.ID) != "" {
			locked = true
			break
		}
	}
	if !locked {
		t.Fatal("expected RF equipment class lock")
	}
	got := esm.RFEquipmentClass(mv.ID)
	if got != "Commercial X-band Nav" {
		t.Fatalf("RF equip=%q want Commercial X-band Nav", got)
	}
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if c.SourceEntityID != mv.ID {
			continue
		}
		if c.ConfirmedClass != "" {
			t.Fatalf("ESM must not set ConfirmedClass (got %q)", c.ConfirmedClass)
		}
		if c.BestMatchID != "" || c.BestMatchName != "" {
			t.Fatalf("ESM must not seed acoustic BestMatch (%q/%q)", c.BestMatchID, c.BestMatchName)
		}
	}
}

func TestEnemyRadarDetectsSurface(t *testing.T) {
	ship := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "spruance",
		X: 0, Y: 0, HeadingDeg: 0,
	}
	tgt := &world.Entity{
		ID: "grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "grisha",
		X: 0, Y: 5000,
	}
	hit := false
	for step := 0; step < 80; step++ {
		gt := float64(step) * 0.1
		if acoustics.EnemyRadarDetectsSurface(ship, tgt, gt, 0.1) {
			hit = true
			break
		}
	}
	if !hit {
		t.Fatal("expected surface radar paint within MaxRangeYd")
	}
	sub := &world.Entity{
		ID: "ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 2000, DepthFt: 180,
	}
	if acoustics.EnemyRadarDetectsSurface(ship, sub, 10, 0.1) {
		t.Fatal("surface radar must not paint submerged contacts")
	}
}
