package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var raw string

// ScenarioFormatMajor is bumped when scenario JSON schema has breaking changes.
const ScenarioFormatMajor = 3

// Title is the public game name (menu, window title, bundle metadata).
const Title = "SSN-688 Modern Submarine Combat Simulator"

// String is the semantic version of the game binary (from VERSION).
func String() string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "0.0.0"
	}
	return v
}

// Display returns the semver with a leading "v" for UI footers.
func Display() string {
	return "v" + String()
}
