package world

import "testing"

func TestMissionReportAllowedAfterEvent(t *testing.T) {
	sc := &Scenario{
		EndAfterEventID: "provocation",
		FiredEventIDs:   map[string]bool{},
	}
	if sc.MissionReportAllowed() {
		t.Fatal("report should be blocked before provocation event")
	}
	sc.FiredEventIDs["dawn_sinking"] = true
	if sc.MissionReportAllowed() {
		t.Fatal("report should still wait for provocation")
	}
	sc.FiredEventIDs["provocation"] = true
	if !sc.MissionReportAllowed() {
		t.Fatal("report should be allowed after provocation")
	}
}
