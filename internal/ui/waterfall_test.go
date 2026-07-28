package ui

import (
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
)

func TestBearingWaterfallNewestFirst(t *testing.T) {
	var w BearingWaterfall
	a := make([]float64, acoustics.BearingWaterfallBins)
	b := make([]float64, acoustics.BearingWaterfallBins)
	a[0] = 1
	b[0] = 2
	w.PushCopy(a, 10)
	w.PushCopy(b, 20)
	if w.Len() != 2 {
		t.Fatalf("len=%d", w.Len())
	}
	if w.Row(0).Bearings[0] != 2 || w.Row(0).Heading != 20 {
		t.Fatal("newest should be b")
	}
	if w.Row(1).Bearings[0] != 1 || w.Row(1).Heading != 10 {
		t.Fatal("older should be a")
	}
}

func TestSnrToIntensity(t *testing.T) {
	if snrToIntensity(0) != 0 {
		t.Fatal("low SNR should be zero intensity")
	}
	if snrToIntensity(30) <= snrToIntensity(10) {
		t.Fatal("higher SNR should yield higher intensity")
	}
}
