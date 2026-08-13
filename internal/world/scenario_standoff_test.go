package world

import (
	"math"
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
		var fox, grisha, allyDD, allySS *Entity
		civs := 0
		for _, e := range sc.Entities {
			if e == nil {
				continue
			}
			switch e.ID {
			case "enemy_foxtrot":
				fox = e
			case "enemy_grisha":
				grisha = e
			case "ally_spruance":
				allyDD = e
			case "ally_688":
				allySS = e
			case "civ_merchant", "civ_tanker", "civ_trawler":
				civs++
			}
			// Non-ally combatants and traffic need assigned transit / edge routes.
			if e.Side != SidePlayer && e.RouteID == "" {
				t.Fatalf("%s has no route", e.ID)
			}
			if (e.ID == "ally_spruance" || e.ID == "ally_688") && e.RouteID == "" {
				t.Fatalf("%s has no route", e.ID)
			}
		}
		if fox == nil || grisha == nil || civs != 3 {
			t.Fatalf("units missing fox=%v grisha=%v civs=%d", fox != nil, grisha != nil, civs)
		}
		if allyDD == nil || allySS == nil {
			t.Fatal("missing ally Spruance / ally 688")
		}
		if fox.Defcon != DefconPassive || grisha.Defcon != DefconPassive {
			t.Fatal("hostiles should start passive")
		}
		if allyDD.Defcon != DefconHostile || allySS.Defcon != DefconHostile {
			t.Fatalf("allies want DEFCON 2, got dd=%d ss=%d", allyDD.Defcon, allySS.Defcon)
		}
		minX, minY, maxX, maxY := sc.Bathy.BoundsYards()
		seX, seY := maxX-1800, minY+1800
		for _, a := range []*Entity{allyDD, allySS} {
			if a.RouteID != "route_ally_edge" {
				t.Fatalf("%s route=%q want route_ally_edge", a.ID, a.RouteID)
			}
			if a.Side != SidePlayer {
				t.Fatalf("%s side %v", a.ID, a.Side)
			}
			// Near SE (lower-right), not stacked on ownship at SW.
			dSE := math.Hypot(a.X-seX, a.Y-seY)
			dSW := math.Hypot(a.X-(minX+1800), a.Y-(minY+1800))
			if dSE > dSW {
				t.Fatalf("%s not near SE: se=%.0f sw=%.0f", a.ID, dSE, dSW)
			}
			if dSE > 12000 {
				t.Fatalf("%s too far from SE corner: %.0f yd", a.ID, dSE)
			}
		}
		_ = maxY
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
