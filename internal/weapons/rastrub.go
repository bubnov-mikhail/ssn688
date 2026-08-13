package weapons

import (
	"fmt"
	"math"

	"github.com/ssn688/sim/internal/world"
)

// Lightweight ASW (UMGT-1) + Udaloy/Krivak delivery (Rastrub rocket / ship tubes).
const (
	UMGT1CruiseKts   = 42.0
	UMGT1MaxAgeSec   = 210.0 // ~4.5 kyd at cruise
	UMGT1SeekRangeYd = 1000.0
	UMGT1TubeClearYd = 40.0
	UMGT1ProximityYd = 70.0
	UMGT1ExitKts     = 22.0

	RastrubMagazineDefault  = 8
	ShipTubeMagazineDefault = 6

	RastrubMinRangeYd  = 2800.0
	RastrubMaxRangeYd  = 10000.0 // ~5 nm rocket reach (gameplay)
	ShipTubeMinRangeYd = 700.0
	ShipTubeMaxRangeYd = 2600.0

	RastrubFlightMinSec = 14.0
	RastrubFlightMaxSec = 28.0
)

// WeaponClass distinguishes heavy sub fish from lightweight ASW UMGT-1.
type WeaponClass int

const (
	ClassHeavy WeaponClass = iota // Mk48 / 53-65-style
	ClassUMGT1
)

// RastrubFlight is an in-air URPK-5 Rastrub rocket before the UMGT-1 enters the water.
type RastrubFlight struct {
	ID           string
	ParentID     string
	ParentSig    string // launcher SignatureID for splash fish type
	TargetID     string
	Side         world.Side
	X0, Y0       float64
	X1, Y1       float64
	LaunchAt     float64
	FlightSec    float64
	RunDepthFt   float64
	Alive        bool
}

func (a *RastrubFlight) Progress(gameTime float64) float64 {
	if a == nil || a.FlightSec <= 0 {
		return 1
	}
	t := (gameTime - a.LaunchAt) / a.FlightSec
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

func (a *RastrubFlight) Pos(gameTime float64) (x, y float64) {
	if a == nil {
		return 0, 0
	}
	p := a.Progress(gameTime)
	// Ease into splash (not a straight ballistic sim — readable plot trail).
	s := p * p * (3 - 2*p)
	return a.X0 + (a.X1-a.X0)*s, a.Y0 + (a.Y1-a.Y0)*s
}

func (fc *FireControl) rastrubAmmo(ship *world.Entity) int {
	if ship == nil {
		return 0
	}
	if fc.EnemyRastrub == nil {
		fc.EnemyRastrub = map[string]int{}
	}
	if v, ok := fc.EnemyRastrub[ship.ID]; ok {
		return v
	}
	n := RastrubMagazineFor(ship.SignatureID)
	fc.EnemyRastrub[ship.ID] = n
	return n
}

func (fc *FireControl) shipTubeAmmo(ship *world.Entity) int {
	if ship == nil {
		return 0
	}
	if fc.EnemyShipTube == nil {
		fc.EnemyShipTube = map[string]int{}
	}
	if v, ok := fc.EnemyShipTube[ship.ID]; ok {
		return v
	}
	n := ShipTubeMagazineFor(ship.SignatureID)
	fc.EnemyShipTube[ship.ID] = n
	return n
}

// LaunchRastrub fires a rocket toward a lead point near the target. Returns nil if dry.
func (fc *FireControl) LaunchRastrub(ship, target *world.Entity, gameTime float64) *RastrubFlight {
	if ship == nil || target == nil || !ship.Alive() || !target.Alive() {
		return nil
	}
	if !SurfaceHasRastrub(ship.SignatureID) {
		return nil
	}
	left := fc.rastrubAmmo(ship)
	if left <= 0 {
		return nil
	}
	rangeYd := math.Hypot(target.X-ship.X, target.Y-ship.Y)
	if rangeYd < RastrubMinRangeYd || rangeYd > RastrubMaxRangeYd {
		return nil
	}
	fc.EnemyRastrub[ship.ID] = left - 1

	// Crude lead along target heading.
	flight := RastrubFlightMinSec + (rangeYd-RastrubMinRangeYd)/(RastrubMaxRangeYd-RastrubMinRangeYd)*(RastrubFlightMaxSec-RastrubFlightMinSec)
	spdYdps := target.SpeedKts * world.YardsPerNM / 3600
	hrad := target.HeadingDeg * math.Pi / 180
	leadX := target.X + math.Sin(hrad)*spdYdps*flight*0.45
	leadY := target.Y + math.Cos(hrad)*spdYdps*flight*0.45
	splashBrg := bearing(ship.X, ship.Y, leadX, leadY)
	splashR := math.Hypot(leadX-ship.X, leadY-ship.Y)
	if splashR < RastrubMinRangeYd {
		splashR = RastrubMinRangeYd
	}
	if splashR > RastrubMaxRangeYd {
		splashR = RastrubMaxRangeYd
	}
	rad := splashBrg * math.Pi / 180
	x1 := ship.X + math.Sin(rad)*splashR
	y1 := ship.Y + math.Cos(rad)*splashR

	flight = RastrubFlightMinSec + (splashR-RastrubMinRangeYd)/(RastrubMaxRangeYd-RastrubMinRangeYd)*(RastrubFlightMaxSec-RastrubFlightMinSec)
	fc.torpedoSeq++
	a := &RastrubFlight{
		ID:         fmt.Sprintf("RASTRUB-%d", fc.torpedoSeq),
		ParentID:   ship.ID,
		ParentSig:  ship.SignatureID,
		TargetID:   target.ID,
		Side:       ship.Side,
		X0:         ship.X,
		Y0:         ship.Y,
		X1:         x1,
		Y1:         y1,
		LaunchAt:   gameTime,
		FlightSec:  flight,
		RunDepthFt: target.DepthFt,
		Alive:      true,
	}
	fc.ActiveRastrub = append(fc.ActiveRastrub, a)
	return a
}

// LaunchShipTube drops a short-range lightweight ASW fish from ship tubes.
func (fc *FireControl) LaunchShipTube(ship, target *world.Entity) *Torpedo {
	if ship == nil || target == nil || !ship.Alive() || !target.Alive() {
		return nil
	}
	left := fc.shipTubeAmmo(ship)
	if left <= 0 {
		return nil
	}
	rangeYd := math.Hypot(target.X-ship.X, target.Y-ship.Y)
	if rangeYd < ShipTubeMinRangeYd || rangeYd > ShipTubeMaxRangeYd {
		return nil
	}
	fc.EnemyShipTube[ship.ID] = left - 1

	brg := bearing(ship.X, ship.Y, target.X, target.Y)
	rad := brg * math.Pi / 180
	x := ship.X + math.Sin(rad)*40
	y := ship.Y + math.Cos(rad)*40
	return fc.spawnLightweight(ship.ID, ship.SignatureID, ship.Side, target.ID, x, y, brg, target.DepthFt, true)
}

// AdvanceRastrub moves rockets and returns newly waterborne lightweight ASW fish.
func (fc *FireControl) AdvanceRastrub(gameTime float64) []*Torpedo {
	if len(fc.ActiveRastrub) == 0 {
		return nil
	}
	var spawned []*Torpedo
	alive := fc.ActiveRastrub[:0]
	for _, a := range fc.ActiveRastrub {
		if a == nil || !a.Alive {
			continue
		}
		if gameTime < a.LaunchAt+a.FlightSec {
			alive = append(alive, a)
			continue
		}
		a.Alive = false
		fish := fc.spawnLightweight(a.ParentID, a.ParentSig, a.Side, a.TargetID, a.X1, a.Y1, bearing(a.X0, a.Y0, a.X1, a.Y1), a.RunDepthFt, false)
		if fish != nil {
			spawned = append(spawned, fish)
		}
	}
	fc.ActiveRastrub = alive
	return spawned
}

func (fc *FireControl) spawnLightweight(parentID, parentSig string, side world.Side, targetID string, x, y, headingDeg, runDepthFt float64, fromShipTube bool) *Torpedo {
	fc.torpedoSeq++
	if runDepthFt < 60 {
		runDepthFt = 60
	}
	clear := 0.0
	if !fromShipTube {
		clear = UMGT1TubeClearYd
	}
	sig := LightweightTorpedoSignature(parentSig)
	idPrefix := "UMGT1"
	if sig == "set40" {
		idPrefix = "SET40"
	}
	if sig == "mk46" {
		idPrefix = "MK46"
	}
	torp := &Torpedo{
		ID:                     fmt.Sprintf("%s-%d", idPrefix, fc.torpedoSeq),
		ParentSubID:            parentID,
		TargetID:               targetID,
		Side:                   side,
		Class:                  ClassUMGT1,
		AcousticSig:            sig,
		X:                      x,
		Y:                      y,
		DepthFt:                math.Min(40, runDepthFt),
		HeadingDeg:             normalizeAngle(headingDeg),
		OrderedHead:            normalizeAngle(headingDeg),
		LaunchHeadDeg:          normalizeAngle(headingDeg),
		GyroCourseDeg:          normalizeAngle(headingDeg),
		SpeedKts:               UMGT1ExitKts,
		CruiseKts:              UMGT1CruiseKts,
		RunDepthFt:             runDepthFt,
		SeekerOn:               !fromShipTube,
		WireCut:                true,
		Armed:                  true,
		Mode:                   ModeSearch,
		Alive:                  true,
		LastPingTime:           -1,
		ClearDistYd:            clear,
		EnableSearchAfterClear: true,
	}
	if !fromShipTube {
		torp.MarkGyroEnabled(true)
		torp.SeekerOn = true
		torp.Mode = ModeSearch
	}
	fc.ActiveTorpedoes = append(fc.ActiveTorpedoes, torp)
	return torp
}

// HasRecentSurfaceASW is true if this ship recently launched Rastrub, RBU, or lightweight fish.
func (fc *FireControl) HasRecentSurfaceASW(shipID string, maxAge float64) bool {
	for _, t := range fc.ActiveTorpedoes {
		if t != nil && t.Alive && t.ParentSubID == shipID && t.Class == ClassUMGT1 && t.Age < maxAge {
			return true
		}
	}
	for _, a := range fc.ActiveRastrub {
		if a != nil && a.Alive && a.ParentID == shipID {
			return true
		}
	}
	for _, a := range fc.ActiveRBU {
		if a != nil && a.Alive && a.ParentID == shipID {
			return true
		}
	}
	return false
}
