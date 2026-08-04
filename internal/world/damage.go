package world

import (
	"fmt"
	"math"
	"math/rand"
)

// Platform systems that can be damaged by a warhead hit.
const (
	SysPassiveHull = iota // hull-mounted passive array
	SysTowed              // towed array
	SysActive             // active sonar
	SysTube1
	SysTube2
	SysTube3
	SysTube4
	SysDepth      // planes / ballast
	SysSteering   // rudder
	SysPropulsion // shaft / screw / plant
	SysHull       // pressure hull / buoyancy
	SysCount
)

const (
	// RepairThresholdPct — systems at or below this cannot be repaired
	// (destroyed ≥75% of effectiveness).
	RepairThresholdPct = 25.0
	// RepairMinToFullSec — game seconds to restore a system from RepairThresholdPct to 100%.
	RepairMinToFullSec = 45 * 60.0
	// CrushDepthFt — catastrophic depth for submarines.
	CrushDepthFt = 1200.0
	// SysNone marks "not repairing".
	SysNone = -1
)

// PlatformDamage tracks per-system effectiveness (0..100) and repair state.
type PlatformDamage struct {
	Initialized     bool
	Eff             [SysCount]float64
	Repairing       int     // Sys* index or SysNone
	DepthRunawayFPM float64 // when depth control critically lost (+dive / −rise)
	SteeringJamDeg  float64 // locked heading when rudder critically lost
	SteeringJammed  bool
}

// NewFullHealth returns an undamaged systems board.
func NewFullHealth() PlatformDamage {
	var d PlatformDamage
	d.Initialized = true
	d.Repairing = SysNone
	for i := range d.Eff {
		d.Eff[i] = 100
	}
	return d
}

// EnsureDamage initializes damage state for combatants that never received it.
func (e *Entity) EnsureDamage() {
	if e == nil {
		return
	}
	if !e.Damage.Initialized {
		e.Damage = NewFullHealth()
	}
}

// InitCombatantDamage sets full health for a living combatant.
func InitCombatantDamage(e *Entity) {
	if e == nil {
		return
	}
	e.Damage = NewFullHealth()
}

func (d *PlatformDamage) EffOf(sys int) float64 {
	if d == nil || sys < 0 || sys >= SysCount {
		return 100
	}
	return d.Eff[sys]
}

// Operational is true when the system retains usable capability (repairable band).
func (d *PlatformDamage) Operational(sys int) bool {
	return d.EffOf(sys) > RepairThresholdPct
}

// Repairable is true when efficiency is above the destruction threshold.
func (d *PlatformDamage) Repairable(sys int) bool {
	eff := d.EffOf(sys)
	return eff > RepairThresholdPct && eff < 100
}

func (d *PlatformDamage) Destroyed(sys int) bool {
	return d.EffOf(sys) <= RepairThresholdPct
}

func SystemName(sys int) string {
	switch sys {
	case SysPassiveHull:
		return "Hull Array"
	case SysTowed:
		return "Towed Array"
	case SysActive:
		return "Active Sonar"
	case SysTube1:
		return "Tube 1"
	case SysTube2:
		return "Tube 2"
	case SysTube3:
		return "Tube 3"
	case SysTube4:
		return "Tube 4"
	case SysDepth:
		return "Depth Control"
	case SysSteering:
		return "Steering"
	case SysPropulsion:
		return "Propulsion"
	case SysHull:
		return "Pressure Hull"
	default:
		return "Unknown"
	}
}

func SystemStatusLabel(d *PlatformDamage, sys int) string {
	eff := d.EffOf(sys)
	switch {
	case eff >= 99.5:
		return "NOMINAL"
	case eff > RepairThresholdPct:
		return "DEGRADED"
	case eff > 0.5:
		return "CRITICAL"
	default:
		return "DESTROYED"
	}
}

// StartRepair begins repairing one system. Returns false if busy/invalid.
func (d *PlatformDamage) StartRepair(sys int) (ok bool, reason string) {
	if d == nil || sys < 0 || sys >= SysCount {
		return false, "Invalid system."
	}
	if d.Repairing != SysNone {
		return false, fmt.Sprintf("Already repairing %s.", SystemName(d.Repairing))
	}
	if d.Eff[sys] >= 100 {
		return false, "System already at full efficiency."
	}
	if !d.Repairable(sys) {
		return false, fmt.Sprintf("%s destroyed beyond repair.", SystemName(sys))
	}
	d.Repairing = sys
	return true, ""
}

func (d *PlatformDamage) CancelRepair() {
	if d != nil {
		d.Repairing = SysNone
	}
}

// AdvanceRepair restores the active system. Rate: (100-25)/45min.
func (d *PlatformDamage) AdvanceRepair(dt float64) {
	if d == nil || d.Repairing < 0 || d.Repairing >= SysCount {
		return
	}
	sys := d.Repairing
	if !d.Repairable(sys) && d.Eff[sys] <= RepairThresholdPct {
		d.Repairing = SysNone
		return
	}
	rate := (100.0 - RepairThresholdPct) / RepairMinToFullSec // % per second
	d.Eff[sys] = math.Min(100, d.Eff[sys]+rate*dt)
	if d.Eff[sys] >= 100 {
		d.Eff[sys] = 100
		d.Repairing = SysNone
	}
}

// TubeIndex maps SysTube1..4 → 0..3, or -1.
func TubeIndex(sys int) int {
	if sys >= SysTube1 && sys <= SysTube4 {
		return sys - SysTube1
	}
	return -1
}

// TubeSys maps tube number 1..4 → SysTube*.
func TubeSys(tubeNum int) int {
	if tubeNum < 1 || tubeNum > 4 {
		return SysNone
	}
	return SysTube1 + (tubeNum - 1)
}

// ApplyTorpedoHit rolls random subsystem casualties. Returns true if the platform is killed.
func ApplyTorpedoHit(e *Entity, rng *rand.Rand) (fatal bool, events []string) {
	if e == nil || !e.Alive() {
		return false, nil
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	e.EnsureDamage()
	d := &e.Damage

	// Candidate systems by platform class.
	cands := hitCandidates(e.Kind)
	nHits := 3 + rng.Intn(4) // 3–6 systems touched
	if nHits > len(cands) {
		nHits = len(cands)
	}
	rng.Shuffle(len(cands), func(i, j int) { cands[i], cands[j] = cands[j], cands[i] })

	for i := 0; i < nHits; i++ {
		sys := cands[i]
		loss := 25.0 + rng.Float64()*75.0 // lose 25–100 points of efficiency
		// Hull hits tend to be serious but not always lethal on first fish.
		if sys == SysHull {
			if rng.Float64() < 0.12 {
				loss = 100
			} else {
				loss = 15.0 + rng.Float64()*55.0
			}
		}
		before := d.Eff[sys]
		d.Eff[sys] = math.Max(0, before-loss)
		after := d.Eff[sys]
		if after < before {
			events = append(events, fmt.Sprintf("%s: %s %.0f%% → %.0f%%", e.Name, SystemName(sys), before, after))
		}
		applySystemSideEffects(e, sys, before, after, rng)
	}

	// Always nudge hull a little on any hit (shock).
	if d.Eff[SysHull] > 0 {
		shock := 5.0 + rng.Float64()*15.0
		before := d.Eff[SysHull]
		d.Eff[SysHull] = math.Max(0, before-shock)
		if d.Eff[SysHull] < before {
			events = append(events, fmt.Sprintf("%s: hull shock %.0f%% → %.0f%%", e.Name, before, d.Eff[SysHull]))
		}
	}

	if d.Eff[SysHull] <= 0 {
		events = append(events, fmt.Sprintf("%s: HULL FATAL — flooding uncontrolled", e.Name))
		return true, events
	}
	return false, events
}

func hitCandidates(kind EntityKind) []int {
	base := []int{
		SysPassiveHull, SysActive,
		SysTube1, SysTube2, SysTube3, SysTube4,
		SysSteering, SysPropulsion, SysHull,
	}
	if kind == KindSubmarine {
		base = append(base, SysTowed, SysDepth)
	}
	return base
}

func applySystemSideEffects(e *Entity, sys int, before, after float64, rng *rand.Rand) {
	d := &e.Damage
	critNow := after <= RepairThresholdPct && before > RepairThresholdPct
	switch sys {
	case SysDepth:
		if critNow || (after <= RepairThresholdPct && d.DepthRunawayFPM == 0) {
			// Uncontrolled dive is more common than broach.
			if rng.Float64() < 0.7 {
				d.DepthRunawayFPM = 40 + rng.Float64()*90
			} else {
				d.DepthRunawayFPM = -(20 + rng.Float64()*50)
			}
		}
		if after > RepairThresholdPct {
			d.DepthRunawayFPM = 0
		}
	case SysSteering:
		if after <= RepairThresholdPct {
			d.SteeringJammed = true
			d.SteeringJamDeg = e.HeadingDeg
			e.OrderedHead = e.HeadingDeg
		} else {
			d.SteeringJammed = false
		}
	case SysPropulsion:
		if after <= 0 {
			e.OrderedSpeed = 0
		} else {
			maxSpd := 30.0 * after / 100
			if e.Kind == KindSurfaceShip {
				maxSpd = 28 * after / 100
			}
			if e.OrderedSpeed > maxSpd {
				e.OrderedSpeed = maxSpd
			}
		}
	}
}

// MaxSpeedKts returns ordered-speed ceiling from propulsion damage.
func (e *Entity) MaxSpeedKts() float64 {
	base := 30.0
	if e.Kind == KindSurfaceShip {
		base = 28
	}
	eff := e.Damage.EffOf(SysPropulsion)
	return base * eff / 100
}

// TurnRateScale 0..1 from steering damage.
func (e *Entity) TurnRateScale() float64 {
	eff := e.Damage.EffOf(SysSteering)
	if eff <= RepairThresholdPct {
		return 0
	}
	return eff / 100
}

// DepthRateScale 0..1 from depth-control damage (subs only).
func (e *Entity) DepthRateScale() float64 {
	if e.Kind != KindSubmarine {
		return 1
	}
	eff := e.Damage.EffOf(SysDepth)
	if eff <= RepairThresholdPct {
		return 0
	}
	return eff / 100
}
