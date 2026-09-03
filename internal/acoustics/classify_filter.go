package acoustics

import (
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// ClassifyFilter restricts which signature-library entries the SPECTRUM
// classifier may offer for the current analyzer look.
type ClassifyFilter int

const (
	// ClassifyIndistinct — no usable tonals; manual classify disabled.
	ClassifyIndistinct ClassifyFilter = iota
	// ClassifyTorpedoOnly — HF fish-like spectrum.
	ClassifyTorpedoOnly
	// ClassifyPlatformOnly — LF/MF hull-like spectrum (sub + surface).
	ClassifyPlatformOnly
	// ClassifyFull — mixed bearing or ambiguous broadband with clear tonals.
	ClassifyFull
)

// ClassifyClarityMin — below this, harmonics are too buried to classify.
// Calibrated with SpectrumClarity01 so LOFAR tips become usable near SNR≈9–10 dB
// (roughly mid-teens kyd on TOWED for a noisy combatant; a few nm closer on HULL).
const ClassifyClarityMin = 0.18

// AnalyzeClassifyFilter inspects analyzer bins and bearing mix.
// mixContacts is CountSpectrumMixContacts for the analysis bearing.
func AnalyzeClassifyFilter(bins []float64, mixContacts int) ClassifyFilter {
	if len(bins) == 0 {
		return ClassifyIndistinct
	}
	peak := 0.0
	for _, v := range bins {
		if v > peak {
			peak = v
		}
	}
	clarity := SpectrumClarity01(peak)
	peaks := countClassifyTonals(bins, clarity)
	if clarity < ClassifyClarityMin || peaks < 2 {
		return ClassifyIndistinct
	}
	// Overlapping contacts on the beam — operator gets the full book.
	if mixContacts >= 2 {
		return ClassifyFull
	}

	lf := bandPower(bins, MinFreqHz, 480)
	hf := bandPower(bins, 750, MaxFreqHz)
	sum := lf + hf
	if sum < 1e-6 {
		return ClassifyIndistinct
	}
	hfShare := hf / sum

	// Both halves clearly present → ambiguous / full library.
	lfDom := lf > hf*1.15
	hfDom := hf > lf*1.25
	if lfDom && hfShare < 0.42 {
		return ClassifyPlatformOnly
	}
	if hfDom && hfShare > 0.58 {
		return ClassifyTorpedoOnly
	}
	return ClassifyFull
}

func bandPower(bins []float64, fLo, fHi float64) float64 {
	var sum float64
	for i, v := range bins {
		f := BandCenterHz(i)
		if f < fLo || f > fHi {
			continue
		}
		if v < 0 {
			continue
		}
		// Approx linear power from SNR-like dB bins.
		sum += math.Pow(10, v/10)
	}
	return sum
}

// countClassifyTonals counts local peaks that rise above the muddy floor.
func countClassifyTonals(bins []float64, clarity float64) int {
	if len(bins) < 3 {
		return 0
	}
	med := classifyMedian(bins)
	margin := 2.0 + (1-clarity)*3.0
	n := 0
	for i := 1; i < len(bins)-1; i++ {
		v := bins[i]
		if v < med+margin {
			continue
		}
		if v >= bins[i-1] && v >= bins[i+1] {
			n++
		}
	}
	return n
}

func classifyMedian(bins []float64) float64 {
	tmp := make([]float64, len(bins))
	copy(tmp, bins)
	// Insertion sort — NumBands is tiny.
	for i := 1; i < len(tmp); i++ {
		v := tmp[i]
		j := i - 1
		for j >= 0 && tmp[j] > v {
			tmp[j+1] = tmp[j]
			j--
		}
		tmp[j+1] = v
	}
	return tmp[len(tmp)/2]
}

// ProfileAllowedByFilter reports whether a library entry may be offered.
func ProfileAllowedByFilter(p world.SignatureProfile, f ClassifyFilter) bool {
	switch f {
	case ClassifyIndistinct:
		return false
	case ClassifyTorpedoOnly:
		return p.Kind == world.KindTorpedo
	case ClassifyPlatformOnly:
		return p.Kind == world.KindSubmarine || p.Kind == world.KindSurfaceShip
	default:
		return true
	}
}

// ClassificationLibraryIndices returns SignatureLibrary indices for filter f.
func ClassificationLibraryIndices(f ClassifyFilter) []int {
	var out []int
	for i, p := range world.SignatureLibrary {
		if ProfileAllowedByFilter(p, f) {
			out = append(out, i)
		}
	}
	return out
}

func (f ClassifyFilter) Label() string {
	switch f {
	case ClassifyIndistinct:
		return "INSUFFICIENT TONAL DATA"
	case ClassifyTorpedoOnly:
		return "HF — TORPEDO CLASS SET"
	case ClassifyPlatformOnly:
		return "LF/MF — PLATFORM CLASS SET"
	default:
		return "FULL LIBRARY"
	}
}
