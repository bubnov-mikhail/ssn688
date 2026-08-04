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
		ClipCaptMissionBrief, ClipSonarPassiveOn, ClipSonarDeployTowed, ClipSonarActiveOnline,
		ClipWepsWireCut, ClipWepsTorpedoInWater, ClipWepsTorpedoHeadingOwnship,
		ClipDiveMakeDepth, ClipDiveUnableDeeper, ClipDiveHoldDepth, ClipNavSpeedDouble,
	} {
		if _, ok := clips[id]; !ok {
			t.Fatalf("missing clip %s", id)
		}
	}
	if _, ok := clips[TubeClip("torpedo_away", 2)]; !ok {
		t.Fatal("missing tube clip")
	}
}

func TestPlayClipDoesNotDropPendingDifferentLine(t *testing.T) {
	m := NewManager(44100)
	m.PlayClip(ClipWepsTorpedoInWater, "Torpedo in the water.")
	m.PlayClip(ClipWepsTorpedoHeadingOwnship, "Incomming torpedo!")

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pending == nil || m.pending.id != ClipWepsTorpedoHeadingOwnship {
		t.Fatalf("expected incoming pending, got %#v", m.pending)
	}
	if len(m.playing) == 0 && len(m.queue) == 0 {
		t.Fatal("torpedo-in-water line was dropped when incoming was queued")
	}
	if len(m.queue) > 0 && m.queue[0].clipID != ClipWepsTorpedoInWater && len(m.playing) == 0 {
		t.Fatalf("unexpected queue %#v", m.queue)
	}
}
