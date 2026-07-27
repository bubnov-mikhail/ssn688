package acoustics

// Acoustic band layout shared by detection, classification, and spectrum UI.
const (
	NumBands       = 32
	MinFreqHz      = 10.0
	MaxFreqHz      = 2000.0
	DetectThreshold = 3.0  // dB SNR per band
	MinDetectBands  = 4    // bands above threshold for contact
	PeakDetectSNR   = 10.0 // alternative detection via peak SNR
)

// BandCenterHz returns the center frequency of band i.
func BandCenterHz(i int) float64 {
	if i < 0 {
		i = 0
	}
	if i >= NumBands {
		i = NumBands - 1
	}
	t := float64(i) / float64(NumBands-1)
	return MinFreqHz + t*(MaxFreqHz-MinFreqHz)
}
