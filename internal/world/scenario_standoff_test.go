package world

import (
	"math/rand"
	"testing"
)

func TestTrainingScenarioSurfaceStandoff(t *testing.T) {
	for i := 0; i < 20; i++ {
		sc := NewTrainingScenario()
		var corvette *Entity
		for _, e := range sc.Entities {
			if e != nil && e.ID == "enemy_grisha" {
				corvette = e
				break
			}
		}
		if corvette == nil {
			t.Fatal("missing enemy_grisha")
		}
		dist := sc.Player.RangeYardsTo(corvette)
		if dist < 1100 || dist > 3000 {
			t.Fatalf("attempt %d: Grisha at %.0f yd (want ~1200–2800 near ownship)", i, dist)
		}
		if corvette.Defcon != DefconPassive {
			t.Fatalf("Grisha DEFCON=%d want 0", corvette.Defcon)
		}
		var fox *Entity
		nearCiv := 0
		for _, e := range sc.Entities {
			if e == nil {
				continue
			}
			if e.ID == "enemy_foxtrot" {
				fox = e
			}
			if e.Side == SideNeutral && (e.ID == "civ_merchant" || e.ID == "civ_tanker") {
				d := sc.Player.RangeYardsTo(e)
				if d < 1200 || d > 3500 {
					t.Fatalf("attempt %d: %s at %.0f yd (want near band)", i, e.ID, d)
				}
				nearCiv++
			}
		}
		if nearCiv != 2 {
			t.Fatalf("want 2 near civilians, got %d", nearCiv)
		}
		if fox == nil || fox.SignatureID != "foxtrot" {
			t.Fatal("missing Foxtrot in training scenario")
		}
		if fox.Defcon != DefconPassive {
			t.Fatalf("Foxtrot DEFCON=%d want 0", fox.Defcon)
		}
		nHostile := 0
		for _, e := range sc.Entities {
			if e != nil && e.Side == SideEnemy {
				nHostile++
			}
		}
		if nHostile != 2 {
			t.Fatalf("want 2 hostiles (Grisha+Foxtrot), got %d", nHostile)
		}
	}
}

func TestPlaceAwayFromMinRange(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	player := &Entity{ID: "player", X: 1000, Y: -500, Kind: KindSubmarine, DepthFt: 100}
	dd := &Entity{ID: "dd", Kind: KindSurfaceShip}
	bathy := DefaultBathy
	placeAwayFrom(rng, dd, player, YardsPerNM, 3*YardsPerNM, nil, &bathy)
	if player.RangeYardsTo(dd) < YardsPerNM-1 {
		t.Fatalf("placed at %.0f yd, want ≥ 1 nm", player.RangeYardsTo(dd))
	}
}
