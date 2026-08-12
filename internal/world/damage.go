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
	SysESM        // ESM / EW mast (688 sail antenna)
	SysCOMM       // HF/VLF communications mast
	SysPeriscope  // optical / photonics mast
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

// FirstNewCriticalSystem returns the first system that crossed into Destroyed
// since beforeCrit (index by Sys*). Returns SysNone if none.
func FirstNewCriticalSystem(beforeCrit [SysCount]bool, d *PlatformDamage) int {
	if d == nil {
		return SysNone
	}
	for sys := 0; sys < SysCount; sys++ {
		if !beforeCrit[sys] && d.Destroyed(sys) {
			return sys
		}
	}
	return SysNone
}

// SnapshotCritical marks which systems are currently Destroyed.
func SnapshotCritical(d *PlatformDamage) (out [SysCount]bool) {
	if d == nil {
		return out
	}
	for sys := 0; sys < SysCount; sys++ {
		out[sys] = d.Destroyed(sys)
	}
	return out
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
	case SysESM:
		return "ESM Mast"
	case SysCOMM:
		return "COMM Mast"
	case SysPeriscope:
		return "Periscope"
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

// ApplyTorpedoHit applies heavy (Mk48 / 53-65) or light (UMGT-1 / SET-40) fish damage.
// Heavy warheads always breach hull hard — small combatants die to one good hit.
func ApplyTorpedoHit(e *Entity, rng *rand.Rand, lightWarhead bool) (fatal bool, events []string) {
	if e == nil || !e.Alive() {
		return false, nil
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	e.EnsureDamage()
	d := &e.Damage

	hullLoss := torpedoHullLoss(e, rng, lightWarhead)
	beforeHull := d.Eff[SysHull]
	d.Eff[SysHull] = math.Max(0, beforeHull-hullLoss)
	tag := "Mk48"
	if lightWarhead {
		tag = "LW torpedo"
	}
	events = append(events, fmt.Sprintf("%s: hull %.0f%% → %.0f%% (%s)", e.Name, beforeHull, d.Eff[SysHull], tag))

	cands := hitCandidates(e.Kind)
	nHits := 2 + rng.Intn(3) // 2–4 subsystems
	if lightWarhead {
		nHits = 1 + rng.Intn(3)
	}
	if nHits > len(cands) {
		nHits = len(cands)
	}
	rng.Shuffle(len(cands), func(i, j int) { cands[i], cands[j] = cands[j], cands[i] })

	for i := 0; i < nHits; i++ {
		sys := cands[i]
		if sys == SysHull {
			continue // already applied primary hull damage
		}
		loss := 30.0 + rng.Float64()*55.0
		if lightWarhead {
			loss = 20.0 + rng.Float64()*40.0
		}
		before := d.Eff[sys]
		d.Eff[sys] = math.Max(0, before-loss)
		after := d.Eff[sys]
		if after < before {
			events = append(events, fmt.Sprintf("%s: %s %.0f%% → %.0f%%", e.Name, SystemName(sys), before, after))
		}
		applySystemSideEffects(e, sys, before, after, rng)
	}

	if d.Eff[SysHull] <= RepairThresholdPct {
		d.Eff[SysHull] = 0
		events = append(events, fmt.Sprintf("%s: HULL FATAL — flooding uncontrolled", e.Name))
		return true, events
	}
	return false, events
}

// torpedoHullLoss is the primary kill roll for a fish warhead.
func torpedoHullLoss(e *Entity, rng *rand.Rand, light bool) float64 {
	if light {
		// Lightweight ASW fish: punish but rarely one-shot a 688.
		base := 22.0 + rng.Float64()*18.0 // ~22–40%
		if e.Kind == KindSurfaceShip {
			base = 28.0 + rng.Float64()*22.0
		}
		return base
	}
	switch e.SignatureID {
	case "grisha", "fishing":
		// Corvette / small hull: under-keel Mk48 is decisive.
		return 85.0 + rng.Float64()*20.0 // ~85–105%
	case "krivak", "merchant", "gorshkov":
		return 58.0 + rng.Float64()*22.0 // ~58–80% → usually 1–2 hits
	case "udaloy", "kresta2", "tanker":
		return 48.0 + rng.Float64()*18.0 // ~48–66% → typically 2 hits
	case "foxtrot":
		return 75.0 + rng.Float64()*30.0 // older diesel — fragile
	case "kilo":
		return 58.0 + rng.Float64()*28.0
	case "victor_iii", "los_angeles", "yasen_m":
		return 48.0 + rng.Float64()*22.0 // peer SSN: often 2 hits
	default:
		if e.Kind == KindSurfaceShip {
			return 55.0 + rng.Float64()*25.0
		}
		return 52.0 + rng.Float64()*28.0
	}
}

// ApplyHarpoonHit applies anti-ship warhead damage (1–3 hits kill a destroyer).
func ApplyHarpoonHit(e *Entity, rng *rand.Rand) (fatal bool, events []string) {
	if e == nil || !e.Alive() {
		return false, nil
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	e.EnsureDamage()
	d := &e.Damage

	hullLoss := 38.0 + rng.Float64()*14.0 // ~38–52% per hit
	if e.Kind == KindSurfaceShip {
		hullLoss = 42.0 + rng.Float64()*12.0
	}
	before := d.Eff[SysHull]
	d.Eff[SysHull] = math.Max(0, before-hullLoss)
	events = append(events, fmt.Sprintf("%s: hull %.0f%% → %.0f%% (Harpoon)", e.Name, before, d.Eff[SysHull]))

	cands := hitCandidates(e.Kind)
	nHits := 2 + rng.Intn(3)
	if nHits > len(cands) {
		nHits = len(cands)
	}
	rng.Shuffle(len(cands), func(i, j int) { cands[i], cands[j] = cands[j], cands[i] })
	for i := 0; i < nHits; i++ {
		sys := cands[i]
		if sys == SysHull {
			continue
		}
		loss := 30.0 + rng.Float64()*50.0
		b := d.Eff[sys]
		d.Eff[sys] = math.Max(0, b-loss)
		if d.Eff[sys] < b {
			events = append(events, fmt.Sprintf("%s: %s %.0f%% → %.0f%%", e.Name, SystemName(sys), b, d.Eff[sys]))
		}
		applySystemSideEffects(e, sys, b, d.Eff[sys], rng)
	}

	if d.Eff[SysHull] <= RepairThresholdPct {
		d.Eff[SysHull] = 0
		events = append(events, fmt.Sprintf("%s: HULL FATAL — Harpoon kill", e.Name))
		return true, events
	}
	return false, events
}

// ApplyDebrisHit applies light fragment damage from a close-in missile intercept.
func ApplyDebrisHit(e *Entity, rng *rand.Rand) (fatal bool, events []string) {
	if e == nil || !e.Alive() {
		return false, nil
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	e.EnsureDamage()
	d := &e.Damage

	hullLoss := 8.0 + rng.Float64()*14.0 // ~8–22%
	before := d.Eff[SysHull]
	d.Eff[SysHull] = math.Max(0, before-hullLoss)
	events = append(events, fmt.Sprintf("%s: hull %.0f%% → %.0f%% (missile debris)", e.Name, before, d.Eff[SysHull]))

	cands := hitCandidates(e.Kind)
	nHits := 1 + rng.Intn(2)
	if nHits > len(cands) {
		nHits = len(cands)
	}
	rng.Shuffle(len(cands), func(i, j int) { cands[i], cands[j] = cands[j], cands[i] })
	for i := 0; i < nHits; i++ {
		sys := cands[i]
		if sys == SysHull {
			continue
		}
		loss := 15.0 + rng.Float64()*30.0
		b := d.Eff[sys]
		d.Eff[sys] = math.Max(0, b-loss)
		if d.Eff[sys] < b {
			events = append(events, fmt.Sprintf("%s: %s %.0f%% → %.0f%%", e.Name, SystemName(sys), b, d.Eff[sys]))
		}
		applySystemSideEffects(e, sys, b, d.Eff[sys], rng)
	}

	if d.Eff[SysHull] <= RepairThresholdPct {
		d.Eff[SysHull] = 0
		events = append(events, fmt.Sprintf("%s: HULL FATAL — debris", e.Name))
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
		base = append(base, SysTowed, SysDepth, SysESM, SysCOMM, SysPeriscope)
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
			maxSpd := e.MaxSpeedKts()
			maxAstern := e.MaxAsternKts()
			if e.OrderedSpeed > maxSpd {
				e.OrderedSpeed = maxSpd
			}
			if e.OrderedSpeed < -maxAstern {
				e.OrderedSpeed = -maxAstern
			}
		}
	}
}

// MaxSpeedKts returns ordered-speed ceiling from propulsion damage + hull class.
func (e *Entity) MaxSpeedKts() float64 {
	base := 30.0
	switch e.SignatureID {
	case "los_angeles":
		base = 32
	case "victor_iii":
		base = 30
	case "yasen_m":
		base = 31
	case "kilo":
		base = 17
	case "foxtrot":
		base = 15
	case "udaloy", "kresta2":
		base = 32
	case "gorshkov":
		base = 30
	case "krivak":
		base = 30
	case "grisha":
		base = 34
	case "merchant":
		base = 16
	case "tanker":
		base = 14
	case "fishing":
		base = 12
	default:
		if e.Kind == KindSurfaceShip {
			base = 28
		}
	}
	eff := e.Damage.EffOf(SysPropulsion)
	return base * eff / 100
}

// MaxAsternKts is the reverse-speed ceiling (subs are limited vs ahead flank).
func (e *Entity) MaxAsternKts() float64 {
	return e.MaxSpeedKts() * 0.55
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
