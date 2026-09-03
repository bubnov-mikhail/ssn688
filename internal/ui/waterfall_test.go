package ui

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
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

func TestBearingWaterfallResetThenPush(t *testing.T) {
	var w BearingWaterfall
	row := make([]float64, acoustics.BearingWaterfallBins)
	row[0] = 3
	w.PushCopy(row, 1)
	w.Reset()
	if w.Len() != 0 {
		t.Fatalf("len after reset=%d", w.Len())
	}
	row[0] = 9
	w.PushCopy(row, 45)
	if w.Len() != 1 {
		t.Fatalf("len=%d", w.Len())
	}
	if w.Latest() == nil || w.Latest().Bearings[0] != 9 || w.Latest().Heading != 45 {
		t.Fatalf("latest after reset: %+v", w.Latest())
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
