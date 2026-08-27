package audio

import (
	"encoding/binary"
	"math"
)

const (
	loopGrainSamples = 2048
	loopSynthHop     = 512
	loopSpeedMin     = 0.8  // ±20% from nominal recording speed
	loopSpeedMax     = 1.2
)

// PropellerMinSpeedKts — below this, propeller/bow-wash loops are silent (PASSIVE contacts).
const PropellerMinSpeedKts = 2.0

// HelmPropellerMinSpeedKts — ownship HELM propulsion loop starts here (gain near 0).
const HelmPropellerMinSpeedKts = 0.1

// HelmPropellerFullGainSpeedKts — HELM loop reaches max gain (0.5) at this speed.
const HelmPropellerFullGainSpeedKts = 8.0

// HelmPropellerRefSpeedKts — playback rate 1.0 (former LA ~12 kt "nominal") at this speed.
const HelmPropellerRefSpeedKts = 32.0

// HelmPropellerMaxGain is peak loudness for the HELM ownship propulsion loop.
const HelmPropellerMaxGain = 0.5

// loopTrack is one ambient FX loop mixed under voices (propeller, bow wash, …).
type loopTrack struct {
	pcm   []byte
	gain  float64
	speed float64 // pitch-neutral playback rate; 1 = nominal

	olaRing    []float64
	weightRing []float64
	outPhase   int
	synRemain  int
	srcIdx     float64
}

func newLoopTrack(pcm []byte, gain, speed float64) *loopTrack {
	tr := &loopTrack{
		pcm:        pcm,
		gain:       gain,
		speed:      clampLoopSpeed(speed),
		olaRing:    make([]float64, loopGrainSamples),
		weightRing: make([]float64, loopGrainSamples),
		synRemain:  0,
	}
	tr.addGrain()
	return tr
}

func clampLoopSpeed(s float64) float64 {
	if s <= 0 || math.IsNaN(s) || math.IsInf(s, 0) {
		return 1
	}
	if s < loopSpeedMin {
		return loopSpeedMin
	}
	if s > loopSpeedMax {
		return loopSpeedMax
	}
	return s
}

func pcmSampleAt(pcm []byte, idx float64) float64 {
	n := len(pcm) / 2
	if n < 1 {
		return 0
	}
	for idx < 0 {
		idx += float64(n)
	}
	idx = math.Mod(idx, float64(n))
	i0 := int(idx)
	i1 := (i0 + 1) % n
	frac := idx - float64(i0)
	s0 := float64(int16(binary.LittleEndian.Uint16(pcm[i0*2:])))
	s1 := float64(int16(binary.LittleEndian.Uint16(pcm[i1*2:])))
	return (s0*(1-frac) + s1*frac) / 32768
}

func hannWindow(i, n int) float64 {
	if n <= 1 {
		return 1
	}
	return 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
}

func (tr *loopTrack) analysisHop() int {
	ha := int(float64(loopSynthHop)*tr.speed + 0.5)
	minHop := 64
	maxHop := loopGrainSamples - loopSynthHop
	if ha < minHop {
		return minHop
	}
	if ha > maxHop {
		return maxHop
	}
	return ha
}

func (tr *loopTrack) addGrain() {
	start := tr.outPhase
	for i := 0; i < loopGrainSamples; i++ {
		pos := (start + i) % loopGrainSamples
		w := hannWindow(i, loopGrainSamples)
		s := pcmSampleAt(tr.pcm, tr.srcIdx+float64(i)) * w
		tr.olaRing[pos] += s
		tr.weightRing[pos] += w
	}
	tr.srcIdx += float64(tr.analysisHop())
	n := len(tr.pcm) / 2
	if n > 0 {
		tr.srcIdx = math.Mod(tr.srcIdx, float64(n))
	}
}

func (tr *loopTrack) nextSample() float64 {
	if tr == nil || tr.gain <= 0 || len(tr.pcm) < 4 {
		return 0
	}
	idx := tr.outPhase
	sample := 0.0
	if tr.weightRing[idx] > 1e-8 {
		sample = tr.olaRing[idx] / tr.weightRing[idx]
	}
	tr.olaRing[idx] = 0
	tr.weightRing[idx] = 0
	tr.outPhase = (tr.outPhase + 1) % loopGrainSamples

	tr.synRemain--
	if tr.synRemain <= 0 {
		tr.synRemain = loopSynthHop
		tr.addGrain()
	}
	return sample
}

// PropellerListenSpeed maps observed speed (kts) to pitch-neutral loop playback rate.
// At the class reference speed the rate is 1.0; clamped to ±20%.
func PropellerListenSpeed(speedKts float64, signatureID string) float64 {
	if speedKts < PropellerMinSpeedKts {
		return 1
	}
	ref := propellerRefSpeedKts(signatureID)
	rate := 1 + 0.4*(speedKts/ref-1)
	return clampLoopSpeed(rate)
}

// HelmPropellerListenSpeed maps ownship speed for the HELM propulsion loop.
// Rate 1.0 (the old ~12 kt LA nominal) is the ceiling, reached at HelmPropellerRefSpeedKts.
func HelmPropellerListenSpeed(speedKts float64) float64 {
	if speedKts < HelmPropellerMinSpeedKts {
		return 1
	}
	spd := speedKts
	if spd > HelmPropellerRefSpeedKts {
		spd = HelmPropellerRefSpeedKts
	}
	rate := 1 + 0.4*(spd/HelmPropellerRefSpeedKts-1)
	if rate > 1 {
		rate = 1
	}
	return clampLoopSpeed(rate)
}

// HelmPropellerGain ramps loudness from 0 at HelmPropellerMinSpeedKts to
// HelmPropellerMaxGain at HelmPropellerFullGainSpeedKts (and above).
func HelmPropellerGain(speedKts float64) float64 {
	if speedKts < HelmPropellerMinSpeedKts {
		return 0
	}
	span := HelmPropellerFullGainSpeedKts - HelmPropellerMinSpeedKts
	if span <= 0 {
		return HelmPropellerMaxGain
	}
	t := (speedKts - HelmPropellerMinSpeedKts) / span
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return HelmPropellerMaxGain * t
}

func propellerRefSpeedKts(signatureID string) float64 {
	switch signatureID {
	case "fishing":
		return 9
	case "merchant":
		return 13
	case "tanker":
		return 11
	case "kilo", "victor", "turbinia", "whiskey":
		return 11
	case "grisha", "krivak", "udaloy", "kresta2", "gorshkov", "spruance":
		return 20
	case "yasen_m", "victor_iii", "los_angeles":
		return 12
	default:
		return 15
	}
}

// TorpedoListenSpeed maps torpedo speed to loop rate (±20%, 1.0 at ~32 kts).
func TorpedoListenSpeed(speedKts float64) float64 {
	if speedKts < 1 {
		return 1
	}
	const ref = 32.0
	rate := 1 + 0.4*(speedKts/ref-1)
	return clampLoopSpeed(rate)
}
