package acoustics

import "math"

// Spectrum is power level in dB re 1 µPa per frequency band.
type Spectrum [NumBands]float64

func NewSpectrumFlat(levelDB float64) Spectrum {
	var s Spectrum
	for i := range s {
		s[i] = levelDB
	}
	return s
}

func (s Spectrum) Clone() Spectrum {
	return s
}

func (s *Spectrum) Clear() {
	for i := range s {
		s[i] = -200
	}
}

func (s Spectrum) Peak() float64 {
	peak := -200.0
	for _, v := range s {
		if v > peak {
			peak = v
		}
	}
	return peak
}

func (s Spectrum) BandsAbove(threshold float64) int {
	n := 0
	for _, v := range s {
		if v >= threshold {
			n++
		}
	}
	return n
}

// AddPower combines two spectra in the linear power domain.
func (s Spectrum) AddPower(other Spectrum) Spectrum {
	var out Spectrum
	for i := range out {
		out[i] = combineDB(s[i], other[i])
	}
	return out
}

func (s Spectrum) SubNoise(noise Spectrum) Spectrum {
	var out Spectrum
	for i := range out {
		out[i] = s[i] - noise[i]
	}
	return out
}

func (s Spectrum) AddAttenuation(db float64) Spectrum {
	var out Spectrum
	for i := range out {
		out[i] = s[i] - db
	}
	return out
}

func (s Spectrum) AddScalar(db float64) Spectrum {
	var out Spectrum
	for i := range out {
		out[i] = s[i] + db
	}
	return out
}

// combineDeltaLUT[i] ≈ 10*log10(1+10^(-d/10)) where d = i/4 dB (0..25).
var combineDeltaLUT [101]float64

func init() {
	for i := range combineDeltaLUT {
		d := float64(i) / 4.0
		combineDeltaLUT[i] = 10 * math.Log10(1+math.Pow(10, -d/10))
	}
}

// combineDB merges two power levels without per-call Pow/Log10.
func combineDB(a, b float64) float64 {
	if a < -150 {
		return b
	}
	if b < -150 {
		return a
	}
	if a < b {
		a, b = b, a
	}
	d := a - b
	if d >= 25 {
		return a // weaker contributes <0.3%
	}
	idx := int(d*4 + 0.5)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(combineDeltaLUT) {
		idx = len(combineDeltaLUT) - 1
	}
	return a + combineDeltaLUT[idx]
}

// NormalizeShape scales spectrum to 0..1 per-band shape for correlation.
func (s Spectrum) NormalizeShape() [NumBands]float64 {
	var out [NumBands]float64
	min, max := 1e9, -1e9
	for _, v := range s {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min
	if span < 1 {
		span = 1
	}
	for i, v := range s {
		out[i] = (v - min) / span
	}
	return out
}

func correlateShape(a, b [NumBands]float64) float64 {
	var sumA, sumB, sumAB float64
	for i := 0; i < NumBands; i++ {
		sumA += a[i] * a[i]
		sumB += b[i] * b[i]
		sumAB += a[i] * b[i]
	}
	if sumA < 1e-6 || sumB < 1e-6 {
		return 0
	}
	return sumAB / (math.Sqrt(sumA) * math.Sqrt(sumB))
}
