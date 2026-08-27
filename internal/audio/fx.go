package audio

import (
	"embed"
	"fmt"
	"log"
	"strings"
)

//go:embed fx/*.wav
var fxFS embed.FS

// FXID identifies an embedded environmental / listen FX sample (not officer voice).
type FXID string

const (
	FXPropellerHydrophone FXID = "propeller_hydrophone" // surface combatants / default
	FXPropellerSubmarine  FXID = "propeller_submarine"
	FXPropellerHelm       FXID = "propeller_helm" // ownship feel on HELM (sub+combatant mix)
	FXPropellerFishing    FXID = "propeller_fishing"
	FXPropellerMerchant   FXID = "propeller_merchant"
	FXPropellerTanker     FXID = "propeller_tanker"
	FXBowWash             FXID = "bow_wash"
	FXPassiveAmbient      FXID = "passive_ambient"
	FXTorpedoRun          FXID = "torpedo_run"
	FXTorpedoLaunch       FXID = "torpedo_launch"
	FXUnderwaterExplosion FXID = "underwater_explosion"
	FXTubeDoorOpen        FXID = "tube_door_open"
	FXTubeDoorClose       FXID = "tube_door_close"
	FXMastHydraulic       FXID = "mast_hydraulic"
)

// LoadFXClips decodes embedded FX WAVs into mono PCM at the target sample rate.
func LoadFXClips(sampleRate int) (map[FXID][]byte, error) {
	out := make(map[FXID][]byte)
	entries, err := fxFS.ReadDir("fx")
	if err != nil {
		return nil, err
	}
	for _, f := range entries {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".wav") {
			continue
		}
		raw, err := fxFS.ReadFile("fx/" + f.Name())
		if err != nil {
			return nil, err
		}
		pcm, err := decodeWAVMono(raw, sampleRate)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name(), err)
		}
		id := FXID(strings.TrimSuffix(f.Name(), ".wav"))
		out[id] = pcm
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no FX clips found")
	}
	return out, nil
}

func MustLoadFXClips(sampleRate int) map[FXID][]byte {
	clips, err := LoadFXClips(sampleRate)
	if err != nil {
		log.Fatalf("fx clips: %v", err)
	}
	return clips
}
