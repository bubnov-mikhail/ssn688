package acoustics

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

// Documented SSN-688 #1 attack-scope height of eye above the keel (ft), from
// Submarine School / NSL close-aboard ranging notes. At periscope depth the
// keel is ~60 ft so only the tip is above water; EyeAboveWaterFt applies a
// gameplay floor so the geometric horizon stays useful in the IR view.
const PeriEyeKeelFt = 40.0

// periMinEyeAboveWaterFt — when fully raised at PD, pretend ~6 ft of optic
// above the surface (~2.9 nm horizon) so coast/shipping remain readable.
const periMinEyeAboveWaterFt = 6.0

// EyeAboveWaterFt returns optic height above the sea surface for ranging /
// horizon math. Scales with mast Extension (0 = fully stowed).
func EyeAboveWaterFt(depthFt, extension float64) float64 {
	if extension <= 0 {
		return 0
	}
	if extension > 1 {
		extension = 1
	}
	raw := PeriEyeKeelFt - depthFt
	if raw < periMinEyeAboveWaterFt {
		raw = periMinEyeAboveWaterFt
	}
	if raw < 1.5 {
		raw = 1.5
	}
	return raw * extension
}

// GeometricHorizonYd is the distance to the sea horizon for a given eye height.
// Uses the nautical approximation d(nm) ≈ 1.17 * sqrt(h_ft).
func GeometricHorizonYd(eyeAboveWaterFt float64) float64 {
	if eyeAboveWaterFt <= 0 {
		return 0
	}
	return 1.17 * math.Sqrt(eyeAboveWaterFt) * world.YardsPerNM
}

// OpticalMaxRangeYd clips draw distance by horizon and weather haze.
func OpticalMaxRangeYd(eyeAboveWaterFt float64, weather world.Weather) float64 {
	h := GeometricHorizonYd(eyeAboveWaterFt)
	if h < 800 {
		h = 800
	}
	switch weather {
	case world.WeatherStorm:
		return h * 0.55
	case world.WeatherCalm:
		return h * 1.15
	default:
		return h
	}
}

// AngleDiffSigned returns shortest signed difference a-b in (−180, 180].
func AngleDiffSigned(a, b float64) float64 {
	d := math.Mod(a-b+180, 360)
	if d < 0 {
		d += 360
	}
	return d - 180
}

// BearingToViewX maps a true bearing into a horizontal pixel in a frame of
// width W looking along lookBrg with horizontal FOV fovDeg. ok is false when
// outside the FOV.
func BearingToViewX(brgDeg, lookBrgDeg, fovDeg float64, width int) (x int, ok bool) {
	if width <= 0 || fovDeg <= 0 {
		return 0, false
	}
	rel := AngleDiffSigned(brgDeg, lookBrgDeg)
	half := fovDeg * 0.5
	if rel < -half || rel > half {
		return 0, false
	}
	t := (rel + half) / fovDeg
	x = int(t * float64(width))
	if x < 0 {
		x = 0
	}
	if x >= width {
		x = width - 1
	}
	return x, true
}

// ViewXToBearing is the inverse of BearingToViewX (pixel center).
func ViewXToBearing(x, width int, lookBrgDeg, fovDeg float64) float64 {
	if width <= 1 {
		return lookBrgDeg
	}
	t := (float64(x) + 0.5) / float64(width)
	rel := t*fovDeg - fovDeg*0.5
	return normalizeDeg360(lookBrgDeg + rel)
}

// ShipAspectDeg is the acute angle (0..90) between LOS and the ship's heading
// (0 = bow/stern-on, 90 = beam-on).
func ShipAspectDeg(losBearingDeg, shipHeadingDeg float64) float64 {
	rel := math.Abs(AngleDiffSigned(losBearingDeg, shipHeadingDeg))
	if rel > 90 {
		rel = 180 - rel
	}
	return rel
}

// ShipAspectBin5 quantizes aspect to the nearest 5° bin in [0, 90].
func ShipAspectBin5(aspectDeg float64) int {
	b := int(math.Round(aspectDeg/5.0)) * 5
	if b < 0 {
		b = 0
	}
	if b > 90 {
		b = 90
	}
	return b
}

// ShipBeamAspect01 is 0 for bow/stern-on silhouette and 1 for full beam-on,
// given the LOS bearing from the observer to the ship and the ship's heading.
func ShipBeamAspect01(losBearingDeg, shipHeadingDeg float64) float64 {
	return math.Sin(ShipAspectDeg(losBearingDeg, shipHeadingDeg) * math.Pi / 180)
}

// ApparentShipLengthFt blends length and beam by aspect for angular size.
func ApparentShipLengthFt(lengthFt, aspect01 float64) float64 {
	if lengthFt <= 0 {
		lengthFt = 200
	}
	beam := lengthFt / 8
	if beam < 20 {
		beam = 20
	}
	if aspect01 < 0 {
		aspect01 = 0
	}
	if aspect01 > 1 {
		aspect01 = 1
	}
	return beam + (lengthFt-beam)*aspect01
}

// SeaSurfacePixelY places a sea-level point at rangeYd on the optic frame.
// With periscope-depth eye heights (~6 ft), atan(h/R) is tiny, so almost all
// contacts hug the horizon — only very close targets drop into the lower FOV.
func SeaSurfacePixelY(eyeAboveWaterFt, rangeYd float64, frameW, frameH, horizonY int, fovDeg float64) int {
	if frameH <= 0 {
		return horizonY
	}
	vFov := fovDeg * float64(frameH) / float64(maxInt(1, frameW))
	if vFov < 1 {
		vFov = 1
	}
	hYd := eyeAboveWaterFt / world.FeetPerYard
	if hYd < 0.2 {
		hYd = 0.2
	}
	if rangeYd < 20 {
		rangeYd = 20
	}
	// Flat-sea dip below the horizontal (horizon ≈ 0 elev at these heights).
	dipDeg := math.Atan(hYd/rangeYd) * 180 / math.Pi
	y := horizonY + int(dipDeg/vFov*float64(frameH)+0.5)
	if y < horizonY {
		y = horizonY
	}
	if y >= frameH {
		y = frameH - 1
	}
	return y
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// PeriShipProj is a projected surface-ship silhouette for the IR frame.
type PeriShipProj struct {
	CenterX    int
	WaterY     int // waterline pixel row
	RangeYd    float64
	WidthPx    int
	HullHPx    int
	SuperHPx   int
	Aspect01   float64
	AspectDeg  float64 // 0..90, for 5° silhouette bins
	Brightness float64 // 0..1 after haze
	Signature  string
	Sinking    bool
	SpeedKts   float64
	// BowRight: bow lies toward +X in the optic frame (false → toward −X).
	BowRight bool
}

// ProjectSurfaceShip projects a surface contact into the optic frame, or ok=false
// when outside FOV / beyond max range / not a surface ship.
func ProjectSurfaceShip(
	eyeX, eyeY float64,
	lookBrgDeg, fovDeg float64,
	frameW, frameH, horizonY int,
	maxRangeYd, eyeAboveWaterFt float64,
	ship *world.Entity,
) (PeriShipProj, bool) {
	var out PeriShipProj
	if ship == nil || ship.Kind != world.KindSurfaceShip {
		return out, false
	}
	if !ship.Alive() && ship.Status != world.StatusSinking {
		return out, false
	}
	dx := ship.X - eyeX
	dy := ship.Y - eyeY
	rangeYd := math.Hypot(dx, dy)
	if rangeYd < 30 || rangeYd > maxRangeYd {
		return out, false
	}
	brg := math.Atan2(dx, dy) * 180 / math.Pi
	if brg < 0 {
		brg += 360
	}
	cx, ok := BearingToViewX(brg, lookBrgDeg, fovDeg, frameW)
	if !ok {
		return out, false
	}
	aspectDeg := ShipAspectDeg(brg, ship.HeadingDeg)
	aspect := math.Sin(aspectDeg * math.Pi / 180)
	appLen := ApparentShipLengthFt(ship.LengthFt, aspect)
	angW := (appLen / world.FeetPerYard) / rangeYd * (180 / math.Pi)
	pxW := int(angW / fovDeg * float64(frameW))
	if pxW < 2 {
		pxW = 2
	}
	if pxW > frameW/2 {
		pxW = frameW / 2
	}
	t := rangeYd / maxRangeYd
	if t > 1 {
		t = 1
	}
	waterY := SeaSurfacePixelY(eyeAboveWaterFt, rangeYd, frameW, frameH, horizonY, fovDeg)
	mastFt := ship.LengthFt / 12
	if mastFt < 25 {
		mastFt = 25
	}
	if ship.Status == world.StatusSinking {
		mastFt *= 0.45
	}
	angH := (mastFt / world.FeetPerYard) / rangeYd * (180 / math.Pi)
	vFov := fovDeg * float64(frameH) / float64(frameW)
	if vFov < 1 {
		vFov = 1
	}
	pxH := int(angH / vFov * float64(frameH))
	if pxH < 3 {
		pxH = 3
	}
	hull := pxH * 2 / 5
	if hull < 2 {
		hull = 2
	}
	super := pxH - hull
	haze := 1 - t*t
	if haze < 0.15 {
		haze = 0.15
	}
	if haze > 1 {
		haze = 1
	}
	// Heading clockwise of LOS → bow toward increasing bearing → +X on screen.
	relHdg := AngleDiffSigned(ship.HeadingDeg, brg)
	return PeriShipProj{
		CenterX:    cx,
		WaterY:     waterY,
		RangeYd:    rangeYd,
		WidthPx:    pxW,
		HullHPx:    hull,
		SuperHPx:   super,
		Aspect01:   aspect,
		AspectDeg:  aspectDeg,
		Brightness: haze,
		Signature:  ship.SignatureID,
		Sinking:    ship.Status == world.StatusSinking,
		SpeedKts:   ship.SpeedKts,
		BowRight:   relHdg >= 0,
	}, true
}
