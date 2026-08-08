package acoustics

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

// Per-band data derived from a static signature profile. Avoids recomputing
// math.Mod / tonal lobes on every Detect and Classify call.
type profileBandCache struct {
	template   Spectrum
	baseLevel  [NumBands]float64 // static band floor before speed/cavitation
	tonalBoost [NumBands]float64
	// bladeNear marks bands near blade-rate harmonics (SourceSpectrum ~12% window).
	bladeNear [NumBands]bool
	// bladeClass marks bands for bladeRateMatch (symmetric 10% window).
	bladeClass [NumBands]bool
	// bladeTemplate matches historical templateSpectrum (low-side 10% only).
	bladeTemplate [NumBands]bool
	bladeHz       float64
}

var (
	profileCaches    []profileBandCache
	profileCacheByID map[string]*profileBandCache
)

func init() {
	rebuildProfileCaches()
}

func rebuildProfileCaches() {
	lib := world.SignatureLibrary
	profileCaches = make([]profileBandCache, len(lib))
	profileCacheByID = make(map[string]*profileBandCache, len(lib))
	for i, p := range lib {
		c := buildProfileBandCache(p)
		profileCaches[i] = c
		profileCacheByID[p.ID] = &profileCaches[i]
	}
}

func buildProfileBandCache(p world.SignatureProfile) profileBandCache {
	var c profileBandCache
	c.bladeHz = p.BladeRateHz
	for i := 0; i < NumBands; i++ {
		freq := BandCenterHz(i)
		level := -200.0
		for _, b := range p.Bands {
			if freq >= b.LowHz && freq <= b.HighHz {
				level = combineDB(level, b.LevelDB)
			}
		}
		if level < -100 {
			level = 70
		}
		c.baseLevel[i] = level
		c.tonalBoost[i] = TonalBoostDB(p, freq)
		if p.BladeRateHz > 0 {
			rem := math.Mod(freq, p.BladeRateHz)
			c.bladeNear[i] = rem < p.BladeRateHz*0.12 || p.BladeRateHz-rem < p.BladeRateHz*0.12
			c.bladeClass[i] = rem < p.BladeRateHz*0.1 || p.BladeRateHz-rem < p.BladeRateHz*0.1
			c.bladeTemplate[i] = rem < p.BladeRateHz*0.1
		}
	}
	c.template = computeTemplateSpectrum(p, &c)
	return c
}

func cacheForProfile(p world.SignatureProfile) *profileBandCache {
	if c := profileCacheByID[p.ID]; c != nil {
		return c
	}
	// Unknown / test-only profile: compute once without storing.
	tmp := buildProfileBandCache(p)
	return &tmp
}

func cacheForSignatureID(id string) *profileBandCache {
	return profileCacheByID[id]
}
