package audio

import (
	"encoding/binary"
	"math"
	"testing"
)

func synthToneLoop(sampleRate int, freqHz float64, cycles float64) []byte {
	samples := int(float64(sampleRate) * cycles / freqHz)
	if samples < sampleRate/4 {
		samples = sampleRate / 4
	}
	out := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		s := math.Sin(2 * math.Pi * freqHz * t)
		binary.LittleEndian.PutUint16(out[i*2:], uint16(int16(s*30000)))
	}
	return out
}

func estimateFundamentalHz(samples []float64, sampleRate int) float64 {
	if len(samples) < sampleRate/8 {
		return 0
	}
	// Autocorrelation peak in 40..2000 Hz.
	bestLag := 0
	best := 0.0
	minLag := sampleRate / 2000
	maxLag := sampleRate / 40
	if maxLag >= len(samples)/2 {
		maxLag = len(samples)/2 - 1
	}
	for lag := minLag; lag <= maxLag; lag++ {
		var sum float64
		for i := 0; i < len(samples)-lag; i++ {
			sum += samples[i] * samples[i+lag]
		}
		if sum > best {
			best = sum
			bestLag = lag
		}
	}
	if bestLag < 1 {
		return 0
	}
	return float64(sampleRate) / float64(bestLag)
}

func renderLoopSamples(tr *loopTrack, n, sampleRate int) []float64 {
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = tr.nextSample()
	}
	_ = sampleRate
	return out
}

func TestLoopStretchSpeedChangesSourceRate(t *testing.T) {
	pcm := synthToneLoop(44100, 120, 8)
	slow := newLoopTrack(pcm, 1, 0.85)
	fast := newLoopTrack(pcm, 1, 1.15)
	const n = 44100 * 2
	dSlow := trackSourceAdvance(slow, n)
	dFast := trackSourceAdvance(fast, n)
	if dFast < dSlow*1.15 {
		t.Fatalf("fast source advance %.0f should exceed slow %.0f", dFast, dSlow)
	}
}

func trackSourceAdvance(tr *loopTrack, n int) float64 {
	var total float64
	pcmLen := float64(len(tr.pcm) / 2)
	prev := tr.srcIdx
	for i := 0; i < n; i++ {
		tr.nextSample()
		if tr.srcIdx < prev {
			total += pcmLen - prev + tr.srcIdx
		} else {
			total += tr.srcIdx - prev
		}
		prev = tr.srcIdx
	}
	return total
}

func TestLoopStretchPitchStable(t *testing.T) {
	const (
		sr   = 44100
		freq = 180.0
		n    = sr * 2
	)
	pcm := synthToneLoop(sr, freq, 6)
	nom := newLoopTrack(pcm, 1, 1.0)
	slow := newLoopTrack(pcm, 1, 0.9)
	fast := newLoopTrack(pcm, 1, 1.1)
	fNom := estimateFundamentalHz(renderLoopSamples(nom, n, sr), sr)
	fSlow := estimateFundamentalHz(renderLoopSamples(slow, n, sr), sr)
	fFast := estimateFundamentalHz(renderLoopSamples(fast, n, sr), sr)
	if fNom < 80 {
		t.Fatalf("could not estimate nominal pitch: %.1f Hz", fNom)
	}
	for _, tc := range []struct {
		name string
		f    float64
	}{
		{"slow", fSlow},
		{"fast", fFast},
	} {
		ratio := tc.f / fNom
		if ratio < 0.85 || ratio > 1.15 {
			t.Fatalf("%s pitch shifted: nominal=%.1f Hz got=%.1f Hz ratio=%.2f", tc.name, fNom, tc.f, ratio)
		}
	}
}

func TestPropellerListenSpeed(t *testing.T) {
	if s := PropellerListenSpeed(13, "merchant"); s < 0.95 || s > 1.05 {
		t.Fatalf("merchant at ref speed want ~1.0, got %.2f", s)
	}
	if PropellerListenSpeed(6, "fishing") >= PropellerListenSpeed(12, "fishing") {
		t.Fatal("faster target should have higher playback rate")
	}
	if s := PropellerListenSpeed(1.5, "merchant"); s != 1 {
		t.Fatalf("below min speed should return 1 (unused), got %.2f", s)
	}
	// ref=20 → 1.2 at 30 kts (1 + 0.4*(1.5-1))
	if s := PropellerListenSpeed(30, "grisha"); s != loopSpeedMax {
		t.Fatalf("high speed should clamp to %.2f, got %.2f", loopSpeedMax, s)
	}
	if s := PropellerListenSpeed(2, "merchant"); s < loopSpeedMin || s > loopSpeedMax {
		t.Fatalf("2 kts merchant rate %.2f outside [%.2f, %.2f]", s, loopSpeedMin, loopSpeedMax)
	}
}

func TestHelmPropellerListenSpeed(t *testing.T) {
	if s := HelmPropellerListenSpeed(0.05); s != 1 {
		t.Fatalf("below helm min want unused 1, got %.2f", s)
	}
	at01 := HelmPropellerListenSpeed(0.1)
	at12 := HelmPropellerListenSpeed(12)
	at32 := HelmPropellerListenSpeed(32)
	at40 := HelmPropellerListenSpeed(40)
	if at01 < loopSpeedMin-0.001 || at01 > loopSpeedMin+0.05 {
		t.Fatalf("0.1 kt should sit near min rate, got %.2f", at01)
	}
	if at32 < 0.99 || at32 > 1.01 {
		t.Fatalf("32 kt should be nominal 1.0, got %.2f", at32)
	}
	if at40 > at32+0.001 {
		t.Fatalf("above 32 kt must not exceed peak (got %.2f > %.2f)", at40, at32)
	}
	if at12 >= at32 {
		t.Fatalf("12 kt (%.2f) should be slower than 32 kt (%.2f)", at12, at32)
	}
}

func TestHelmPropellerGain(t *testing.T) {
	if g := HelmPropellerGain(0); g != 0 {
		t.Fatalf("0 kt want 0, got %.3f", g)
	}
	if g := HelmPropellerGain(0.05); g != 0 {
		t.Fatalf("below min want 0, got %.3f", g)
	}
	g1 := HelmPropellerGain(0.1)
	if g1 > 0.01 {
		t.Fatalf("at min speed gain should be ~0, got %.3f", g1)
	}
	mid := HelmPropellerGain(4.05) // midpoint of 0.1..8
	if mid < 0.20 || mid > 0.30 {
		t.Fatalf("mid ramp want ~0.25, got %.3f", mid)
	}
	if g := HelmPropellerGain(8); math.Abs(g-HelmPropellerMaxGain) > 1e-9 {
		t.Fatalf("8 kt want max %.2f, got %.3f", HelmPropellerMaxGain, g)
	}
	if g := HelmPropellerGain(20); math.Abs(g-HelmPropellerMaxGain) > 1e-9 {
		t.Fatalf("above full-gain speed want max %.2f, got %.3f", HelmPropellerMaxGain, g)
	}
}
