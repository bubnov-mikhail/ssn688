package world

// Mast raise limits for 688-class ESM/optical masts (gameplay; open sources vary).
// Modern photonics masts cite ~12 kn operate / 16 kn survive; Cold War optical/ESM
// masts are closer to Dangerous Waters / anecdotal ~8 kn shear risk.
const (
	ESMMastMaxDepthFt  = 60.0 // periscope depth
	ESMMastMaxSpeedKts = 8.0
	ESMMastRaiseSec    = 4.0 // time to fully extend
	ESMMastLowerSec    = 3.0
)

// RadarProfile describes a surface search / navigation radar for ESM + counter-detect.
type RadarProfile struct {
	ID           string  // unique radar id (e.g. "mr320")
	Name         string  // display / library name
	ClassLabel   string  // short class for classification (e.g. "SURFACE SEARCH")
	Band         string  // "S", "X", "L", ...
	ScanPeriodSec float64 // full 360° rotation period
	BeamHalfDeg  float64 // half-power half-beam in azimuth
	PeakPowerKW  float64 // relative peak (gameplay)
	MaxRangeYd   float64 // instrumented / practical surface range
	MastDetectYd float64 // max range to detect a thin raised mast (calm baseline)
}

// RadarBySignature returns the search/nav radar fitted to a platform signature.
// Periods from open sources: SPS-49 class 6/12 rpm (10/5 s); surface/nav ~15–24 rpm (2.5–4 s).
func RadarBySignature(sigID string) (RadarProfile, bool) {
	switch sigID {
	case "udaloy", "kresta2":
		// MR-320 Top Plate / Fregat-class air+surface — ~6–12 rpm long-range modes.
		return RadarProfile{
			ID: "mr320_fregat", Name: "MR-320 Fregat (Top Plate)",
			ClassLabel: "AIR/SURFACE SEARCH", Band: "S",
			ScanPeriodSec: 6.0, BeamHalfDeg: 2.0, PeakPowerKW: 90,
			MaxRangeYd: 45000, MastDetectYd: 14000,
		}, true
	case "krivak":
		return RadarProfile{
			ID: "mr310u", Name: "MR-310U Angara-M",
			ClassLabel: "AIR/SURFACE SEARCH", Band: "S",
			ScanPeriodSec: 5.0, BeamHalfDeg: 2.2, PeakPowerKW: 55,
			MaxRangeYd: 32000, MastDetectYd: 11000,
		}, true
	case "gorshkov":
		// Poliment 5P-20K / Furke-4 class air+surface suite.
		return RadarProfile{
			ID: "poliment_furke", Name: "Poliment / Furke-4",
			ClassLabel: "AIR/SURFACE SEARCH", Band: "S/X",
			ScanPeriodSec: 4.5, BeamHalfDeg: 1.6, PeakPowerKW: 70,
			MaxRangeYd: 40000, MastDetectYd: 13000,
		}, true
	case "grisha":
		// Compact coastal ASW — faster surface search (~15 rpm ≈ 4 s).
		return RadarProfile{
			ID: "mr302", Name: "MR-302 Rubka",
			ClassLabel: "SURFACE SEARCH", Band: "X",
			ScanPeriodSec: 4.0, BeamHalfDeg: 1.8, PeakPowerKW: 35,
			MaxRangeYd: 22000, MastDetectYd: 9000,
		}, true
	case "merchant":
		return RadarProfile{
			ID: "nav_xband", Name: "Commercial X-band Nav",
			ClassLabel: "NAV RADAR", Band: "X",
			ScanPeriodSec: 2.5, BeamHalfDeg: 1.5, PeakPowerKW: 12,
			MaxRangeYd: 16000, MastDetectYd: 4500,
		}, true
	case "tanker":
		return RadarProfile{
			ID: "nav_sband", Name: "Commercial S-band Nav",
			ClassLabel: "NAV RADAR", Band: "S",
			ScanPeriodSec: 3.0, BeamHalfDeg: 1.8, PeakPowerKW: 18,
			MaxRangeYd: 20000, MastDetectYd: 5000,
		}, true
	case "fishing":
		return RadarProfile{
			ID: "fishfinder_nav", Name: "Small-craft Nav Radar",
			ClassLabel: "NAV RADAR", Band: "X",
			ScanPeriodSec: 2.5, BeamHalfDeg: 2.0, PeakPowerKW: 6,
			MaxRangeYd: 10000, MastDetectYd: 2800,
		}, true
	default:
		if sigID == "" {
			return RadarProfile{}, false
		}
		return RadarProfile{}, false
	}
}

// SurfaceHasSearchRadar is true for platforms that radiate a search/nav sweep.
func SurfaceHasSearchRadar(e *Entity) bool {
	if e == nil || e.Kind != KindSurfaceShip || !e.Alive() {
		return false
	}
	_, ok := RadarBySignature(e.SignatureID)
	return ok
}

// RadarBeamDeg returns the instantaneous main-beam azimuth (true) for a rotating radar.
func RadarBeamDeg(e *Entity, gameTime float64) float64 {
	if e == nil {
		return 0
	}
	p, ok := RadarBySignature(e.SignatureID)
	if !ok || p.ScanPeriodSec <= 0 {
		return e.HeadingDeg
	}
	phase := mathMod(gameTime/p.ScanPeriodSec, 1) * 360
	return normalizeDeg360(e.HeadingDeg + phase)
}

// RadarIlluminates reports whether the main beam currently covers bearingToTarget.
func RadarIlluminates(e *Entity, gameTime, bearingToTarget float64) bool {
	p, ok := RadarBySignature(e.SignatureID)
	if !ok {
		return false
	}
	beam := RadarBeamDeg(e, gameTime)
	diff := mathAbs(shortestDeg(beam, bearingToTarget))
	return diff <= p.BeamHalfDeg
}

// RadarBeamPassed is true if the rotating main beam covered bearingToTarget
// at any point during (gameTime-dt, gameTime]. Needed because narrow nav beams
// dwell shorter than a 10 Hz sim tick and would otherwise be invisible to ESM.
func RadarBeamPassed(e *Entity, gameTime, dt, bearingToTarget float64) bool {
	p, ok := RadarBySignature(e.SignatureID)
	if !ok || p.ScanPeriodSec <= 0 {
		return false
	}
	if dt <= 1e-6 {
		return RadarIlluminates(e, gameTime, bearingToTarget)
	}
	half := p.BeamHalfDeg
	if half < 0.5 {
		half = 0.5
	}
	rate := 360.0 / p.ScanPeriodSec
	sweep := rate * dt
	steps := int(sweep/half) + 2
	if steps < 2 {
		steps = 2
	}
	if steps > 32 {
		steps = 32
	}
	for i := 0; i <= steps; i++ {
		t := gameTime - dt + dt*float64(i)/float64(steps)
		if t < 0 {
			t = 0
		}
		beam := RadarBeamDeg(e, t)
		if mathAbs(shortestDeg(beam, bearingToTarget)) <= p.BeamHalfDeg {
			return true
		}
	}
	return false
}

func mathMod(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	v := a - float64(int(a/b))*b
	if v < 0 {
		v += b
	}
	return v
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func shortestDeg(from, to float64) float64 {
	d := to - from
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}

func normalizeDeg360(d float64) float64 {
	for d < 0 {
		d += 360
	}
	for d >= 360 {
		d -= 360
	}
	return d
}
