// Package fastmath provides cheap approximations for hot game loops.
package fastmath

import "math"

const sinLUTBits = 10
const sinLUTSize = 1 << sinLUTBits // 1024
const sinLUTMask = sinLUTSize - 1

// Two-pi mapped onto the LUT period.
const sinScale = float64(sinLUTSize) / (2 * math.Pi)

var sinLUT [sinLUTSize]float64

func init() {
	for i := 0; i < sinLUTSize; i++ {
		sinLUT[i] = math.Sin(2 * math.Pi * float64(i) / float64(sinLUTSize))
	}
}

// Sin returns a LUT-approximated sine (period 2π). Good enough for display noise.
func Sin(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}
	// Floor via int truncation after positive wrap — avoid Mod for speed.
	idx := x * sinScale
	i := int(idx)
	if idx < 0 {
		i = int(idx) - 1
	}
	return sinLUT[i&sinLUTMask]
}

// Hash01 maps integers to [0,1) with a cheap integer mix (no trig).
func Hash01(a, b, c int) float64 {
	n := a*374761393 + b*668265263 + c*1274126177
	n = (n ^ (n >> 13)) * 1274126177
	return float64((n^(n>>16))&255) / 256.0
}
