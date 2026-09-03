package weapons

import (
	"math"
	"math/rand"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// Udaloy/Krivak point defense vs inbound anti-ship missiles (Kinzhal/Osa-M + AK-630).
const (
	SAMMagazineDefault  = 8
	CIWSBurstDefault    = 10
	SAMCooldownSec      = 12.0
	CIWSCooldownSec     = 4.0

	SAMMinRangeYd  = 2000.0
	SAMMaxRangeYd  = 8000.0
	CIWSMinRangeYd = 200.0
	CIWSMaxRangeYd = 2000.0 // meets SAM min — continuous coverage

	// DebrisDamageMaxYd — intercept inside this range can rain fragments on the ship.
	DebrisDamageMaxYd = 900.0

	pointDefenseAimConeDeg = 48.0

	// Kill probability band (user: 15–50%).
	pdPkMin = 0.15
	pdPkMax = 0.50
)

// PointDefenseIntercept describes a successful SAM/CIWS kill of a Harpoon.
type PointDefenseIntercept struct {
	ShipID   string
	Layer    string // "SAM" or "CIWS"
	Missile  *HarpoonMissile
	Debris   bool // close-in kill — debris damage to ship
	RangeYd  float64
}

func (fc *FireControl) samAmmo(ship *world.Entity) int {
	if ship == nil {
		return 0
	}
	if fc.EnemySAM == nil {
		fc.EnemySAM = map[string]int{}
	}
	if v, ok := fc.EnemySAM[ship.ID]; ok {
		return v
	}
	n := SAMMagazineFor(ship.SignatureID)
	fc.EnemySAM[ship.ID] = n
	return n
}

func (fc *FireControl) ciwsAmmo(ship *world.Entity) int {
	if ship == nil {
		return 0
	}
	if fc.EnemyCIWS == nil {
		fc.EnemyCIWS = map[string]int{}
	}
	if v, ok := fc.EnemyCIWS[ship.ID]; ok {
		return v
	}
	n := CIWSMagazineFor(ship.SignatureID)
	fc.EnemyCIWS[ship.ID] = n
	return n
}

func (fc *FireControl) lastPDEngage(shipID string) float64 {
	if fc.EnemyPDEngageAt == nil {
		return 0
	}
	return fc.EnemyPDEngageAt[shipID]
}

func (fc *FireControl) markPDEngage(shipID string, gameTime float64) {
	if fc.EnemyPDEngageAt == nil {
		fc.EnemyPDEngageAt = map[string]float64{}
	}
	fc.EnemyPDEngageAt[shipID] = gameTime
}

// TryPointDefense attempts SAM then CIWS against a cruise-phase Harpoon.
// Returns a detonation if the missile is killed (optional debris damage on Hit).
func (fc *FireControl) TryPointDefense(h *HarpoonMissile, ships []*world.Entity, gameTime float64, rng *rand.Rand) *Detonation {
	if h == nil || !h.Alive || h.Phase != HarpoonCruise {
		return nil
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	var best *world.Entity
	bestDist := math.MaxFloat64
	for _, ship := range ships {
		if ship == nil || !ship.Alive() || ship.Kind != world.KindSurfaceShip {
			continue
		}
		if ship.Side != world.SideEnemy && ship.Side != world.SidePlayer {
			continue
		}
		if h.Side == ship.Side {
			continue // IFF — do not engage own-side missiles
		}
		if !ship.CanDefconAttack() {
			continue
		}
		if !harpoonThreatensShip(h, ship) {
			continue
		}
		d := math.Hypot(ship.X-h.X, ship.Y-h.Y)
		if d < bestDist {
			bestDist = d
			best = ship
		}
	}
	if best == nil {
		return nil
	}

	layer := ""
	cooldown := 0.0
	pkLo, pkHi := pdPkMin, pdPkMax
	inSAM := bestDist >= SAMMinRangeYd && bestDist <= SAMMaxRangeYd
	inCIWS := bestDist >= CIWSMinRangeYd && bestDist <= CIWSMaxRangeYd
	switch {
	case inSAM && fc.samAmmo(best) > 0:
		cooldown = SAMCooldownSec
		layer = "SAM"
		// Mid-band a bit better than long edge; still within 15–50%.
		pkLo, pkHi = 0.15, 0.40
	case inCIWS && fc.ciwsAmmo(best) > 0:
		cooldown = CIWSCooldownSec
		layer = "CIWS"
		pkLo, pkHi = 0.25, 0.50
	}
	if layer == "" {
		return nil
	}
	if gameTime-fc.lastPDEngage(best.ID) < cooldown {
		return nil
	}

	fc.markPDEngage(best.ID, gameTime)
	label := "s"
	if layer == "CIWS" {
		label = "c"
	}
	fc.PushDebugMapFlash(h.X, h.Y, label, gameTime)
	if layer == "SAM" {
		fc.EnemySAM[best.ID] = fc.samAmmo(best) - 1
	} else {
		fc.EnemyCIWS[best.ID] = fc.ciwsAmmo(best) - 1
	}

	pk := pointDefensePk(bestDist, layer, h.RadarOn || h.LockedTargetID != "", pkLo, pkHi)
	if rng.Float64() > pk {
		return nil // shot expended, miss
	}

	h.Alive = false
	// Keep VisibleOnWEPS — player only sees the assumed track until DSTR / blast heard.
	debris := bestDist <= DebrisDamageMaxYd
	det := &Detonation{
		X: h.X, Y: h.Y, DepthFt: 0,
		SelfKill:    true, // no full warhead blast / deaf
		Intercepted: true,
		ShooterID:   h.ParentSubID,
	}
	if debris {
		det.SelfKill = false
		det.Debris = true
		det.Hit = best
		det.Harpoon = false
	}
	return det
}

func harpoonThreatensShip(h *HarpoonMissile, ship *world.Entity) bool {
	if h == nil || ship == nil {
		return false
	}
	brg := bearing(h.X, h.Y, ship.X, ship.Y)
	if math.Abs(shortestAngleDiff(h.HeadingDeg, brg)) > pointDefenseAimConeDeg {
		return false
	}
	return true
}

func pointDefensePk(rangeYd float64, layer string, radarHot bool, pkLo, pkHi float64) float64 {
	var t float64
	switch layer {
	case "SAM":
		span := SAMMaxRangeYd - SAMMinRangeYd
		if span < 1 {
			span = 1
		}
		// Closer to min range → higher Pk.
		t = 1 - (rangeYd-SAMMinRangeYd)/span
	default:
		span := CIWSMaxRangeYd - CIWSMinRangeYd
		if span < 1 {
			span = 1
		}
		t = 1 - (rangeYd-CIWSMinRangeYd)/span
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	pk := pkLo + t*(pkHi-pkLo)
	if radarHot {
		pk += 0.05
	}
	if pk < pdPkMin {
		pk = pdPkMin
	}
	if pk > pdPkMax {
		pk = pdPkMax
	}
	return pk
}
