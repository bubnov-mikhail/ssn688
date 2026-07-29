package ui

import (
	"image/color"
	"math"
)

// sonarHeatColor maps 0..1 intensity to a marine sonar heat palette
// (dark blue → cyan → green → yellow → red), matching commercial PPI/waterfall displays.
func sonarHeatColor(intensity float64) color.RGBA {
	return sonarHeatColorFast(intensity)
}

var heatLUT [256]color.RGBA

func init() {
	for i := 0; i < 256; i++ {
		heatLUT[i] = sonarHeatColorRaw(float64(i) / 255)
	}
}

func sonarHeatColorFast(intensity float64) color.RGBA {
	if intensity <= 0 {
		return heatLUT[0]
	}
	if intensity >= 1 {
		return heatLUT[255]
	}
	return heatLUT[int(intensity*255+0.5)]
}

func sonarHeatColorRaw(intensity float64) color.RGBA {
	if intensity <= 0.02 {
		return color.RGBA{0, 2, 18, 255}
	}
	if intensity > 1 {
		intensity = 1
	}
	stops := []struct {
		t       float64
		r, g, b float64
	}{
		{0.00, 0, 4, 32},
		{0.22, 0, 28, 105},
		{0.40, 0, 95, 165},
		{0.58, 0, 175, 155},
		{0.72, 60, 210, 95},
		{0.83, 200, 210, 45},
		{0.92, 245, 175, 35},
		{0.965, 255, 115, 38},
		{0.992, 255, 70, 35},
		{1.00, 255, 70, 35},
	}
	for i := 0; i < len(stops)-1; i++ {
		a, b := stops[i], stops[i+1]
		if intensity <= b.t {
			u := (intensity - a.t) / (b.t - a.t)
			return color.RGBA{
				uint8(a.r + (b.r-a.r)*u),
				uint8(a.g + (b.g-a.g)*u),
				uint8(a.b + (b.b-a.b)*u),
				255,
			}
		}
	}
	last := stops[len(stops)-1]
	return color.RGBA{uint8(last.r), uint8(last.g), uint8(last.b), 255}
}

func snrToIntensity(snr float64) float64 {
	const lo, hi = 5.5, 38.0
	t := (snr - lo) / (hi - lo)
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	// Keep weak-signal (blue) mapping; compress the top so red is rare.
	exp := 1.48 + 0.52*t*t
	v := math.Pow(t, exp)
	if v > 0.80 {
		u := (v - 0.80) / 0.20
		v = 0.80 + 0.17*u // peak SNR tops out ~0.97, not full red
	}
	return v
}

// waterfallSNRToIntensity maps waterfall bin energy to color. Uses a gentler curve
// tuned for self-noise (roughly 4–22 dB) so high speed visibly washes the display.
func waterfallSNRToIntensity(snr float64) float64 {
	const lo, hi = 2.0, 24.0
	t := (snr - lo) / (hi - lo)
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return math.Pow(t, 1.12)
}
