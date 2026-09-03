package acoustics

import (
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// SoundSpeedYdPerSec is nominal sound speed in seawater (~1480 m/s).
const SoundSpeedYdPerSec = 1619.0

// EchoRangeYd is the maximum one-way range from which a two-way echo
// could have returned after ageSec since transmit.
func EchoRangeYd(ageSec float64) float64 {
	if ageSec <= 0 {
		return 0
	}
	return SoundSpeedYdPerSec * ageSec * 0.5
}

// TwoWayTravelSec is round-trip travel time for a target at rangeYd.
func TwoWayTravelSec(rangeYd float64) float64 {
	if rangeYd <= 0 {
		return 0
	}
	return 2 * rangeYd / SoundSpeedYdPerSec
}

// Propagate applies spreading, absorption, layer, and water-column effects to a source spectrum.
// When bathy is set, dry land along the horizontal path blocks propagation.
func Propagate(env Environment, source Spectrum, emitter, listener *world.Entity, bathy *world.Bathymetry) Spectrum {
	if pathBlockedByLand(bathy, emitter, listener) {
		return silentSpectrum()
	}
	rangeYd := emitter.RangeYardsTo(listener)
	rangeKy := rangeYd / 1000
	spread := passiveSpreadingLossDB(rangeYd)
	layerLoss := env.LayerCrossingLoss(emitter.DepthFt, listener.DepthFt)

	var out Spectrum
	for i := 0; i < NumBands; i++ {
		freq := BandCenterHz(i)
		abs := absorptionDBPerKy(freq, rangeKy)
		column := env.ColumnAttenuationDB(emitter.DepthFt, listener.DepthFt, freq)
		out[i] = source[i] - spread - abs - layerLoss - column
	}
	return out
}

// PropagateActive models outbound ping + echo return (two-way loss).
func PropagateActive(env Environment, emitter, target *world.Entity, sourceLevelDB float64, bathy *world.Bathymetry) Spectrum {
	if pathBlockedByLand(bathy, emitter, target) {
		return silentSpectrum()
	}
	rangeYd := emitter.RangeYardsTo(target)
	rangeKy := rangeYd / 1000
	spread := spreadingLossDB(rangeYd) * 2
	layerLoss := env.LayerCrossingLoss(emitter.DepthFt, target.DepthFt) * 2

	ts := targetStrengthDB(target)

	var out Spectrum
	for i := 0; i < NumBands; i++ {
		freq := BandCenterHz(i)
		abs := absorptionDBPerKy(freq, rangeKy) * 2
		column := env.ColumnAttenuationDB(emitter.DepthFt, target.DepthFt, freq) * 2
		out[i] = sourceLevelDB + ts - spread - abs - layerLoss - column
	}
	return out
}

func targetStrengthDB(target *world.Entity) float64 {
	switch target.Kind {
	case world.KindSubmarine:
		return -8
	case world.KindSurfaceShip:
		return 15
	case world.KindTorpedo:
		return -16 // small reflector — close-range active only
	default:
		return 0
	}
}

// PingSourceLevel converts active sonar power setting to source level.
func PingSourceLevel(power01 float64) float64 {
	if power01 < 0.05 {
		power01 = 0.05
	}
	return 210 + 20*math.Log10(power01)
}

func pathBlockedByLand(bathy *world.Bathymetry, a, b *world.Entity) bool {
	if bathy == nil || !bathy.Valid() || a == nil || b == nil {
		return false
	}
	return bathy.AcousticPathBlocked(a.X, a.Y, b.X, b.Y)
}

func silentSpectrum() Spectrum {
	var s Spectrum
	s.Clear()
	return s
}
