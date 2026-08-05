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
		if dist < YardsPerNM-1 {
			t.Fatalf("attempt %d: Grisha too close: %.0f yd (want ≥ %.0f / 1 nm)", i, dist, YardsPerNM)
		}
		var fox *Entity
		for _, e := range sc.Entities {
			if e != nil && e.ID == "enemy_foxtrot" {
				fox = e
				break
			}
		}
		if fox == nil || fox.SignatureID != "foxtrot" {
			t.Fatal("missing Foxtrot in training scenario")
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
