package world

import "math"

const (
	// ShoreRayMaxYd caps shore-distance ray casts.
	ShoreRayMaxYd = 6000.0
	// ShoreRayStepYd is the march increment along each bearing ray.
	ShoreRayStepYd = 80.0
)

// DistanceToShoreYd returns yards to the nearest dry land / off-chart edge.
// Open ocean with no land within ShoreRayMaxYd returns ShoreRayMaxYd.
func (b Bathymetry) DistanceToShoreYd(x, y float64) float64 {
	if !b.Valid() {
		return ShoreRayMaxYd
	}
	minDist := ShoreRayMaxYd
	for i := 0; i < 36; i++ {
		if d := b.rayDistanceToLandYd(x, y, float64(i)*10); d < minDist {
			minDist = d
		}
	}
	return minDist
}

// NearestShoreBearingDeg returns the compass bearing toward the closest shore.
func (b Bathymetry) NearestShoreBearingDeg(x, y float64) float64 {
	if !b.Valid() {
		return 0
	}
	best := ShoreRayMaxYd
	var bearing float64
	for i := 0; i < 36; i++ {
		brg := float64(i) * 10
		if d := b.rayDistanceToLandYd(x, y, brg); d < best {
			best = d
			bearing = brg
		}
	}
	return bearing
}

func (b Bathymetry) rayDistanceToLandYd(x, y, bearingDeg float64) float64 {
	rad := bearingDeg * math.Pi / 180
	sinB := math.Sin(rad)
	cosB := math.Cos(rad)
	step := ShoreRayStepYd
	if b.CellSize > 0 && b.CellSize < step {
		step = b.CellSize
	}
	for d := step; d <= ShoreRayMaxYd; d += step {
		px := x + sinB*d
		py := y + cosB*d
		if b.IsShoreBlocked(px, py) {
			return d - step*0.5
		}
	}
	return ShoreRayMaxYd
}

// IsSurfaceBlocked reports dry land, shoal, or off-chart for a surface ship.
// Uses cell centers (not bilinear depth) so the coast line stays crisp for AI.
func (b Bathymetry) IsSurfaceBlocked(x, y float64) bool {
	if !b.Valid() {
		return true
	}
	fx := (x - b.OriginX) / b.CellSize
	fy := (y - b.OriginY) / b.CellSize
	if fx < 0 || fy < 0 || fx >= float64(b.Width) || fy >= float64(b.Height) {
		return true
	}
	i := int(fx)
	j := int(fy)
	return b.cellDepthBlocked(i, j)
}

// IsShoreBlocked is land/shoal on-chart only (chart edge is not treated as shore).
func (b Bathymetry) IsShoreBlocked(x, y float64) bool {
	if !b.Valid() {
		return false
	}
	fx := (x - b.OriginX) / b.CellSize
	fy := (y - b.OriginY) / b.CellSize
	if fx < 0 || fy < 0 || fx >= float64(b.Width) || fy >= float64(b.Height) {
		return false
	}
	return b.cellDepthBlocked(int(fx), int(fy))
}

func (b Bathymetry) cellDepthBlocked(i, j int) bool {
	if i < 0 || j < 0 || i >= b.Width || j >= b.Height {
		return true
	}
	return float64(b.Depths[b.index(i, j)]) < 40
}

// AcousticPathBlocked reports whether a horizontal path between two world
// points crosses dry land (depth <= 0) or off-chart cells. Used to prevent
// passive/active sonar from hearing targets through an island or peninsula.
func (b Bathymetry) AcousticPathBlocked(x0, y0, x1, y1 float64) bool {
	if !b.Valid() {
		return false
	}
	dx := x1 - x0
	dy := y1 - y0
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		return false
	}
	step := b.CellSize * 0.45
	if step < 40 {
		step = 40
	}
	if step > 120 {
		step = 120
	}
	n := int(dist/step) + 1
	if n < 2 {
		n = 2
	}
	for i := 1; i < n; i++ {
		t := float64(i) / float64(n)
		if b.IsLand(x0+dx*t, y0+dy*t) {
			return true
		}
	}
	return false
}

// HorizonBlocked reports dry land between two chart points (radar, ESM, optics).
func (b Bathymetry) HorizonBlocked(x0, y0, x1, y1 float64) bool {
	return b.AcousticPathBlocked(x0, y0, x1, y1)
}

func (b Bathymetry) OnChart(x, y float64) bool {
	if !b.Valid() {
		return false
	}
	fx := (x - b.OriginX) / b.CellSize
	fy := (y - b.OriginY) / b.CellSize
	return fx >= 0 && fy >= 0 && fx < float64(b.Width) && fy < float64(b.Height)
}
