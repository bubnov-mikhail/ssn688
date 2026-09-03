package acoustics

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestAnalyzeClassifyFilterIndistinct(t *testing.T) {
	bins := make([]float64, NumBands)
	for i := range bins {
		bins[i] = 3.5 // below clarity floor
	}
	if got := AnalyzeClassifyFilter(bins, 1); got != ClassifyIndistinct {
		t.Fatalf("got %v want indistinct", got)
	}
}

func TestAnalyzeClassifyFilterMixIsFull(t *testing.T) {
	bins := make([]float64, NumBands)
	for i := range bins {
		f := BandCenterHz(i)
		bins[i] = 8
		if f < 200 {
			bins[i] = 22
		}
		if f > 1000 {
			bins[i] = 20
		}
	}
	// Make a few clear peaks.
	bins[4] = 28
	bins[8] = 26
	bins[24] = 27
	if got := AnalyzeClassifyFilter(bins, 2); got != ClassifyFull {
		t.Fatalf("mixed bearing should be full, got %v", got)
	}
}

func TestAnalyzeClassifyFilterHFTorpedo(t *testing.T) {
	bins := make([]float64, NumBands)
	for i := range bins {
		f := BandCenterHz(i)
		bins[i] = 6
		if f >= 800 {
			bins[i] = 18 + float64(i%3)
		}
	}
	bins[22] = 26
	bins[26] = 25
	bins[28] = 24
	if got := AnalyzeClassifyFilter(bins, 1); got != ClassifyTorpedoOnly {
		t.Fatalf("HF-dominant got %v want torpedo", got)
	}
}

func TestAnalyzeClassifyFilterLFPlatform(t *testing.T) {
	bins := make([]float64, NumBands)
	for i := range bins {
		f := BandCenterHz(i)
		bins[i] = 5
		if f <= 400 {
			bins[i] = 18 + float64(i%3)
		}
	}
	bins[2] = 26
	bins[5] = 25
	bins[8] = 24
	if got := AnalyzeClassifyFilter(bins, 1); got != ClassifyPlatformOnly {
		t.Fatalf("LF-dominant got %v want platform", got)
	}
}

func TestClassificationLibraryIndices(t *testing.T) {
	torp := ClassificationLibraryIndices(ClassifyTorpedoOnly)
	if len(torp) == 0 {
		t.Fatal("expected torpedo entries")
	}
	for _, i := range torp {
		if world.SignatureLibrary[i].Kind != world.KindTorpedo {
			t.Fatalf("non-torpedo in filter: %s", world.SignatureLibrary[i].ID)
		}
	}
	plat := ClassificationLibraryIndices(ClassifyPlatformOnly)
	if len(plat) == 0 {
		t.Fatal("expected platform entries")
	}
	for _, i := range plat {
		k := world.SignatureLibrary[i].Kind
		if k != world.KindSubmarine && k != world.KindSurfaceShip {
			t.Fatalf("non-platform in filter: %s", world.SignatureLibrary[i].ID)
		}
	}
	if len(ClassificationLibraryIndices(ClassifyIndistinct)) != 0 {
		t.Fatal("indistinct must offer no profiles")
	}
}
