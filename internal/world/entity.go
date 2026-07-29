package world

import "math"

const (
	YardsPerNM  = 2025.37183
	FeetPerYard = 3.0
	KnotsToYPS  = YardsPerNM / 3600.0 // yards per second at 1 knot
	// KnotsToFPM converts forward speed in knots to feet per minute
	// (1 nm ≈ 6076.12 ft → 101.2687 ft/min per knot).
	KnotsToFPM = YardsPerNM * FeetPerYard / 60.0

	// Depth-change model (open sources; exact 688 rates are classified):
	// Vertical speed ≈ forward_speed × sin(hull_angle).
	// Edward L. Beach / NSL: routine ~10°, normal dive limit ~15°, battle ~30°.
	// USNA / USNI: planes dominate; large angles restricted at high speed.
	// Aggressive diesel drills: ~250–350 ft/min sustained (e.g. 400→PD in ~90 s).
	DepthAngleRoutineDeg = 10.0 // modern routine acceptance (~Beach)
	DepthAngleMaxDeg     = 15.0 // normal dive limit (Amberjack doctrine)
	DepthRateMinFPM      = 35.0 // trim/pump crawl when nearly stopped
	DepthRateMaxFPM      = 320.0 // aggressive tactical ceiling (~drill figures)
)

// Entity is any simulated object in the battlespace.
type Entity struct {
	ID            string
	Name          string
	Kind          EntityKind
	Side          Side
	Status        Status
	SignatureID   string
	X, Y          float64 // yards from origin (east, north)
	DepthFt       float64
	HeadingDeg    float64
	SpeedKts      float64
	OrderedSpeed  float64
	OrderedDepth  float64
	OrderedHead   float64
	LengthFt      float64
	ActiveSonar   bool
	LastPingTime  float64
	LastPingPower float64 // 0..1 transmit power of last active ping
	AIState       string
	// Short-lived transient signature (tube doors, flooding valves, etc).
	TransientUntil   float64
	TransientFreqHz  float64
	TransientLevelDB float64
	// Sinking wreck noise / motion (StatusSinking).
	SinkRateFPM     float64 // feet per minute downward
	WreckNoiseUntil float64
}

func (e *Entity) Alive() bool {
	return e.Status == StatusActive
}

// InWater reports platforms that still occupy the battlespace (incl. sinking wrecks).
func (e *Entity) InWater() bool {
	return e.Status == StatusActive || e.Status == StatusSinking
}

func (e *Entity) RangeYardsTo(other *Entity) float64 {
	dx := other.X - e.X
	dy := other.Y - e.Y
	horiz := math.Hypot(dx, dy)
	dz := (other.DepthFt - e.DepthFt) / FeetPerYard
	return math.Hypot(horiz, dz)
}

func (e *Entity) BearingDegTo(other *Entity) float64 {
	dx := other.X - e.X
	dy := other.Y - e.Y
	deg := math.Atan2(dx, dy) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}

func (e *Entity) RelativeBearingDeg(other *Entity) float64 {
	b := e.BearingDegTo(other) - e.HeadingDeg
	for b > 180 {
		b -= 360
	}
	for b < -180 {
		b += 360
	}
	return b
}

func (e *Entity) Advance(dt float64) {
	if e.Status == StatusSinking {
		e.DepthFt += e.SinkRateFPM / 60 * dt
		e.SpeedKts *= math.Max(0, 1-0.15*dt)
		return
	}
	// Cap acceleration by platform class (open-order figures; exact rates classified).
	maxA := MaxSpeedAccelKtsPerSec(e)
	errSpd := e.OrderedSpeed - e.SpeedKts
	e.SpeedKts += clamp(errSpd, -maxA*dt, maxA*dt)
	if e.Kind != KindSurfaceShip {
		rateFPS := DepthChangeRateFPM(e.SpeedKts) / 60
		err := e.OrderedDepth - e.DepthFt
		e.DepthFt += clamp(err, -rateFPS*dt, rateFPS*dt)
	} else {
		e.DepthFt = 0
		e.OrderedDepth = 0
	}
	diff := shortestAngleDiff(e.HeadingDeg, e.OrderedHead)
	e.HeadingDeg += clamp(diff*dt*0.25, -dt*3, dt*3)
	e.HeadingDeg = normalizeAngle(e.HeadingDeg)

	rad := e.HeadingDeg * math.Pi / 180
	speedYPS := e.SpeedKts * KnotsToYPS
	e.X += math.Sin(rad) * speedYPS * dt
	e.Y += math.Cos(rad) * speedYPS * dt
}

// MaxSpeedAccelKtsPerSec is the magnitude limit on |dSpeed/dt| in knots/second.
// Order-of-magnitude from open naval literature (not classified 688 performance):
//   SSN: minutes to change speed bands (~0.05–0.12 kts/s)
//   Surface warship: somewhat quicker; large merchants slower
//   Torpedo: reaches cruise in ~10–20 s
func MaxSpeedAccelKtsPerSec(e *Entity) float64 {
	if e == nil {
		return 0.08
	}
	switch e.Kind {
	case KindSubmarine:
		return 0.08 // ~5 kts/min — several minutes all-stop → flank
	case KindTorpedo:
		return 4.0
	case KindSurfaceShip:
		if e.LengthFt >= 700 {
			return 0.035 // large merchant / tanker
		}
		if e.LengthFt >= 400 {
			return 0.06 // destroyer / cruiser class
		}
		return 0.09 // small craft / fishing
	default:
		return 0.08
	}
}

// DepthChangeRateFPM is the max sustained |dDepth/dt| in feet/minute for a
// plane-driven depth change at the given forward speed.
func DepthChangeRateFPM(speedKts float64) float64 {
	spd := math.Abs(speedKts)
	// Effective hull angle: routine ~10°, tapering at higher speed for control
	// stability (USNI: large plane/rudder angles restricted above ~15 kts).
	var angle float64
	if spd >= 6 && spd <= 12 {
		// Mild boost toward normal-dive limit when planes have good authority.
		angle = DepthAngleRoutineDeg + (DepthAngleMaxDeg-DepthAngleRoutineDeg)*((spd-6)/6)
	} else if spd > 12 {
		angle = DepthAngleMaxDeg - (spd-12)*0.5
		if angle < 5 {
			angle = 5
		}
	} else {
		// Slow speed: little hydrodynamic lift; rely on trim/pumps.
		angle = DepthAngleRoutineDeg * (spd / 6)
		if angle < 2 {
			angle = 2
		}
	}
	fpm := spd * math.Sin(angle*math.Pi/180) * KnotsToFPM
	if fpm < DepthRateMinFPM {
		fpm = DepthRateMinFPM
	}
	if fpm > DepthRateMaxFPM {
		fpm = DepthRateMaxFPM
	}
	return fpm
}

func normalizeAngle(a float64) float64 {
	for a >= 360 {
		a -= 360
	}
	for a < 0 {
		a += 360
	}
	return a
}

func shortestAngleDiff(from, to float64) float64 {
	diff := to - from
	for diff > 180 {
		diff -= 360
	}
	for diff < -180 {
		diff += 360
	}
	return diff
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
