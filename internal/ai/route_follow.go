package ai

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

// followAssignedRoute steers toward the current waypoint. Returns false if no route.
// On resume after interrupt, selects the waypoint nearer the unit and nearer route end.
// PingPong routes reverse at either terminus instead of cutting a closing chord.
func followAssignedRoute(e *world.Entity, routes []*world.Route, state string, speedKts float64) bool {
	if e == nil || e.RouteID == "" {
		return false
	}
	r := world.FindRoute(routes, e.RouteID)
	if r == nil || r.UniqueCount() == 0 {
		return false
	}
	if e.RouteNeedResume {
		e.RouteWP, e.RouteDir = r.ResumeWaypoint(e.X, e.Y)
		e.RouteNeedResume = false
	}
	e.RouteDir = world.NormalizeRouteDir(e.RouteDir)
	wp, ok := r.Target(e.RouteWP)
	if !ok {
		return false
	}
	if world.RangeYdToWaypoint(e.X, e.Y, wp) <= world.WaypointVisitYd {
		e.RouteWP, e.RouteDir = r.Advance(e.RouteWP, e.RouteDir)
		wp, ok = r.Target(e.RouteWP)
		if !ok {
			return false
		}
		// Still inside visit radius of the new target (short legs) — keep advancing once more.
		if world.RangeYdToWaypoint(e.X, e.Y, wp) <= world.WaypointVisitYd && r.UniqueCount() > 2 {
			e.RouteWP, e.RouteDir = r.Advance(e.RouteWP, e.RouteDir)
			wp, ok = r.Target(e.RouteWP)
			if !ok {
				return false
			}
		}
	}
	if !e.Damage.Destroyed(world.SysSteering) {
		e.OrderedHead = world.BearingDegToWaypoint(e.X, e.Y, wp)
	}
	maxSpd := e.MaxSpeedKts()
	if speedKts > maxSpd {
		speedKts = maxSpd
	}
	e.OrderedSpeed = speedKts
	e.AIState = state
	return true
}

func markRouteInterrupted(e *world.Entity) {
	world.InterruptRoute(e)
}

func routeCruiseSpeed(e *world.Entity) float64 {
	switch e.SignatureID {
	case "tanker":
		return 9
	case "fishing":
		return 7
	case "grisha", "udaloy", "kresta2", "krivak", "gorshkov", "spruance":
		return math.Min(14, e.MaxSpeedKts())
	case "foxtrot":
		return 5
	case "victor_iii", "yasen_m", "los_angeles":
		return 8
	default:
		if e.Kind == world.KindSubmarine {
			return 6
		}
		return 11
	}
}
