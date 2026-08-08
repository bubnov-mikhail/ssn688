package weapons

import (
	"math"
	"math/rand"
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func testDD(id string, x, y, heading float64, defcon int) *world.Entity {
	e := &world.Entity{
		ID: id, Name: "DD "+id, Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		X: x, Y: y, HeadingDeg: heading, SpeedKts: 12, DepthFt: 0, Defcon: defcon,
	}
	e.EnsureDamage()
	return e
}

func inboundHarpoon(x, y, heading float64) *HarpoonMissile {
	return &HarpoonMissile{
		ID: "H-1", ParentSubID: "P", Alive: true, Phase: HarpoonCruise,
		X: x, Y: y, LaunchX: x, LaunchY: y, HeadingDeg: heading, ProgrammedHead: heading,
		SpeedKts: HarpoonCruiseKts, VisibleOnWEPS: true, RadarOn: true,
		DestructRangeYd: HarpoonMaxRangeYd, BeamHalfDeg: HarpoonWideBeamDeg,
	}
}

func TestPointDefensePkBand(t *testing.T) {
	for _, layer := range []string{"SAM", "CIWS"} {
		for _, r := range []float64{CIWSMinRangeYd, 500, 900, SAMMinRangeYd, 5000, SAMMaxRangeYd} {
			pk := pointDefensePk(r, layer, false, pdPkMin, pdPkMax)
			if pk < pdPkMin-1e-9 || pk > pdPkMax+1e-9 {
				t.Fatalf("%s @ %.0f yd: pk=%.3f outside [%.2f, %.2f]", layer, r, pk, pdPkMin, pdPkMax)
			}
		}
		pkHot := pointDefensePk(SAMMinRangeYd, layer, true, 0.15, 0.40)
		if pkHot < pdPkMin || pkHot > pdPkMax {
			t.Fatalf("radar-hot pk clamped out of band: %.3f", pkHot)
		}
	}
}

func TestPointDefenseThreatCone(t *testing.T) {
	ship := testDD("dd1", 5000, 0, 0, world.DefconWeaponsFree)
	h := inboundHarpoon(0, 0, 90) // +X east
	if !harpoonThreatensShip(h, ship) {
		t.Fatal("expected inbound heading to threaten ship")
	}
	h.HeadingDeg = 0
	if harpoonThreatensShip(h, ship) {
		t.Fatal("abeam heading should not threaten")
	}
}

func TestPointDefenseRequiresDefconAttack(t *testing.T) {
	fc := NewFireControl()
	ship := testDD("dd1", 4000, 0, 0, world.DefconHostile)
	h := inboundHarpoon(0, 0, 90)
	rng := rand.New(rand.NewSource(1))
	if det := fc.TryPointDefense(h, []*world.Entity{ship}, 100, rng); det != nil {
		t.Fatal("DEFCON < weapons-free must not engage")
	}
	ship.Defcon = world.DefconWeaponsFree
	hits := 0
	for i := 0; i < 40; i++ {
		fc.EnemyPDEngageAt = map[string]float64{}
		fc.EnemySAM["dd1"] = SAMMagazineDefault
		m := inboundHarpoon(0, 0, 90)
		if det := fc.TryPointDefense(m, []*world.Entity{ship}, float64(100+i*20), rng); det != nil {
			hits++
			if !det.Intercepted {
				t.Fatal("expected Intercepted flag")
			}
			break
		}
	}
	if hits == 0 {
		t.Fatal("expected at least one SAM intercept with weapons-free DEFCON")
	}
}

func TestPointDefenseDebrisCloseIn(t *testing.T) {
	fc := NewFireControl()
	ship := testDD("dd1", 600, 0, 0, world.DefconWeaponsFree)
	rng := rand.New(rand.NewSource(42))
	var det *Detonation
	for i := 0; i < 80; i++ {
		fc.EnemyPDEngageAt = map[string]float64{}
		fc.EnemyCIWS["dd1"] = CIWSBurstDefault
		h := inboundHarpoon(0, 0, 90)
		if d := fc.TryPointDefense(h, []*world.Entity{ship}, float64(i*10), rng); d != nil {
			det = d
			break
		}
	}
	if det == nil {
		t.Fatal("expected CIWS intercept within debris range")
	}
	if !det.Debris || det.Hit == nil {
		t.Fatalf("close-in kill should flag debris damage: %+v", det)
	}
	dist := math.Hypot(ship.X-0, ship.Y-0)
	if dist > DebrisDamageMaxYd {
		t.Fatalf("test setup range %.0f > debris max", dist)
	}
}

func TestPointDefenseSAMNoDebrisAtLongRange(t *testing.T) {
	fc := NewFireControl()
	ship := testDD("dd1", 5000, 0, 0, world.DefconWeaponsFree)
	rng := rand.New(rand.NewSource(7))
	var det *Detonation
	for i := 0; i < 80; i++ {
		fc.EnemyPDEngageAt = map[string]float64{}
		fc.EnemySAM["dd1"] = SAMMagazineDefault
		h := inboundHarpoon(0, 0, 90)
		if d := fc.TryPointDefense(h, []*world.Entity{ship}, float64(i*15), rng); d != nil {
			det = d
			break
		}
	}
	if det == nil {
		t.Fatal("expected SAM intercept")
	}
	if det.Debris {
		t.Fatal("long-range SAM kill should not apply debris")
	}
	if !det.Intercepted || !det.SelfKill {
		t.Fatal("clean intercept should be Intercepted+SelfKill")
	}
}
