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

	for i, p := range world.SignatureLibrary {
		var template Spectrum
		var blade float64
		if i < len(profileCaches) {
			c := &profileCaches[i]
			template = c.template
			blade = bladeRateMatchCached(signal, c)
		} else {
			template = templateSpectrum(p)
			blade = bladeRateMatch(signal, p.BladeRateHz)
		}
		dist := spectralDistance(signal, template)

		conf := math.Max(0.3, 1.0-dist/35) + blade*0.1
		if peakSNR > 10 {
			conf += 0.04
		}
		// Favor fingerprints inside typical LOFAR class ranges (~5 nm / ~10 kyd).
		if rangeYd < 10000 {
			conf += 0.03
		}
		// Weak waterfall-class SNR: tonals are unreliable — suppress auto-ID.
		clarity := SpectrumClarity01(peakSNR)
		conf *= 0.40 + 0.60*clarity
		if peakSNR < 8 {
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

// IsTorpedoProfile reports whether a signature library ID is a torpedo class.
func IsTorpedoProfile(id string) bool {
	return id == "mk48" || id == "type53" || id == "umgt1" || id == "set40" || id == "mk46"
}

// TryAutoClassifyTorpedo confirms TORP when harmonics / match look like a fish.
// Returns true when this call newly confirmed the contact as a torpedo.
func TryAutoClassifyTorpedo(c *Contact, class Classification) bool {
	if c == nil || !IsTorpedoProfile(class.ProfileID) {
		return false
	}
	// Need clear HF blade harmonics and a usable match — not a weak broadband guess.
	if class.BladeMatch < 0.40 && class.Confidence < 0.58 {
		return false
	}
	if class.Confidence < 0.50 {
		return false
	}
	was := c.ConfirmedClass != "" && c.Kind == world.KindTorpedo
	c.BestMatchID = class.ProfileID
	c.BestMatchName = class.ProfileName
	c.ConfirmedID = class.ProfileID
	c.ConfirmedClass = "TORP"
	c.Kind = world.KindTorpedo
	if c.Confidence < class.Confidence {
		c.Confidence = class.Confidence
	}
	if c.Confidence < 0.72 {
		c.Confidence = 0.72
	}
	return !was
}

// KindFromMatch returns library Kind for a confident match; unknown otherwise.
func KindFromMatch(class Classification) world.EntityKind {
	if class.Confidence < 0.50 {
		return world.EntityKind(-1)
	}
	if p, ok := world.ProfileByID(class.ProfileID); ok {
		return p.Kind
	}
	return world.EntityKind(-1)
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
	if c := cacheForProfile(p); c != nil {
		return c.template
	}
	return computeTemplateSpectrum(p, nil)
}

func computeTemplateSpectrum(p world.SignatureProfile, c *profileBandCache) Spectrum {
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
		if c != nil {
			level += c.tonalBoost[i]
			if c.bladeTemplate[i] {
				level += 6
			}
		} else {
			level += TonalBoostDB(p, freq)
			if p.BladeRateHz > 0 {
				rem := math.Mod(freq, p.BladeRateHz)
				if rem < p.BladeRateHz*0.1 {
					level += 6
				}
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

func bladeRateMatchCached(signal Spectrum, c *profileBandCache) float64 {
	if c == nil || c.bladeHz <= 0 {
		return 0
	}
	peak := 0.0
	sigPeak := signal.Peak()
	for i := 0; i < NumBands; i++ {
		if !c.bladeClass[i] {
			continue
		}
		if signal[i] > peak {
			peak = signal[i]
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
