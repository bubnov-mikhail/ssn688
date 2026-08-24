package weapons

// Tube ordnance types (688-class torpedo tubes).
const (
	OrdnanceMk48           = "Mk48"
	OrdnanceMk48Exercise   = "Mk48 EX"
	OrdnanceHarpoon        = "Harpoon"
	EnemyOrdnanceSSN688Decoy = "SSN688 Decoy"
)

// PlayerHarpoonMagazine — typical Sub-Harpoon loadout order of magnitude.
const PlayerHarpoonMagazine = 8

// AllTubeOrdnance lists selectable reload types.
func AllTubeOrdnance() []string {
	return []string{OrdnanceMk48, OrdnanceMk48Exercise, OrdnanceHarpoon}
}

func normalizeOrdnance(name string) string {
	switch name {
	case OrdnanceMk48Exercise:
		return OrdnanceMk48Exercise
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

func OrdnanceUsesHarpoonMagazine(name string) bool {
	return normalizeOrdnance(name) == OrdnanceHarpoon
}

func OrdnanceIsTorpedo(name string) bool {
	switch normalizeOrdnance(name) {
	case OrdnanceMk48, OrdnanceMk48Exercise:
		return true
	default:
		return false
	}
}
