package world

import "testing"

func TestPrimaryObjectivesComplete(t *testing.T) {
	sc := NewTrainingScenario()
	if sc.PrimaryObjectivesComplete() {
		t.Fatal("fresh mission should not have primaries complete")
	}
	for _, e := range sc.Entities {
		switch e.ID {
		case "enemy_foxtrot", "enemy_grisha":
			e.Status = StatusSunk
		}
	}
	sc.NoteIdentified("enemy_grisha")
	sc.CheckObjectives()
	if !sc.PrimaryObjectivesComplete() {
		t.Fatal("primaries should complete after sink + ID")
	}
}
