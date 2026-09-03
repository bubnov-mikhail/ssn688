package simreplay

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	rep := &Replay{
		FormatVersion: FormatVersion,
		ScenarioID:    "test",
		MissionID:     "m1",
		MissionTitle:  "Test",
		DurationSec:   10,
		SampleSec:     1,
		Frames: []Frame{{
			Time: 0,
			Units: []UnitSnap{{
				ID: "player", Side: "PLAYER", Status: "ACTIVE", X: 100, Y: 200, Alive: true,
			}},
			Weapons: []WeaponSnap{{
				Kind: WeaponTorpedo, Label: "MK48", Side: "PLAYER", X: 1, Y: 2, Alive: true,
			}},
		}},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "r.replay.json")
	if err := Save(path, rep); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Frames) != 1 || len(got.Frames[0].Weapons) != 1 {
		t.Fatalf("got %#v", got.Frames)
	}
}

func TestRecordMissionShort(t *testing.T) {
	path := filepath.Join("..", "..", "scenarios_generated", "taiwan_formosa_watch.json")
	if _, err := os.Stat(path); err != nil {
		t.Skip("taiwan scenario not present")
	}
	rep, err := RecordMission(RecordOptions{
		ScenarioPath: path,
		MissionID:    "tw_attribution",
		Seed:         1,
		MaxSec:       30,
		SampleSec:    5,
		PlayerIdle:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Frames) < 2 {
		t.Fatalf("expected frames, got %d", len(rep.Frames))
	}
	u := rep.Frames[0].Units[0]
	if u.X == 0 && u.Y == 0 {
		t.Fatalf("expected non-zero spawn position, got %#v", u)
	}
}
