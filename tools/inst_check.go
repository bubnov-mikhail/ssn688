//go:build ignore

package main

import (
	"fmt"
	"math"
	"os"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
)

func main() {
	data, _ := os.ReadFile("scenarios_generated/taiwan_formosa_watch.json")
	sc, err := campaign.ParseScenarioJSON(data, "test")
	if err != nil {
		panic(err)
	}
	var m *campaign.MissionDef
	for i := range sc.Missions {
		if sc.Missions[i].ID == "tw_twin_exercises" {
			m = &sc.Missions[i]
			break
		}
	}
	rt := campaign.Instantiate(&sc, m, campaign.BuildContext{})
	if rt == nil {
		panic("instantiate failed")
	}
	var spr, hulk *struct{ id string; x, y float64 }
	for _, e := range rt.AllEntities() {
		if e == nil {
			continue
		}
		fmt.Printf("%s %.0f,%.0f\n", e.ID, e.X, e.Y)
		switch e.ID {
		case "ally_spruance":
			spr = &struct{ id string; x, y float64 }{e.ID, e.X, e.Y}
		case "ex_hulk_a":
			hulk = &struct{ id string; x, y float64 }{e.ID, e.X, e.Y}
		}
	}
	p := rt.Player
	if p != nil && spr != nil && hulk != nil {
		fmt.Printf("spr->hulk %.0f player->spr %.0f CPA %.0f\n",
			math.Hypot(hulk.x-spr.x, hulk.y-spr.y),
			math.Hypot(p.X-spr.x, p.Y-spr.y),
			distPointToSeg(p.X, p.Y, spr.x, spr.y, hulk.x, hulk.y))
	}
}

func distPointToSeg(px, py, x0, y0, x1, y1 float64) float64 {
	dx, dy := x1-x0, y1-y0
	if dx == 0 && dy == 0 {
		return math.Hypot(px-x0, py-y0)
	}
	t := ((px-x0)*dx + (py-y0)*dy) / (dx*dx + dy*dy)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(px-(x0+t*dx), py-(y0+t*dy))
}
