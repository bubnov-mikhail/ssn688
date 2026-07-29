package acoustics

import (
	"math"
	"math/rand"
)

// BioTransient is a short-lived natural sound (whale, dolphin) for passive display.
// It is never promoted to a Contact track.
type BioTransient struct {
	BearingDeg float64
	PeakSNR    float64
	ExpireAt   float64
	Kind       string  // "whale" | "dolphin"
	FreqBiasHz float64 // characteristic frequency for band filtering
}

const (
	bioMinIntervalSec = 55.0
	bioMaxIntervalSec = 140.0
	bioMinDurationSec = 8.0
	bioMaxDurationSec = 22.0
)

// UpdateBiology occasionally spawns faint natural sounds; keeps them rare and short.
func UpdateBiology(sonar *SonarState, gameTime float64, rng *rand.Rand) {
	if sonar == nil {
		return
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(int64(gameTime*1000) ^ 0xB10))
	}
	kept := sonar.BioTransients[:0]
	for _, b := range sonar.BioTransients {
		if gameTime < b.ExpireAt {
			kept = append(kept, b)
		}
	}
	sonar.BioTransients = kept

	if sonar.nextBioAt <= 0 {
		sonar.nextBioAt = gameTime + bioMinIntervalSec + rng.Float64()*(bioMaxIntervalSec-bioMinIntervalSec)
	}
	if gameTime < sonar.nextBioAt || len(sonar.BioTransients) >= 2 {
		return
	}
	sonar.nextBioAt = gameTime + bioMinIntervalSec + rng.Float64()*(bioMaxIntervalSec-bioMinIntervalSec)

	kind := "whale"
	freq := 40.0 + rng.Float64()*80 // LF moan
	snr := 6.0 + rng.Float64()*6    // faint — rarely masks contacts
	if rng.Float64() < 0.4 {
		kind = "dolphin"
		freq = 900 + rng.Float64()*700 // clicks more HF
		snr = 5.0 + rng.Float64()*5
	}
	dur := bioMinDurationSec + rng.Float64()*(bioMaxDurationSec-bioMinDurationSec)
	sonar.BioTransients = append(sonar.BioTransients, BioTransient{
		BearingDeg: rng.Float64() * 360,
		PeakSNR:    snr,
		ExpireAt:   gameTime + dur,
		Kind:       kind,
		FreqBiasHz: freq,
	})
}

// AddPassiveTransient injects a short-lived waterfall-only transient cue.
func AddPassiveTransient(sonar *SonarState, bearingDeg, peakSNR, durationSec float64, kind string, freqBiasHz float64, gameTime float64) {
	if sonar == nil || durationSec <= 0 || peakSNR <= 0 {
		return
	}
	sonar.BioTransients = append(sonar.BioTransients, BioTransient{
		BearingDeg: bearingDeg,
		PeakSNR:    peakSNR,
		ExpireAt:   gameTime + durationSec,
		Kind:       kind,
		FreqBiasHz: freqBiasHz,
	})
}

// BioWaterfallContribution returns display energy for a bio transient under the listen band.
func BioWaterfallContribution(b BioTransient, band ListenBand, gameTime float64) float64 {
	remain := b.ExpireAt - gameTime
	if remain <= 0 {
		return 0
	}
	env := 1.0
	if remain < 3 {
		env = remain / 3
	}
	total := bioMaxDurationSec
	if total < remain {
		total = remain + 1
	}
	lived := total - remain
	ageFade := 1.0
	if lived < 2 {
		ageFade = math.Max(0.15, lived/2)
	}
	gain := BandDisplayGain(band, b.FreqBiasHz)
	return b.PeakSNR * env * ageFade * gain * 0.85
}

// BioSpectrumFloor adds a little natural hash near a bio bearing for the analyzer.
func BioSpectrumFloor(bins []float64, sonar *SonarState, bearingDeg, gameTime float64) {
	if sonar == nil || len(bins) == 0 {
		return
	}
	for _, b := range sonar.BioTransients {
		if math.Abs(normalizeBearingDiff(b.BearingDeg-bearingDeg)) > 18 {
			continue
		}
		gain := BandDisplayGain(sonar.ListenBand, b.FreqBiasHz)
		if gain < 0.15 {
			continue
		}
		level := BioWaterfallContribution(b, sonar.ListenBand, gameTime) * 0.35
		for i := range bins {
			freq := BandCenterHz(i)
			w := 1.0 - math.Min(1, math.Abs(freq-b.FreqBiasHz)/(b.FreqBiasHz*0.8+80))
			if w <= 0 {
				continue
			}
			v := level * w * gain
			if v > bins[i] {
				bins[i] = bins[i]*0.7 + v*0.3
			} else {
				bins[i] += v * 0.08
			}
		}
	}
}
