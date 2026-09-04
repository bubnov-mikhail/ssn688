package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/platform"
)

// Settings holds user preferences persisted between sessions.
type Settings struct {
	Fullscreen    bool    `json:"fullscreen"`
	MasterVolume  float64 `json:"master_volume"`
	VoiceVolume   float64 `json:"voice_volume"`
	EffectsVolume float64 `json:"effects_volume"`
	WindowWidth   int     `json:"window_width"`
	WindowHeight  int     `json:"window_height"`
	Debug         bool    `json:"debug"`
	Language      string  `json:"language"`
}

func DefaultSettings() Settings {
	return Settings{
		Fullscreen:    false,
		MasterVolume:  0.8,
		VoiceVolume:   0.9,
		EffectsVolume: 0.7,
		WindowWidth:   1600,
		WindowHeight:  900,
		Debug:         true,
		Language:      i18n.LangEN,
	}
}

var (
	dataRootMu sync.RWMutex
	dataRoot   string // Android/iOS files dir from mobile.SetDataRoot; empty on desktop
)

// SetDataRoot overrides the app data parent directory (mobile: Context.getFilesDir()).
// Call before Load / SavesDir / ScenariosDir. Pass "" to clear.
func SetDataRoot(dir string) {
	dataRootMu.Lock()
	dataRoot = filepath.Clean(dir)
	if dataRoot == "." || dataRoot == "" {
		dataRoot = ""
	}
	dataRootMu.Unlock()
}

// DataRoot returns the override set via SetDataRoot (empty if unset).
func DataRoot() string {
	dataRootMu.RLock()
	defer dataRootMu.RUnlock()
	return dataRoot
}

// appDataDir is <configRoot>/ssn688 — settings, saves, scenarios live under it.
func appDataDir() (string, error) {
	dataRootMu.RLock()
	root := dataRoot
	dataRootMu.RUnlock()
	if root != "" {
		return filepath.Join(root, "ssn688"), nil
	}
	if platform.Mobile() {
		// ebitenmobile Activity should call SetDataRoot(getFilesDir()) early.
		// Until then, keep writes inside a relative sandbox rather than "/".
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, "ssn688-data"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ssn688"), nil
}

func SettingsPath() (string, error) {
	dir, err := appDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

func Load() (Settings, error) {
	s := DefaultSettings()
	path, err := SettingsPath()
	if err != nil {
		return s, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	var raw struct {
		Fullscreen    bool     `json:"fullscreen"`
		MasterVolume  float64  `json:"master_volume"`
		VoiceVolume   float64  `json:"voice_volume"`
		EffectsVolume float64  `json:"effects_volume"`
		WindowWidth   int      `json:"window_width"`
		WindowHeight  int      `json:"window_height"`
		Debug         *bool    `json:"debug"`
		Language      string   `json:"language"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return DefaultSettings(), err
	}
	s.Fullscreen = raw.Fullscreen
	s.MasterVolume = raw.MasterVolume
	s.VoiceVolume = raw.VoiceVolume
	s.EffectsVolume = raw.EffectsVolume
	if raw.WindowWidth > 0 {
		s.WindowWidth = raw.WindowWidth
	}
	if raw.WindowHeight > 0 {
		s.WindowHeight = raw.WindowHeight
	}
	if raw.Debug != nil {
		s.Debug = *raw.Debug
	}
	if raw.Language != "" {
		s.Language = i18n.NormalizeLang(raw.Language)
	}
	return s, nil
}

func Save(s Settings) error {
	path, err := SettingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	s.Language = i18n.NormalizeLang(s.Language)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func SavesDir() (string, error) {
	dir, err := appDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "saves"), nil
}

// ScenariosDir is the user folder for installed (imported) scenario JSON files.
func ScenariosDir() (string, error) {
	dir, err := appDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "scenarios"), nil
}

// ScenarioImportDir is the inbox for mobile scenario drops (USB / share / adb).
// Desktop import still uses the OS file picker; this folder is created for parity.
func ScenarioImportDir() (string, error) {
	dir, err := appDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "import"), nil
}
