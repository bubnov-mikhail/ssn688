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
	if !listener.Alive() || !target.Alive() || listener.ID == target.ID {
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
		if math.Abs(target.RelativeBearingDeg(listener)) > 70 {
			return DetectionResult{TrueRangeYd: rangeYd, BearingDeg: bearing}
		}
		received = PropagateActive(m.Env, listener, target, PingSourceLevel(activePower))
	default:
		return DetectionResult{}
	}

	selfNoise := SelfNoiseSpectrum(listener, m.Env, PassiveArrayHull, 0)
	snr := received.SubNoise(selfNoise)

	peak := snr.Peak()
	bands := snr.BandsAbove(DetectThreshold)
	detected := bands >= MinDetectBands || peak >= PeakDetectSNR

	ambient := m.Env.AmbientSpectrum(listener.DepthFt)
	signalClassify := received.SubNoise(ambient)

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
