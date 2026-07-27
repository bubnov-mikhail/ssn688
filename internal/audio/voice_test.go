package audio

import "testing"

func TestLoadVoiceClips(t *testing.T) {
	clips, err := LoadVoiceClips(44100)
	if err != nil {
		t.Fatal(err)
	}
	if len(clips) < 30 {
		t.Fatalf("expected at least 30 voice clips, got %d", len(clips))
	}
	for _, id := range []ClipID{
		ClipCaptMissionBrief, ClipSonarPassiveOn, ClipWepsWireCut, ClipDiveMakeDepth, ClipNavSpeedDouble,
	} {
		if _, ok := clips[id]; !ok {
			t.Fatalf("missing clip %s", id)
		}
	}
	if _, ok := clips[TubeClip("torpedo_away", 2)]; !ok {
		t.Fatal("missing tube clip")
	}
}
