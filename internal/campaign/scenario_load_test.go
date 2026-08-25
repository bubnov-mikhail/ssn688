package campaign

import (
	"testing"
)

func TestParseBundledDemoJSON(t *testing.T) {
	ReloadScenarios()
	sc := ScenarioByID(DemoScenarioID)
	if sc == nil {
		t.Fatal("demo scenario missing")
	}
	if !sc.Compatible {
		t.Fatalf("demo should be compatible: %s", sc.IncompatibleReason)
	}
	if sc.Version.String() != "1.1.0" {
		t.Fatalf("version %s", sc.Version)
	}
	if sc.FormatVersion.Major != 2 {
		t.Fatalf("format major %d", sc.FormatVersion.Major)
	}
	if len(sc.Missions[0].Routes) == 0 || len(sc.Missions[0].Routes[0].Waypoints) < 2 {
		t.Fatal("expected waypoint routes")
	}
	if len(sc.CoverData) == 0 {
		t.Fatal("expected cover bytes from data_b64")
	}
	if len(sc.Theaters) != 1 || sc.Theaters[0].Chart == nil || !sc.Theaters[0].Chart.Valid() {
		t.Fatal("expected inline theater bathy")
	}
	if len(sc.Missions[0].CommSchedule) == 0 {
		t.Fatal("comm schedule from events")
	}
}

func TestIncompatibleFormatRejected(t *testing.T) {
	sc := ScenarioDef{
		FormatVersion:  ParseSemVer("0.9.0"),
		Version:        ParseSemVer("1.0.0"),
		MinGameVersion: ParseSemVer("1.0.0"),
	}
	sc.ApplyCompatibility()
	if sc.Compatible {
		t.Fatal("format 0.9 should be incompatible with game format 1")
	}
}

func TestImportRejectsIncompatible(t *testing.T) {
	_, err := ImportScenarioJSON("/nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUseGameDefaultRejected(t *testing.T) {
	raw := []byte(`{
		"format_version": "1.0.0",
		"version": "1.0.0",
		"min_game_version": "1.0.0",
		"id": "no_default",
		"title": "Bad",
		"backstory": "x",
		"theaters": [{"id": "t1", "bathy": {"use_game_default": true}}],
		"missions": [{
			"id": "m1", "title": "M", "description": "d", "theater_id": "t1",
			"routes": [],
			"player": {"id":"player","name":"P","kind":"submarine","side":"player","signature_id":"los_angeles","spawn":"corner"},
			"units": [],
			"objectives": [{"id":"o1","description":"x","target_id":"player"}]
		}]
	}`)
	_, err := ParseScenarioJSON(raw, "test")
	if err == nil {
		t.Fatal("expected use_game_default to be rejected")
	}
}
