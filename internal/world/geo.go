package world

import (
	"fmt"
	"math"
)

// Geographic origin of the mission chart (Santa Catalina, matches tools/gen_hormuz_bathy.py).
// World (0,0) yards = this lat/lon; +X east, +Y north.
const (
	GeoOriginLatDeg = 33.30
	GeoOriginLonDeg = -118.45 // standard signed longitude (W negative)
	metersPerDegLat = 111320.0
	yardsPerMeter   = 1.0936133
)

// WorldToLatLon converts chart yards (east, north) to WGS84 degrees.
func WorldToLatLon(xYd, yYd float64) (latDeg, lonDeg float64) {
	mPerDegLon := metersPerDegLat * math.Cos(GeoOriginLatDeg*math.Pi/180)
	latDeg = GeoOriginLatDeg + (yYd / yardsPerMeter) / metersPerDegLat
	lonDeg = GeoOriginLonDeg + (xYd / yardsPerMeter) / mPerDegLon
	return latDeg, lonDeg
}

// FormatNavLatLon formats a position in maritime degrees–decimal-minutes form,
// e.g. 33°18.05'N 118°27.12'W.
func FormatNavLatLon(latDeg, lonDeg float64) string {
	return formatHemisphere(latDeg, "N", "S") + " " + formatHemisphere(lonDeg, "E", "W")
}

func formatHemisphere(deg float64, pos, neg string) string {
	hems := pos
	if deg < 0 {
		hems = neg
		deg = -deg
	}
	d := int(deg)
	minutes := (deg - float64(d)) * 60
	// Avoid 60.00' after rounding.
	if minutes >= 59.995 {
		d++
		minutes = 0
	}
	return fmt.Sprintf("%d°%05.2f'%s", d, minutes, hems)
}
