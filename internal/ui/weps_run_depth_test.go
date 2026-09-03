package ui

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestWepsSuggestedRunDepthSurfaceFromConfirmedID(t *testing.T) {
	player := &world.Entity{DepthFt: 250}
	// Track Kind still looks like a sub (pre-classify lag), but SPECTRUM confirmed a DDG.
	c := &acoustics.Contact{
		Kind:           world.KindSubmarine,
		ConfirmedID:    "udaloy",
		ConfirmedClass: "Udaloy DDG",
	}
	got := wepsSuggestedRunDepth(player, c)
	if got != 40 {
		t.Fatalf("surface confirm want 40 ft, got %.0f (used sub depth?)", got)
	}
}

func TestWepsSuggestedRunDepthSubmarine(t *testing.T) {
	player := &world.Entity{DepthFt: 250}
	c := &acoustics.Contact{
		Kind:           world.KindSubmarine,
		ConfirmedID:    "foxtrot",
		ConfirmedClass: "Foxtrot SS",
	}
	got := wepsSuggestedRunDepth(player, c)
	if got != 250 {
		t.Fatalf("sub confirm want player depth 250, got %.0f", got)
	}
}

func TestContactConfirmedKindPrefersProfile(t *testing.T) {
	c := &acoustics.Contact{
		Kind:           world.KindSubmarine,
		ConfirmedID:    "grisha",
		ConfirmedClass: "Grisha Corvette",
	}
	if contactConfirmedKind(c) != world.KindSurfaceShip {
		t.Fatalf("want surface from profile, got %v", contactConfirmedKind(c))
	}
}
