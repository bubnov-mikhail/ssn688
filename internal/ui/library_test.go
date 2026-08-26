package ui

import (
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/i18n"
	"github.com/ssn688/sim/internal/world"
)

func TestLibraryTableRowsGrouping(t *testing.T) {
	rows := libraryTableRows()
	if len(rows) < 5 {
		t.Fatalf("expected headers+entries, got %d", len(rows))
	}
	var sawHostile, sawNeutral, sawFriendly bool
	var lastAll libraryAllegiance = -1
	sawSub := false
	for _, r := range rows {
		if r.Header {
			switch r.Label.GetText(i18n.LangEN) {
			case "HOSTILE":
				sawHostile = true
				lastAll = libHostile
			case "NEUTRAL":
				sawNeutral = true
				lastAll = libNeutral
			case "FRIENDLY":
				sawFriendly = true
				lastAll = libFriendly
			default:
				t.Fatalf("unexpected header %q", r.Label.GetText(i18n.LangEN))
			}
			sawSub = false
			continue
		}
		e := libraryEntryByID(r.EntryID)
		if e == nil {
			t.Fatalf("missing entry %s", r.EntryID)
		}
		if e.Allegiance != lastAll {
			t.Fatalf("entry %s under wrong section", e.ID)
		}
		if e.Kind == world.KindSubmarine {
			sawSub = true
		}
		if e.Kind == world.KindSurfaceShip && sawSub {
			t.Fatalf("surface %s after submarine in section", e.ID)
		}
	}
	if !sawHostile || !sawNeutral || !sawFriendly {
		t.Fatal("missing allegiance headers")
	}
}

func TestClassifiedLibraryIDFromConfirmed(t *testing.T) {
	c := &acoustics.Contact{ConfirmedID: "udaloy", ConfirmedClass: "Udaloy DDG"}
	if id := classifiedLibraryID(c); id != "udaloy" {
		t.Fatalf("got %q", id)
	}
	torp := &acoustics.Contact{ConfirmedID: "mk48", ConfirmedClass: "Mk48 ADCAP"}
	if classifiedLibraryID(torp) != "" {
		t.Fatal("torpedo should not map to platform handbook")
	}
	if classifiedLibraryID(&acoustics.Contact{}) != "" {
		t.Fatal("unclassified should be empty")
	}
}

func TestLibraryCatalogCoversPlatforms(t *testing.T) {
	want := []string{
		"udaloy", "krivak", "kresta2", "grisha", "gorshkov", "spruance",
		"kilo", "foxtrot", "victor_iii", "yasen_m",
		"merchant", "tanker", "fishing",
		"los_angeles",
	}
	for _, id := range want {
		if libraryEntryByID(id) == nil {
			t.Fatalf("missing catalog entry %s", id)
		}
	}
}
