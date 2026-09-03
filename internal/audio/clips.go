package audio

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"strings"

	"github.com/bubnov-mikhail/ssn688/internal/i18n"
)

//go:embed voices
var voiceFS embed.FS

// ClipID is a language→path map for officer voice lines (TranslatedAudio).
type ClipID = i18n.TranslatedAudio

var (
	ClipCaptHoldSimulation     = i18n.A("capt/hold_simulation")
	ClipCaptSaveComplete       = i18n.A("capt/save_complete")
	ClipCaptOwnshipHit         = i18n.A("capt/ownship_hit")
	ClipCaptCriticalDamage     = i18n.A("capt/critical_damage")
	ClipCaptOwnshipLost        = i18n.A("capt/ownship_lost")
	ClipCaptCommMessage        = i18n.A("capt/comm_message")
	ClipCaptCommTrafficWaiting = i18n.A("capt/comm_traffic_waiting")

	ClipSonarPassiveOn           = i18n.A("sonar/passive_on")
	ClipSonarPassiveOff          = i18n.A("sonar/passive_off")
	ClipSonarActiveStandby       = i18n.A("sonar/active_standby")
	ClipSonarActiveOnline        = i18n.A("sonar/active_online")
	ClipSonarEnemyPing           = i18n.A("sonar/enemy_ping")
	ClipSonarDeployTowed         = i18n.A("sonar/deploy_towed")
	ClipSonarTowedHeld           = i18n.A("sonar/towed_held")
	ClipSonarRetractTowed        = i18n.A("sonar/retract_towed")
	ClipSonarBTLaunch            = i18n.A("sonar/bt_launch")
	ClipSonarLayerSurveyComplete = i18n.A("sonar/layer_survey_complete")
	ClipSonarContactClassified   = i18n.A("sonar/contact_classified")

	ClipWepsImpactConfirmed       = i18n.A("weps/impact_confirmed")
	ClipWepsTorpedoInWater        = i18n.A("weps/torpedo_in_water")
	ClipWepsTorpedoHeadingOwnship = i18n.A("weps/torpedo_heading_ownship")
	ClipWepsOuterDoorClosed       = i18n.A("weps/outer_door_closed")
	ClipWepsRunDepthSet           = i18n.A("weps/run_depth_set")
	ClipWepsSpeedHigh             = i18n.A("weps/speed_high")
	ClipWepsSpeedLow              = i18n.A("weps/speed_low")
	ClipWepsSeekerOn              = i18n.A("weps/seeker_on")
	ClipWepsSeekerOff             = i18n.A("weps/seeker_off")
	ClipWepsWireCut               = i18n.A("weps/wire_cut")

	ClipDiveComeLeft     = i18n.A("dive/come_left")
	ClipDiveComeRight    = i18n.A("dive/come_right")
	ClipDiveMakeDepth    = i18n.A("dive/make_depth")
	ClipDiveHoldDepth    = i18n.A("dive/hold_depth")
	ClipDiveUnableDeeper = i18n.A("dive/unable_deeper")

	ClipNavSpeedHalf   = i18n.A("nav/speed_half")
	ClipNavSpeedNormal = i18n.A("nav/speed_normal")
	ClipNavSpeedDouble = i18n.A("nav/speed_double")
	ClipNavSpeedQuad   = i18n.A("nav/speed_quad")
	ClipNavSpeedEight  = i18n.A("nav/speed_eight")
)

func clipPath(id ClipID) string {
	return id.GetWav(i18n.LangEN)
}

func clipCompartment(id ClipID) Compartment {
	p := strings.Split(clipPath(id), "/")
	if len(p) == 0 {
		return CompCaptain
	}
	switch p[0] {
	case "sonar":
		return CompSonar
	case "weps":
		return CompWeps
	case "dive":
		return CompDive
	case "nav":
		return CompNav
	default:
		return CompCaptain
	}
}

func TubeClip(prefix string, tube int) ClipID {
	return tubeClip(prefix, tube)
}

func tubeClip(prefix string, tube int) ClipID {
	if tube < 1 {
		tube = 1
	}
	if tube > 4 {
		tube = 4
	}
	return i18n.A(fmt.Sprintf("weps/%s_%d", prefix, tube))
}

// LoadVoiceClips decodes embedded WAV assets into mono PCM at the target sample rate.
// Keys are clip paths: "capt/comm_message" (EN) or "ru/capt/comm_message" (RU).
func LoadVoiceClips(sampleRate int) (map[string][]byte, error) {
	clips := make(map[string][]byte)
	err := fs.WalkDir(voiceFS, "voices", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".wav") {
			return nil
		}
		data, err := voiceFS.ReadFile(p)
		if err != nil {
			return err
		}
		pcm, err := decodeWAVMono(data, sampleRate)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		id := strings.TrimSuffix(strings.TrimPrefix(p, "voices/"), ".wav")
		clips[id] = pcm
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(clips) == 0 {
		return nil, fmt.Errorf("no voice clips found")
	}
	return clips, nil
}

func decodeWAVMono(data []byte, sampleRate int) ([]byte, error) {
	stream, err := wavDecoder(sampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	pcm, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	// Ebiten WAV decoder outputs stereo interleaved int16.
	mono := make([]byte, 0, len(pcm)/2)
	for i := 0; i+3 < len(pcm); i += 4 {
		mono = append(mono, pcm[i], pcm[i+1])
	}
	return mono, nil
}

// wavDecoder is set in voice.go to avoid circular imports in tests.
var wavDecoder = func(sampleRate int, r io.Reader) (io.Reader, error) {
	return nil, fmt.Errorf("wav decoder not initialized")
}

func MustLoadVoiceClips(sampleRate int) map[string][]byte {
	clips, err := LoadVoiceClips(sampleRate)
	if err != nil {
		log.Fatalf("voice clips: %v", err)
	}
	return clips
}
