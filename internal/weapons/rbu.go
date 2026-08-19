package weapons

import (
	"fmt"
	"math"

	"github.com/ssn688/sim/internal/world"
)

// RBU-6000-style short-range ASW rocket barrage (Grisha). No swimming fish —
// a patterned splash that can shock a submerged contact near the aim point.
const (
	RBUMinRangeYd       = 400.0
	RBUMaxRangeYd       = 2200.0
	RBUFlightMinSec     = 4.0
	RBUFlightMaxSec     = 9.0
	RBUBlastRadiusYd    = 280.0
	RBUMagazineDefault  = 10
	// RBUMaxTargetDepthFt — RBU-6000 only shocks shallow boats (periscope depth is in the envelope).
	RBUMaxTargetDepthFt = 120.0
)

// PreferRBUOverShipTubes is true when a Grisha-class ship should bracket with rockets
// instead of lightweight tubes (overlap band is 700–2200 yd).
func PreferRBUOverShipTubes(ship *world.Entity, aiState string, targetDepthFt float64) bool {
	if ship == nil || !SurfaceHasRBU(ship.SignatureID) {
		return false
	}
	if aiState == "RBU" {
		return true
	}
	return targetDepthFt > 0 && targetDepthFt <= RBUMaxTargetDepthFt
}

// RBUSalvo is an in-air ASW rocket pattern before underwater detonation.
type RBUSalvo struct {
	ID         string
	ParentID   string
	TargetID   string
	Side       world.Side
	X0, Y0     float64
	X1, Y1     float64
	LaunchAt   float64
	FlightSec  float64
	Alive      bool
}

func (a *RBUSalvo) Progress(gameTime float64) float64 {
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

func (a *RBUSalvo) Pos(gameTime float64) (x, y float64) {
	if a == nil {
		return 0, 0
	}
	p := a.Progress(gameTime)
	s := p * p * (3 - 2*p)
	return a.X0 + (a.X1-a.X0)*s, a.Y0 + (a.Y1-a.Y0)*s
}

func (fc *FireControl) rbuAmmo(ship *world.Entity) int {
	if ship == nil {
		return 0
	}
	if fc.EnemyRBU == nil {
		fc.EnemyRBU = map[string]int{}
	}
	if v, ok := fc.EnemyRBU[ship.ID]; ok {
		return v
	}
	n := RBUMagazineFor(ship.SignatureID)
	fc.EnemyRBU[ship.ID] = n
	return n
}

// LaunchRBU fires a short-range ASW rocket pattern toward a lead near the target.
func (fc *FireControl) LaunchRBU(ship, target *world.Entity, gameTime float64) *RBUSalvo {
	if ship == nil || target == nil || !ship.Alive() || !target.Alive() {
		return nil
	}
	if !SurfaceHasRBU(ship.SignatureID) {
		return nil
	}
	left := fc.rbuAmmo(ship)
	if left <= 0 {
		return nil
	}
	rangeYd := math.Hypot(target.X-ship.X, target.Y-ship.Y)
	if rangeYd < RBUMinRangeYd || rangeYd > RBUMaxRangeYd {
		return nil
	}
	fc.EnemyRBU[ship.ID] = left - 1

	flight := RBUFlightMinSec + (rangeYd-RBUMinRangeYd)/(RBUMaxRangeYd-RBUMinRangeYd)*(RBUFlightMaxSec-RBUFlightMinSec)
	spdYdps := target.SpeedKts * world.YardsPerNM / 3600
	hrad := target.HeadingDeg * math.Pi / 180
	leadX := target.X + math.Sin(hrad)*spdYdps*flight*0.35
	leadY := target.Y + math.Cos(hrad)*spdYdps*flight*0.35
	splashBrg := bearing(ship.X, ship.Y, leadX, leadY)
	splashR := math.Hypot(leadX-ship.X, leadY-ship.Y)
	if splashR < RBUMinRangeYd {
		splashR = RBUMinRangeYd
	}
	if splashR > RBUMaxRangeYd {
		splashR = RBUMaxRangeYd
	}
	rad := splashBrg * math.Pi / 180
	fc.torpedoSeq++
	a := &RBUSalvo{
		ID:        fmt.Sprintf("RBU-%d", fc.torpedoSeq),
		ParentID:  ship.ID,
		TargetID:  target.ID,
		Side:      ship.Side,
		X0:        ship.X,
		Y0:        ship.Y,
		X1:        ship.X + math.Sin(rad)*splashR,
		Y1:        ship.Y + math.Cos(rad)*splashR,
		LaunchAt:  gameTime,
		FlightSec: flight,
		Alive:     true,
	}
	fc.ActiveRBU = append(fc.ActiveRBU, a)
	fc.PushDebugMapFlash(a.X1, a.Y1, "RBU", gameTime)
	return a
}

// AdvanceRBU returns underwater detonations when salvos splash.
func (fc *FireControl) AdvanceRBU(gameTime float64, targets []*world.Entity) []*Detonation {
	if len(fc.ActiveRBU) == 0 {
		return nil
	}
	var dets []*Detonation
	alive := fc.ActiveRBU[:0]
	for _, a := range fc.ActiveRBU {
		if a == nil || !a.Alive {
			continue
		}
		if gameTime < a.LaunchAt+a.FlightSec {
			alive = append(alive, a)
			continue
		}
		a.Alive = false
		det := &Detonation{
			X: a.X1, Y: a.Y1, DepthFt: 80,
			ShooterID: a.ParentID,
			RBU:       true,
		}
		for _, t := range targets {
			if t == nil || !t.Alive() || t.Kind != world.KindSubmarine {
				continue
			}
			if t.DepthFt > RBUMaxTargetDepthFt {
				continue
			}
			if math.Hypot(t.X-a.X1, t.Y-a.Y1) <= RBUBlastRadiusYd {
				det.Hit = t
				break
			}
		}
		dets = append(dets, det)
	}
	fc.ActiveRBU = alive
	return dets
}
