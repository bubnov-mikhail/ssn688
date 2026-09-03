package world

import "testing"

func TestIsExerciseTarget(t *testing.T) {
	if !IsExerciseTarget(&Entity{ID: "ex_hulk_a", ExerciseTarget: false}) {
		t.Fatal("ex_hulk_ prefix")
	}
	if !IsExerciseTarget(&Entity{ID: "foo", ExerciseTarget: true}) {
		t.Fatal("exercise_target flag")
	}
	if IsExerciseTarget(&Entity{ID: "plan_grisha"}) {
		t.Fatal("combatant not exercise")
	}
}
