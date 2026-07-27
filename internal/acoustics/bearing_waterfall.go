package acoustics

import (
	"math"

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
	synth := *sonar
	synth.PassiveArray = array

	for _, em := range emitters {
		if !em.Alive() || em.ID == listener.ID {
			continue
		}
		result := model.Detect(listener, em, ModePassive, 0)
		applyPassiveArrayModifiers(&result, &synth)
		if result.PeakSNR <= 0 {
			continue
		}
		rel := angleDiffDeg(result.BearingDeg, listener.HeadingDeg)
		sens := arraySensitivity(array, rel, sonar.TowedCablePct)
		peak := result.PeakSNR * sens
		spreadBearingEnergy(bins, result.BearingDeg, peak, array, sonar.TowedCablePct)
	}
	addAmbientNoise(bins, gameTime, array, sonar.TowedCablePct)
	return BearingWaterfallRow{Bearings: bins, Heading: listener.HeadingDeg}
}

func spreadBearingEnergy(bins []float64, bearingDeg, peak float64, array PassiveArrayKind, towedPct float64) {
	if peak <= 0 {
		return
	}
	n := len(bins)
	sigma := hullBeamSigmaDeg
	if array == PassiveArrayTowed {
		sigma = towedBeamSigmaDeg(towedPct)
	}
	inv2Sigma2 := 1.0 / (2 * sigma * sigma)

	for bi := 0; bi < n; bi++ {
		binBearing := float64(bi) / float64(n) * 360
		delta := angleDiffDeg(bearingDeg, binBearing)
		gain := math.Exp(-delta * delta * inv2Sigma2)
		// Soft skirts so contacts fade gradually at the beam edge.
		gain *= 1.0 - math.Min(1, math.Abs(delta)/(sigma*3.2))*0.35
		v := peak * gain
		if v > bins[bi] {
			bins[bi] = v
		}
	}
}

const hullBeamSigmaDeg = 7.5

func towedBeamSigmaDeg(cablePct float64) float64 {
	if cablePct < 0 {
		cablePct = 0
	}
	if cablePct > 1 {
		cablePct = 1
	}
	// Deployed towed array: tighter bearing, partial cable: broader smear.
	return 10.0 - cablePct*5.5
}

// PassiveArraySensitivity returns listen sensitivity for a contact at relative bearing.
func PassiveArraySensitivity(array PassiveArrayKind, relativeBearingDeg, towedPct float64) float64 {
	return arraySensitivity(array, relativeBearingDeg, towedPct)
}

func arraySensitivity(array PassiveArrayKind, relativeBearingDeg, towedPct float64) float64 {
	abs := math.Abs(relativeBearingDeg)
	switch array {
	case PassiveArrayTowed:
		effect := 0.25 + 0.75*towedPct
		if abs < 25 {
			return effect * 0.88
		}
		if abs > 160 {
			return effect * 1.05
		}
		return effect
	default:
		switch {
		case abs <= 55:
			return 1.0
		case abs >= 155:
			return 0.12
		default:
			t := (abs - 55) / 100
			return 1.0 - t*0.88
		}
	}
}

func addAmbientNoise(bins []float64, gameTime float64, array PassiveArrayKind, towedPct float64) {
	base := 2.6
	switch array {
	case PassiveArrayTowed:
		base = 2.0 - towedPct*0.55
	default:
		base = 3.1
	}
	for bi := range bins {
		phase := gameTime*5.3 + float64(bi)*0.173
		flutter := math.Sin(phase)*0.55 +
			math.Sin(phase*2.17+0.8)*0.32 +
			math.Sin(phase*0.41+float64(bi)*0.07)*0.22 +
			math.Sin(gameTime*1.1+float64(bi)*0.53)*0.18
		noise := base + flutter
		if noise > bins[bi] {
			bins[bi] = noise
		} else {
			// Blend a little noise into existing returns for a living display.
			bins[bi] = bins[bi]*0.92 + noise*0.08
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
