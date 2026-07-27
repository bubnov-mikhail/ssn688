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
	if c.BestMatchName != "" {
		return "~" + c.BestMatchName
	}
	return "—"
}

func contactTypeLabel(c *acoustics.Contact) string {
	switch c.Kind {
	case world.KindSubmarine:
		return "SUB"
	case world.KindSurfaceShip:
		return "SURF"
	default:
		return "UNK"
	}
}

func contactShortLabel(c *acoustics.Contact) string {
	return c.ID
}

func contactLongLabel(c *acoustics.Contact) string {
	if c.ConfirmedClass != "" {
		return fmt.Sprintf("%s %s", c.ID, c.ConfirmedClass)
	}
	return fmt.Sprintf("%s %s", c.ID, contactClassLabel(c))
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
