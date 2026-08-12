package acoustics

import (
	"math"
	"testing"
)

func TestCombineDBMatchesReference(t *testing.T) {
	ref := func(a, b float64) float64 {
		if a < -150 {
			return b
		}
		if b < -150 {
			return a
		}
		return 10 * math.Log10(math.Pow(10, a/10)+math.Pow(10, b/10))
	}
	cases := [][2]float64{
		{60, 60}, {70, 40}, {55, 54.5}, {80, -200}, {-200, 50}, {90, 65}, {45, 44},
	}
	for _, c := range cases {
		got := combineDB(c[0], c[1])
		want := ref(c[0], c[1])
		if math.Abs(got-want) > 0.15 {
			t.Fatalf("combineDB(%v,%v)=%v want~%v", c[0], c[1], got, want)
		}
	}
}

func TestShouldReclassifyThrottle(t *testing.T) {
	c := &Contact{BestMatchID: "kilo", SNR: 12, LastClassifyAt: 10}
	if shouldReclassify(c, 12.5, 10.2) {
		t.Fatal("expected throttle within interval and small SNR delta")
	}
	if !shouldReclassify(c, 12.5, 10.6) {
		t.Fatal("expected reclassify after interval")
	}
	if !shouldReclassify(c, 15, 10.1) {
		t.Fatal("expected reclassify on SNR jump")
	}
	if !shouldReclassify(&Contact{}, 10, 1) {
		t.Fatal("empty match must classify")
	}
}
