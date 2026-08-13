package acoustics

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

const (
	// VisualIdentifyMaxYd — periscope hull ID inside this true range.
	VisualIdentifyMaxYd = 800.0
	// HarmonicIdentifyMatchFrac — fraction of library fingerprint lines that
	// must be present in the received spectrum.
	HarmonicIdentifyMatchFrac = 0.80
	// HarmonicIdentifyHoldSec — consecutive (accumulated) good-match time.
	HarmonicIdentifyHoldSec = 120.0
)

const (
	IdentifiedByVisual   = "visual"
	IdentifiedByAcoustic = "acoustic"
)

// HarmonicMatchFraction is how many of profile's эталон tonals are visible
// in the received classify spectrum (0..1).
func HarmonicMatchFraction(signal Spectrum, profile world.SignatureProfile) float64 {
	refs := ProfileReferencePeaks(profile)
	if len(refs) == 0 {
		return 0
	}
	matched := 0
	for _, peak := range refs {
		if tonalPresentInSignal(signal, peak.FreqHz) {
			matched++
		}
	}
	return float64(matched) / float64(len(refs))
}

func tonalPresentInSignal(signal Spectrum, freqHz float64) bool {
	if freqHz < MinFreqHz || freqHz > MaxFreqHz {
		return false
	}
	i := BandIndexForHz(freqHz)
	v := signal[i]
	if i > 0 && signal[i-1] > v {
		v = signal[i-1]
	}
	if i+1 < NumBands && signal[i+1] > v {
		v = signal[i+1]
	}
	if v < DetectThreshold {
		return false
	}
	med := spectrumMedian(signal)
	return v >= med+2.0
}

func spectrumMedian(signal Spectrum) float64 {
	tmp := make([]float64, NumBands)
	copy(tmp, signal[:])
	for i := 1; i < len(tmp); i++ {
		v := tmp[i]
		j := i
		for j > 0 && tmp[j-1] > v {
			tmp[j] = tmp[j-1]
			j--
		}
		tmp[j] = v
	}
	if NumBands%2 == 0 {
		return 0.5 * (tmp[NumBands/2-1] + tmp[NumBands/2])
	}
	return tmp[NumBands/2]
}

// IdentifyContact locks the contact to the emitter's true library class.
// Returns true when this call newly identified the track.
func IdentifyContact(c *Contact, em *world.Entity, by string, gameTime float64) bool {
	if c == nil || em == nil || c.Identified {
		return false
	}
	if em.Kind == world.KindTorpedo {
		return false
	}
	p, ok := world.ProfileByID(em.SignatureID)
	if !ok {
		return false
	}
	c.Identified = true
	c.IdentifiedBy = by
	c.IdentifiedAt = gameTime
	c.ConfirmedID = p.ID
	c.ConfirmedClass = p.Name
	c.Kind = p.Kind
	c.BestMatchID = p.ID
	c.BestMatchName = p.Name
	if c.Confidence < 0.90 {
		c.Confidence = 0.90
	}
	return true
}

func tryAcousticIdentify(c *Contact, signal Spectrum, em *world.Entity, dt, gameTime float64) {
	if c == nil || em == nil || c.Identified {
		return
	}
	if em.Kind == world.KindTorpedo {
		return
	}
	p, ok := world.ProfileByID(em.SignatureID)
	if !ok {
		return
	}
	frac := HarmonicMatchFraction(signal, p)
	c.HarmonicMatch = frac
	if dt <= 0 || dt > 1 {
		dt = 0.1
	}
	if frac >= HarmonicIdentifyMatchFrac {
		c.HarmonicHoldSec += dt
	} else if c.HarmonicHoldSec > 0 && frac < HarmonicIdentifyMatchFrac*0.5 {
		// Lost the fingerprint — drop the hold rather than bank a lucky minute.
		c.HarmonicHoldSec = 0
	}
	if c.HarmonicHoldSec >= HarmonicIdentifyHoldSec {
		IdentifyContact(c, em, IdentifiedByAcoustic, gameTime)
	}
}

func tryVisualIdentify(c *Contact, em *world.Entity, rangeYd, gameTime float64) {
	if c == nil || em == nil || c.Identified {
		return
	}
	if rangeYd <= 0 || rangeYd >= VisualIdentifyMaxYd {
		return
	}
	IdentifyContact(c, em, IdentifiedByVisual, gameTime)
}

// NewlyIdentifiedContacts returns tracks that locked ID on this tick.
func NewlyIdentifiedContacts(sonar *SonarState, gameTime float64) []Contact {
	if sonar == nil {
		return nil
	}
	var out []Contact
	for i := range sonar.Contacts {
		c := sonar.Contacts[i]
		if !c.Identified || c.IdentifiedAt <= 0 {
			continue
		}
		if math.Abs(gameTime-c.IdentifiedAt) <= 0.12 {
			out = append(out, c)
		}
	}
	return out
}
