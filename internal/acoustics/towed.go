package acoustics

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
	return s.towedEffectiveness() * 5.0
}

func (s *SonarState) passiveSelfNoiseCutDB() float64 {
	return s.towedEffectiveness() * 4.0
}

func (s *SonarState) passiveBearingSigmaScale() float64 {
	return 1.0 - s.towedEffectiveness()*0.55
}

func applyPassiveArrayModifiers(result *DetectionResult, sonar *SonarState) {
	bonus := sonar.passiveSNRBonusDB()
	if bonus <= 0 {
		return
	}
	result.PeakSNR += bonus
	for i := range result.SNR {
		result.SNR[i] += bonus
	}
	for i := range result.SignalForClassify {
		result.SignalForClassify[i] += bonus * 0.6
	}
	result.BandsAbove = result.SNR.BandsAbove(DetectThreshold)
	if result.BandsAbove >= MinDetectBands || result.PeakSNR >= PeakDetectSNR {
		result.Detected = true
	}
}
