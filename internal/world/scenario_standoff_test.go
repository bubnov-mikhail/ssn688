package world

import (
	"math/rand"
	"testing"

	"github.com/ssn688/sim/assets"
)

func TestTrainingScenarioDiagonalPlacement(t *testing.T) {
	b, err := LoadBathymetry(assets.BathyChart)
	if err != nil {
		t.Fatal(err)
	}
	SetDefaultBathymetry(b)

	for i := 0; i < 8; i++ {
		sc := NewTrainingScenario()
		if sc.Player == nil {
			t.Fatal("missing player")
		}
		if d := MinDistToRoutesYd(sc.Player.X, sc.Player.Y, sc.Routes); d < 500 || d > 3000 {
			t.Fatalf("attempt %d: player–route=%.0f want ≤3000 yd", i, d)
		}
		var fox, grisha *Entity
		civs := 0
		for _, e := range sc.Entities {
			if e == nil {
				continue
			}
			if e.RouteID == "" {
				t.Fatalf("%s has no route", e.ID)
			}
			switch e.ID {
			case "enemy_foxtrot":
				fox = e
			case "enemy_grisha":
				grisha = e
			case "civ_merchant", "civ_tanker", "civ_trawler":
				civs++
			}
		}
		if fox == nil || grisha == nil || civs != 3 {
			t.Fatalf("units missing fox=%v grisha=%v civs=%d", fox != nil, grisha != nil, civs)
		}
		if fox.Defcon != DefconPassive || grisha.Defcon != DefconPassive {
			t.Fatal("hostiles should start passive")
		}
		if fox.CrewSkill < 20 || fox.CrewSkill > 40 {
			t.Fatalf("foxtrot crew skill %.1f want 30±10", fox.CrewSkill)
		}
		if grisha.CrewSkill < 40 || grisha.CrewSkill > 80 {
			t.Fatalf("grisha crew skill %.1f want 60±20", grisha.CrewSkill)
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
