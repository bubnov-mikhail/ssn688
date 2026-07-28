package world

import "math"

const (
	YardsPerNM   = 2025.37183
	FeetPerYard  = 3.0
	KnotsToYPS    = YardsPerNM / 3600.0 // yards per second at 1 knot
)

// Entity is any simulated object in the battlespace.
type Entity struct {
	ID           string
	Name         string
	Kind         EntityKind
	Side         Side
	Status       Status
	SignatureID  string
	X, Y         float64 // yards from origin (east, north)
	DepthFt      float64
	HeadingDeg   float64
	SpeedKts     float64
	OrderedSpeed float64
	OrderedDepth float64
	OrderedHead  float64
	LengthFt     float64
	ActiveSonar   bool
	LastPingTime  float64
	LastPingPower float64 // 0..1 transmit power of last active ping
	AIState       string
}

func (e *Entity) Alive() bool {
	return e.Status == StatusActive
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
	e.SpeedKts += clamp((e.OrderedSpeed-e.SpeedKts)*dt*0.15, -2*dt*10, 2*dt*10)
	e.DepthFt += clamp((e.OrderedDepth-e.DepthFt)*dt*8, -dt*30, dt*30)
	diff := shortestAngleDiff(e.HeadingDeg, e.OrderedHead)
	e.HeadingDeg += clamp(diff*dt*0.25, -dt*3, dt*3)
	e.HeadingDeg = normalizeAngle(e.HeadingDeg)

	rad := e.HeadingDeg * math.Pi / 180
	speedYPS := e.SpeedKts * KnotsToYPS
	e.X += math.Sin(rad) * speedYPS * dt
	e.Y += math.Cos(rad) * speedYPS * dt
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
