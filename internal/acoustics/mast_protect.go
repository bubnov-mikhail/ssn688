package acoustics

import (
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const mastExposeEpsilon = 0.05

// Auto-retract triggers before hydrodynamic shear damage (75 ft / 9.5 kn).
const (
	MastAutoRetractDepthFt  = 65.0
	MastAutoRetractSpeedKts = 8.5
	MastShearDepthFt        = world.ESMMastMaxDepthFt + 15 // 75 ft
	MastShearSpeedKts       = world.ESMMastMaxSpeedKts + 1.5 // 9.5 kn
)

const (
	EventAutoRetractMasts = "AUTO-RETRACT — masts lowering to prevent damage"
	EventAutoRetractTowed = "AUTO-RETRACT — towed array recovering to prevent cable damage"
	EventAutoRetractBoth  = "AUTO-RETRACT — masts and towed array stowed to prevent damage"
)

func mastSpeedOverLimit(player *world.Entity) bool {
	if player == nil {
		return false
	}
	spd := math.Max(math.Abs(player.SpeedKts), math.Abs(player.OrderedSpeed))
	return spd >= MastAutoRetractSpeedKts
}

func mastDepthOverLimit(player *world.Entity) bool {
	if player == nil {
		return false
	}
	depth := math.Max(player.DepthFt, player.OrderedDepth)
	return depth >= MastAutoRetractDepthFt
}

func mastNeedsAutoRetract(player *world.Entity) bool {
	return mastSpeedOverLimit(player) || mastDepthOverLimit(player)
}

// AutoProtectExtendedGear lowers masts and recovers towed cable before hydrodynamic
// damage would occur. Returns at most one notification event per call.
func AutoProtectExtendedGear(player *world.Entity, esm *ESMState, comm *COMMState, peri *PeriscopeState, sonar *SonarState) []string {
	if player == nil || player.Kind != world.KindSubmarine {
		return nil
	}
	var masts, towed bool

	if notify := autoRetractESM(esm, player); notify {
		masts = true
	}
	if notify := autoRetractCOMM(comm, player); notify {
		masts = true
	}
	if notify := autoRetractPeriscope(peri, player); notify {
		masts = true
	}
	if notify := autoRetractTowed(sonar, player); notify {
		towed = true
	}

	switch {
	case masts && towed:
		return []string{EventAutoRetractBoth}
	case masts:
		return []string{EventAutoRetractMasts}
	case towed:
		return []string{EventAutoRetractTowed}
	default:
		return nil
	}
}

func autoRetractESM(esm *ESMState, player *world.Entity) bool {
	if !esmShouldRetract(esm, player) {
		return false
	}
	notify := esm.Order != ESMMastStow || esm.Extension <= mastExposeEpsilon
	esm.OrderLowerESM()
	return notify
}

func autoRetractCOMM(comm *COMMState, player *world.Entity) bool {
	if !commShouldRetract(comm, player) {
		return false
	}
	notify := comm.Order != COMMMastStow || comm.Extension <= mastExposeEpsilon
	comm.OrderLowerCOMM()
	return notify
}

func autoRetractPeriscope(peri *PeriscopeState, player *world.Entity) bool {
	if !periShouldRetract(peri, player) {
		return false
	}
	notify := peri.Order != PeriMastStow || peri.Extension <= mastExposeEpsilon
	peri.OrderLower()
	return notify
}

func autoRetractTowed(sonar *SonarState, player *world.Entity) bool {
	if !towedShouldRetract(sonar, player) {
		return false
	}
	sonar.StartRetract()
	return true
}

func esmShouldRetract(esm *ESMState, player *world.Entity) bool {
	if esm == nil || player == nil || esm.Sheared {
		return false
	}
	player.EnsureDamage()
	if player.Damage.Destroyed(world.SysESM) {
		return false
	}
	if !mastNeedsAutoRetract(player) {
		return false
	}
	return esm.Extension > mastExposeEpsilon || esm.Order == ESMMastRaise
}

func commShouldRetract(comm *COMMState, player *world.Entity) bool {
	if comm == nil || player == nil || comm.Sheared {
		return false
	}
	player.EnsureDamage()
	if player.Damage.Destroyed(world.SysCOMM) {
		return false
	}
	if !mastNeedsAutoRetract(player) {
		return false
	}
	return comm.Extension > mastExposeEpsilon || comm.Order == COMMMastRaise
}

func periShouldRetract(peri *PeriscopeState, player *world.Entity) bool {
	if peri == nil || player == nil || peri.Sheared {
		return false
	}
	player.EnsureDamage()
	if player.Damage.Destroyed(world.SysPeriscope) {
		return false
	}
	if !mastNeedsAutoRetract(player) {
		return false
	}
	return peri.Extension > mastExposeEpsilon || peri.Order == PeriMastRaise
}

func towedShouldRetract(sonar *SonarState, player *world.Entity) bool {
	if sonar == nil || player == nil || sonar.TowedDamaged {
		return false
	}
	if sonar.TowedCablePct < towedShearMinPct {
		return false
	}
	if sonar.TowedCablePct <= 0 && !(sonar.TowedInMotion() && sonar.TowedCableRate > 0) {
		return false
	}
	if sonar.TowedInMotion() && sonar.TowedCableRate < 0 {
		return false
	}
	spd := math.Max(math.Abs(player.SpeedKts), math.Abs(player.OrderedSpeed))
	return spd >= TowedWarnSpeedKts(sonar.TowedCablePct)
}
