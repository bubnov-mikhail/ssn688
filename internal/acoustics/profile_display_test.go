package acoustics

import "testing"

func TestObservedPeaksWeakSNRStaysMuddy(t *testing.T) {
	weak := make([]float64, NumBands)
	for i := range weak {
		weak[i] = 3.0
	}
	// Distinct tonal spike that would dominate after max-normalization.
	weak[8] = 8.5
	weak[16] = 7.5

	peaks := ObservedPeaksFromBins(weak)
	if len(peaks) != NumBands {
		t.Fatalf("len=%d", len(peaks))
	}
	maxLvl := 0.0
	for _, p := range peaks {
		if p.Level > maxLvl {
			maxLvl = p.Level
		}
	}
	if maxLvl > 0.62 {
		t.Fatalf("weak spectrum should not stretch to bright peaks, max=%.2f", maxLvl)
	}
	if peaks[8].Level-peaks[0].Level > 0.35 {
		t.Fatalf("weak tonal contrast too high: peak=%.2f floor=%.2f", peaks[8].Level, peaks[0].Level)
	}
}

func TestObservedPeaksStrongSNRShowsTonals(t *testing.T) {
	strong := make([]float64, NumBands)
	for i := range strong {
		strong[i] = 6.0
	}
	strong[10] = 28.0
	peaks := ObservedPeaksFromBins(strong)
	if peaks[10].Level < 0.7 {
		t.Fatalf("strong tonal should read clearly, lvl=%.2f", peaks[10].Level)
	}
}

func TestSpectrumClarity01(t *testing.T) {
	if SpectrumClarity01(5) > 0.05 {
		t.Fatal("below waterfall floor should be ~0 clarity")
	}
	if SpectrumClarity01(28) < 0.9 {
		t.Fatal("strong SNR should be near full clarity")
	}
	if SpectrumClarity01(14) <= SpectrumClarity01(9) {
		t.Fatal("clarity should rise with SNR")
	}
}
