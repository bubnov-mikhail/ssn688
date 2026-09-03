package weapons

// Platform loadouts — Soviet/Russian ASW classes used by AI surface/sub units.

func SurfaceHasRastrub(signatureID string) bool {
	switch signatureID {
	case "grisha":
		return false // RBU + tubes only
	case "exercise_hulk":
		return false // practice target — no standoff ASW
	case "udaloy", "krivak", "kresta2":
		return true // Metel / Rastrub
	case "gorshkov":
		return true // Otvet PLUR from UKSK (same splash mechanics)
	case "spruance":
		return true // ASROC (same splash mechanics)
	default:
		// Unknown combatant: assume Metel/Rastrub-capable ASW escort.
		return true
	}
}

// SurfaceASWRocketLabel is the player-facing name for UKSK/Metel/ASROC rocket ASW.
func SurfaceASWRocketLabel(signatureID string) string {
	switch signatureID {
	case "gorshkov":
		return "Otvet"
	case "spruance":
		return "ASROC"
	default:
		return "Rastrub"
	}
}

func SurfaceHasRBU(signatureID string) bool {
	return signatureID == "grisha"
}

func RastrubMagazineFor(signatureID string) int {
	switch signatureID {
	case "kresta2":
		return 12
	case "udaloy", "krivak":
		return RastrubMagazineDefault
	case "gorshkov":
		return 8 // Otvet loadout share of UKSK
	case "spruance":
		return 8 // ASROC cells / reload set in sim
	default:
		return RastrubMagazineDefault
	}
}

func ShipTubeMagazineFor(signatureID string) int {
	switch signatureID {
	case "grisha":
		return 4
	case "kresta2":
		return 8
	case "gorshkov":
		return 8 // Paket-NK MTT cells
	case "spruance":
		return 6 // Mk32 SVTT / Nulka-era tubes stand-in
	case "exercise_hulk":
		return 0
	default:
		return ShipTubeMagazineDefault
	}
}

func RBUMagazineFor(signatureID string) int {
	if signatureID == "grisha" {
		return 10
	}
	return 0
}

func SAMMagazineFor(signatureID string) int {
	switch signatureID {
	case "grisha":
		return 4
	case "kresta2":
		return 12
	case "udaloy", "krivak":
		return SAMMagazineDefault
	case "gorshkov":
		return 32 // Poliment-Redut VLS
	case "spruance":
		return 24 // NATO Sea Sparrow / VLS era stand-in
	default:
		return SAMMagazineDefault
	}
}

func CIWSMagazineFor(signatureID string) int {
	switch signatureID {
	case "grisha":
		return 6
	case "kresta2":
		return 14
	case "gorshkov":
		return 16 // Palma / Pantsir-M class PD
	case "spruance":
		return 12 // Phalanx bursts
	default:
		return CIWSBurstDefault
	}
}

// AllySubHarpoonMagazine is the standard Sub-Harpoon load for Los Angeles-class allies.
const AllySubHarpoonMagazine = 8

// AllyHarpoonMagazineFor returns default Sub-Harpoon stowage by signature.
func AllyHarpoonMagazineFor(signatureID string) int {
	switch signatureID {
	case "los_angeles":
		return AllySubHarpoonMagazine
	default:
		return 0
	}
}

func EnemySubMagazineFor(signatureID string) int {
	switch signatureID {
	case "victor_iii":
		return 18
	case "yasen_m":
		return 24
	case "los_angeles":
		return 22
	case "foxtrot":
		return 10
	case "kilo":
		return EnemySubMagazine
	default:
		return EnemySubMagazine
	}
}

// HostileTorpedoCruiseKts — heavy fish speed by launcher class.
func HostileTorpedoCruiseKts(signatureID string) float64 {
	switch signatureID {
	case "victor_iii", "yasen_m", "los_angeles":
		return 55
	case "foxtrot":
		return 40
	default:
		return 48 // Kilo / unknown
	}
}

// LightweightTorpedoSignature for ship-tube / Rastrub splash fish.
func LightweightTorpedoSignature(launcherSig string) string {
	switch launcherSig {
	case "grisha":
		return "set40"
	case "spruance":
		return "mk46"
	default:
		return "umgt1" // includes Paket MTT stand-in for Gorshkov
	}
}
