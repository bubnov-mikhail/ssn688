package ui

import (
	"image/color"
	"math"
)

// sonarHeatColor maps 0..1 intensity to a marine sonar heat palette
// (dark blue → cyan → green → yellow → red), matching commercial PPI/waterfall displays.
func sonarHeatColor(intensity float64) color.RGBA {
	if intensity <= 0.02 {
		return color.RGBA{0, 2, 18, 255}
	}
	if intensity > 1 {
		intensity = 1
	}
	stops := []struct {
		t    float64
		r, g, b float64
	}{
		{0.00, 0, 4, 36},
		{0.18, 0, 30, 110},
		{0.35, 0, 110, 170},
		{0.50, 0, 190, 140},
		{0.65, 40, 220, 60},
		{0.80, 220, 220, 30},
		{0.92, 255, 120, 20},
		{1.00, 255, 40, 30},
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
	const lo, hi = 1.0, 22.0
	t := (snr - lo) / (hi - lo)
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return math.Pow(t, 0.72)
}
