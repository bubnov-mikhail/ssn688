package world

import "math"

// WaypointVisitYd is the radius at which a route waypoint counts as visited.
const WaypointVisitYd = 500.0

// Waypoint is a navigational fix in yards (east, north).
type Waypoint struct {
	X, Y float64
}

// Route is an ordered list of waypoints.
// Looped: last equals first — wrap forward only (may chord across land if poorly authored).
// PingPong: open polyline — at either end reverse direction (last becomes the new "start").
type Route struct {
	ID        string
	Waypoints []Waypoint
	Looped    bool
	PingPong  bool
}

// FindRoute returns the route with the given ID, or nil.
func FindRoute(routes []*Route, id string) *Route {
	if id == "" {
		return nil
	}
	for _, r := range routes {
		if r != nil && r.ID == id {
			return r
		}
	}
	return nil
}

// UniqueCount is the number of distinct navigation fixes (loop closing point excluded).
func (r *Route) UniqueCount() int {
	if r == nil || len(r.Waypoints) == 0 {
		return 0
	}
	n := len(r.Waypoints)
	if r.Looped && n >= 2 {
		last := r.Waypoints[n-1]
		first := r.Waypoints[0]
		if math.Hypot(last.X-first.X, last.Y-first.Y) < 1 {
			return n - 1
		}
	}
	return n
}

// Target returns the waypoint at index (clamped; looped indices wrap).
func (r *Route) Target(index int) (Waypoint, bool) {
	n := r.UniqueCount()
	if n == 0 {
		return Waypoint{}, false
	}
	if index < 0 {
		index = 0
	}
	if r.Looped {
		index = index % n
	} else if index >= n {
		index = n - 1
	}
	return r.Waypoints[index], true
}

// Advance moves past a visited waypoint. dir is +1 forward / -1 reverse (0 → +1).
// For PingPong, hitting an end flips direction so the terminus becomes the next origin.
func (r *Route) Advance(index, dir int) (newIndex, newDir int) {
	n := r.UniqueCount()
	if n == 0 {
		return 0, 1
	}
	if dir >= 0 {
		dir = 1
	} else {
		dir = -1
	}
	if r.PingPong {
		if n == 1 {
			return 0, 1
		}
		next := index + dir
		if next >= n {
			return n - 2, -1
		}
		if next < 0 {
			return 1, 1
		}
		return next, dir
	}
	next := index + 1
	if r.Looped {
		return next % n, 1
	}
	if next >= n {
		return n - 1, 1
	}
	return next, 1
}

// AdvanceIndex moves past a visited waypoint (forward-only; prefer Advance for PingPong).
func (r *Route) AdvanceIndex(index int) int {
	idx, _ := r.Advance(index, 1)
	return idx
}

// CumLengthYd returns path length from waypoint 0 to index along unique points.
func (r *Route) CumLengthYd(index int) float64 {
	n := r.UniqueCount()
	if n == 0 || index <= 0 {
		return 0
	}
	if index >= n {
		index = n - 1
	}
	sum := 0.0
	for i := 1; i <= index; i++ {
		a := r.Waypoints[i-1]
		b := r.Waypoints[i]
		sum += math.Hypot(b.X-a.X, b.Y-a.Y)
	}
	return sum
}

// TotalLengthYd is one-way unique-path length (plus closing chord only for Looped).
func (r *Route) TotalLengthYd() float64 {
	n := r.UniqueCount()
	if n < 2 {
		return 0
	}
	sum := r.CumLengthYd(n - 1)
	if r.Looped && !r.PingPong {
		a := r.Waypoints[n-1]
		b := r.Waypoints[0]
		sum += math.Hypot(b.X-a.X, b.Y-a.Y)
	}
	return sum
}

// ResumeWaypoint picks a nearby waypoint and travel direction after leaving the route.
// Prefers points near the unit; for PingPong chooses dir toward the longer remaining arm
// (so a terminus becomes the new start when that is the sensible continuation).
func (r *Route) ResumeWaypoint(x, y float64) (index, dir int) {
	n := r.UniqueCount()
	if n == 0 {
		return 0, 1
	}
	total := r.TotalLengthYd()
	best := 0
	bestScore := math.MaxFloat64
	for i := 0; i < n; i++ {
		wp := r.Waypoints[i]
		dist := math.Hypot(wp.X-x, wp.Y-y)
		remaining := total - r.CumLengthYd(i)
		if remaining < 0 {
			remaining = 0
		}
		score := dist + 0.25*remaining
		if score < bestScore {
			bestScore = score
			best = i
		}
	}
	dir = 1
	if r.PingPong {
		forward := n - 1 - best
		reverse := best
		if reverse > forward || best == n-1 {
			dir = -1
		}
		if best == 0 {
			dir = 1
		}
	}
	return best, dir
}

// ResumeWaypointIndex picks a waypoint near the unit and nearer the route end.
func (r *Route) ResumeWaypointIndex(x, y float64) int {
	idx, _ := r.ResumeWaypoint(x, y)
	return idx
}

// BearingDegToWaypoint is compass bearing from (x,y) to wp.
func BearingDegToWaypoint(x, y float64, wp Waypoint) float64 {
	dx := wp.X - x
	dy := wp.Y - y
	brg := math.Atan2(dx, dy) * 180 / math.Pi
	if brg < 0 {
		brg += 360
	}
	return brg
}

// RangeYdToWaypoint is horizontal yards from (x,y) to wp.
func RangeYdToWaypoint(x, y float64, wp Waypoint) float64 {
	return math.Hypot(wp.X-x, wp.Y-y)
}

// InterruptRoute marks that the unit left its route and must reselect on return.
func InterruptRoute(e *Entity) {
	if e == nil || e.RouteID == "" {
		return
	}
	e.RouteNeedResume = true
}

// NormalizeRouteDir returns +1 or -1.
func NormalizeRouteDir(dir int) int {
	if dir < 0 {
		return -1
	}
	return 1
}
