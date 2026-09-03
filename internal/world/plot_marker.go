package world

import "github.com/bubnov-mikhail/ssn688/internal/i18n"

// PlotMarker is a player-placed annotation on the tactical chart.
type PlotMarker struct {
	ID    string
	Label i18n.TranslatedText // optional; shown instead of ID when set
	X, Y  float64             // yards east / north
}

// DisplayLabel returns the chart label for lang (falls back to ID).
func (m PlotMarker) DisplayLabel(lang string) string {
	if t := m.Label.GetText(lang); t != "" {
		return t
	}
	return m.ID
}

// PlotMarkerSizeYd is the square side length drawn for a user marker (0.5 kyd).
const PlotMarkerSizeYd = 500.0
