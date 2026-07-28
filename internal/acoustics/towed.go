package acoustics

import "math"

// PassiveArrayKind selects which passive array feeds the display and detectors.
type PassiveArrayKind int

const (
	PassiveArrayHull PassiveArrayKind = iota
	PassiveArrayTowed
)

const (
	towedDeploySec  = 15.0
	towedRetractSec = 10.0
)

// UpdateTowed advances towed-array cable pay-out or recovery.
func (s *SonarState) UpdateTowed(dt float64) {
	if s.TowedCableRate == 0 {
		return
	}
	s.TowedCablePct += s.TowedCableRate * dt
	if s.TowedCablePct >= 1 {
		s.TowedCablePct = 1
		s.TowedCableRate = 0
	}
	if s.TowedCablePct <= 0 {
		s.TowedCablePct = 0
		s.TowedCableRate = 0
	}
}

// StopTowed halts cable motion at the present length.
func (s *SonarState) StopTowed() {
	s.TowedCableRate = 0
}

// StartDeploy begins paying out the towed array.
func (s *SonarState) StartDeploy() {
	if s.TowedCablePct >= 1 {
		return
	}
	s.TowedCableRate = 1.0 / towedDeploySec
}

// StartRetract begins recovering the towed array.
func (s *SonarState) StartRetract() {
	if s.TowedCablePct <= 0 {
		return
	}
	s.TowedCableRate = -1.0 / towedRetractSec
}

// TowedDeployed reports whether the cable is fully streamed.
func (s *SonarState) TowedDeployed() bool {
	return s.TowedCablePct >= 1 && s.TowedCableRate == 0
}

// TowedStowed reports whether the array is fully housed.
func (s *SonarState) TowedStowed() bool {
	return s.TowedCablePct <= 0 && s.TowedCableRate == 0
}

// TowedInMotion reports deploy or retract in progress.
func (s *SonarState) TowedInMotion() bool {
	return s.TowedCableRate != 0
}

func (s *SonarState) towedEffectiveness() float64 {
	if s.PassiveArray != PassiveArrayTowed {
		return 0
	}
	return s.TowedCablePct
}

func (s *SonarState) passiveSNRBonusDB() float64 {
	return s.towedEffectiveness() * 8.0
}

func (s *SonarState) passiveSelfNoiseCutDB() float64 {
	return s.towedEffectiveness() * 6.0
}

func (s *SonarState) passiveBearingSigmaScale() float64 {
	return 1.0 - s.towedEffectiveness()*0.62
}

// PassiveSelfNoisePenaltyDB models loss of passive SNR from ownship speed.
// Below ~8 kts machinery dominates; above that, flow noise rises sharply.
// Hull arrays suffer most on the bow quarters; towed arrays are cleaner astern.
func PassiveSelfNoisePenaltyDB(array PassiveArrayKind, relativeBearingDeg, speedKts, depthFt, towedPct float64) float64 {
	if speedKts <= 4 {
		return 0
	}
	base := math.Max(0, speedKts-4) * 0.08
	if speedKts > 8 {
		base += math.Pow(speedKts-8, 1.2) * 0.22
	}
	// Deeper pressure delays cavitation onset, but flow noise still rises with speed.
	base += CavitationSeverity(depthFt, speedKts) * 4.5

	abs := math.Abs(relativeBearingDeg)
	dir := 1.0
	switch array {
	case PassiveArrayTowed:
		dir = 0.55 - towedPct*0.18
		if abs < 30 {
			dir += 0.65 // endfire / tow-line turbulence ahead hurts most
		} else if abs > 140 {
			dir *= 0.65 // astern remains the cleanest sector for a streamed tow
		}
		if dir < 0.2 {
			dir = 0.2
		}
	default:
		switch {
		case abs < 35:
			dir = 1.55
		case abs < 70:
			dir = 1.2
		case abs > 145:
			dir = 0.7
		default:
			dir = 0.95
		}
	}
	return base * dir
}

func PassiveSelfNoiseDeltaDB(array PassiveArrayKind, relativeBearingDeg, speedKts, depthFt, towedPct float64) float64 {
	return PassiveSelfNoisePenaltyDB(array, relativeBearingDeg, speedKts, depthFt, towedPct) -
		PassiveSelfNoisePenaltyDB(PassiveArrayHull, relativeBearingDeg, speedKts, depthFt, 0)
}

func PassiveSelfNoiseSeverity(array PassiveArrayKind, speedKts, depthFt, towedPct float64) float64 {
	maxPen := PassiveSelfNoisePenaltyDB(array, 0, speedKts, depthFt, towedPct)
	if array == PassiveArrayTowed {
		maxPen = PassiveSelfNoisePenaltyDB(array, 0, speedKts, depthFt, towedPct)
	}
	if maxPen <= 0 {
		return 0
	}
		return math.Min(1, maxPen/8.0)
}

func applyPassiveArrayModifiers(result *DetectionResult, sonar *SonarState) {
	bonus := sonar.passiveSNRBonusDB()
	for i := range result.SNR {
		result.SNR[i] += bonus
	}
	for i := range result.SignalForClassify {
		result.SignalForClassify[i] += bonus * 0.6
	}
	result.PeakSNR += bonus
	result.BandsAbove = result.SNR.BandsAbove(DetectThreshold)
	if result.BandsAbove >= MinDetectBands || result.PeakSNR >= PeakDetectSNR {
		result.Detected = true
	}
}
