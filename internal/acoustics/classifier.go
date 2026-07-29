package acoustics

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

// Classification matches a received signal shape against the signature library.
type Classification struct {
	ProfileID   string
	ProfileName string
	Confidence  float64
	BladeMatch  float64
}

// Classify compares received signal spectrum to known profiles.
func Classify(signal Spectrum, peakSNR, rangeYd float64) Classification {
	best := Classification{Confidence: 0.3}
	bestDist := math.MaxFloat64

	for _, p := range world.SignatureLibrary {
		template := templateSpectrum(p)
		dist := spectralDistance(signal, template)
		blade := bladeRateMatch(signal, p.BladeRateHz)

		conf := math.Max(0.3, 1.0-dist/35) + blade*0.1
		if peakSNR > 12 {
			conf += 0.04
		}
		if rangeYd < 6000 {
			conf += 0.03
		}
		// Weak waterfall-class SNR: tonals are unreliable — suppress auto-ID.
		clarity := SpectrumClarity01(peakSNR)
		conf *= 0.35 + 0.65*clarity
		if peakSNR < 9 {
			conf = math.Min(conf, 0.42)
		}
		if conf > 0.96 {
			conf = 0.96
		}

		if dist < bestDist {
			bestDist = dist
			best = Classification{
				ProfileID:   p.ID,
				ProfileName: p.Name,
				Confidence:  conf,
				BladeMatch:  blade,
			}
		}
	}
	return best
}

func spectralDistance(a, b Spectrum) float64 {
	var sum float64
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum / NumBands)
}

// AccumulateConfidence grows classification certainty while tracking a contact.
func AccumulateConfidence(current float64, measured float64, listenTime float64) float64 {
	rate := 0.015 + listenTime*0.0002
	if rate > 0.04 {
		rate = 0.04
	}
	next := current + (measured-current)*rate
	if next > 0.98 {
		next = 0.98
	}
	return next
}

func templateSpectrum(p world.SignatureProfile) Spectrum {
	var s Spectrum
	for i := 0; i < NumBands; i++ {
		freq := BandCenterHz(i)
		level := -200.0
		for _, b := range p.Bands {
			if freq >= b.LowHz && freq <= b.HighHz {
				level = combineDB(level, b.LevelDB)
			}
		}
		if level < -100 {
			level = 75
		}
		level += TonalBoostDB(p, freq)
		if p.BladeRateHz > 0 {
			rem := math.Mod(freq, p.BladeRateHz)
			if rem < p.BladeRateHz*0.1 {
				level += 6
			}
		}
		s[i] = level
	}
	return s
}

func bladeRateMatch(signal Spectrum, bladeHz float64) float64 {
	if bladeHz <= 0 {
		return 0
	}
	peak := 0.0
	sigPeak := signal.Peak()
	for i := 0; i < NumBands; i++ {
		freq := BandCenterHz(i)
		rem := math.Mod(freq, bladeHz)
		if rem < bladeHz*0.1 || bladeHz-rem < bladeHz*0.1 {
			if signal[i] > peak {
				peak = signal[i]
			}
		}
	}
	if sigPeak < -50 {
		return 0
	}
	return math.Min(1, math.Max(0, (peak-(sigPeak-18))/18))
}

// TemplateSpectrumForTest exposes template generation for unit tests in this package.
func TemplateSpectrumForTest(p world.SignatureProfile) Spectrum {
	return templateSpectrum(p)
}
