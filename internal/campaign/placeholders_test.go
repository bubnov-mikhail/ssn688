package campaign

import (
	"strings"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestExpandUnitPlaceholders(t *testing.T) {
	byID := map[string]*world.Entity{
		"civ_tanker": {
			ID: "civ_tanker", Name: "MT Horizon",
			X: 12340, Y: -5601, HeadingDeg: 227, SpeedKts: 9, DepthFt: 0,
		},
	}
	in := "Track **{{unit.civ_tanker.pos}}** course {{unit.civ_tanker.course}} ({{unit.civ_tanker.name}})"
	got := ExpandUnitPlaceholders(in, byID)
	lat, lon := world.WorldToLatLon(12340, -5601)
	wantPos := world.FormatNavLatLon(lat, lon)
	for _, w := range []string{wantPos, "course 225", "MT Horizon"} {
		if !strings.Contains(got, w) {
			t.Fatalf("missing %q in %q", w, got)
		}
	}
	if strings.Contains(got, "{{") {
		t.Fatalf("unexpanded token left: %q", got)
	}
	raw := "{{unit.missing.x}} {{unit.civ_tanker.bogus}}"
	if ExpandUnitPlaceholders(raw, byID) != raw {
		t.Fatalf("expected intact unknowns, got %q", ExpandUnitPlaceholders(raw, byID))
	}
}

func TestExpandMissionTimePlaceholder(t *testing.T) {
	start := 4*3600 + 30*60
	got := ExpandPlaceholders("Zulu {{mission_time}} execute.", nil, float64(start), 90)
	if got != "Zulu 04:31:30 execute." {
		t.Fatalf("got %q", got)
	}
}
