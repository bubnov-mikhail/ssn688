package world

import (
	"math/rand"
	"testing"
)

func TestPlaceAwayFromMinRange(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	player := &Entity{ID: "player", X: 1000, Y: -500, Kind: KindSubmarine, DepthFt: 100}
	dd := &Entity{ID: "dd", Kind: KindSurfaceShip}
	raw := encodeTestBathy(32, 32, -4000, -4000, 250, 2000)
	bathy, err := LoadBathymetry(raw)
	if err != nil {
		t.Fatal(err)
	}
	PlaceAwayFrom(rng, dd, player, YardsPerNM, 3*YardsPerNM, nil, &bathy)
	if player.RangeYardsTo(dd) < YardsPerNM-1 {
		t.Fatalf("placed at %.0f yd, want ≥ 1 nm", player.RangeYardsTo(dd))
	}
}
