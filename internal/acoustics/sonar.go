package acoustics

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/ssn688/sim/internal/world"
)

// Contact is a sonar detection track.
type Contact struct {
	ID               string
	BearingDeg       float64
	EstimatedRangeYd float64
	UncBearingDeg    float64 // display bearing uncertainty (shrinks with track quality)
	UncRangeYd       float64 // display range uncertainty radius
	SNR              float64
	BandsAbove       int
	BestMatchID      string
	BestMatchName    string
	ConfirmedID      string
	ConfirmedClass   string
	Confidence       float64
	SourceEntityID   string
	Kind             world.EntityKind
	DetectedBy       string // passive, active
	LastUpdate       float64
	FirstSeen        float64
	ListenTime       float64
}

// SonarState holds player sonar equipment state.
type SonarState struct {
	PassiveEnabled  bool
	ActiveEnabled   bool
	ActivePower     float64
	LastPingTime    float64
	PingInterval    float64
	SpectrumBearing float64
	PassiveArray    PassiveArrayKind
	TowedCablePct   float64
	TowedCableRate  float64
	Contacts        []Contact
	contactSeq      int
}

func NewSonarState() SonarState {
	return SonarState{
		PassiveEnabled:  true,
		ActiveEnabled:   false,
		ActivePower:     0.7,
		PingInterval:    12.0,
		SpectrumBearing: 0,
	}
}

// UpdatePassive processes passive detections for the listener.
func UpdatePassive(model Model, listener *world.Entity, emitters []*world.Entity, sonar *SonarState, gameTime float64) {
	if !sonar.PassiveEnabled {
		return
	}

	existing := map[string]int{}
	for i, c := range sonar.Contacts {
		existing[c.SourceEntityID] = i
	}

	for _, em := range emitters {
		if em.ID == listener.ID || !em.Alive() {
			continue
		}

		result := model.Detect(listener, em, ModePassive, 0)
		applyPassiveArrayModifiers(&result, sonar)

		// Active ping from target greatly increases passive detectability.
		if em.ActiveSonar && gameTime-em.LastPingTime < 2.5 {
			result.PeakSNR += 10
			result.BandsAbove += 3
			if result.PeakSNR >= PeakDetectSNR || result.BandsAbove >= MinDetectBands {
				result.Detected = true
			}
		}

		if !result.Detected {
			continue
		}

		class := Classify(result.SignalForClassify, result.PeakSNR, result.TrueRangeYd)
		class = refineClassification(class, result.SignalForClassify, em.SignatureID)
		bearing := bearingWithError(result.BearingDeg, result.PeakSNR, result.BandsAbove, sonar.passiveBearingSigmaScale())
		estRange := estimatePassiveRange(model.Env, listener, em, result.TrueRangeYd, result.PeakSNR)

		if idx, ok := existing[em.ID]; ok {
			c := &sonar.Contacts[idx]
			updateContactTrack(c, bearing, estRange, result.PeakSNR, result.BandsAbove, class, gameTime, em)
			continue
		}

		sonar.contactSeq++
		c := Contact{
			ID:               formatContactID(sonar.contactSeq),
			BearingDeg:       bearing,
			EstimatedRangeYd: estRange,
			SNR:              result.PeakSNR,
			BandsAbove:       result.BandsAbove,
			BestMatchID:      class.ProfileID,
			BestMatchName:    class.ProfileName,
			Confidence:       class.Confidence,
			SourceEntityID:   em.ID,
			Kind:             em.Kind,
			DetectedBy:       "passive",
			LastUpdate:       gameTime,
			FirstSeen:        gameTime,
		}
		if em.ActiveSonar && gameTime-em.LastPingTime < 2.5 {
			c.DetectedBy = "passive/ping"
		}
		initContactUncertainty(&c)
		sonar.Contacts = append(sonar.Contacts, c)
	}

	filtered := sonar.Contacts[:0]
	for _, c := range sonar.Contacts {
		if gameTime-c.LastUpdate < 50 {
			filtered = append(filtered, c)
		}
	}
	sonar.Contacts = filtered
}

// FireActivePing emits active sonar and creates or updates contacts.
func FireActivePing(model Model, listener *world.Entity, emitters []*world.Entity, sonar *SonarState, gameTime float64) {
	if !sonar.ActiveEnabled || gameTime-sonar.LastPingTime < sonar.PingInterval {
		return
	}
	sonar.LastPingTime = gameTime
	listener.ActiveSonar = true
	listener.LastPingTime = gameTime

	for _, em := range emitters {
		if em.ID == listener.ID || !em.Alive() {
			continue
		}

		result := model.Detect(listener, em, ModeActive, sonar.ActivePower)
		if !result.Detected {
			continue
		}

		class := Classify(result.SignalForClassify, result.PeakSNR, result.TrueRangeYd)
		class = refineClassification(class, result.SignalForClassify, em.SignatureID)
		class.Confidence = math.Min(0.99, class.Confidence+0.12)

		found := false
		for i := range sonar.Contacts {
			if sonar.Contacts[i].SourceEntityID == em.ID {
				c := &sonar.Contacts[i]
				measRange := result.TrueRangeYd + rand.NormFloat64()*result.TrueRangeYd*0.02
				updateContactTrack(c, result.BearingDeg, measRange, result.PeakSNR, result.BandsAbove, class, gameTime, em)
				c.DetectedBy = "active"
				c.Confidence = math.Max(c.Confidence, class.Confidence)
				// Active ranging tightens the blob quickly.
				c.UncBearingDeg = math.Min(c.UncBearingDeg, 2.5)
				c.UncRangeYd = math.Min(c.UncRangeYd, math.Max(80, result.TrueRangeYd*0.04))
				found = true
				break
			}
		}
		if !found {
			sonar.contactSeq++
			c := Contact{
				ID:               formatContactID(sonar.contactSeq),
				BearingDeg:       result.BearingDeg,
				EstimatedRangeYd: result.TrueRangeYd,
				SNR:              result.PeakSNR,
				BandsAbove:       result.BandsAbove,
				BestMatchID:      class.ProfileID,
				BestMatchName:    class.ProfileName,
				Confidence:       class.Confidence,
				SourceEntityID:   em.ID,
				Kind:             em.Kind,
				DetectedBy:       "active",
				LastUpdate:       gameTime,
				FirstSeen:        gameTime,
				UncBearingDeg:    2.5,
				UncRangeYd:       math.Max(80, result.TrueRangeYd*0.04),
			}
			sonar.Contacts = append(sonar.Contacts, c)
		}
	}
}

// SpectrumAtBearing returns SNR spectrum for the analyzer display at a bearing.
func SpectrumAtBearing(model Model, listener *world.Entity, emitters []*world.Entity, sonar *SonarState, bearingDeg float64) []float64 {
	selfNoise := SelfNoiseSpectrum(listener, model.Env, sonar.PassiveArray, sonar.TowedCablePct)
	out := make([]float64, NumBands)

	for _, em := range emitters {
		if !em.Alive() || em.ID == listener.ID {
			continue
		}
		b := listener.BearingDegTo(em)
		if math.Abs(normalizeBearingDiff(b-bearingDeg)) > 6 {
			continue
		}
		src := SourceSpectrum(em)
		recv := Propagate(model.Env, src, em, listener)
		snr := recv.SubNoise(selfNoise)
		bonus := sonar.passiveSNRBonusDB()
		for i := 0; i < NumBands; i++ {
			v := snr[i] + bonus
			if v > out[i] {
				out[i] = v
			}
		}
	}

	// Ambient noise floor visible in analyzer.
	for i := 0; i < NumBands; i++ {
		out[i] = math.Max(out[i], 0) + 1.5 + rand.Float64()
	}
	return out
}

func bearingWithError(trueBearing, peakSNR float64, bands int, sigmaScale float64) float64 {
	sigma := 3.5 - float64(bands)*0.15
	if peakSNR > 14 {
		sigma -= 1.2
	}
	if peakSNR > 20 {
		sigma -= 0.8
	}
	sigma *= sigmaScale
	if sigma < 0.4 {
		sigma = 0.4
	}
	return trueBearing + rand.NormFloat64()*sigma
}

func estimatePassiveRange(env Environment, listener, emitter *world.Entity, trueRange, peakSNR float64) float64 {
	uncertainty := 0.18 + (PeakDetectSNR-peakSNR)*0.02
	if uncertainty < 0.12 {
		uncertainty = 0.12
	}
	if env.layerIndex(listener.DepthFt) != env.layerIndex(emitter.DepthFt) {
		uncertainty += 0.22
	}
	return trueRange * (1 + rand.NormFloat64()*uncertainty)
}

func formatContactID(n int) string {
	return fmt.Sprintf("C%02d", n)
}

func lerpAngleDeg(from, to, alpha float64) float64 {
	d := normalizeBearingDiff(to - from)
	out := from + d*alpha
	for out < 0 {
		out += 360
	}
	for out >= 360 {
		out -= 360
	}
	return out
}

func initContactUncertainty(c *Contact) {
	c.UncBearingDeg = 28
	c.UncRangeYd = math.Max(600, c.EstimatedRangeYd*0.45)
}

func targetUncertainty(listenTime, confidence, snr float64, rangeYd float64) (bearDeg, rangeYdUnc float64) {
	// ~2 minutes of listening + strong SNR shrinks the blob.
	t := math.Min(1, listenTime/120)
	snrT := math.Min(1, math.Max(0, (snr-6)/18))
	q := 0.45*t + 0.35*confidence + 0.20*snrT
	bearDeg = 26*(1-q) + 1.8
	rangeYdUnc = (0.42*(1-q)+0.05)*math.Max(400, rangeYd)
	if rangeYdUnc < 120 {
		rangeYdUnc = 120
	}
	return bearDeg, rangeYdUnc
}

func shrinkUncertainty(cur, target float64) float64 {
	if cur <= 0 {
		return target
	}
	// Monotonic shrink toward better estimate; never jump larger suddenly.
	if target < cur {
		return cur*0.90 + target*0.10
	}
	return cur*0.995 + target*0.005
}

func updateContactTrack(c *Contact, measBearing, measRange, snr float64, bands int, class Classification, gameTime float64, em *world.Entity) {
	c.ListenTime = gameTime - c.FirstSeen
	c.LastUpdate = gameTime
	c.SNR = snr
	c.BandsAbove = bands
	c.BestMatchID = class.ProfileID
	c.BestMatchName = class.ProfileName
	c.Confidence = AccumulateConfidence(c.Confidence, class.Confidence, c.ListenTime)
	c.DetectedBy = "passive"
	if em.ActiveSonar && gameTime-em.LastPingTime < 2.5 {
		c.DetectedBy = "passive/ping"
	}

	// Smooth track so the blob centre does not jump each noisy sample.
	alpha := 0.06 + math.Min(0.28, snr/50)
	c.BearingDeg = lerpAngleDeg(c.BearingDeg, measBearing, alpha)
	if c.EstimatedRangeYd <= 0 {
		c.EstimatedRangeYd = measRange
	} else {
		c.EstimatedRangeYd = c.EstimatedRangeYd*(1-alpha) + measRange*alpha
	}

	tb, tr := targetUncertainty(c.ListenTime, c.Confidence, snr, c.EstimatedRangeYd)
	c.UncBearingDeg = shrinkUncertainty(c.UncBearingDeg, tb)
	c.UncRangeYd = shrinkUncertainty(c.UncRangeYd, tr)
}

func normalizeBearingDiff(d float64) float64 {
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}

func refineClassification(class Classification, signal Spectrum, trueSignatureID string) Classification {
	profile, ok := world.ProfileByID(trueSignatureID)
	if !ok {
		return class
	}
	dist := spectralDistance(signal, templateSpectrum(profile))
	if dist < 30 {
		class.ProfileID = profile.ID
		class.ProfileName = profile.Name
		class.Confidence = math.Max(class.Confidence, 0.55)
	}
	return class
}
