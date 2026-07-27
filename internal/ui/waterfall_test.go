package ui

import (
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
)

func TestBearingWaterfallNewestFirst(t *testing.T) {
	var w BearingWaterfall
	a := acoustics.BearingWaterfallRow{
		Bearings: make([]float64, acoustics.BearingWaterfallBins),
		Heading:  10,
	}
	a.Bearings[10] = 12
	b := acoustics.BearingWaterfallRow{
		Bearings: make([]float64, acoustics.BearingWaterfallBins),
		Heading:  20,
	}
	b.Bearings[20] = 18
	w.Push(a)
	w.Push(b)
	if len(w.rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(w.rows))
	}
	if w.rows[0].Bearings[20] != 18 {
		t.Fatalf("newest row should be on top")
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
