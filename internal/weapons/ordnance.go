package weapons

// Tube ordnance types (688-class torpedo tubes).
const (
	OrdnanceMk48    = "Mk48"
	OrdnanceHarpoon = "Harpoon"
)

// PlayerHarpoonMagazine — typical Sub-Harpoon loadout order of magnitude.
const PlayerHarpoonMagazine = 8

// AllTubeOrdnance lists selectable reload types.
func AllTubeOrdnance() []string {
	return []string{OrdnanceMk48, OrdnanceHarpoon}
}

func normalizeOrdnance(name string) string {
	switch name {
	case OrdnanceHarpoon:
		return OrdnanceHarpoon
	default:
		return OrdnanceMk48
	}
}

// NormalizeOrdnance is the exported alias for UI/save callers.
func NormalizeOrdnance(name string) string {
	return normalizeOrdnance(name)
}
