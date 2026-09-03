package world

import "strings"

// IsExerciseTarget reports practice hulks that must not employ warshot ASW.
func IsExerciseTarget(e *Entity) bool {
	if e == nil {
		return false
	}
	if e.ExerciseTarget {
		return true
	}
	return strings.HasPrefix(e.ID, "ex_hulk_")
}
