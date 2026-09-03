package campaign

import (
	"strings"

	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// SnapshotObjectiveOutcomes records identify/destroy/complete for debrief text.
func SnapshotObjectiveOutcomes(sc *world.Scenario) []ObjectiveOutcome {
	if sc == nil {
		return nil
	}
	sc.CheckObjectives()
	out := make([]ObjectiveOutcome, 0, len(sc.Objectives))
	for _, o := range sc.Objectives {
		destroyed := false
		for _, e := range sc.Entities {
			if e != nil && e.ID == o.TargetID && e.Status != world.StatusActive {
				destroyed = true
				break
			}
		}
		out = append(out, ObjectiveOutcome{
			ID:         o.ID,
			Identified: o.Identified,
			Destroyed:  destroyed,
			Complete:   o.Complete,
		})
	}
	return out
}

// ComposeMissionDebrief builds after-action text: lead paragraph plus
// success/fail snippets for secondary and hidden tasks.
func ComposeMissionDebrief(m MissionDef, outcomes []ObjectiveOutcome, lang string) string {
	lang = i18n.NormalizeLang(lang)
	byID := make(map[string]ObjectiveOutcome, len(outcomes))
	for _, o := range outcomes {
		byID[o.ID] = o
	}
	var parts []string
	if lead := strings.TrimSpace(m.DebriefLead.GetText(lang)); lead != "" {
		parts = append(parts, lead)
	}
	for _, line := range m.DebriefLines {
		oc := byID[line.ObjectiveID]
		text := line.OnFail.GetText(lang)
		if oc.Complete {
			text = line.OnSuccess.GetText(lang)
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n\n")
}
