package simreplay

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestTerminalProgress(t *testing.T) {
	var buf bytes.Buffer
	p := &TerminalProgress{Label: "test", Out: &buf, BarWide: 10}
	p.Update(0, 100)
	p.Update(50, 100)
	p.Update(100, 100)
	p.Finish()
	out := buf.String()
	if !strings.Contains(out, "50%") {
		t.Fatalf("expected 50%% in output: %q", out)
	}
	if !strings.Contains(out, "100%") {
		t.Fatalf("expected 100%% in output: %q", out)
	}
}

func TestRecordMissionReportsProgress(t *testing.T) {
	path := "../../scenarios_generated/taiwan_formosa_watch.json"
	if _, err := os.Stat(path); err != nil {
		t.Skip("taiwan scenario not present")
	}
	var calls int
	_, err := RecordMission(RecordOptions{
		ScenarioPath: path,
		MissionID:    "tw_attribution",
		Seed:         1,
		MaxSec:       5,
		SampleSec:    1,
		PlayerIdle:   true,
		Progress: func(gameTime, maxSec float64) {
			calls++
			if gameTime < 0 || maxSec != 5 {
				t.Fatalf("bad progress args: %.2f %.2f", gameTime, maxSec)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls < 2 {
		t.Fatalf("expected multiple progress callbacks, got %d", calls)
	}
}
