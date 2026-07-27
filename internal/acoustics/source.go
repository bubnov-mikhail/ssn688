package acoustics

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

// SourceSpectrum computes the radiated noise spectrum of a platform.
func SourceSpectrum(e *world.Entity) Spectrum {
	profile, ok := world.ProfileByID(e.SignatureID)
	if !ok {
		return NewSpectrumFlat(80)
	}

	var s Spectrum
	for i := 0; i < NumBands; i++ {
		freq := BandCenterHz(i)
		level := -200.0
		for _, b := range profile.Bands {
			if freq >= b.LowHz && freq <= b.HighHz {
				level = combineDB(level, b.LevelDB)
			}
		}
		if level < -100 {
			level = 70
		}

		// Machinery grows with speed.
		level += machineryNoiseDB(e.SpeedKts)

		// Discrete LOFAR/DEMON tonals (Cold Waters fingerprint lines).
		level += TonalBoostDB(profile, freq)

		// Propeller blade rate harmonics (broadband reinforcement).
		if profile.BladeRateHz > 0 {
			rem := math.Mod(freq, profile.BladeRateHz)
			if rem < profile.BladeRateHz*0.12 || profile.BladeRateHz-rem < profile.BladeRateHz*0.12 {
				level += 4 + e.SpeedKts*0.12
			}
		}

		// Cavitation broadband, stronger at higher frequencies.
		cav := CavitationSeverity(e.DepthFt, e.SpeedKts)
		if cav > 0 {
			highFreqBoost := (freq / MaxFreqHz) * profile.CavitationDB * 0.08 * cav
			level += highFreqBoost
			level += cav * 12
		}

		// Flow noise at high speed.
		if e.SpeedKts > 15 {
			level += (e.SpeedKts - 15) * 0.35 * (freq / MaxFreqHz)
		}

		s[i] = level
	}
	return s
}

func machineryNoiseDB(speedKts float64) float64 {
	if speedKts < 3 {
		return 0
	}
	return (speedKts - 3) * 0.9
}

// SelfNoiseSpectrum is the noise floor produced by the listening platform.
func SelfNoiseSpectrum(e *world.Entity, env Environment, array PassiveArrayKind, towedCablePct float64) Spectrum {
	base := 62.0 + e.SpeedKts*1.1
	if e.SpeedKts > 12 {
		base += (e.SpeedKts - 12) * 2.2
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
		level += (freq / MaxFreqHz) * e.SpeedKts * 0.25

		cav := CavitationSeverity(e.DepthFt, e.SpeedKts)
		level += cav * 18
		level += cav * (freq / MaxFreqHz) * 10

		s[i] = level
	}

	ambient := env.AmbientSpectrum(e.DepthFt)
	return s.AddPower(ambient)
}
