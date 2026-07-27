package acoustics

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

// Propagate applies spreading, absorption, and layer effects to a source spectrum.
func Propagate(env Environment, source Spectrum, emitter, listener *world.Entity) Spectrum {
	rangeYd := emitter.RangeYardsTo(listener)
	rangeKy := rangeYd / 1000
	spread := spreadingLossDB(rangeYd)
	layerLoss := env.LayerCrossingLoss(emitter.DepthFt, listener.DepthFt)

	var out Spectrum
	for i := 0; i < NumBands; i++ {
		freq := BandCenterHz(i)
		abs := absorptionDBPerKy(freq, rangeKy)
		depthDiff := math.Abs(emitter.DepthFt-listener.DepthFt) * 0.0015
		out[i] = source[i] - spread - abs - layerLoss - depthDiff
	}
	return out
}

// PropagateActive models outbound ping + echo return (two-way loss).
func PropagateActive(env Environment, emitter, target *world.Entity, sourceLevelDB float64) Spectrum {
	rangeYd := emitter.RangeYardsTo(target)
	rangeKy := rangeYd / 1000
	spread := spreadingLossDB(rangeYd) * 2
	layerLoss := env.LayerCrossingLoss(emitter.DepthFt, target.DepthFt) * 2

	ts := targetStrengthDB(target)

	var out Spectrum
	for i := 0; i < NumBands; i++ {
		freq := BandCenterHz(i)
		abs := absorptionDBPerKy(freq, rangeKy) * 2
		out[i] = sourceLevelDB + ts - spread - abs - layerLoss
	}
	return out
}

func targetStrengthDB(target *world.Entity) float64 {
	switch target.Kind {
	case world.KindSubmarine:
		return -8
	case world.KindSurfaceShip:
		return 15
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
