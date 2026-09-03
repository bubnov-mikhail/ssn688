package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const scenariosDir = "scenarios_generated"

// resolveScenarioPath picks a scenario JSON under scenarios_generated/.
func resolveScenarioPath(flagPath string) (string, error) {
	if flagPath != "" {
		if _, err := os.Stat(flagPath); err != nil {
			return "", fmt.Errorf("%s: %w", flagPath, err)
		}
		return flagPath, nil
	}
	paths, err := listScenarioJSONs()
	if err != nil {
		return "", fmt.Errorf("scenario: %w", err)
	}
	switch len(paths) {
	case 0:
		return "", fmt.Errorf("no scenario JSON in %s/ — generate one first", scenariosDir)
	case 1:
		fmt.Printf("scenario: %s\n", paths[0])
		return paths[0], nil
	default:
		picked, err := pickScenarioFileDialog()
		if err != nil {
			return "", err
		}
		if picked == "" {
			return "", fmt.Errorf("scenario selection cancelled")
		}
		fmt.Printf("scenario: %s\n", picked)
		return picked, nil
	}
}

func listScenarioJSONs() ([]string, error) {
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		out = append(out, filepath.Join(scenariosDir, e.Name()))
	}
	return out, nil
}

type replayEntry struct {
	Path         string
	ScenarioID   string
	MissionID    string
	MissionTitle string
	Label        string
}

func listReplaysForScenario(scenarioID string) ([]replayEntry, error) {
	dir := filepath.Join(scenariosDir, "sim_replays")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []replayEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".replay.json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		meta, err := peekReplay(path)
		if err != nil {
			continue
		}
		if meta.ScenarioID != scenarioID {
			continue
		}
		label := meta.MissionID
		if meta.MissionTitle != "" {
			label = meta.MissionTitle + " (" + meta.MissionID + ")"
		}
		out = append(out, replayEntry{
			Path:         path,
			ScenarioID:   meta.ScenarioID,
			MissionID:    meta.MissionID,
			MissionTitle: meta.MissionTitle,
			Label:        label,
		})
	}
	return out, nil
}

type replayPeek struct {
	ScenarioID   string `json:"scenario_id"`
	MissionID    string `json:"mission_id"`
	MissionTitle string `json:"mission_title"`
}

func peekReplay(path string) (replayPeek, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return replayPeek{}, err
	}
	var p replayPeek
	if err := json.Unmarshal(b, &p); err != nil {
		return replayPeek{}, err
	}
	return p, nil
}

// resolveReplayPath picks a replay for the given scenario.
func resolveReplayPath(scenarioID, flagPath string) (string, error) {
	if flagPath != "" {
		if _, err := os.Stat(flagPath); err != nil {
			return "", err
		}
		return flagPath, nil
	}
	replays, err := listReplaysForScenario(scenarioID)
	if err != nil {
		return "", err
	}
	switch len(replays) {
	case 0:
		return "", nil
	case 1:
		fmt.Printf("replay: %s\n", replays[0].Path)
		return replays[0].Path, nil
	default:
		labels := make([]string, len(replays))
		for i, r := range replays {
			labels[i] = r.Label
		}
		idx, err := pickFromList("Select replay", labels)
		if err != nil || idx < 0 {
			return "", err
		}
		if idx >= len(replays) {
			return "", fmt.Errorf("replay selection cancelled")
		}
		fmt.Printf("replay: %s\n", replays[idx].Path)
		return replays[idx].Path, nil
	}
}

func pickFromList(prompt string, items []string) (int, error) {
	if len(items) == 0 {
		return -1, fmt.Errorf("empty list")
	}
	if len(items) == 1 {
		return 0, nil
	}
	switch runtime.GOOS {
	case "darwin":
		return pickFromListOSX(prompt, items)
	default:
		return pickFromListZenity(prompt, items)
	}
}

func pickFromListOSX(prompt string, items []string) (int, error) {
	var b strings.Builder
	b.WriteString("set theList to {")
	for i, item := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(`"`)
		b.WriteString(strings.ReplaceAll(item, `"`, `\"`))
		b.WriteString(`"`)
	}
	b.WriteString("}\n")
	b.WriteString(`set picked to choose from list theList with prompt "`)
	b.WriteString(strings.ReplaceAll(prompt, `"`, `\"`))
	b.WriteString(`"`)
	b.WriteString("\nif picked is false then return \"\"\nreturn item 1 of picked\n")
	out, err := exec.Command("osascript", "-e", b.String()).Output()
	if err != nil {
		return -1, err
	}
	choice := strings.TrimSpace(string(out))
	if choice == "" {
		return -1, nil
	}
	for i, item := range items {
		if item == choice {
			return i, nil
		}
	}
	return -1, fmt.Errorf("unexpected choice %q", choice)
}

func pickFromListZenity(prompt string, items []string) (int, error) {
	args := []string{"--list", "--title=" + prompt, "--column=Replay"}
	args = append(args, items...)
	out, err := exec.Command("zenity", args...).Output()
	if err != nil {
		return -1, err
	}
	choice := strings.TrimSpace(string(out))
	for i, item := range items {
		if item == choice {
			return i, nil
		}
	}
	return -1, fmt.Errorf("unexpected choice %q", choice)
}

func pickScenarioFileDialog() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		dir := scenariosDir
		if workspaceRoot != "" {
			dir = filepath.Join(workspaceRoot, scenariosDir)
		}
		out, err := exec.Command("osascript", "-e",
			fmt.Sprintf(`POSIX path of (choose file with prompt "Select scenario JSON" of type {"json", "public.json"} default location (POSIX file "%s/"))`, dir),
		).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	default:
		dir := scenariosDir
		if workspaceRoot != "" {
			dir = filepath.Join(workspaceRoot, scenariosDir)
		}
		abs, _ := filepath.Abs(dir)
		out, err := exec.Command("zenity", "--file-selection",
			"--title=Select scenario JSON",
			"--filename="+abs+"/",
			"--file-filter=JSON | *.json",
		).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}
}
