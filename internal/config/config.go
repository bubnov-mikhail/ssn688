package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bubnov-mikhail/ssn688/internal/i18n"
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

func SettingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ssn688", "settings.json"), nil
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
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ssn688", "saves"), nil
}

// ScenariosDir is the user folder for imported scenario JSON files.
func ScenariosDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ssn688", "scenarios"), nil
}
