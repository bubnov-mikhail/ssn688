package ui

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestLibraryTableRowsGrouping(t *testing.T) {
	rows := libraryTableRows()
	if len(rows) < 5 {
		t.Fatalf("expected headers+entries, got %d", len(rows))
	}
	var sawHostile, sawNeutral, sawFriendly, sawWeapons bool
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
			case "WEAPONS":
				sawWeapons = true
				lastAll = libWeapons
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
	if !sawHostile || !sawNeutral || !sawFriendly || !sawWeapons {
		t.Fatal("missing section headers")
	}
}

func TestLibraryCatalogCoversWeapons(t *testing.T) {
	want := []string{
		"wpn_mk48", "wpn_harpoon", "wpn_klub", "wpn_oniks", "wpn_kalibr",
		"wpn_53_65", "wpn_umgt1", "wpn_rastrub", "wpn_rbu",
		"wpn_sam", "wpn_ciws", "wpn_adc", "wpn_jitter", "wpn_nixie",
	}
	for _, id := range want {
		if libraryEntryByID(id) == nil {
			t.Fatalf("missing weapon entry %s", id)
		}
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

func TestSyncReferenceToClassifiedContact(t *testing.T) {
	a := &App{referenceProfileIdx: 0} // Los Angeles default
	wantIdx := -1
	for i, p := range world.SignatureLibrary {
		if p.ID == "grisha" {
			wantIdx = i
			break
		}
	}
	if wantIdx < 0 {
		t.Fatal("grisha missing from SignatureLibrary")
	}
	c := &acoustics.Contact{ConfirmedID: "grisha", ConfirmedClass: "Grisha Corvette"}
	a.syncReferenceToContact(c)
	if a.referenceProfileIdx != wantIdx {
		t.Fatalf("reference=%d want grisha idx %d", a.referenceProfileIdx, wantIdx)
	}
	// BestMatch should not override confirmed.
	c.BestMatchID = "los_angeles"
	a.syncReferenceToContact(c)
	if a.referenceProfileIdx != wantIdx {
		t.Fatal("confirmed id should win over BestMatchID")
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
