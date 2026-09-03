package campaign

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/bubnov-mikhail/ssn688/internal/config"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/scenarios"
)

var (
	scenarioMu    sync.RWMutex
	scenarioCache []ScenarioDef
)

// ReloadScenarios clears cached scenario list (after import).
func ReloadScenarios() {
	scenarioMu.Lock()
	scenarioCache = nil
	scenarioMu.Unlock()
	render.ClearScenarioCoverCache()
}

// AllScenarios returns bundled + user scenarios sorted by title.
func AllScenarios() []ScenarioDef {
	scenarioMu.RLock()
	if scenarioCache != nil {
		out := append([]ScenarioDef(nil), scenarioCache...)
		scenarioMu.RUnlock()
		return out
	}
	scenarioMu.RUnlock()

	scenarioMu.Lock()
	defer scenarioMu.Unlock()
	ensureScenarioCacheLocked()
	return append([]ScenarioDef(nil), scenarioCache...)
}

func ensureScenarioCacheLocked() {
	if scenarioCache == nil {
		scenarioCache = loadAllScenarios()
	}
}

func loadAllScenarios() []ScenarioDef {
	byID := map[ScenarioID]ScenarioDef{}

	loadDir := func(dir string, entries []os.DirEntry) {
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".json" || e.Name() == "schema.json" {
				continue
			}
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			sc, err := ParseScenarioJSON(data, path)
			if err != nil {
				continue
			}
			mergeScenario(byID, sc)
		}
	}

	// Bundled JSON from embed.
	bundledNames, _ := scenarios.Bundled.ReadDir(".")
	for _, e := range bundledNames {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" || e.Name() == "schema.json" {
			continue
		}
		data, err := scenarios.Bundled.ReadFile(e.Name())
		if err != nil {
			continue
		}
		sc, err := ParseScenarioJSON(data, "bundled:"+e.Name())
		if err != nil {
			continue
		}
		mergeScenario(byID, sc)
	}

	if userDir, err := config.ScenariosDir(); err == nil {
		if entries, err := os.ReadDir(userDir); err == nil {
			loadDir(userDir, entries)
		}
	}

	out := make([]ScenarioDef, 0, len(byID))
	for _, sc := range byID {
		out = append(out, sc)
	}
	sort.Slice(out, func(i, j int) bool {
		ti := out[i].Title.GetText(i18n.LangEN)
		tj := out[j].Title.GetText(i18n.LangEN)
		if ti == tj {
			return out[i].ID < out[j].ID
		}
		return ti < tj
	})
	return out
}

func mergeScenario(byID map[ScenarioID]ScenarioDef, sc ScenarioDef) {
	prev, ok := byID[sc.ID]
	if !ok || sc.Version.Compare(prev.Version) >= 0 {
		byID[sc.ID] = sc
	}
}

// ScenarioByID finds a loaded campaign definition.
func ScenarioByID(id ScenarioID) *ScenarioDef {
	scenarioMu.RLock()
	defer scenarioMu.RUnlock()
	ensureScenarioCacheLocked()
	for i := range scenarioCache {
		if scenarioCache[i].ID == id {
			return &scenarioCache[i]
		}
	}
	return nil
}

// ScenarioByIDCompatible returns nil for incompatible scenarios.
func ScenarioByIDCompatible(id ScenarioID) *ScenarioDef {
	sc := ScenarioByID(id)
	if sc == nil || !sc.Compatible {
		return nil
	}
	return sc
}

// HasUserScenarioFile reports whether an imported JSON exists for id
// under the user scenarios directory (bundled-only scenarios return false).
func HasUserScenarioFile(id ScenarioID) bool {
	_, ok := userScenarioPath(id)
	return ok
}

func userScenarioPath(id ScenarioID) (string, bool) {
	if id == "" {
		return "", false
	}
	userDir, err := config.ScenariosDir()
	if err != nil {
		return "", false
	}
	path := filepath.Join(userDir, string(id)+".json")
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

// DeleteUserScenario removes an imported scenario JSON and its saves.
// Bundled scenarios cannot be deleted; returns an error if no user file exists.
func DeleteUserScenario(id ScenarioID) error {
	path, ok := userScenarioPath(id)
	if !ok {
		return fmt.Errorf("scenario %s is not an imported file", id)
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	_ = DeleteScenarioSaves(id)
	ReloadScenarios()
	return nil
}

// ImportScenarioJSON validates and installs a user scenario file.
func ImportScenarioJSON(srcPath string) (ScenarioDef, error) {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return ScenarioDef{}, err
	}
	sc, err := ParseScenarioJSON(data, srcPath)
	if err != nil {
		return ScenarioDef{}, err
	}
	if !sc.Compatible {
		return ScenarioDef{}, fmt.Errorf("incompatible: %s", sc.IncompatibleReason)
	}
	userDir, err := config.ScenariosDir()
	if err != nil {
		return ScenarioDef{}, err
	}
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return ScenarioDef{}, err
	}
	dest := filepath.Join(userDir, string(sc.ID)+".json")
	if prev, err := os.ReadFile(dest); err == nil {
		old, perr := ParseScenarioJSON(prev, dest)
		if perr == nil && sc.Version.Compare(old.Version) < 0 {
			return ScenarioDef{}, fmt.Errorf("installed version %s is newer than %s", old.Version, sc.Version)
		}
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return ScenarioDef{}, err
	}
	ReloadScenarios()
	return sc, nil
}
