package world

// PlotMarker is a player-placed annotation on the tactical chart.
type PlotMarker struct {
	ID   string
	X, Y float64 // yards east / north
}

// PlotMarkerSizeYd is the square side length drawn for a user marker (0.5 kyd).
const PlotMarkerSizeYd = 500.0
