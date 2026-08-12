package world

import "math"

const (
	transitMinClearanceYd = 1000.0
	transitCornerMarginYd = 1800.0
)

// BuildNWSETransit builds a coarse PingPong lane from chart NW (upper-left)
// toward SE (lower-right). lateralOffsetYd shifts the lane along the perpendicular
// so parallel routes can cross near the island without hugging the coastline.
func BuildNWSETransit(bathy *Bathymetry, id string, lateralOffsetYd float64, numWP int) *Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	if numWP < 3 {
		numWP = 5
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	m := transitCornerMarginYd
	start := Waypoint{X: minX + m, Y: maxY - m}
	end := Waypoint{X: maxX - m, Y: minY + m}
	dx := end.X - start.X
	dy := end.Y - start.Y
	span := math.Hypot(dx, dy)
	if span < 1 {
		return nil
	}
	px, py := -dy/span, dx/span // left-normal of NW→SE
	start.X += px * lateralOffsetYd
	start.Y += py * lateralOffsetYd
	end.X += px * lateralOffsetYd
	end.Y += py * lateralOffsetYd

	wps := make([]Waypoint, 0, numWP)
	for i := 0; i < numWP; i++ {
		t := float64(i) / float64(numWP-1)
		x := start.X + (end.X-start.X)*t
		y := start.Y + (end.Y-start.Y)*t
		x, y = snapNavigableClear(bathy, x, y, transitMinClearanceYd)
		wps = append(wps, Waypoint{X: x, Y: y})
	}
	// Drop near-duplicate samples after snapping (keep dense detours around land).
	wps = dedupeWaypoints(wps, 120)
	if len(wps) < 3 {
		return nil
	}
	return &Route{ID: id, Waypoints: wps, PingPong: true}
}

func dedupeWaypoints(wps []Waypoint, minSepYd float64) []Waypoint {
	if len(wps) == 0 {
		return wps
	}
	out := []Waypoint{wps[0]}
	for i := 1; i < len(wps); i++ {
		prev := out[len(out)-1]
		if math.Hypot(wps[i].X-prev.X, wps[i].Y-prev.Y) >= minSepYd {
			out = append(out, wps[i])
		}
	}
	if len(out) >= 2 {
		last := wps[len(wps)-1]
		prev := out[len(out)-1]
		if math.Hypot(last.X-prev.X, last.Y-prev.Y) >= minSepYd*0.5 {
			if prev.X != last.X || prev.Y != last.Y {
				out = append(out, last)
			}
		} else {
			out[len(out)-1] = last
		}
	}
	return out
}

// snapNavigableClear nudges (x,y) onto water with at least minClear shore distance.
func snapNavigableClear(b *Bathymetry, x, y, minClear float64) (float64, float64) {
	if b.NavigableFor(x, y, KindSurfaceShip, 0) && b.DistanceToShoreYd(x, y) >= minClear {
		return x, y
	}
	bestX, bestY := x, y
	bestScore := math.MaxFloat64
	for rad := 200.0; rad <= 8000; rad += 200 {
		for brg := 0; brg < 360; brg += 15 {
			a := float64(brg) * math.Pi / 180
			nx := x + math.Sin(a)*rad
			ny := y + math.Cos(a)*rad
			if !b.NavigableFor(nx, ny, KindSurfaceShip, 0) {
				continue
			}
			d := b.DistanceToShoreYd(nx, ny)
			if d < minClear {
				continue
			}
			// Prefer staying near the intended point; slight bias to more clearance.
			score := rad - d*0.15
			if score < bestScore {
				bestScore = score
				bestX, bestY = nx, ny
			}
		}
		if bestScore < math.MaxFloat64/2 {
			return bestX, bestY
		}
	}
	// Last resort: any navigable nearby.
	for rad := 200.0; rad <= 10000; rad += 250 {
		for brg := 0; brg < 360; brg += 20 {
			a := float64(brg) * math.Pi / 180
			nx := x + math.Sin(a)*rad
			ny := y + math.Cos(a)*rad
			if b.NavigableFor(nx, ny, KindSurfaceShip, 0) {
				return nx, ny
			}
		}
	}
	return x, y
}

// PlaceOnRouteFraction puts e on route r at fraction t∈[0,1] (nearest waypoint).
// bathy is used to re-snap if the stored WP is somehow unnavigable for e.
func PlaceOnRouteFraction(e *Entity, r *Route, t float64, bathy *Bathymetry) bool {
	if e == nil || r == nil {
		return false
	}
	n := r.UniqueCount()
	if n == 0 {
		return false
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	idx := int(t*float64(n-1) + 0.5)
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	// Search outward from the preferred index for a WP navigable for this platform.
	for delta := 0; delta < n; delta++ {
		for _, sign := range []int{1, -1} {
			i := idx
			if delta > 0 {
				i = idx + sign*delta
			}
			if i < 0 || i >= n {
				continue
			}
			wp := r.Waypoints[i]
			x, y := wp.X, wp.Y
			if bathy != nil && bathy.Valid() {
				if !bathy.NavigableFor(x, y, e.Kind, e.DepthFt) {
					x, y = snapNavigableClear(bathy, x, y, transitMinClearanceYd)
					if !bathy.NavigableFor(x, y, e.Kind, e.DepthFt) {
						continue
					}
				}
			}
			e.X, e.Y = x, y
			AssignRoute(e, r)
			return true
		}
		if delta == 0 {
			continue
		}
	}
	return false
}

// DistToRouteYd is the minimum distance from (x,y) to any segment of r.
func DistToRouteYd(x, y float64, r *Route) float64 {
	if r == nil {
		return math.MaxFloat64
	}
	n := r.UniqueCount()
	if n == 0 {
		return math.MaxFloat64
	}
	if n == 1 {
		return math.Hypot(r.Waypoints[0].X-x, r.Waypoints[0].Y-y)
	}
	best := math.MaxFloat64
	for i := 1; i < n; i++ {
		d := distPointToSegYd(x, y, r.Waypoints[i-1], r.Waypoints[i])
		if d < best {
			best = d
		}
	}
	return best
}

// MinDistToRoutesYd is min DistToRouteYd over all routes.
func MinDistToRoutesYd(x, y float64, routes []*Route) float64 {
	best := math.MaxFloat64
	for _, r := range routes {
		if d := DistToRouteYd(x, y, r); d < best {
			best = d
		}
	}
	return best
}

func distPointToSegYd(px, py float64, a, b Waypoint) float64 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	len2 := dx*dx + dy*dy
	if len2 < 1e-6 {
		return math.Hypot(a.X-px, a.Y-py)
	}
	t := ((px-a.X)*dx + (py-a.Y)*dy) / len2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	qx := a.X + t*dx
	qy := a.Y + t*dy
	return math.Hypot(qx-px, qy-py)
}

// PlaceNearChartCorner puts e on navigable water near a chart corner.
// corner: "SW" lower-left, "NW" upper-left, "SE" lower-right, "NE" upper-right.
// Route distance must be in [minRouteYd, maxRouteYd]; maxRouteYd<=0 disables the upper bound.
func PlaceNearChartCorner(e *Entity, bathy *Bathymetry, corner string, routes []*Route, minRouteYd, maxRouteYd float64) bool {
	if e == nil || bathy == nil || !bathy.Valid() {
		return false
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	m := transitCornerMarginYd
	tx, ty := minX+m, minY+m // SW default
	switch corner {
	case "NW":
		tx, ty = minX+m, maxY-m
	case "NE":
		tx, ty = maxX-m, maxY-m
	case "SE":
		tx, ty = maxX-m, minY+m
	}
	bestX, bestY := tx, ty
	bestScore := math.MaxFloat64
	step := bathy.CellSize
	if step < 250 {
		step = 250
	}
	for y := minY + step; y < maxY-step; y += step {
		for x := minX + step; x < maxX-step; x += step {
			if !bathy.NavigableFor(x, y, e.Kind, e.DepthFt) {
				continue
			}
			if bathy.DistanceToShoreYd(x, y) < transitMinClearanceYd {
				continue
			}
			dRoute := MinDistToRoutesYd(x, y, routes)
			if dRoute < minRouteYd {
				continue
			}
			if maxRouteYd > 0 && dRoute > maxRouteYd {
				continue
			}
			// Prefer SW corner, then closer to the band mid (~half of max clear).
			targetBand := minRouteYd
			if maxRouteYd > minRouteYd {
				targetBand = (minRouteYd + maxRouteYd) * 0.5
			}
			prox := math.Hypot(x-tx, y-ty) + math.Abs(dRoute-targetBand)*0.35
			if prox < bestScore {
				bestScore = prox
				bestX, bestY = x, y
			}
		}
	}
	if bestScore > 1e12 {
		return false
	}
	e.X, e.Y = bestX, bestY
	e.HeadingDeg = 45
	e.OrderedHead = e.HeadingDeg
	return true
}
