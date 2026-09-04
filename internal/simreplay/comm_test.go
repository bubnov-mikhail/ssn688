package simreplay

import (
	"os"
	"testing"
)

func TestCaptureCommTimelineContested(t *testing.T) {
	path := "../../scenarios_generated/taiwan_formosa_watch.json"
	if _, err := os.Stat(path); err != nil {
		t.Skip(err)
	}
	msgs, _, err := CaptureCommTimeline(RecordOptions{
		ScenarioPath: path,
		MissionID:    "tw_contested",
		Seed:         1,
		MaxSec:       60,
		PlayerIdle:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range msgs {
		if m.ID == "tasking_m4" {
			found = true
			if m.TimeSec < 24 || m.TimeSec > 26 {
				t.Fatalf("tasking_m4 time=%v", m.TimeSec)
			}
		}
	}
	if !found {
		t.Fatalf("tasking_m4 missing from %d msgs (deep start must still deliver COMM)", len(msgs))
	}
}

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

func TestCommPlayheadScrubBackRemovesMessages(t *testing.T) {
	ph := NewCommPlayhead([]CommSnap{
		{TimeSec: 0, ID: "briefing", Body: map[string]string{"en": "a", "ru": "а"}},
		{TimeSec: 25, ID: "tasking", Body: map[string]string{"en": "b", "ru": "б"}},
		{TimeSec: 90, ID: "cue", Body: map[string]string{"en": "c", "ru": "в"}},
	})
	ph.Sync(100)
	if len(ph.Inbox()) != 3 {
		t.Fatalf("inbox=%d want 3", len(ph.Inbox()))
	}
	ph.Sync(30)
	got := ph.Inbox()
	if len(got) != 2 {
		t.Fatalf("after scrub back inbox=%d want 2", len(got))
	}
	if got[1].ID != "tasking" {
		t.Fatalf("last=%q", got[1].ID)
	}
	ph.Sync(10)
	if len(ph.Inbox()) != 1 || ph.Inbox()[0].ID != "briefing" {
		t.Fatalf("inbox=%v", ph.Inbox())
	}
	ph.Sync(0)
	if len(ph.Inbox()) != 1 {
		t.Fatalf("briefing at t=0: %d", len(ph.Inbox()))
	}
}
