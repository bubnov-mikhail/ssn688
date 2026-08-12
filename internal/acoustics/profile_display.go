package acoustics

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

// SpectrumPeak is a display sample for Cold Waters-style signature panels.
type SpectrumPeak struct {
	FreqHz float64
	Level  float64 // 0..1
}

// ProfileReferencePeaks returns sharp library fingerprint lines.
func ProfileReferencePeaks(profile world.SignatureProfile) []SpectrumPeak {
	if len(profile.Tonals) == 0 {
		return legacyBladePeaks(profile)
	}
	out := make([]SpectrumPeak, 0, len(profile.Tonals))
	for _, t := range profile.Tonals {
		if t.FreqHz < MinFreqHz || t.FreqHz > MaxFreqHz {
			continue
		}
		lvl := t.RelLevel
		if lvl < 0.15 {
			lvl = 0.15
		}
		if lvl > 1 {
			lvl = 1
		}
		out = append(out, SpectrumPeak{FreqHz: t.FreqHz, Level: lvl})
	}
	return out
}

func legacyBladePeaks(profile world.SignatureProfile) []SpectrumPeak {
	var out []SpectrumPeak
	for _, f := range ProfileBladeHarmonics(profile, 12) {
		out = append(out, SpectrumPeak{FreqHz: f, Level: 0.7})
	}
	return out
}

// ProfileBladeHarmonics returns blade-rate harmonic frequencies for markers.
func ProfileBladeHarmonics(profile world.SignatureProfile, maxHarmonics int) []float64 {
	if profile.BladeRateHz <= 0 || maxHarmonics < 1 {
		return nil
	}
	var freqs []float64
	for n := 1; n <= maxHarmonics; n++ {
		f := profile.BladeRateHz * float64(n)
		if f > MaxFreqHz {
			break
		}
		if f >= MinFreqHz {
			freqs = append(freqs, f)
		}
	}
	return freqs
}

// ObservedPeaksFromBins converts analyzer SNR bins into a dense peak list for
// the fuzzy live-signal panel (Cold Waters lower strip).
// Weak absolute SNR (same regime as a faint waterfall trace) keeps harmonics
// buried in the noise floor instead of auto-stretching them to full height.
func ObservedPeaksFromBins(bins []float64) []SpectrumPeak {
	if len(bins) == 0 {
		return nil
	}
	peak := 0.0
	for _, v := range bins {
		if v > peak {
			peak = v
		}
	}
	clarity := SpectrumClarity01(peak)
	const refDB = 24.0 // absolute full-scale reference (not per-frame max)
	out := make([]SpectrumPeak, len(bins))
	for i, v := range bins {
		abs := v / refDB
		if abs < 0 {
			abs = 0
		}
		if abs > 1 {
			abs = 1
		}
		// Rising noise floor as clarity falls — tonals drown first.
		floor := 0.10 + (1-clarity)*0.48
		// Absolute amplitude, compressed when the track is weak.
		lvl := abs * (0.18 + 0.82*clarity)
		detect := 0.10 + 0.40*(1-clarity)
		if abs < detect {
			// Below detection margin: show only noisy floor, no harmonic tip.
			lvl = floor * (0.55 + 0.45*abs/math.Max(detect, 0.01))
		} else {
			lvl = floor + (lvl-floor)*clarity
		}
		// Local-peak emphasis only when the waterfall would also look strong.
		if clarity > 0.38 && i > 0 && i < len(bins)-1 && v >= bins[i-1] && v >= bins[i+1] {
			lvl = math.Min(1, lvl*(1+0.28*clarity)+0.06*clarity)
		}
		if lvl < 0 {
			lvl = 0
		}
		if lvl > 1 {
			lvl = 1
		}
		out[i] = SpectrumPeak{FreqHz: BandCenterHz(i), Level: lvl}
	}
	return out
}

// SpectrumClarity01 maps peak analyzer SNR to 0..1 tonal readability.
//
// Open-source LOFAR/DEMON guidance (harbor DEMON lines to ~7 km; hull passive
// often a few nm for useful lines; LF towed arrays much farther for the same
// machinery fingerprint). Tuned so ClassifyClarityMin (~0.18) is reached near:
//   noisy ASW corvette ~8–10 kyd on HULL / ~15–18 kyd on full TOWED,
//   merchant / noisy diesel ~5–8 kyd HULL / ~10–14 kyd TOWED —
// rather than forcing the operator inside weapons range to read harmonics.
func SpectrumClarity01(peakSNR float64) float64 {
	t := (peakSNR - 5.0) / 22.0
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return math.Pow(t, 1.1)
}

// DegradeSpectrumBinsForClarity smears and buries weak spectra so distant /
// quiet contacts do not retain crisp LOFAR lines in the analyzer.
func DegradeSpectrumBinsForClarity(bins []float64) {
	if len(bins) < 3 {
		return
	}
	peak := 0.0
	for _, v := range bins {
		if v > peak {
			peak = v
		}
	}
	clarity := SpectrumClarity01(peak)
	if clarity >= 0.88 {
		return
	}
	mud := 1 - clarity
	tmp := make([]float64, len(bins))
	copy(tmp, bins)
	for i := range bins {
		// Milder smear — keep LOFAR tips readable once SNR clears the floor.
		left, right := tmp[i], tmp[i]
		if i > 0 {
			left = tmp[i-1]
		}
		if i < len(tmp)-1 {
			right = tmp[i+1]
		}
		smeared := tmp[i]*(1-0.40*mud) + (left+right)*0.5*(0.40*mud)
		floor := 1.5 + mud*4.5
		noise := (float64((i*37+11)%17)/17.0 - 0.5) * mud * 3.2
		v := smeared*(0.45+0.55*clarity) + floor*mud*0.50 + noise
		if v < floor*0.35 {
			v = floor * 0.35
		}
		bins[i] = v
	}
}

// ProfileDisplayBins keeps a coarse bar view for tests / legacy callers.
func ProfileDisplayBins(profile world.SignatureProfile) []float64 {
	s := templateSpectrum(profile)
	out := make([]float64, NumBands)
	for i := range s {
		out[i] = math.Max(0, (s[i]-55)*0.12)
	}
	return out
}

// TonalBoostDB returns extra source level if freq is near a library tonal.
func TonalBoostDB(profile world.SignatureProfile, freqHz float64) float64 {
	best := 0.0
	for _, t := range profile.Tonals {
		bw := math.Max(4, t.FreqHz*0.03)
		d := math.Abs(freqHz - t.FreqHz)
		if d > bw {
			continue
		}
		boost := (8 + t.RelLevel*14) * (1 - d/bw)
		if boost > best {
			best = boost
		}
	}
	return best
}
