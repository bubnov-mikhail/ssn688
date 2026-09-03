package campaign

import (
	"encoding/base64"
	"encoding/binary"
	"math"
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
	if sc.Version.String() != "1.3.1" {
		t.Fatalf("version %s", sc.Version)
	}
	if sc.FormatVersion.Major != 3 {
		t.Fatalf("format major %d", sc.FormatVersion.Major)
	}
	if len(sc.Missions) != 2 {
		t.Fatalf("want 2 missions, got %d", len(sc.Missions))
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
	if len(sc.Missions[0].Events) == 0 {
		t.Fatal("expected mission events")
	}
	rt := DemoRuntime()
	if rt == nil || len(rt.CommSchedule) == 0 {
		t.Fatal("comm schedule from events at instantiate")
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

func TestParseMissionBriefMap(t *testing.T) {
	bathyB64 := base64.StdEncoding.EncodeToString(minimalTestBathyB64())
	mapB64 := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
	raw := []byte(`{
		"format_version": "3.0.0",
		"version": "1.0.0",
		"min_game_version": "1.0.0",
		"id": "brief_map_test",
		"title": {"en":"T","ru":"T"},
		"backstory": {"en":"x","ru":"x"},
		"theaters": [{"id": "t1", "bathy": {"mime":"application/octet-stream","data_b64":"` + bathyB64 + `"}}],
		"missions": [{
			"id": "m1",
			"title": {"en":"M","ru":"M"},
			"description": {"en":"d","ru":"d"},
			"brief_map": {"mime":"image/png","data_b64":"` + mapB64 + `"},
			"theater_id": "t1",
			"routes": [],
			"player": {"id":"player","name":{"en":"P","ru":"P"},"kind":"submarine","side":"player","signature_id":"los_angeles","spawn":"corner"},
			"units": [],
			"objectives": [{"id":"o1","description":{"en":"x","ru":"x"},"target_id":"player"}]
		}]
	}`)
	sc, err := ParseScenarioJSON(raw, "test")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	data, key := MissionBriefMap(&sc.Missions[0])
	if len(data) != len([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}) {
		t.Fatalf("brief_map bytes len %d", len(data))
	}
	if key != "brief_map:m1" {
		t.Fatalf("cache key %q", key)
	}
	warmed := WarmMissionBriefMap(&sc.Missions[0])
	if warmed != "brief_map:m1" {
		t.Fatalf("warm key %q", warmed)
	}
	data, key = MissionBriefMap(&sc.Missions[0])
	if len(data) != 0 {
		t.Fatalf("expected warmed data cleared, got %d bytes", len(data))
	}
	if key != "brief_map:m1" {
		t.Fatalf("warmed cache key %q", key)
	}
}

func TestUseGameDefaultRejected(t *testing.T) {
	raw := []byte(`{
		"format_version": "3.0.0",
		"version": "1.0.0",
		"min_game_version": "1.0.0",
		"id": "no_default",
		"title": {"en":"Bad","ru":"Bad"},
		"backstory": {"en":"x","ru":"x"},
		"theaters": [{"id": "t1", "bathy": {"use_game_default": true}}],
		"missions": [{
			"id": "m1",
			"title": {"en":"M","ru":"M"},
			"description": {"en":"d","ru":"d"},
			"theater_id": "t1",
			"routes": [],
			"player": {"id":"player","name":{"en":"P","ru":"P"},"kind":"submarine","side":"player","signature_id":"los_angeles","spawn":"corner"},
			"units": [],
			"objectives": [{"id":"o1","description":{"en":"x","ru":"x"},"target_id":"player"}]
		}]
	}`)
	_, err := ParseScenarioJSON(raw, "test")
	if err == nil {
		t.Fatal("expected use_game_default to be rejected")
	}
}

func TestParseExerciseTorpedoesPayload(t *testing.T) {
	bathyB64 := base64.StdEncoding.EncodeToString(minimalTestBathyB64())
	raw := []byte(`{
		"format_version": "3.0.0",
		"version": "1.0.0",
		"min_game_version": "1.0.0",
		"id": "ex_payload",
		"title": {"en":"T","ru":"T"},
		"backstory": {"en":"x","ru":"x"},
		"theaters": [{"id": "t1", "bathy": {"mime":"application/octet-stream","data_b64":"` + bathyB64 + `"}}],
		"missions": [{
			"id": "m1",
			"title": {"en":"M","ru":"M"},
			"description": {"en":"d","ru":"d"},
			"theater_id": "t1",
			"routes": [],
			"player": {"id":"player","name":{"en":"P","ru":"P"},"kind":"submarine","side":"player","signature_id":"los_angeles","spawn":"corner"},
			"units": [{
				"id": "ex_hulk_a",
				"name": {"en":"H","ru":"H"},
				"kind": "surface",
				"side": "enemy",
				"signature_id": "exercise_hulk",
				"spawn": "route",
				"exercise_target": true,
				"payload": {"ship_tubes": 0, "exercise_torpedoes": 2}
			}],
			"objectives": [{"id":"o1","description":{"en":"x","ru":"x"},"target_id":"ex_hulk_a"}]
		}]
	}`)
	sc, err := ParseScenarioJSON(raw, "test")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	p := sc.Missions[0].Units[0].Payload
	if p == nil || p.ExerciseTorpedoes == nil || *p.ExerciseTorpedoes != 2 {
		t.Fatalf("exercise_torpedoes: %+v", p)
	}
}

func minimalTestBathyB64() []byte {
	const w, h = 2, 2
	buf := make([]byte, 40+w*h*4)
	copy(buf[0:4], "BATH")
	binary.LittleEndian.PutUint32(buf[4:8], 1)
	binary.LittleEndian.PutUint32(buf[8:12], w)
	binary.LittleEndian.PutUint32(buf[12:16], h)
	binary.LittleEndian.PutUint64(buf[16:24], math.Float64bits(-1000))
	binary.LittleEndian.PutUint64(buf[24:32], math.Float64bits(-1000))
	binary.LittleEndian.PutUint64(buf[32:40], math.Float64bits(250))
	for i := 0; i < w*h; i++ {
		off := 40 + i*4
		binary.LittleEndian.PutUint32(buf[off:off+4], math.Float32bits(2000))
	}
	return buf
}
