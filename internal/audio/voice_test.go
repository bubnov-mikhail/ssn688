package audio

import (
	"testing"

	"github.com/ssn688/sim/internal/i18n"
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
		ClipCaptCommTrafficWaiting,
		ClipSonarPassiveOn, ClipSonarDeployTowed, ClipSonarActiveOnline,
		ClipWepsWireCut, ClipWepsTorpedoInWater, ClipWepsTorpedoHeadingOwnship,
		ClipDiveMakeDepth, ClipDiveUnableDeeper, ClipDiveHoldDepth, ClipNavSpeedDouble,
	} {
		if _, ok := clips[clipPath(id)]; !ok {
			t.Fatalf("missing clip %s", clipPath(id))
		}
	}
	if _, ok := clips[clipPath(TubeClip("torpedo_away", 2))]; !ok {
		t.Fatal("missing tube clip")
	}
}

func TestPlayClipPendingUsesPath(t *testing.T) {
	m := NewManager(44100)
	m.PlayClip(ClipWepsTorpedoHeadingOwnship, "Incomming torpedo!")
	if m.pending == nil || m.pending.id != clipPath(ClipWepsTorpedoHeadingOwnship) {
		t.Fatalf("expected incoming pending, got %#v", m.pending)
	}

	i18n.SetLang(i18n.LangRU)
	defer i18n.SetLang(i18n.LangEN)
	m.PlayClip(ClipCaptCommMessage, "")
	if m.pending == nil || m.pending.id != "ru/capt/comm_message" {
		t.Fatalf("expected RU WAV path, got %#v", m.pending)
	}

	// FX-only clip has no RU asset → EN fallback.
	m.PlayClip(ClipSonarEnemyPing, "ping")
	if m.pending == nil || m.pending.id != "sonar/enemy_ping" {
		t.Fatalf("expected EN FX fallback, got %#v", m.pending)
	}
}

func TestGetWavRUPath(t *testing.T) {
	if ClipCaptCommMessage.GetWav(i18n.LangEN) != "capt/comm_message" {
		t.Fatal(ClipCaptCommMessage.GetWav(i18n.LangEN))
	}
	if ClipCaptCommMessage.GetWav(i18n.LangRU) != "ru/capt/comm_message" {
		t.Fatal(ClipCaptCommMessage.GetWav(i18n.LangRU))
	}
}

func TestLoadVoiceClipsIncludesRU(t *testing.T) {
	clips, err := LoadVoiceClips(44100)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := clips["ru/capt/comm_message"]; !ok {
		t.Fatal("missing embedded RU clip ru/capt/comm_message")
	}
	if _, ok := clips["ru/sonar/enemy_ping"]; ok {
		t.Fatal("enemy_ping must not have a RU TTS copy")
	}
}
