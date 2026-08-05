package audio

import (
	"testing"
	"time"
)

func TestLoadVoiceClips(t *testing.T) {
	clips, err := LoadVoiceClips(44100)
	if err != nil {
		t.Fatal(err)
	}
	if len(clips) < 30 {
		t.Fatalf("expected at least 30 voice clips, got %d", len(clips))
	}
	for _, id := range []ClipID{
		ClipCaptMissionBrief, ClipCaptOwnshipHit, ClipCaptCriticalDamage, ClipCaptOwnshipLost,
		ClipSonarPassiveOn, ClipSonarDeployTowed, ClipSonarActiveOnline,
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

func newTestManager() *Manager {
	m := &Manager{
		sampleRate: 44100,
		masterVol:  0.8,
		voiceVol:   0.9,
		fxVol:      0.7,
		clips:      MustLoadVoiceClips(44100),
	}
	return m
}

func TestPlayClipDoesNotDropPendingDifferentLine(t *testing.T) {
	m := newTestManager()
	m.PlayClip(ClipWepsTorpedoInWater, "Torpedo in the water.")
	m.PlayClip(ClipWepsTorpedoHeadingOwnship, "Incomming torpedo!")

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pending == nil || m.pending.id != ClipWepsTorpedoHeadingOwnship {
		t.Fatalf("expected incoming pending, got %#v", m.pending)
	}
	if len(m.voicePlaying) == 0 && len(m.queue) == 0 {
		t.Fatal("torpedo-in-water line was dropped when incoming was queued")
	}
}

func TestFXDoesNotBlockVoiceQueue(t *testing.T) {
	m := newTestManager()
	m.PlayTorpedoLaunch()
	m.PlayClip(ClipDiveMakeDepth, "Make depth 200 feet.")
	m.mu.Lock()
	m.pending.playAt = time.Now().Add(-time.Millisecond)
	m.flushPendingLocked()
	if len(m.voicePlaying) == 0 {
		m.mu.Unlock()
		t.Fatal("voice should start even while launch FX is playing")
	}
	if len(m.fxPlaying) == 0 {
		m.mu.Unlock()
		t.Fatal("expected FX still playing")
	}
	m.mu.Unlock()
}

func TestRoutineClipsDoNotBacklog(t *testing.T) {
	m := newTestManager()
	// Occupy voice channel with a long synthetic buffer.
	m.mu.Lock()
	long := make([]byte, 44100*2*4) // ~4s mono
	m.voicePlaying = []*playerVoice{{data: long, volume: 1}}
	m.activeClipID = ClipDiveMakeDepth
	m.mu.Unlock()

	for i := 0; i < 12; i++ {
		m.PlayClip(ClipDiveComeLeft, "Come left.")
		m.mu.Lock()
		if m.pending != nil {
			m.pending.playAt = time.Now().Add(-time.Millisecond)
			m.flushPendingLocked()
		}
		m.mu.Unlock()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.queue) > maxVoiceQueue {
		t.Fatalf("queue grew unbounded: %d", len(m.queue))
	}
}
