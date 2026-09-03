package acoustics

import (
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// PassiveArrayKind selects which passive array feeds the display and detectors.
type PassiveArrayKind int

const (
	PassiveArrayHull PassiveArrayKind = iota
	PassiveArrayTowed
)

const (
	towedDeploySec  = 15.0
	towedRetractSec = 10.0

	// TowedCableFullYd is TB-16-class tow-cable length (~2400 ft ≈ 800 yd).
	TowedCableFullYd = 800.0
	// Minimum streamed fraction before hydrodynamic shear risk applies.
	towedShearMinPct = 0.20
)

// UpdateTowed advances towed-array cable pay-out or recovery.
func (s *SonarState) UpdateTowed(dt float64) {
	if s.TowedDamaged || s.TowedCableRate == 0 {
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
	if s.TowedDamaged || s.TowedCablePct >= 1 {
		return
	}
	s.TowedCableRate = 1.0 / towedDeploySec
}

// StartRetract begins recovering the towed array.
func (s *SonarState) StartRetract() {
	if s.TowedDamaged || s.TowedCablePct <= 0 {
		return
	}
	s.TowedCableRate = -1.0 / towedRetractSec
}

// TowedDeployed reports whether the cable is fully streamed.
func (s *SonarState) TowedDeployed() bool {
	return !s.TowedDamaged && s.TowedCablePct >= 1 && s.TowedCableRate == 0
}

// TowedStowed reports whether the array is fully housed.
func (s *SonarState) TowedStowed() bool {
	return s.TowedCablePct <= 0 && s.TowedCableRate == 0
}

// TowedInMotion reports deploy or retract in progress.
func (s *SonarState) TowedInMotion() bool {
	return !s.TowedDamaged && s.TowedCableRate != 0
}

// TowedBaselineYd is the horizontal lever arm from ownship to the array center.
func TowedBaselineYd(cablePct float64) float64 {
	if cablePct < 0 {
		cablePct = 0
	}
	if cablePct > 1 {
		cablePct = 1
	}
	return TowedCableFullYd * cablePct
}

// PlaceTowedListener copies ownship into dst and shifts position to the streamed
// array center (astern along heading by TowedBaselineYd).
func PlaceTowedListener(dst, ownship *world.Entity, cablePct float64) bool {
	if dst == nil || ownship == nil || cablePct < 0.05 {
		return false
	}
	*dst = *ownship
	base := TowedBaselineYd(cablePct)
	rad := ownship.HeadingDeg * math.Pi / 180
	dst.X = ownship.X - math.Sin(rad)*base
	dst.Y = ownship.Y - math.Cos(rad)*base
	return true
}

func (s *SonarState) towedEffectiveness() float64 {
	if s.TowedDamaged || s.PassiveArray != PassiveArrayTowed {
		return 0
	}
	return s.TowedCablePct
}

func (s *SonarState) passiveSNRBonusDB() float64 {
	// Full TB-16 stream: ~+11 dB vs hull for LF machinery lines (self-noise
	// stand-off + aperture). Open sources put TAS detection far beyond hull;
	// classification of harmonics is shorter than detection but still clearly
	// favors the towed array — keep a solid edge without god-mode ID at 20 kyd.
	return s.towedEffectiveness() * 11.0
}

func (s *SonarState) passiveSelfNoiseCutDB() float64 {
	return s.towedEffectiveness() * 7.0
}

func (s *SonarState) passiveBearingSigmaScale() float64 {
	return 1.0 - s.towedEffectiveness()*0.62
}

// TowedWarnSpeedKts — above this, UI warns that further speed risks parting the cable.
// Open sources: acoustic sweet spot ~12 kn; safely streamable toward ~25 kn (Navy Lookout / TAS).
// Full cable: warn ~20 kn; short scope: higher.
func TowedWarnSpeedKts(cablePct float64) float64 {
	if cablePct < towedShearMinPct {
		return 99
	}
	return 20 + (1-cablePct)*6 // 20..~25
}

// TowedShearSpeedKts — hydrodynamic drag ~v² can part cable / handling gear.
// Full stream: ~24 kn failure; shorter scope tolerates a few knots more.
func TowedShearSpeedKts(cablePct float64) float64 {
	if cablePct < towedShearMinPct {
		return 99
	}
	return 24 + (1-cablePct)*5 // 24..~28
}


// TriangulationQuality 0..1 from hull↔towed baseline vs range and geometry.
// Best abeam with a long stream; near zero ahead/astern or with short cable.
func TriangulationQuality(baselineYd, rangeYd, relBearingDeg float64) float64 {
	if baselineYd < 80 || rangeYd < 200 {
		return 0
	}
	geom := math.Abs(math.Sin(relBearingDeg * math.Pi / 180))
	if geom < 0.12 {
		return 0
	}
	// Parallax angle scale: B/R; saturates as baseline becomes a useful fraction of range.
	parallax := baselineYd / (baselineYd + rangeYd*0.22)
	q := parallax * geom
	if q > 1 {
		q = 1
	}
	return q
}

// ApplyTriangulationBonus shrinks passive track uncertainty when a long towed
// baseline can be fused with the hull aperture (dual-array TMA).
func ApplyTriangulationBonus(c *Contact, baselineYd, rangeYd, relBearingDeg float64) {
	if c == nil {
		return
	}
	q := TriangulationQuality(baselineYd, rangeYd, relBearingDeg)
	if q < 0.05 {
		return
	}
	// Full cable abeam mid-range: up to ~55% tighter range blob, ~30% bearing.
	c.UncRangeYd *= 1 - 0.55*q
	c.UncBearingDeg *= 1 - 0.30*q
	if c.UncRangeYd < 90 {
		c.UncRangeYd = 90
	}
	if c.UncBearingDeg < 1.2 {
		c.UncBearingDeg = 1.2
	}
}

// PassiveSelfNoisePenaltyDB models loss of passive SNR from ownship speed.
// Below ~8 kts machinery dominates; above that, flow noise rises sharply.
// Hull arrays suffer most on the bow quarters; towed arrays are cleaner astern.
func PassiveSelfNoisePenaltyDB(array PassiveArrayKind, relativeBearingDeg, speedKts, depthFt, towedPct float64) float64 {
	speedKts = math.Abs(speedKts)
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
