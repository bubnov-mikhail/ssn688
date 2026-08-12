package world

import "math"

// AITrack is an enemy crew's estimated solution on the player (not truth).
// Quality is driven by CrewSkill: green crews get noisy bearings, weak TMA,
// and low classification confidence; veterans converge near truth.
type AITrack struct {
	Valid     bool
	X, Y      float64 // estimated target world position (yards)
	CourseDeg float64
	SpeedKts  float64
	DepthFt   float64
	// ClassConf 0..1 — trust that the contact is a hostile combatant / usable fire solution.
	ClassConf float64
	HoldSec   float64 // continuous detection dwell
	UpdatedAt float64
}

// CrewSkill01 returns CrewSkill as 0..1.
func (e *Entity) CrewSkill01() float64 {
	if e == nil {
		return 0.5
	}
	s := e.CrewSkill / 100
	if s < 0 {
		return 0
	}
	if s > 1 {
		return 1
	}
	return s
}

// RandomCrewSkill returns base ± jitter, clamped to [0,100].
func RandomCrewSkill(base, jitter, u01 float64) float64 {
	if u01 < 0 {
		u01 = 0
	}
	if u01 > 1 {
		u01 = 1
	}
	v := base + (u01*2-1)*jitter
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// BearingDegFrom returns estimated true bearing from observer to the track.
func (t AITrack) BearingDegFrom(ox, oy float64) float64 {
	deg := math.Atan2(t.X-ox, t.Y-oy) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}

// RangeYdFrom returns estimated range from observer to the track.
func (t AITrack) RangeYdFrom(ox, oy float64) float64 {
	return math.Hypot(t.X-ox, t.Y-oy)
}

// GhostTarget builds a disposable aim entity from the track for weapons helpers.
func (t AITrack) GhostTarget(id string) *Entity {
	return &Entity{
		ID: id, Name: "AI-TRACK", Kind: KindSubmarine, Side: SidePlayer,
		Status: StatusActive, X: t.X, Y: t.Y, DepthFt: t.DepthFt,
		HeadingDeg: t.CourseDeg, SpeedKts: t.SpeedKts,
		OrderedHead: t.CourseDeg, OrderedSpeed: t.SpeedKts, OrderedDepth: t.DepthFt,
	}
}
