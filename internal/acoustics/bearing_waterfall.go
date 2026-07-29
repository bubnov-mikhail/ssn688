package acoustics

import (
	"math"

	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

// BearingWaterfallBins is angular resolution for the passive bearing waterfall (1° per bin).
const BearingWaterfallBins = 360

// BearingWaterfallRow is one time slice of omnidirectional passive energy vs bearing.
type BearingWaterfallRow struct {
	Bearings []float64
	Heading  float64
}

// BearingWaterfallSlice samples passive SNR around the horizon for waterfall display.
func BearingWaterfallSlice(model Model, listener *world.Entity, emitters []*world.Entity, sonar *SonarState, array PassiveArrayKind, gameTime float64) BearingWaterfallRow {
	bins := make([]float64, BearingWaterfallBins)
	BearingWaterfallInto(bins, model, listener, emitters, sonar, array, gameTime)
	return BearingWaterfallRow{Bearings: bins, Heading: listener.HeadingDeg}
}

// BearingWaterfallInto fills dst (len >= BearingWaterfallBins) without allocating.
func BearingWaterfallInto(dst []float64, model Model, listener *world.Entity, emitters []*world.Entity, sonar *SonarState, array PassiveArrayKind, gameTime float64) {
	if len(dst) < BearingWaterfallBins {
		return
	}
	bins := dst[:BearingWaterfallBins]
	for i := range bins {
		bins[i] = 0
	}
	synth := *sonar
	synth.PassiveArray = array

	for _, em := range emitters {
		if em == nil || em.ID == listener.ID {
			continue
		}
		if !em.Alive() && em.Status != world.StatusSinking {
			continue
		}
		result := model.Detect(listener, em, ModePassive, 0)
		ApplyListenBand(&result, synth.ListenBand)
		applyPassiveArrayModifiers(&result, &synth)
		rel := angleDiffDeg(result.BearingDeg, listener.HeadingDeg)
		sens := arraySensitivity(array, rel, sonar.TowedCablePct)
		sensDB := 20 * math.Log10(math.Max(sens, 0.001))

		peak := result.PeakSNR + sensDB - 4.0
		if peak < 0 {
			peak = 0
		}
		sigmaScale := 1.0

		if age := EnemyActivePingAgeSec(em, gameTime); age >= 0 {
			if ping := enemyActivePingPeak(result.TrueRangeYd, age); ping > 0 {
				pingPeak := ping + sensDB
				if pingPeak > peak {
					peak = pingPeak
				}
				sigmaScale = 1.7
			}
		}

		if peak <= 0.05 {
			continue
		}
		spreadBearingEnergy(bins, result.BearingDeg, peak, array, sonar.TowedCablePct, sigmaScale)
	}
	for _, bio := range sonar.BioTransients {
		peak := BioWaterfallContribution(bio, sonar.ListenBand, gameTime)
		if peak <= 0.2 {
			continue
		}
		spreadBearingEnergy(bins, bio.BearingDeg, peak, array, sonar.TowedCablePct, 2.2)
	}
	addAmbientNoise(bins, gameTime, array, sonar.TowedCablePct, listener.HeadingDeg, listener.SpeedKts)
	AddOwnshipFlowNoise(bins, listener.SpeedKts, listener.DepthFt, listener.HeadingDeg, array, sonar.TowedCablePct)
	applyBlastWashout(bins, sonar, listener, gameTime)
}

func applyBlastWashout(bins []float64, sonar *SonarState, listener *world.Entity, gameTime float64) {
	if sonar == nil || listener == nil || sonar.LastBlastAt <= 0 {
		return
	}
	age := gameTime - sonar.LastBlastAt
	flashSec := sonar.LastBlastFlashSec
	if flashSec <= 0 {
		flashSec = 8
	}
	if age < 0 || age > flashSec {
		return
	}
	maxR := sonar.LastBlastRangeYd
	if maxR <= 0 {
		maxR = weapons.BlastDeafRadiusYd * 1.2
	}
	dist := math.Hypot(listener.X-sonar.LastBlastX, listener.Y-sonar.LastBlastY)
	if dist > maxR {
		return
	}
	flash := math.Exp(-age*(0.55*8/flashSec)) * (1 - dist/maxR)
	wash := 35 + 55*flash
	// Absolute bearing from listener toward detonation (same convention as entity bearings).
	blastBrg := math.Atan2(sonar.LastBlastX-listener.X, sonar.LastBlastY-listener.Y) * 180 / math.Pi
	if blastBrg < 0 {
		blastBrg += 360
	}
	n := len(bins)
	for i := range bins {
		binBrg := float64(i) / float64(n) * 360
		ang := math.Abs(AngleDiffDeg(binBrg, blastBrg))
		// Peak toward blast (~±40–60°), weak opposite side.
		dir := math.Exp(-(ang * ang) / (2 * 50 * 50))
		w := wash * (0.10 + 0.90*dir)
		if w > bins[i] {
			bins[i] = w
		} else {
			bins[i] = bins[i]*0.3 + w*0.7
		}
	}
}

// enemyActivePingPeak returns display SNR for a recent active transmission (one-way path).
// Uses LastPingTime only so the flash persists after the AI clears ActiveSonar.
func enemyActivePingPeak(rangeYd, ageSec float64) float64 {
	if ageSec < 0 || ageSec > 3.5 {
		return 0
	}
	flash := math.Exp(-ageSec * 1.25)
	rangeAtt := 1.0
	if rangeYd > 1500 {
		rangeAtt = 1500 / rangeYd
	}
	if rangeAtt < 0.35 {
		rangeAtt = 0.35
	}
	// Direct-path active pulse dwarfs broadband machinery noise on the display.
	return (55 + 90*flash) * rangeAtt
}

// EnemyActivePingAgeSec is time since emitter last transmitted, or -1 if never.
func EnemyActivePingAgeSec(em *world.Entity, gameTime float64) float64 {
	if em == nil || em.LastPingTime <= 0 {
		return -1
	}
	return gameTime - em.LastPingTime
}

func spreadBearingEnergy(bins []float64, bearingDeg, peak float64, array PassiveArrayKind, towedPct, sigmaScale float64) {
	if peak <= 0 {
		return
	}
	if sigmaScale < 0.5 {
		sigmaScale = 0.5
	}
	n := len(bins)
	sigma := hullBeamSigmaDeg
	if array == PassiveArrayTowed {
		sigma = towedBeamSigmaDeg(towedPct)
	}
	sigma *= sigmaScale
	inv2Sigma2 := 1.0 / (2 * sigma * sigma)
	// Only touch bins under the beam (was a full 360° scan).
	// Tighter beam — contacts read as narrow lines, not fat bands.
	halfW := int(sigma*2.4*(float64(n)/360.0)) + 1
	center := int(bearingDeg / 360 * float64(n))
	for d := -halfW; d <= halfW; d++ {
		bi := center + d
		for bi < 0 {
			bi += n
		}
		for bi >= n {
			bi -= n
		}
		binBearing := float64(bi) / float64(n) * 360
		delta := angleDiffDeg(bearingDeg, binBearing)
		gain := math.Exp(-delta * delta * inv2Sigma2)
		gain *= 1.0 - math.Min(1, math.Abs(delta)/(sigma*2.4))*0.55
		v := peak * gain
		if v > bins[bi] {
			bins[bi] = v
		}
	}
}

const hullBeamSigmaDeg = 2.4

func towedBeamSigmaDeg(cablePct float64) float64 {
	if cablePct < 0 {
		cablePct = 0
	}
	if cablePct > 1 {
		cablePct = 1
	}
	return 3.8 - cablePct*1.8
}

// PassiveArraySensitivity returns listen sensitivity for a contact at relative bearing.
func PassiveArraySensitivity(array PassiveArrayKind, relativeBearingDeg, towedPct float64) float64 {
	return arraySensitivity(array, relativeBearingDeg, towedPct)
}

func arraySensitivity(array PassiveArrayKind, relativeBearingDeg, towedPct float64) float64 {
	abs := math.Abs(relativeBearingDeg)
	switch array {
	case PassiveArrayTowed:
		// Stowed / short cable: nearly deaf. Fully streamed: strong abeam/astern, weak endfire ahead.
		effect := 0.08 + 0.92*towedPct
		if towedPct < 0.15 {
			return effect * 0.2
		}
		switch {
		case abs < 18: // endfire null ahead of tow
			return effect * 0.22
		case abs < 35:
			return effect * 0.55
		case abs > 155: // strong aft of beam
			return effect * 1.08
		default:
			return effect
		}
	default: // hull spherical / conformal — stern baffle
		switch {
		case abs <= 50:
			return 1.0
		case abs >= 150: // deep baffle "deaf zone"
			return 0.06
		case abs >= 120:
			t := (abs - 120) / 30
			return 0.35 - t*0.29
		default:
			t := (abs - 50) / 70
			return 1.0 - t*0.65
		}
	}
}

func addAmbientNoise(bins []float64, gameTime float64, array PassiveArrayKind, towedPct, headingDeg, speedKts float64) {
	base := 2.6
	switch array {
	case PassiveArrayTowed:
		base = 1.6 + towedPct*0.5
	default:
		base = 3.0
	}
	if speedKts > 8 {
		base += math.Pow(speedKts-8, 1.15) * 0.28
	}
	n := len(bins)
	for bi := range bins {
		bearing := float64(bi) / float64(n) * 360
		rel := angleDiffDeg(bearing, headingDeg)
		sens := arraySensitivity(array, rel, towedPct)
		phase := gameTime*5.3 + float64(bi)*0.173
		flutter := math.Sin(phase)*0.55 +
			math.Sin(phase*2.17+0.8)*0.32 +
			math.Sin(phase*0.41+float64(bi)*0.07)*0.22 +
			math.Sin(gameTime*1.1+float64(bi)*0.53)*0.18
		noise := (base + flutter) * (0.08 + 0.92*sens)
		if noise > bins[bi] {
			bins[bi] = noise
		} else if speedKts > 12 {
			bins[bi] = bins[bi]*0.7 + noise*0.3
		} else {
			bins[bi] += noise * 0.04
		}
	}
}

// AddOwnshipFlowNoise projects speed-induced self-noise onto the bearing display.
// For hull arrays this is strongest on the bow quarters; for towed arrays it is
// reduced overall but still prominent near the forward endfire / tow-line sector.
func AddOwnshipFlowNoise(bins []float64, speedKts, depthFt, headingDeg float64, array PassiveArrayKind, towedPct float64) {
	if len(bins) == 0 || speedKts < 5 {
		return
	}
	n := len(bins)
	for bi := range bins {
		bearing := float64(bi) / float64(n) * 360
		rel := angleDiffDeg(bearing, headingDeg)
		penalty := PassiveSelfNoisePenaltyDB(array, rel, speedKts, depthFt, towedPct)
		if penalty <= 0 {
			continue
		}
		noise := penalty * 1.45
		if noise > bins[bi] {
			bins[bi] = noise
		} else if speedKts > 14 {
			bins[bi] = math.Max(bins[bi], noise*0.82)
		} else {
			bins[bi] += noise * 0.12
		}
	}
}

// AngleDiffDeg returns the shortest signed angle from b to a in degrees.
func AngleDiffDeg(a, b float64) float64 {
	return angleDiffDeg(a, b)
}

func angleDiffDeg(a, b float64) float64 {
	d := a - b
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}

func BearingBinToDeg(bin int) float64 {
	if BearingWaterfallBins <= 0 {
		return 0
	}
	return float64(bin%BearingWaterfallBins) / float64(BearingWaterfallBins) * 360
}

func HeadingToWaterfallX(heading float64, plotW int) int {
	h := math.Mod(heading, 360)
	if h < 0 {
		h += 360
	}
	return int(h / 360 * float64(plotW))
}
