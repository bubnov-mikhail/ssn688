package acoustics

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

// DetectionMode is passive listening or active echo-ranging.
type DetectionMode int

const (
	ModePassive DetectionMode = iota
	ModeActive
)

// DetectionResult holds per-band analysis for one listener–target pair.
type DetectionResult struct {
	Detected           bool
	PeakSNR            float64
	BandsAbove         int
	BearingDeg         float64
	TrueRangeYd        float64
	Received           Spectrum // propagated signal at listener
	SNR                Spectrum
	SignalForClassify  Spectrum // received minus ambient at listener depth
}

// Model performs unified acoustic detection for any platform.
type Model struct {
	Env Environment
}

func NewModel(env Environment) Model {
	return Model{Env: env}
}

// Detect evaluates whether listener detects target using passive or active sonar.
func (m Model) Detect(listener, target *world.Entity, mode DetectionMode, activePower float64) DetectionResult {
	return m.detect(listener, target, mode, activePower, nil, nil)
}

// detect is the shared implementation. Optional selfNoise/ambient avoid recomputing
// listener floors when scanning many emitters in one passive pass.
func (m Model) detect(listener, target *world.Entity, mode DetectionMode, activePower float64, selfNoise, ambient *Spectrum) DetectionResult {
	if !listener.Alive() || target == nil || listener.ID == target.ID {
		return DetectionResult{}
	}
	if !target.Alive() && target.Status != world.StatusSinking {
		return DetectionResult{}
	}

	bearing := listener.BearingDegTo(target)
	rangeYd := listener.RangeYardsTo(target)

	var received Spectrum
	switch mode {
	case ModePassive:
		src := SourceSpectrum(target)
		received = Propagate(m.Env, src, target, listener)
	case ModeActive:
		received = PropagateActive(m.Env, listener, target, PingSourceLevel(activePower))
		// Beam aspect: strong return abeam, weaker bow/stern — never a hard cutoff.
		aspect := math.Abs(AngleDiffDeg(target.BearingDegTo(listener), target.HeadingDeg))
		rcsGain := 0.0
		switch {
		case aspect >= 55 && aspect <= 125:
			rcsGain = 8
		case aspect >= 35 && aspect <= 145:
			rcsGain = 4
		case aspect > 160:
			rcsGain = -6
		}
		for i := range received {
			received[i] += rcsGain
		}
	default:
		return DetectionResult{}
	}

	var snFloor Spectrum
	if selfNoise != nil {
		snFloor = *selfNoise
	} else {
		snFloor = SelfNoiseSpectrum(listener, m.Env, PassiveArrayHull, 0)
	}
	snr := received.SubNoise(snFloor)
	if mode == ModePassive {
		rel := AngleDiffDeg(bearing, listener.HeadingDeg)
		if penalty := PassiveSelfNoisePenaltyDB(PassiveArrayHull, rel, listener.SpeedKts, listener.DepthFt, 0); penalty > 0 {
			for i := range snr {
				snr[i] -= penalty
			}
		}
	}

	peak := snr.Peak()
	bands := snr.BandsAbove(DetectThreshold)
	detected := bands >= MinDetectBands || peak >= PeakDetectSNR

	var amb Spectrum
	if ambient != nil {
		amb = *ambient
	} else {
		amb = m.Env.AmbientSpectrum(listener.DepthFt)
	}
	signalClassify := received.SubNoise(amb)

	return DetectionResult{
		Detected:          detected,
		PeakSNR:           peak,
		BandsAbove:        bands,
		BearingDeg:        bearing,
		TrueRangeYd:       rangeYd,
		Received:          received,
		SNR:               snr,
		SignalForClassify: signalClassify,
	}
}

// PeakPassiveSNR is a convenience for AI and quick checks.
func (m Model) PeakPassiveSNR(listener, target *world.Entity) float64 {
	return m.Detect(listener, target, ModePassive, 0).PeakSNR
}

// PeakActiveSNR evaluates active echo return strength.
func (m Model) PeakActiveSNR(listener, target *world.Entity, power float64) float64 {
	return m.Detect(listener, target, ModeActive, power).PeakSNR
}

// CanDetectPassive returns true when listener can reliably hear target.
func (m Model) CanDetectPassive(listener, target *world.Entity) bool {
	return m.Detect(listener, target, ModePassive, 0).Detected
}

// CanDetectActive returns true when active return exceeds threshold.
func (m Model) CanDetectActive(listener, target *world.Entity, power float64) bool {
	return m.Detect(listener, target, ModeActive, power).Detected
}

// passiveDetectCacheTTLSec — reuse AI passive detect result for this long when
// the player is not illuminating (ping bonus changes detectability rapidly).
const passiveDetectCacheTTLSec = 0.35

// CanDetectPlayerPassive applies a temporary SNR bonus after the player active-pings.
func (m Model) CanDetectPlayerPassive(listener, player *world.Entity, gameTime float64) bool {
	if listener == nil || player == nil {
		return false
	}
	heardAge := PlayerPingHeardAge(m.Env, listener, player, gameTime)
	// Cache only the quiet case — ping bonuses must be live.
	if heardAge < 0 &&
		listener.PassiveDetectCacheAt > 0 &&
		gameTime-listener.PassiveDetectCacheAt < passiveDetectCacheTTLSec {
		return listener.PassiveDetectCached
	}

	r := m.Detect(listener, player, ModePassive, 0)
	detected := r.Detected
	if heardAge >= 0 {
		power := player.LastPingPower
		if power <= 0 {
			power = 0.7
		}
		bonus := PlayerPingPassiveBonusDB(heardAge, power)
		peak := r.PeakSNR + bonus
		bands := r.BandsAbove
		if bonus > 10 {
			bands += 2
		}
		detected = bands >= MinDetectBands || peak >= PeakDetectSNR
	} else {
		listener.PassiveDetectCacheAt = gameTime
		listener.PassiveDetectCached = detected
	}
	return detected
}
