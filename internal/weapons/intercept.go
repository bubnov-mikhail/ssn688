package weapons

import (
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// CruiseSpeedKts maps fire-control speed setting to ordered Mk48 run speed.
func CruiseSpeedKts(setting string) float64 {
	return speedKts(setting)
}

// InterceptCourseDeg solves a constant-speed collision course from the origin to a
// target at (relX, relY) yards moving at tgtCourseDeg / tgtSpeedKts. weaponKts is
// the interceptor speed. Returns false when no positive-time intercept exists.
func InterceptCourseDeg(relX, relY, tgtCourseDeg, tgtSpeedKts, weaponKts float64) (courseDeg float64, ok bool) {
	if weaponKts < 1 {
		return 0, false
	}
	rangeYd := math.Hypot(relX, relY)
	if rangeYd < 1 {
		return 0, false
	}
	wYPS := weaponKts * world.KnotsToYPS
	tr := tgtCourseDeg * math.Pi / 180
	tvx := math.Sin(tr) * tgtSpeedKts * world.KnotsToYPS
	tvy := math.Cos(tr) * tgtSpeedKts * world.KnotsToYPS

	// |P + Vt·t| = Vw·t  →  (Vt·Vt − Vw²)t² + 2(P·Vt)t + P·P = 0
	a := tvx*tvx + tvy*tvy - wYPS*wYPS
	b := 2 * (relX*tvx + relY*tvy)
	c := relX*relX + relY*relY

	tHit := math.NaN()
	if math.Abs(a) < 1e-9 {
		if math.Abs(b) < 1e-9 {
			return 0, false
		}
		t := -c / b
		if t > 0.05 {
			tHit = t
		}
	} else {
		disc := b*b - 4*a*c
		if disc < 0 {
			return 0, false
		}
		root := math.Sqrt(disc)
		t1 := (-b - root) / (2 * a)
		t2 := (-b + root) / (2 * a)
		for _, t := range []float64{t1, t2} {
			if t > 0.05 && (math.IsNaN(tHit) || t < tHit) {
				tHit = t
			}
		}
	}
	if math.IsNaN(tHit) {
		return 0, false
	}

	ix := relX + tvx*tHit
	iy := relY + tvy*tHit
	course := math.Atan2(ix, iy) * 180 / math.Pi
	if course < 0 {
		course += 360
	}
	return course, true
}

// TorpedoInterceptGyro advances geometry through tube-clear run along ownship
// heading, then returns the gyro course that intercepts the moving target.
func TorpedoInterceptGyro(ownX, ownY, ownHeadDeg, tgtX, tgtY, tgtCourseDeg, tgtSpeedKts, weaponKts float64) (courseDeg float64, ok bool) {
	if weaponKts < 1 {
		return 0, false
	}
	const exitKts = 18.0
	avgClear := 0.5 * (exitKts + weaponKts)
	if avgClear < 1 {
		avgClear = weaponKts
	}
	clearSec := TubeClearYd / (avgClear * world.KnotsToYPS)
	hrad := ownHeadDeg * math.Pi / 180
	ox := ownX + math.Sin(hrad)*TubeClearYd
	oy := ownY + math.Cos(hrad)*TubeClearYd
	trad := tgtCourseDeg * math.Pi / 180
	tx := tgtX + math.Sin(trad)*tgtSpeedKts*world.KnotsToYPS*clearSec
	ty := tgtY + math.Cos(trad)*tgtSpeedKts*world.KnotsToYPS*clearSec
	return InterceptCourseDeg(tx-ox, ty-oy, tgtCourseDeg, tgtSpeedKts, weaponKts)
}
