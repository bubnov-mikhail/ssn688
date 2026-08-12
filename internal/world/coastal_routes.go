package world

import "math"

const (
	coastMinClearanceYd = 1000.0
	coastStepYd         = 1200.0
)

// BuildCoastalLoop traces an open coastal lane at approx shoreDistYd clearance.
// Ships PingPong the arc (no land-crossing chord back to the start).
// cw=true walks clockwise (shore on starboard when heading along track).
func BuildCoastalLoop(bathy *Bathymetry, id string, shoreDistYd float64, cw bool) *Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	if shoreDistYd < coastMinClearanceYd {
		shoreDistYd = coastMinClearanceYd
	}
	sx, sy, ok := findCoastSeed(bathy, shoreDistYd)
	if !ok {
		return nil
	}
	side := 90.0
	if !cw {
		side = -90
	}
	const coastArcWPs = 18
	wps := make([]Waypoint, 0, coastArcWPs)
	x, y := sx, sy
	for i := 0; i < coastArcWPs; i++ {
		wps = append(wps, Waypoint{X: x, Y: y})
		shoreBrg := bathy.NearestShoreBearingDeg(x, y)
		head := normalizeDeg(shoreBrg + side)
		nx := x + math.Sin(head*math.Pi/180)*coastStepYd
		ny := y + math.Cos(head*math.Pi/180)*coastStepYd
		nx, ny = projectToShoreBand(bathy, nx, ny, shoreDistYd)
		if !bathy.NavigableFor(nx, ny, KindSurfaceShip, 0) {
			// Nudge offshore and retry once.
			away := normalizeDeg(bathy.NearestShoreBearingDeg(nx, ny) + 180)
			nx += math.Sin(away*math.Pi/180) * 400
			ny += math.Cos(away*math.Pi/180) * 400
			nx, ny = projectToShoreBand(bathy, nx, ny, shoreDistYd)
			if !bathy.NavigableFor(nx, ny, KindSurfaceShip, 0) {
				break
			}
		}
		x, y = nx, ny
	}
	if len(wps) < 4 {
		return nil
	}
	return &Route{ID: id, Waypoints: wps, PingPong: true}
}

func findCoastSeed(b *Bathymetry, shoreDistYd float64) (float64, float64, bool) {
	minX, minY, maxX, maxY := b.BoundsYards()
	step := b.CellSize
	if step < 200 {
		step = 250
	}
	bestX, bestY := 0.0, 0.0
	bestErr := math.MaxFloat64
	found := false
	// Prefer the densest land cluster (Santa Catalina) by sampling.
	for y := minY + step; y < maxY-step; y += step * 2 {
		for x := minX + step; x < maxX-step; x += step * 2 {
			if !b.NavigableFor(x, y, KindSurfaceShip, 0) {
				continue
			}
			d := b.DistanceToShoreYd(x, y)
			if d >= ShoreRayMaxYd-1 {
				continue
			}
			err := math.Abs(d - shoreDistYd)
			if err < bestErr {
				bestErr = err
				bestX, bestY = x, y
				found = true
			}
		}
	}
	if !found || bestErr > shoreDistYd*0.6 {
		return 0, 0, false
	}
	x, y := projectToShoreBand(b, bestX, bestY, shoreDistYd)
	return x, y, true
}

func projectToShoreBand(b *Bathymetry, x, y, shoreDistYd float64) (float64, float64) {
	for i := 0; i < 12; i++ {
		if !b.OnChart(x, y) {
			break
		}
		d := b.DistanceToShoreYd(x, y)
		if math.Abs(d-shoreDistYd) < 80 {
			break
		}
		brg := b.NearestShoreBearingDeg(x, y)
		delta := shoreDistYd - d
		// Positive delta → move toward shore; negative → offshore.
		move := -delta
		x += math.Sin(brg*math.Pi/180) * move * 0.55
		y += math.Cos(brg*math.Pi/180) * move * 0.55
	}
	return x, y
}

func normalizeDeg(deg float64) float64 {
	for deg < 0 {
		deg += 360
	}
	for deg >= 360 {
		deg -= 360
	}
	return deg
}

// PlaceOnShoreBand puts e on water at about shoreDistYd from land near (nearX, nearY).
func PlaceOnShoreBand(e *Entity, bathy *Bathymetry, shoreDistYd, nearX, nearY float64) bool {
	if e == nil || bathy == nil || !bathy.Valid() {
		return false
	}
	if shoreDistYd < coastMinClearanceYd {
		shoreDistYd = coastMinClearanceYd
	}
	bestX, bestY := nearX, nearY
	bestScore := math.MaxFloat64
	minX, minY, maxX, maxY := bathy.BoundsYards()
	step := bathy.CellSize * 2
	if step < 400 {
		step = 400
	}
	for y := minY + step; y < maxY-step; y += step {
		for x := minX + step; x < maxX-step; x += step {
			if !bathy.NavigableFor(x, y, e.Kind, e.DepthFt) {
				continue
			}
			d := bathy.DistanceToShoreYd(x, y)
			if d >= ShoreRayMaxYd-1 {
				continue
			}
			err := math.Abs(d - shoreDistYd)
			prox := math.Hypot(x-nearX, y-nearY) * 0.15
			score := err + prox
			if score < bestScore {
				bestScore = score
				bestX, bestY = x, y
			}
		}
	}
	if bestScore > shoreDistYd {
		return false
	}
	x, y := projectToShoreBand(bathy, bestX, bestY, shoreDistYd)
	if !bathy.NavigableFor(x, y, e.Kind, e.DepthFt) {
		return false
	}
	e.X, e.Y = x, y
	shore := bathy.NearestShoreBearingDeg(x, y)
	// Face along coast (parallel).
	e.HeadingDeg = normalizeDeg(shore + 90)
	e.OrderedHead = e.HeadingDeg
	return true
}

// RouteCentroid returns the average of unique waypoints.
func RouteCentroid(r *Route) (float64, float64) {
	if r == nil {
		return 0, 0
	}
	n := r.UniqueCount()
	if n == 0 {
		return 0, 0
	}
	var sx, sy float64
	for i := 0; i < n; i++ {
		sx += r.Waypoints[i].X
		sy += r.Waypoints[i].Y
	}
	return sx / float64(n), sy / float64(n)
}

// AssignRoute binds an entity to a route and seeds the nearest resume waypoint.
func AssignRoute(e *Entity, r *Route) {
	if e == nil || r == nil {
		return
	}
	e.RouteID = r.ID
	e.RouteWP, e.RouteDir = r.ResumeWaypoint(e.X, e.Y)
	e.RouteNeedResume = false
	if wp, ok := r.Target(e.RouteWP); ok {
		e.HeadingDeg = BearingDegToWaypoint(e.X, e.Y, wp)
		e.OrderedHead = e.HeadingDeg
	}
}
