package ui

import (
	"fmt"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/world"
)

func contactClassLabel(c *acoustics.Contact) string {
	if c.ConfirmedClass != "" {
		return c.ConfirmedClass
	}
	return "—"
}

func contactRangeAccuracy(c *acoustics.Contact) float64 {
	if c == nil || c.EstimatedRangeYd <= 0 {
		return 0
	}
	if acoustics.ContactHasActiveRange(c) {
		return 1
	}
	unc := c.UncRangeYd
	if unc <= 0 {
		unc = c.EstimatedRangeYd * 0.45
	}
	acc := 1 - unc/c.EstimatedRangeYd
	if acc < 0 {
		return 0
	}
	return acc
}

func contactBearingLabel(c *acoustics.Contact) string {
	if c == nil {
		return "—"
	}
	return fmt.Sprintf("%03.0f°", c.BearingDeg)
}

func contactRangeLabel(c *acoustics.Contact) string {
	if c == nil || c.EstimatedRangeYd <= 0 {
		return "—"
	}
	val := fmt.Sprintf("%.1f kyd", c.EstimatedRangeYd/1000)
	if contactRangeAccuracy(c) < 0.80 {
		return "~" + val
	}
	return val
}

func contactShortLabel(c *acoustics.Contact) string {
	return c.ID
}

func contactLongLabel(c *acoustics.Contact) string {
	if c.ConfirmedClass != "" {
		return fmt.Sprintf("%s %s", c.ID, c.ConfirmedClass)
	}
	return c.ID
}

func contactSourceLabel(c *acoustics.Contact, player *world.Entity, sonar *acoustics.SonarState) string {
	if c == nil || sonar == nil || player == nil {
		return "—"
	}
	rel := acoustics.AngleDiffDeg(c.BearingDeg, player.HeadingDeg)
	hull := acoustics.PassiveArraySensitivity(acoustics.PassiveArrayHull, rel, 0) >= 0.18
	towed := sonar.TowedCablePct > 0.15 &&
		acoustics.PassiveArraySensitivity(acoustics.PassiveArrayTowed, rel, sonar.TowedCablePct) >= 0.18
	switch {
	case hull && towed:
		return "ALL"
	case hull:
		return "HLL"
	case towed:
		return "TWD"
	default:
		return "—"
	}
}

func selectedContactLabel(sonar *acoustics.SonarState, selectedID string) string {
	if selectedID == "" {
		return "none"
	}
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if c.SourceEntityID == selectedID {
			return contactLongLabel(c)
		}
	}
	return "none"
}
