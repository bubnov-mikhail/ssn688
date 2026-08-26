package world

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// SecondsPerDay is used when wrapping mission wall-clock display.
const SecondsPerDay = 24 * 3600

// ParseStartTimeHHMM parses "HH:MM" (24h) into seconds from midnight.
func ParseStartTimeHHMM(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("start_time want HH:MM, got %q", s)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, fmt.Errorf("invalid hour in start_time %q", s)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, fmt.Errorf("invalid minute in start_time %q", s)
	}
	return float64(h*3600 + m*60), nil
}

// FormatMissionClock formats wall-clock HH:MM:SS from mission start + elapsed game seconds.
func FormatMissionClock(startSecOfDay, elapsedSec float64) string {
	if elapsedSec < 0 {
		elapsedSec = 0
	}
	total := startSecOfDay + elapsedSec
	sec := int(math.Mod(total, SecondsPerDay))
	if sec < 0 {
		sec += SecondsPerDay
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
