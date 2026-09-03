package world

// RevertMoveIfBlocked restores the previous position when a move ends on
// dry land or water too shallow for the unit's current/planned depth.
func RevertMoveIfBlocked(e *Entity, prevX, prevY float64, bathy *Bathymetry) {
	if e == nil || bathy == nil || !bathy.Valid() {
		return
	}
	depth := NavDepthFt(e)
	if bathy.NavigableFor(e.X, e.Y, e.Kind, depth) {
		return
	}
	if e.Side == SideNeutral {
		// Keep following the stitched route; don't bounce at chart edges.
		return
	}
	e.X, e.Y = prevX, prevY
	if e.SpeedKts > 0 {
		e.SpeedKts *= 0.25
	}
	InterruptRoute(e)
}

func NavDepthFt(e *Entity) float64 {
	if e.OrderedDepth > 0 {
		return e.OrderedDepth
	}
	if e.DepthFt > 0 {
		return e.DepthFt
	}
	if e.Kind == KindSubmarine {
		return 160
	}
	return 0
}
