package simreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Replay is a time-sampled headless mission run (player idle unless noted).
type Replay struct {
	FormatVersion int     `json:"format_version"`
	ScenarioID    string  `json:"scenario_id"`
	MissionID     string  `json:"mission_id"`
	MissionTitle  string  `json:"mission_title"`
	TheaterID     string  `json:"theater_id"`
	Seed          int64   `json:"seed"`
	DurationSec   float64 `json:"duration_sec"`
	SampleSec        float64    `json:"sample_sec"`
	MissionStartSec  float64    `json:"mission_start_sec,omitempty"`
	Comm             []CommSnap `json:"comm,omitempty"`
	Frames           []Frame    `json:"frames"`
}

// Frame is one sampled instant of the battlespace.
type Frame struct {
	Time    float64       `json:"time"`
	Units   []UnitSnap    `json:"units"`
	Weapons []WeaponSnap  `json:"weapons,omitempty"`
	Flashes []FlashSnap   `json:"flashes,omitempty"`
	Markers []MarkerSnap  `json:"markers,omitempty"`
}

// UnitSnap is a platform at sample time.
type UnitSnap struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Side     string  `json:"side"`
	Status   string  `json:"status"`
	AIState  string  `json:"ai_state,omitempty"`
	Defcon   int     `json:"defcon"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Heading  float64 `json:"heading"`
	SpeedKts float64 `json:"speed_kts"`
	Alive    bool    `json:"alive"`
}

// WeaponKind identifies in-flight ordnance for the replay player.
type WeaponKind string

const (
	WeaponTorpedo WeaponKind = "torpedo"
	WeaponHarpoon WeaponKind = "harpoon"
	WeaponRBU     WeaponKind = "rbu"
	WeaponRastrub WeaponKind = "rastrub"
)

// WeaponSnap is a torpedo, missile, or in-flight ASW rocket pattern.
type WeaponSnap struct {
	Kind     WeaponKind `json:"kind"`
	Label    string     `json:"label"`
	Side     string     `json:"side"`
	X        float64    `json:"x"`
	Y        float64    `json:"y"`
	X1       float64    `json:"x1,omitempty"`
	Y1       float64    `json:"y1,omitempty"`
	Heading  float64    `json:"heading"`
	SpeedKts float64    `json:"speed_kts"`
	Alive    bool       `json:"alive"`
	WireCut  bool       `json:"wire_cut,omitempty"`
	HarpoonUW bool      `json:"harpoon_uw,omitempty"`
	HarpoonLock bool    `json:"harpoon_lock,omitempty"`
	HarpoonRadar bool   `json:"harpoon_radar,omitempty"`
}

// FlashSnap is a short-lived launch marker (WEPS debug overlay).
type FlashSnap struct {
	Label string  `json:"label"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}

// MarkerSnap is a mission or player plot marker on the tactical chart.
type MarkerSnap struct {
	ID   string  `json:"id"`
	Name string  `json:"name,omitempty"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

const FormatVersion = 1

// DefaultMaxSec is the standard AFK headless run length (90 minutes).
const DefaultMaxSec = 90 * 60

func Save(path string, r *Replay) error {
	if r == nil {
		return fmt.Errorf("nil replay")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func Load(path string) (*Replay, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !bytes.Contains(b, []byte(`"x"`)) {
		return nil, fmt.Errorf("replay %q has no unit positions — re-record with ./ssn688-player -record", filepath.Base(path))
	}
	var r Replay
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if r.FormatVersion != FormatVersion {
		return nil, fmt.Errorf("unsupported replay format %d", r.FormatVersion)
	}
	if len(r.Frames) == 0 {
		return nil, fmt.Errorf("empty replay")
	}
	return &r, nil
}

// FrameAt returns the frame index and interpolated time for game clock t.
func (r *Replay) FrameAt(t float64) int {
	if r == nil || len(r.Frames) == 0 {
		return 0
	}
	if t <= r.Frames[0].Time {
		return 0
	}
	last := len(r.Frames) - 1
	if t >= r.Frames[last].Time {
		return last
	}
	step := r.SampleSec
	if step <= 0 {
		step = 1
	}
	idx := int(t / step)
	if idx > last {
		return last
	}
	return idx
}
