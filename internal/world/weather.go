package world

import "math/rand"

// Weather is discrete sea state for radar/ESM observability.
type Weather int

const (
	WeatherCalm Weather = iota // flat / glass
	WeatherLight               // light chop
	WeatherStorm               // heavy seas
	weatherCount
)

// RandomWeather picks Calm / Light / Storm uniformly (training mission variety).
func RandomWeather(rng *rand.Rand) Weather {
	if rng == nil {
		return WeatherLight
	}
	return Weather(rng.Intn(int(weatherCount)))
}

func (w Weather) String() string {
	switch w {
	case WeatherCalm:
		return "CALM"
	case WeatherLight:
		return "LIGHT SEAS"
	case WeatherStorm:
		return "STORM"
	default:
		return "UNKNOWN"
	}
}

// SeaStateInt maps weather onto the acoustic SeaState scale (0–6).
func (w Weather) SeaStateInt() int {
	switch w {
	case WeatherCalm:
		return 1
	case WeatherLight:
		return 3
	case WeatherStorm:
		return 6
	default:
		return 2
	}
}

// MastDetectFactor scales how easily a thin raised mast is seen on surface radar.
// Storm seas bury the stalk among wave clutter; calm water makes it stand out.
func (w Weather) MastDetectFactor() float64 {
	switch w {
	case WeatherCalm:
		return 1.35
	case WeatherLight:
		return 1.0
	case WeatherStorm:
		return 0.45
	default:
		return 1.0
	}
}

// ESMReceiveFactor scales received emitter strength at the ESM mast.
func (w Weather) ESMReceiveFactor() float64 {
	switch w {
	case WeatherCalm:
		return 1.15
	case WeatherLight:
		return 1.0
	case WeatherStorm:
		return 0.72
	default:
		return 1.0
	}
}
