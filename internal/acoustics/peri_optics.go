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

// BearingToViewXF maps a true bearing into a fractional horizontal pixel.
func BearingToViewXF(brgDeg, lookBrgDeg, fovDeg float64, width int) (x float64, ok bool) {
	if width <= 0 || fovDeg <= 0 {
		return 0, false
	}
	rel := AngleDiffSigned(brgDeg, lookBrgDeg)
	half := fovDeg * 0.5
	if rel < -half || rel > half {
		return 0, false
	}
	t := (rel + half) / fovDeg
	x = t * float64(width)
	if x < 0 {
		x = 0
	}
	max := float64(width) - 1e-6
	if x > max {
		x = max
	}
	return x, true
}

// BearingToViewX maps a true bearing into a horizontal pixel in a frame of
// width W looking along lookBrg with horizontal FOV fovDeg. ok is false when
// outside the FOV.
func BearingToViewX(brgDeg, lookBrgDeg, fovDeg float64, width int) (x int, ok bool) {
	xf, ok := BearingToViewXF(brgDeg, lookBrgDeg, fovDeg, width)
	if !ok {
		return 0, false
	}
	x = int(math.Floor(xf))
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

// ShipAspectDeg is which face of the ship points at the observer (0..180):
// 0 = bow toward us, 90 = beam, 180 = stern toward us.
// losBearingDeg is eye→ship; we compare heading to the opposite (ship→eye).
func ShipAspectDeg(losBearingDeg, shipHeadingDeg float64) float64 {
	toEye := normalizeDeg360(losBearingDeg + 180)
	return math.Abs(AngleDiffSigned(toEye, shipHeadingDeg))
}

// ShipAspectBin quantizes aspect to the nearest 1° bin in [0, 180].
func ShipAspectBin(aspectDeg float64) int {
	b := int(math.Round(aspectDeg))
	if b < 0 {
		b = 0
	}
	if b > 180 {
		b = 180
	}
	return b
}

// ShipAspectBin5 is kept as an alias for older call sites (now 1° quantization).
func ShipAspectBin5(aspectDeg float64) int {
	return ShipAspectBin(aspectDeg)
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
	CenterX    float64 // fractional pixel — avoids staircase crawl when upscaled
	WaterY     int     // waterline pixel row
	RangeYd    float64
	WidthPx    int
	HullHPx    int
	SuperHPx   int
	Aspect01   float64
	AspectDeg  float64 // 0..180, for 1° silhouette bins (bow→beam→stern)
	Brightness float64 // 0..1 after haze
	Signature  string
	Sinking    bool
	// SinkFrac is DepthFt/air-draft while StatusSinking (0=afloat … 1=just under).
	// Sprite is lowered by this fraction and clipped at WaterY; ≥1 → not projected.
	SinkFrac float64
	SpeedKts float64
	// BowRight: bow lies toward +X in the optic frame (false → toward −X).
	BowRight bool
	// Fire01 is peri IR hull-fire intensity (0..1) from HullFireUntil.
	Fire01 float64
	// FirePhase is game time used to flicker burning patches.
	FirePhase float64
}

// SurfaceShipAirDraftFt is keel-to-mast height (ft) used for peri immersion.
// Matches the floating silhouette scale (LengthFt/12, floor 25 ft).
func SurfaceShipAirDraftFt(lengthFt float64) float64 {
	h := lengthFt / 12
	if h < 25 {
		h = 25
	}
	return h
}

// SurfaceShipSubmergeSec is game-seconds until a sinking surface ship is fully
// under (sprite gone), given SinkRateFPM. Bottom contact may be much later.
func SurfaceShipSubmergeSec(lengthFt, sinkRateFPM float64) float64 {
	if sinkRateFPM <= 0 {
		return 0
	}
	return SurfaceShipAirDraftFt(lengthFt) / sinkRateFPM * 60
}

// ProjectSurfaceShip projects a surface contact into the optic frame from
// entity truth (X/Y/HeadingDeg/LengthFt), or ok=false when outside FOV /
// beyond max range / not a surface ship. Contact TMA estimates are never used
// for silhouette size or aspect — those feed from this optic via
// UpdateContactsFromPeriscope instead.
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
	airDraft := SurfaceShipAirDraftFt(ship.LengthFt)
	sinkFrac := 0.0
	if ship.Status == world.StatusSinking {
		if ship.DepthFt >= airDraft {
			return out, false // superstructure under — no IR silhouette
		}
		if airDraft > 0 {
			sinkFrac = ship.DepthFt / airDraft
			if sinkFrac < 0 {
				sinkFrac = 0
			}
		}
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
	cx, ok := BearingToViewXF(brg, lookBrgDeg, fovDeg, frameW)
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
	mastFt := airDraft
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
		SinkFrac:   sinkFrac,
		SpeedKts:   ship.SpeedKts,
		BowRight:   relHdg >= 0,
	}, true
}
