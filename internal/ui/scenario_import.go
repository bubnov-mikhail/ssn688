package ui

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/ssn688/sim/internal/campaign"
	"github.com/ssn688/sim/internal/i18n"
)

// pickScenarioJSON opens a native file picker for a scenario JSON file.
func pickScenarioJSON() (string, error) {
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

func (a *App) importScenarioFromOS() {
	path, err := pickScenarioJSON()
	if err != nil || path == "" {
		return
	}
	sc, err := campaign.ImportScenarioJSON(path)
	if err != nil {
		a.Statusf(i18n.StatusImportFailed, err.Error())
		return
	}
	a.Statusf(i18n.StatusImported, sc.Title.GetText(a.Lang()), sc.Version.String())
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
