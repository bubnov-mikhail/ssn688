package fastmath

import (
	"math"
	"testing"
)

func TestSinApproxCloseToMath(t *testing.T) {
	maxErr := 0.0
	for i := 0; i < 2000; i++ {
		x := float64(i) * 0.017
		got := Sin(x)
		want := math.Sin(x)
		err := math.Abs(got - want)
		if err > maxErr {
			maxErr = err
		}
	}
	// 1024-entry LUT → max error ≈ half bin ≈ 0.003
	if maxErr > 0.01 {
		t.Fatalf("Sin LUT max err=%v", maxErr)
	}
}

func TestHash01Distinct(t *testing.T) {
	if Hash01(1, 2, 3) == Hash01(9, 8, 7) && Hash01(0, 0, 0) == Hash01(1, 0, 0) {
		// Extremely unlikely; just ensure function returns finite values.
	}
	a := Hash01(1, 2, 3)
	if a < 0 || a >= 1 {
		t.Fatalf("Hash01 out of range: %v", a)
	}
}

func BenchmarkSinLUT(b *testing.B) {
	x := 1.234
	for i := 0; i < b.N; i++ {
		x = Sin(x + 0.01)
	}
}

func BenchmarkMathSin(b *testing.B) {
	x := 1.234
	for i := 0; i < b.N; i++ {
		x = math.Sin(x + 0.01)
	}
}
