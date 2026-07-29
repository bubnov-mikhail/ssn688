package acoustics

import "math"

// ListenBand selects which part of the 10–2000 Hz analyzer the passive suite uses.
type ListenBand int

const (
	// ListenBroadband favors ships/subs; high-frequency torpedo lines are attenuated.
	ListenBroadband ListenBand = iota
	// ListenHF favors torpedo propulsion / HF machinery; mid/low ship noise is cut.
	ListenHF
)

const (
	listenHFCutoffHz   = 800.0
	listenShipHFAtten  = 18.0 // max dB cut of HF in broadband mode
	listenHFLowAtten   = 24.0 // max dB cut of LF/MF in HF mode
)

// ApplyListenBand reshapes SNR for the selected passive frequency mode and
// recomputes peak / band count / detection.
func ApplyListenBand(r *DetectionResult, band ListenBand) {
	if r == nil {
		return
	}
	for i := range r.SNR {
		freq := BandCenterHz(i)
		att := listenBandAttenuationDB(band, freq)
		r.SNR[i] -= att
		r.SignalForClassify[i] -= att * 0.7
	}
	r.PeakSNR = r.SNR.Peak()
	r.BandsAbove = r.SNR.BandsAbove(DetectThreshold)
	r.Detected = r.BandsAbove >= MinDetectBands || r.PeakSNR >= PeakDetectSNR
}

func listenBandAttenuationDB(band ListenBand, freqHz float64) float64 {
	switch band {
	case ListenHF:
		if freqHz >= listenHFCutoffHz {
			return 0
		}
		t := (listenHFCutoffHz - freqHz) / listenHFCutoffHz
		return t * listenHFLowAtten
	default: // broadband — keep ships, soft-cut torpedo HF
		if freqHz <= 650 {
			return 0
		}
		t := (freqHz - 650) / (MaxFreqHz - 650)
		if t > 1 {
			t = 1
		}
		return t * listenShipHFAtten
	}
}

// BandDisplayGain scales waterfall energy for a listen band (bio / torpedo cues).
func BandDisplayGain(band ListenBand, freqBiasHz float64) float64 {
	att := listenBandAttenuationDB(band, freqBiasHz)
	return math.Pow(10, -att/20)
}
