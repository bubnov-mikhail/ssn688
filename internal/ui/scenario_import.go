package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/config"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/platform"
)

// pickScenarioJSON opens a native file picker for a scenario JSON file.
// On mobile, picks the newest *.json from the import inbox (no OS dialog).
func pickScenarioJSON() (string, error) {
	if platform.Mobile() {
		return pickScenarioFromImportInbox()
	}
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("osascript", "-e",
			`POSIX path of (choose file with prompt "Import scenario JSON" of type {"json", "public.json"})`,
		).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	default:
		out, err := exec.Command("zenity", "--file-selection",
			"--title=Import scenario JSON",
			"--file-filter=JSON files | *.json",
		).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}
}

func pickScenarioFromImportInbox() (string, error) {
	dir, err := config.ScenarioImportDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	type cand struct {
		path string
		mod  time.Time
	}
	var cands []cand
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		cands = append(cands, cand{path: filepath.Join(dir, name), mod: info.ModTime()})
	}
	if len(cands) == 0 {
		return "", fmt.Errorf("no JSON in import folder: %s", dir)
	}
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].mod.After(cands[j].mod)
	})
	return cands[0].path, nil
}

func (a *App) importScenarioFromOS() {
	path, err := pickScenarioJSON()
	if err != nil || path == "" {
		if platform.Mobile() && err != nil {
			a.Statusf(i18n.StatusImportFailed, err.Error())
		}
		return
	}
	sc, err := campaign.ImportScenarioJSON(path)
	if err != nil {
		a.Statusf(i18n.StatusImportFailed, err.Error())
		return
	}
	a.Statusf(i18n.StatusImported, sc.Title.GetText(a.Lang()), sc.Version.String())
	a.beginScenarioUI()
	a.Mode = ModeScenarioList
	a.ScenarioListIndex = 0
	for i, d := range a.scenarioDefs() {
		if d.ID == sc.ID {
			a.ScenarioListIndex = i
			break
		}
	}
	a.ensureScenarioSelection()
}
