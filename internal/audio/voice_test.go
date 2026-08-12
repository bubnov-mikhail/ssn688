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
		ClipCaptHoldSimulation, ClipCaptOwnshipHit, ClipCaptCriticalDamage, ClipCaptOwnshipLost, ClipCaptCommMessage,
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

func TestLoadFXClipsPropeller(t *testing.T) {
	fx, err := LoadFXClips(44100)
	if err != nil {
		t.Fatal(err)
	}
	pcm, ok := fx[FXPropellerHydrophone]
	if !ok {
		t.Fatal("missing propeller hydrophone FX")
	}
	// ~14s mono int16 @ 44.1kHz.
	if len(pcm) < 44100*2*5 {
		t.Fatalf("propeller loop too short: %d bytes", len(pcm))
	}
	sub, ok := fx[FXPropellerSubmarine]
	if !ok {
		t.Fatal("missing submarine propeller FX")
	}
	if len(sub) < 44100*2*1 {
		t.Fatalf("submarine propeller loop too short: %d bytes", len(sub))
	}
	bow, ok := fx[FXBowWash]
	if !ok {
		t.Fatal("missing bow wash FX")
	}
	if len(bow) < 44100*2*5 {
		t.Fatalf("bow wash loop too short: %d bytes", len(bow))
	}
	amb, ok := fx[FXPassiveAmbient]
	if !ok {
		t.Fatal("missing passive ambient FX")
	}
	if len(amb) < 44100*2*5 {
		t.Fatalf("passive ambient loop too short: %d bytes", len(amb))
	}
	run, ok := fx[FXTorpedoRun]
	if !ok {
		t.Fatal("missing torpedo run FX")
	}
	if len(run) < 44100*2*2 {
		t.Fatalf("torpedo run loop too short: %d bytes", len(run))
	}
	for _, id := range []FXID{
		FXPropellerFishing, FXPropellerMerchant, FXPropellerTanker,
		FXTorpedoLaunch, FXUnderwaterExplosion,
		FXTubeDoorOpen, FXTubeDoorClose, FXMastHydraulic,
	} {
		if pcm, ok := fx[id]; !ok || len(pcm) < 44100 {
			t.Fatalf("missing or short FX %s", id)
		}
	}
}

func TestSetLoopingFXIndependentTracks(t *testing.T) {
	m := newTestManager()
	m.SetLoopingFX(FXPropellerHydrophone, 0.8, 1)
	m.SetLoopingFX(FXBowWash, 0.25, 1)
	m.mu.Lock()
	if len(m.loops) != 2 {
		m.mu.Unlock()
		t.Fatalf("want 2 loop tracks, got %d", len(m.loops))
	}
	m.mu.Unlock()
	m.SetLoopingFX(FXBowWash, 0, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.loops[FXBowWash]; ok {
		t.Fatal("bow wash should stop when gain is 0")
	}
	if _, ok := m.loops[FXPropellerHydrophone]; !ok {
		t.Fatal("propeller should keep looping after bow wash stop")
	}
}

func newTestManager() *Manager {
	m := &Manager{
		sampleRate: 44100,
		masterVol:  0.8,
		voiceVol:   0.9,
		fxVol:      0.7,
		clips:      MustLoadVoiceClips(44100),
		fxClips:    MustLoadFXClips(44100),
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
