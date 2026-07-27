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

func combineDB(a, b float64) float64 {
	if a < -150 {
		return b
	}
	if b < -150 {
		return a
	}
	pa := math.Pow(10, a/10)
	pb := math.Pow(10, b/10)
	return 10 * math.Log10(pa+pb)
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
