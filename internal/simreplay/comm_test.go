package simreplay

import (
	"os"
	"testing"
)

func TestCaptureCommTimelineAttribution(t *testing.T) {
	path := "../../scenarios_generated/taiwan_formosa_watch.json"
	if _, err := os.Stat(path); err != nil {
		t.Skip(err)
	}
	msgs, _, err := CaptureCommTimeline(RecordOptions{
		ScenarioPath: path,
		MissionID:    "tw_attribution",
		Seed:         1,
		MaxSec:       60,
		PlayerIdle:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected comm traffic")
	}
	if msgs[0].ID != "briefing" {
		t.Fatalf("first=%q", msgs[0].ID)
	}
	found := false
	for _, m := range msgs {
		if m.ID == "tasking_attr" {
			found = true
			if m.TimeSec < 24 || m.TimeSec > 26 {
				t.Fatalf("tasking_attr time=%v", m.TimeSec)
			}
		}
	}
	if !found {
		t.Fatal("tasking_attr comm missing")
	}
}

func TestCommLinesFiltersByTime(t *testing.T) {
	msgs := []CommSnap{
		{TimeSec: 0, ID: "briefing", Body: map[string]string{"en": "hi", "ru": "привет"}},
		{TimeSec: 25, ID: "tasking", Body: map[string]string{"en": "go", "ru": "иди"}},
	}
	lines := CommLines(msgs, 0, 10, "en", 200)
	if len(lines) == 0 {
		t.Fatal("expected lines")
	}
	lines25 := CommLines(msgs, 0, 30, "en", 200)
	if len(lines25) <= len(lines) {
		t.Fatalf("expected more lines at t=30: %d vs %d", len(lines25), len(lines))
	}
}
