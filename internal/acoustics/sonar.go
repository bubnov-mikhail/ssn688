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

	// Last active range-bearing snapshot (held for ActiveFixHoldSec on the plot).
	LastActiveBearingDeg float64
	LastActiveRangeYd    float64
	LastActiveFixAt      float64
}

// ActiveFixHoldSec is how long a solid active fix anchors the tactical plot
// (matches the ACTIVE echo marker fade window).
const ActiveFixHoldSec = 30.0

// ContactActiveFixValid reports whether the last active fix is still within the hold window.
func ContactActiveFixValid(c *Contact, gameTime float64) bool {
	if c == nil || c.LastActiveFixAt <= 0 || c.LastActiveRangeYd < 50 {
		return false
	}
	return gameTime-c.LastActiveFixAt <= ActiveFixHoldSec
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
	ListenBand      ListenBand
	Contacts        []Contact
	BioTransients   []BioTransient
	nextBioAt       float64
	// SonarDeafUntil — temporary washout after nearby underwater detonation.
	SonarDeafUntil   float64
	LastBlastAt      float64
	LastBlastX       float64
	LastBlastY       float64
	LastBlastRangeYd float64 // washout visibility radius for this event
	LastBlastFlashSec float64
	contactSeq       int
	// activeEchoDone marks SourceEntityIDs already processed for the current ping.
	activeEchoDone map[string]bool
	activeEchoAt   float64
	// passiveContactIndex is scratch for UpdatePassive (reused each tick).
	passiveContactIndex map[string]int
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

	if sonar.passiveContactIndex == nil {
		sonar.passiveContactIndex = make(map[string]int)
	} else {
		clear(sonar.passiveContactIndex)
	}
	existing := sonar.passiveContactIndex
	for i, c := range sonar.Contacts {
		existing[c.SourceEntityID] = i
	}

	deafActive := gameTime < sonar.SonarDeafUntil && sonar.LastBlastAt > 0
	var blastBrg float64
	if deafActive {
		blastBrg = math.Atan2(sonar.LastBlastX-listener.X, sonar.LastBlastY-listener.Y) * 180 / math.Pi
		if blastBrg < 0 {
			blastBrg += 360
		}
	}

	for _, em := range emitters {
		if em.ID == listener.ID {
			continue
		}
		if !em.Alive() && em.Status != world.StatusSinking {
			continue
		}

		result := model.Detect(listener, em, ModePassive, 0)
		ApplyListenBand(&result, sonar.ListenBand)
		applyPassiveArrayModifiers(&result, sonar)
		rel := AngleDiffDeg(result.BearingDeg, listener.HeadingDeg)
		sensDB := 20 * math.Log10(math.Max(PassiveArraySensitivity(sonar.PassiveArray, rel, sonar.TowedCablePct), 0.001))
		result.PeakSNR += sensDB
		for i := range result.SNR {
			result.SNR[i] += sensDB
		}
		for i := range result.SignalForClassify {
			result.SignalForClassify[i] += sensDB * 0.65
		}
		result.BandsAbove = result.SNR.BandsAbove(DetectThreshold)
		result.Detected = result.BandsAbove >= MinDetectBands || result.PeakSNR >= PeakDetectSNR
		if penalty := PassiveSelfNoiseDeltaDB(sonar.PassiveArray, rel, listener.SpeedKts, listener.DepthFt, sonar.TowedCablePct); penalty != 0 {
			result.PeakSNR -= penalty
			for i := range result.SNR {
				result.SNR[i] -= penalty
			}
			for i := range result.SignalForClassify {
				result.SignalForClassify[i] -= penalty * 0.65
			}
			result.BandsAbove = result.SNR.BandsAbove(DetectThreshold)
			result.Detected = result.BandsAbove >= MinDetectBands || result.PeakSNR >= PeakDetectSNR
		}

		// Directional blast deaf: heavy toward explosion, mild opposite.
		if deafActive {
			ang := math.Abs(AngleDiffDeg(result.BearingDeg, blastBrg))
			toward := math.Exp(-(ang * ang) / (2 * 55 * 55))
			deafDB := 8 + 42*toward
			result.PeakSNR -= deafDB
			for i := range result.SNR {
				result.SNR[i] -= deafDB
			}
			for i := range result.SignalForClassify {
				result.SignalForClassify[i] -= deafDB * 0.65
			}
			result.BandsAbove = result.SNR.BandsAbove(DetectThreshold)
			result.Detected = result.BandsAbove >= MinDetectBands || result.PeakSNR >= PeakDetectSNR
		}

		// Active ping from target greatly increases passive detectability.
		if age := EnemyActivePingAgeSec(em, gameTime); age >= 0 && age < 2.5 {
			result.PeakSNR += 10 + 20*math.Exp(-age*1.2)
			result.BandsAbove += 3
			if result.PeakSNR >= PeakDetectSNR || result.BandsAbove >= MinDetectBands {
				result.Detected = true
			}
		}

		if !result.Detected {
			continue
		}

		classifySig := ContaminateClassifySignal(result.SignalForClassify, model, listener, em.ID, emitters, result.BearingDeg, sonar)
		class := Classify(classifySig, result.PeakSNR, result.TrueRangeYd)
		class = refineClassification(class, classifySig, em.SignatureID, result.PeakSNR)
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
		if age := EnemyActivePingAgeSec(em, gameTime); age >= 0 && age < 2.5 {
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

const (
	PingIntervalMinSec = 0   // 0 = manual pings only
	PingIntervalMaxSec = 60
	// ActiveDisplayMaxRangeYd is the PPI scale for the active range display.
	ActiveDisplayMaxRangeYd = 12000.0
)

// FireActivePing emits active sonar on the auto-ping schedule.
func FireActivePing(model Model, listener *world.Entity, emitters []*world.Entity, sonar *SonarState, gameTime float64) {
	if !sonar.ActiveEnabled || sonar.PingInterval <= PingIntervalMinSec {
		return
	}
	if gameTime-sonar.LastPingTime < sonar.PingInterval {
		return
	}
	transmitActivePing(listener, sonar, gameTime)
}

// FireActivePingNow transmits a single pulse immediately, even when active mode
// is on standby (auto-ping still requires ActiveEnabled).
func FireActivePingNow(model Model, listener *world.Entity, emitters []*world.Entity, sonar *SonarState, gameTime float64) bool {
	if sonar == nil || listener == nil {
		return false
	}
	transmitActivePing(listener, sonar, gameTime)
	return true
}

func transmitActivePing(listener *world.Entity, sonar *SonarState, gameTime float64) {
	sonar.LastPingTime = gameTime
	listener.ActiveSonar = true
	listener.LastPingTime = gameTime
	listener.LastPingPower = sonar.ActivePower
	if listener.LastPingPower <= 0 {
		listener.LastPingPower = 0.7
	}
	sonar.activeEchoAt = gameTime
	if sonar.activeEchoDone == nil {
		sonar.activeEchoDone = map[string]bool{}
	} else {
		clear(sonar.activeEchoDone)
	}
}

// ExpireActivePingIfBeyondDisplay clears in-flight active ping state once the
// expanding echo front passes the display range cap (12 kyd).
func ExpireActivePingIfBeyondDisplay(sonar *SonarState, listener *world.Entity, gameTime float64) bool {
	if sonar == nil || sonar.LastPingTime <= 0 {
		return false
	}
	age := gameTime - sonar.LastPingTime
	if age < 0 || EchoRangeYd(age) <= ActiveDisplayMaxRangeYd {
		return false
	}
	sonar.LastPingTime = 0
	sonar.activeEchoAt = 0
	sonar.activeEchoDone = nil
	if listener != nil {
		listener.ActiveSonar = false
		listener.LastPingTime = 0
		listener.LastPingPower = 0
	}
	return true
}

// ProcessActiveEchoes applies active detections once the two-way echo can have returned.
func ProcessActiveEchoes(model Model, listener *world.Entity, emitters []*world.Entity, sonar *SonarState, gameTime float64) {
	if sonar == nil || listener == nil || sonar.LastPingTime <= 0 {
		return
	}
	if ExpireActivePingIfBeyondDisplay(sonar, listener, gameTime) {
		return
	}
	if sonar.activeEchoAt != sonar.LastPingTime || sonar.activeEchoDone == nil {
		sonar.activeEchoAt = sonar.LastPingTime
		if sonar.activeEchoDone == nil {
			sonar.activeEchoDone = make(map[string]bool)
		} else {
			clear(sonar.activeEchoDone)
		}
	}
	age := gameTime - sonar.LastPingTime
	if age < 0 {
		return
	}
	echoReach := EchoRangeYd(age)

	for _, em := range emitters {
		if em == nil || em.ID == listener.ID || !em.Alive() {
			continue
		}
		if sonar.activeEchoDone[em.ID] {
			continue
		}
		rangeYd := listener.RangeYardsTo(em)
		if rangeYd > echoReach {
			continue
		}

		result := model.Detect(listener, em, ModeActive, sonar.ActivePower)
		if result.Detected {
			sonar.activeEchoDone[em.ID] = true

			class := Classify(result.SignalForClassify, result.PeakSNR, result.TrueRangeYd)
			class = refineClassification(class, result.SignalForClassify, em.SignatureID, result.PeakSNR)
			class.Confidence = math.Min(0.99, class.Confidence+0.12)

			found := false
			for i := range sonar.Contacts {
				if sonar.Contacts[i].SourceEntityID == em.ID {
					c := &sonar.Contacts[i]
					measRange := result.TrueRangeYd + rand.NormFloat64()*result.TrueRangeYd*0.02
					updateContactTrack(c, result.BearingDeg, measRange, result.PeakSNR, result.BandsAbove, class, gameTime, em)
					c.DetectedBy = "active"
					c.Confidence = math.Max(c.Confidence, class.Confidence)
					c.UncBearingDeg = math.Min(c.UncBearingDeg, 2.5)
					c.UncRangeYd = math.Min(c.UncRangeYd, math.Max(80, result.TrueRangeYd*0.04))
					c.LastActiveBearingDeg = result.BearingDeg
					c.LastActiveRangeYd = measRange
					c.LastActiveFixAt = gameTime
					found = true
					break
				}
			}
			if !found {
				sonar.contactSeq++
				measRange := result.TrueRangeYd
				c := Contact{
					ID:                   formatContactID(sonar.contactSeq),
					BearingDeg:           result.BearingDeg,
					EstimatedRangeYd:     measRange,
					SNR:                  result.PeakSNR,
					BandsAbove:           result.BandsAbove,
					BestMatchID:          class.ProfileID,
					BestMatchName:        class.ProfileName,
					Confidence:           class.Confidence,
					SourceEntityID:       em.ID,
					Kind:                 em.Kind,
					DetectedBy:           "active",
					LastUpdate:           gameTime,
					FirstSeen:            gameTime,
					UncBearingDeg:        2.5,
					UncRangeYd:           math.Max(80, result.TrueRangeYd*0.04),
					LastActiveBearingDeg: result.BearingDeg,
					LastActiveRangeYd:    measRange,
					LastActiveFixAt:      gameTime,
				}
				sonar.Contacts = append(sonar.Contacts, c)
			}
			continue
		}
		// Echo has passed this range but no detection — stop retrying after the return window.
		if age > TwoWayTravelSec(rangeYd)+0.25 {
			sonar.activeEchoDone[em.ID] = true
		}
	}
}

// ContactHasActiveRange reports whether the contact carries a solid active range fix.
func ContactHasActiveRange(c *Contact) bool {
	if c == nil || c.EstimatedRangeYd <= 0 {
		return false
	}
	return c.DetectedBy == "active" || c.DetectedBy == "passive/ping"
}

// SpectrumAtBearing returns SNR spectrum for the analyzer display at a bearing.
func SpectrumAtBearing(model Model, listener *world.Entity, emitters []*world.Entity, sonar *SonarState, bearingDeg, gameTime float64) []float64 {
	return SpectrumAtBearingInto(nil, model, listener, emitters, sonar, bearingDeg, gameTime)
}

// SpectrumBeamSigmaDeg is the soft analyzer beam width (1σ). Contacts within
// ~2σ contribute strongly; harmonics from neighbors mix into the LOFAR trace.
func SpectrumBeamSigmaDeg(sonar *SonarState) float64 {
	base := 6.5 // hull spherical — modest bearing discrimination for LOFAR
	if sonar != nil && sonar.PassiveArray == PassiveArrayTowed && sonar.TowedCablePct > 40 {
		base = 4.2 // streamed TA resolves bearings better
	}
	if sonar != nil {
		base *= math.Max(0.45, sonar.passiveBearingSigmaScale())
	}
	return base
}

// SpectrumBeamWeight returns 0..1 contribution of an emitter at true bearing
// offset deltaDeg from the analyzer look bearing.
func SpectrumBeamWeight(deltaDeg, sigmaDeg float64) float64 {
	if sigmaDeg < 0.5 {
		sigmaDeg = 0.5
	}
	d := math.Abs(normalizeBearingDiff(deltaDeg))
	if d > sigmaDeg*3.5 {
		return 0
	}
	return math.Exp(-(d * d) / (2 * sigmaDeg * sigmaDeg))
}

// SpectrumAtBearingInto fills dst with the analyzer spectrum, reusing dst when len >= NumBands.
// Nearby emitters are power-summed through a soft bearing beam so close contacts
// interleave their tonals and complicate fingerprint matching.
func SpectrumAtBearingInto(dst []float64, model Model, listener *world.Entity, emitters []*world.Entity, sonar *SonarState, bearingDeg, gameTime float64) []float64 {
	if len(dst) < NumBands {
		dst = make([]float64, NumBands)
	} else {
		dst = dst[:NumBands]
		for i := range dst {
			dst[i] = 0
		}
	}
	out := dst
	selfNoise := SelfNoiseSpectrum(listener, model.Env, sonar.PassiveArray, sonar.TowedCablePct)
	sigma := SpectrumBeamSigmaDeg(sonar)
	var accum Spectrum
	for i := range accum {
		accum[i] = -200
	}
	nStrong := 0

	for _, em := range emitters {
		if em == nil || !em.Alive() || em.ID == listener.ID {
			continue
		}
		b := listener.BearingDegTo(em)
		w := SpectrumBeamWeight(b-bearingDeg, sigma)
		if w < 0.02 {
			continue
		}
		src := SourceSpectrum(em)
		recv := Propagate(model.Env, src, em, listener)
		snr := recv.SubNoise(selfNoise)
		bonus := sonar.passiveSNRBonusDB()
		rel := AngleDiffDeg(b, listener.HeadingDeg)
		penalty := PassiveSelfNoiseDeltaDB(sonar.PassiveArray, rel, listener.SpeedKts, listener.DepthFt, sonar.TowedCablePct)
		sens := PassiveArraySensitivity(sonar.PassiveArray, rel, sonar.TowedCablePct)
		if sens < 0.001 {
			sens = 0.001
		}
		sensDB := 20 * math.Log10(sens)
		wDB := 10 * math.Log10(w)
		for i := 0; i < NumBands; i++ {
			v := snr[i] + bonus - penalty + sensDB - listenBandAttenuationDB(sonar.ListenBand, BandCenterHz(i)) + wDB
			accum[i] = combineDB(accum[i], v)
		}
		if w >= 0.22 {
			nStrong++
		}
	}

	for i := 0; i < NumBands; i++ {
		v := accum[i]
		if v < -150 {
			v = 0
		}
		out[i] = math.Max(v, 0) + 1.5 + rand.Float64()
	}
	BioSpectrumFloor(out, sonar, bearingDeg, gameTime)
	DegradeSpectrumBinsForClarity(out)
	if nStrong >= 2 {
		// Multiple contacts in-beam: extra smear so neither fingerprint stays clean.
		degradeSpectrumForBearingMix(out, nStrong)
	}
	return dst
}

// ContaminateClassifySignal power-adds nearby emitters into a classify spectrum
// so auto-matching degrades when contacts share a bearing beam.
func ContaminateClassifySignal(signal Spectrum, model Model, listener *world.Entity, primaryID string, emitters []*world.Entity, bearingDeg float64, sonar *SonarState) Spectrum {
	sigma := SpectrumBeamSigmaDeg(sonar)
	ambient := model.Env.AmbientSpectrum(listener.DepthFt)
	out := signal
	for _, em := range emitters {
		if em == nil || em.ID == listener.ID || em.ID == primaryID {
			continue
		}
		if !em.Alive() && em.Status != world.StatusSinking {
			continue
		}
		b := listener.BearingDegTo(em)
		w := SpectrumBeamWeight(b-bearingDeg, sigma)
		if w < 0.15 {
			continue
		}
		src := SourceSpectrum(em)
		recv := Propagate(model.Env, src, em, listener)
		other := recv.SubNoise(ambient)
		wDB := 10 * math.Log10(w)
		for i := range other {
			other[i] += wDB
			other[i] -= listenBandAttenuationDB(sonar.ListenBand, BandCenterHz(i)) * 0.7
		}
		out = out.AddPower(other)
	}
	return out
}

func degradeSpectrumForBearingMix(bins []float64, nStrong int) {
	if len(bins) < 3 || nStrong < 2 {
		return
	}
	mud := 0.35 + 0.12*float64(nStrong-2)
	if mud > 0.7 {
		mud = 0.7
	}
	tmp := make([]float64, len(bins))
	copy(tmp, bins)
	for i := range bins {
		left, right := tmp[i], tmp[i]
		if i > 0 {
			left = tmp[i-1]
		}
		if i < len(tmp)-1 {
			right = tmp[i+1]
		}
		// Neighbor bleed + raised floor — overlapping tonals wash together.
		smear := tmp[i]*(1-0.65*mud) + (left+right)*0.5*(0.65*mud)
		floor := 3.0 + mud*5.0
		hash := (float64((i*53+7)%19)/19.0 - 0.5) * mud * 5.0
		bins[i] = smear + floor*mud*0.4 + hash
	}
}

// CountSpectrumMixContacts returns how many tracked contacts sit inside the
// analyzer beam around bearingDeg (for UI "MIXED" cues).
func CountSpectrumMixContacts(sonar *SonarState, bearingDeg float64) int {
	if sonar == nil {
		return 0
	}
	sigma := SpectrumBeamSigmaDeg(sonar)
	n := 0
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if SpectrumBeamWeight(c.BearingDeg-bearingDeg, sigma) >= 0.22 {
			n++
		}
	}
	return n
}

func bearingWithError(trueBearing, peakSNR float64, bands int, sigmaScale float64) float64 {
	sigma := 2.8 - float64(bands)*0.12
	if peakSNR > 14 {
		sigma -= 0.9
	}
	if peakSNR > 20 {
		sigma -= 0.6
	}
	sigma *= sigmaScale
	if sigma < 0.35 {
		sigma = 0.35
	}
	return trueBearing + rand.NormFloat64()*sigma
}

func estimatePassiveRange(env Environment, listener, emitter *world.Entity, trueRange, peakSNR float64) float64 {
	uncertainty := 0.14 + (PeakDetectSNR-peakSNR)*0.015
	if uncertainty < 0.09 {
		uncertainty = 0.09
	}
	if env.layerIndex(listener.DepthFt) != env.layerIndex(emitter.DepthFt) {
		uncertainty += 0.16
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
	if age := EnemyActivePingAgeSec(em, gameTime); age >= 0 && age < 2.5 {
		c.DetectedBy = "passive/ping"
	}

	// Smooth track so the plot centre does not jump each noisy sample.
	// Uncertain range fixes (<~90% accuracy) stay heavily averaged.
	alpha := 0.035 + math.Min(0.16, snr/70)
	rangeAcc := 1.0
	if c.EstimatedRangeYd > 1 && c.UncRangeYd > 0 {
		rangeAcc = c.UncRangeYd / c.EstimatedRangeYd
	}
	switch {
	case rangeAcc > 0.18 || c.UncRangeYd <= 0:
		alpha *= 0.28
	case rangeAcc > 0.10:
		alpha *= 0.42
	case rangeAcc > 0.05:
		alpha *= 0.65
	}
	if alpha < 0.012 {
		alpha = 0.012
	}
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

func refineClassification(class Classification, signal Spectrum, trueSignatureID string, peakSNR float64) Classification {
	profile, ok := world.ProfileByID(trueSignatureID)
	if !ok {
		return class
	}
	dist := spectralDistance(signal, templateSpectrum(profile))
	if dist < 30 {
		class.ProfileID = profile.ID
		class.ProfileName = profile.Name
		// Only boost toward a firm ID when the spectrum is actually readable.
		boost := 0.55 * (0.25 + 0.75*SpectrumClarity01(peakSNR))
		class.Confidence = math.Max(class.Confidence, boost)
	}
	return class
}
