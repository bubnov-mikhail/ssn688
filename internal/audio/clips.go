package audio

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"log"
	"path"
	"strings"
)

//go:embed voices/*/*.wav
var voiceFS embed.FS

// ClipID identifies a pre-recorded officer voice line.
type ClipID string

const (
	ClipCaptMissionBrief   ClipID = "capt/mission_brief"
	ClipCaptHoldSimulation ClipID = "capt/hold_simulation"
	ClipCaptSaveComplete   ClipID = "capt/save_complete"

	ClipSonarPassiveOn           ClipID = "sonar/passive_on"
	ClipSonarPassiveOff          ClipID = "sonar/passive_off"
	ClipSonarActiveStandby       ClipID = "sonar/active_standby"
	ClipSonarActiveOnline        ClipID = "sonar/active_online"
	ClipSonarActivePing          ClipID = "sonar/active_ping"
	ClipSonarEnemyPing           ClipID = "sonar/enemy_ping"
	ClipSonarDeployTowed         ClipID = "sonar/deploy_towed"
	ClipSonarTowedHeld           ClipID = "sonar/towed_held"
	ClipSonarRetractTowed        ClipID = "sonar/retract_towed"
	ClipSonarBTLaunch            ClipID = "sonar/bt_launch"
	ClipSonarLayerSurveyComplete ClipID = "sonar/layer_survey_complete"
	ClipSonarContactClassified   ClipID = "sonar/contact_classified"
	ClipSonarEnableActiveFirst   ClipID = "sonar/enable_active_first"

	ClipWepsImpactConfirmed ClipID = "weps/impact_confirmed"
	ClipWepsOuterDoorClosed ClipID = "weps/outer_door_closed"
	ClipWepsGyroSet         ClipID = "weps/gyro_set"
	ClipWepsRunDepthSet     ClipID = "weps/run_depth_set"
	ClipWepsSpeedHigh       ClipID = "weps/speed_high"
	ClipWepsSpeedLow        ClipID = "weps/speed_low"
	ClipWepsSeekerOn        ClipID = "weps/seeker_on"
	ClipWepsSeekerOff       ClipID = "weps/seeker_off"
	ClipWepsWireCut         ClipID = "weps/wire_cut"

	ClipDiveComeLeft      ClipID = "dive/come_left"
	ClipDiveComeRight     ClipID = "dive/come_right"
	ClipDiveMakeDepth     ClipID = "dive/make_depth"
	ClipDiveHoldDepth     ClipID = "dive/hold_depth"
	ClipDiveUnableDeeper  ClipID = "dive/unable_deeper"

	ClipNavSpeedHalf   ClipID = "nav/speed_half"
	ClipNavSpeedNormal ClipID = "nav/speed_normal"
	ClipNavSpeedDouble ClipID = "nav/speed_double"
	ClipNavSpeedQuad   ClipID = "nav/speed_quad"
	ClipNavSpeedEight  ClipID = "nav/speed_eight"
)

func clipCompartment(id ClipID) Compartment {
	p := strings.Split(string(id), "/")
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
	return ClipID(fmt.Sprintf("weps/%s_%d", prefix, tube))
}

// LoadVoiceClips decodes embedded WAV assets into mono PCM at the target sample rate.
func LoadVoiceClips(sampleRate int) (map[ClipID][]byte, error) {
	clips := make(map[ClipID][]byte)
	entries, err := voiceFS.ReadDir("voices")
	if err != nil {
		return nil, err
	}
	for _, dept := range entries {
		if !dept.IsDir() {
			continue
		}
		sub, err := voiceFS.ReadDir(path.Join("voices", dept.Name()))
		if err != nil {
			return nil, err
		}
		for _, f := range sub {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".wav") {
				continue
			}
			rel := path.Join("voices", dept.Name(), f.Name())
			data, err := voiceFS.ReadFile(rel)
			if err != nil {
				return nil, err
			}
			pcm, err := decodeWAVMono(data, sampleRate)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", rel, err)
			}
			id := ClipID(dept.Name() + "/" + strings.TrimSuffix(f.Name(), ".wav"))
			clips[id] = pcm
		}
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

func MustLoadVoiceClips(sampleRate int) map[ClipID][]byte {
	clips, err := LoadVoiceClips(sampleRate)
	if err != nil {
		log.Fatalf("voice clips: %v", err)
	}
	return clips
}
