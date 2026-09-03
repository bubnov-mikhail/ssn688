package campaign

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bubnov-mikhail/ssn688/internal/config"
)

// AutosaveName returns the canonical between-missions autosave filename.
func AutosaveName(scenarioID ScenarioID) string {
	return fmt.Sprintf("campaign_%s_autosave.sav", scenarioID)
}

// ResolveMissionOutputs applies output rules after a mission end.
// outcomes are the debrief snapshots used for when_objective_id rules.
func ResolveMissionOutputs(sc *ScenarioDef, missionID MissionID, primaryComplete bool, outcomes []ObjectiveOutcome) map[string]string {
	out := map[string]string{}
	m := MissionByID(sc.ID, missionID)
	if m == nil {
		return out
	}
	complete := map[string]bool{}
	for _, o := range outcomes {
		if o.Complete {
			complete[o.ID] = true
		}
	}
	for _, rule := range m.Outputs {
		if rule.WhenObjectiveID != "" {
			if complete[rule.WhenObjectiveID] {
				out[rule.Key] = rule.Value
			}
			continue
		}
		if rule.WhenPrimaryComplete && !primaryComplete {
			continue
		}
		out[rule.Key] = rule.Value
	}
	return out
}

// MergeVars copies src into dst.
func MergeVars(dst, src map[string]string) {
	if dst == nil || len(src) == 0 {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}

// ReadSaveCampaignMeta scans a save file header for campaign fields without full load.
func ReadSaveCampaignMeta(path string) (RuntimeMeta, bool) {
	f, err := os.Open(path)
	if err != nil {
		return RuntimeMeta{}, false
	}
	defer f.Close()

	meta := RuntimeMeta{
		Completed: map[MissionID]bool{},
		Vars:      map[string]string{},
	}
	section := ""
	found := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}
		if section != "" && section != "campaign" {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "scenario_id":
			meta.ScenarioID = ScenarioID(val)
			found = true
		case "mission_id":
			meta.MissionID = MissionID(val)
			found = true
		case "mission_hash":
			meta.MissionHash = val
		case "loadout_mix":
			fmt.Sscanf(val, "%f", &meta.LoadoutMix)
		case "report_eligible":
			meta.ReportEligible, _ = parseBool(val)
		case "between_missions":
			meta.BetweenMissions, _ = parseBool(val)
		case "debrief_pending":
			meta.DebriefPending, _ = parseBool(val)
		case "debrief_mission":
			meta.DebriefMission = MissionID(val)
		case "debrief_obj":
			if o, ok := parseDebriefOutcome(val); ok {
				meta.DebriefOutcomes = append(meta.DebriefOutcomes, o)
			}
		case "completed":
			for _, id := range strings.Split(val, ",") {
				id = strings.TrimSpace(id)
				if id != "" {
					meta.Completed[MissionID(id)] = true
				}
			}
		case "var":
			parts := strings.SplitN(val, ":", 2)
			if len(parts) == 2 {
				meta.Vars[parts[0]] = parts[1]
			}
		}
	}
	return meta, found
}

func parseDebriefOutcome(val string) (ObjectiveOutcome, bool) {
	parts := strings.Split(val, "|")
	if len(parts) < 4 || parts[0] == "" {
		return ObjectiveOutcome{}, false
	}
	return ObjectiveOutcome{
		ID:         parts[0],
		Identified: parts[1] == "true",
		Destroyed:  parts[2] == "true",
		Complete:   parts[3] == "true",
	}, true
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// LatestSaveForScenario returns the newest save belonging to a compatible campaign.
func LatestSaveForScenario(scenarioID ScenarioID) (string, error) {
	sc := ScenarioByID(scenarioID)
	if sc == nil || !sc.Compatible {
		return "", os.ErrNotExist
	}
	dir, err := config.SavesDir()
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	type candidate struct {
		path string
		mod  time.Time
	}
	var best *candidate
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sav" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		meta, ok := ReadSaveCampaignMeta(path)
		if !ok || meta.ScenarioID != scenarioID {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if best == nil || info.ModTime().After(best.mod) {
			best = &candidate{path: path, mod: info.ModTime()}
		}
	}
	if best == nil {
		return "", os.ErrNotExist
	}
	return best.path, nil
}

// DeleteScenarioSaves removes all saves tied to a scenario.
func DeleteScenarioSaves(scenarioID ScenarioID) error {
	dir, err := config.SavesDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sav" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		meta, ok := ReadSaveCampaignMeta(path)
		if ok && meta.ScenarioID == scenarioID {
			_ = os.Remove(path)
		}
	}
	return nil
}

// ListSaveFiles returns sorted save paths in the saves directory.
func ListSaveFiles() ([]string, error) {
	dir, err := config.SavesDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sav" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}
