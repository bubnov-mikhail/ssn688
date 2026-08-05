package world

// RestrictedZone marks a future no-go area. Player entry triggers DEFCON 3.
type RestrictedZone struct {
	ID       string
	CenterX  float64
	CenterY  float64
	RadiusYd float64
}

// PlayerInside reports whether the player is inside the zone (horizontal).
func (z RestrictedZone) PlayerInside(player *Entity) bool {
	if z.RadiusYd <= 0 || player == nil {
		return false
	}
	dx := player.X - z.CenterX
	dy := player.Y - z.CenterY
	return dx*dx+dy*dy <= z.RadiusYd*z.RadiusYd
}
