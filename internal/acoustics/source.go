package acoustics

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

// SourceSpectrum computes the radiated noise spectrum of a platform.
func SourceSpectrum(e *world.Entity) Spectrum {
	if e.Status == world.StatusSinking {
		// Breaking bulkheads / secondary explosions — loud broadband wreck noise.
		s := NewSpectrumFlat(118)
		for i := range s {
			freq := BandCenterHz(i)
			s[i] += 8*math.Sin(freq*0.02) + (freq/MaxFreqHz)*12
			if e.WreckNoiseUntil > 0 {
				s[i] += 10
			}
		}
		return s
	}
	profile, ok := world.ProfileByID(e.SignatureID)
	if !ok {
		return NewSpectrumFlat(80)
	}
	pc := cacheForSignatureID(e.SignatureID)

	var s Spectrum
	for i := 0; i < NumBands; i++ {
		freq := BandCenterHz(i)
		var level float64
		if pc != nil {
			level = pc.baseLevel[i]
		} else {
			level = -200.0
			for _, b := range profile.Bands {
				if freq >= b.LowHz && freq <= b.HighHz {
					level = combineDB(level, b.LevelDB)
				}
			}
			if level < -100 {
				level = 70
			}
		}

		if e.Kind == world.KindTorpedo {
			// Torpedo propulsion is HF-heavy; do not apply ship machinery curve.
			level += math.Min(6, (e.SpeedKts-20)*0.08)
			if pc != nil {
				level += pc.tonalBoost[i]
			} else {
				level += TonalBoostDB(profile, freq)
			}
			if profile.BladeRateHz > 0 {
				rem := math.Mod(freq, profile.BladeRateHz)
				if rem < profile.BladeRateHz*0.15 || profile.BladeRateHz-rem < profile.BladeRateHz*0.15 {
					level += 6 + e.SpeedKts*0.04
				}
			}
			// Running torpedoes always cavitate somewhat at the propeller.
			cav := 0.35 + math.Min(0.5, e.SpeedKts/120)
			level += cav * (4 + (freq/MaxFreqHz)*10)
			s[i] = level
			continue
		}

		// Machinery grows with speed.
		spd := math.Abs(e.SpeedKts)
		level += machineryNoiseDB(spd)

		// Discrete LOFAR/DEMON tonals (Cold Waters fingerprint lines).
		if pc != nil {
			level += pc.tonalBoost[i]
		} else {
			level += TonalBoostDB(profile, freq)
		}

		// Propeller blade rate harmonics (broadband reinforcement).
		if profile.BladeRateHz > 0 {
			onBlade := false
			if pc != nil {
				onBlade = pc.bladeNear[i]
			} else {
				rem := math.Mod(freq, profile.BladeRateHz)
				onBlade = rem < profile.BladeRateHz*0.12 || profile.BladeRateHz-rem < profile.BladeRateHz*0.12
			}
			if onBlade {
				level += 4 + spd*0.12
			}
		}

		// Cavitation broadband, stronger at higher frequencies.
		cav := CavitationSeverity(e.DepthFt, spd)
		if cav > 0 {
			highFreqBoost := (freq / MaxFreqHz) * profile.CavitationDB * 0.08 * cav
			level += highFreqBoost
			level += cav * 12
		}

		// Flow noise at high speed.
		if spd > 15 {
			level += (spd - 15) * 0.35 * (freq / MaxFreqHz)
		}

		if e.TransientUntil > 0 && e.TransientLevelDB > 0 {
			// Mechanical transients: strong LF/MF snap with a broadband tail.
			bias := e.TransientFreqHz
			if bias <= 0 {
				bias = 180
			}
			delta := math.Abs(freq - bias)
			width := bias*0.55 + 120
			if width < 120 {
				width = 120
			}
			shape := math.Max(0, 1-delta/width)
			level += e.TransientLevelDB * shape
			level += e.TransientLevelDB * 0.18 * (1 - math.Min(1, freq/MaxFreqHz))
		}

		s[i] = level
	}
	return s
}

func machineryNoiseDB(speedKts float64) float64 {
	speedKts = math.Abs(speedKts)
	if speedKts < 3 {
		return 0
	}
	return (speedKts - 3) * 0.9
}

// SelfNoiseSpectrum is the noise floor produced by the listening platform.
func SelfNoiseSpectrum(e *world.Entity, env Environment, array PassiveArrayKind, towedCablePct float64) Spectrum {
	spd := math.Abs(e.SpeedKts)
	base := 62.0 + spd*1.1
	if spd > 12 {
		base += (spd - 12) * 2.2
	}
	// Surface combatants tow powerful passive arrays with lower flow noise at patrol speed.
	if e.Kind == world.KindSurfaceShip && e.DepthFt < 30 {
		base -= 10
	}
	if array == PassiveArrayTowed && towedCablePct > 0 {
		base -= towedCablePct * 4.0
	}

	var s Spectrum
	for i := 0; i < NumBands; i++ {
		freq := BandCenterHz(i)
		level := base

		// Hull flow noise rises with frequency.
		level += (freq / MaxFreqHz) * spd * 0.25

		cav := CavitationSeverity(e.DepthFt, spd)
		level += cav * 18
		level += cav * (freq / MaxFreqHz) * 10

		s[i] = level
	}

	ambient := env.AmbientSpectrum(e.DepthFt)
	return s.AddPower(ambient)
}
