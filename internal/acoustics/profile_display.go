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
func ObservedPeaksFromBins(bins []float64) []SpectrumPeak {
	if len(bins) == 0 {
		return nil
	}
	maxV := 1.0
	for _, v := range bins {
		if v > maxV {
			maxV = v
		}
	}
	out := make([]SpectrumPeak, len(bins))
	for i, v := range bins {
		lvl := math.Max(0, v/maxV)
		// Emphasize local peaks so tonals read as brighter streaks.
		if i > 0 && i < len(bins)-1 && v >= bins[i-1] && v >= bins[i+1] {
			lvl = math.Min(1, lvl*1.25+0.08)
		}
		out[i] = SpectrumPeak{FreqHz: BandCenterHz(i), Level: lvl}
	}
	return out
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
