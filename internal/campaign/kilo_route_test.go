package campaign

import (
	"os"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestTwAttributionKiloRouteLegsNavigable(t *testing.T) {
	path := "../../scenarios_generated/taiwan_formosa_watch.json"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip(err)
	}
	sc, err := ParseScenarioJSON(data, path)
	if err != nil {
		t.Fatal(err)
	}
	m := FindMission(&sc, "tw_attribution")
	if m == nil {
		t.Fatal("mission missing")
	}
	bathy := TheaterChart(&sc, m.TheaterID)
	routes, _ := RuntimeRoutes(m.Routes)
	var r *world.Route
	for _, rt := range routes {
		if rt.ID == "route_rf_kilo" {
			r = rt
			break
		}
	}
	if r == nil {
		t.Fatal("route_rf_kilo missing")
	}
	const subDepth = 160.0
	for i := 0; i < len(r.Waypoints)-1; i++ {
		a, b := r.Waypoints[i], r.Waypoints[i+1]
		if legCrossesBlocked(bathy, a.X, a.Y, b.X, b.Y, subDepth) {
			t.Fatalf("leg %d->%d crosses land/shallow", i, i+1)
		}
	}
}

func legCrossesBlocked(bathy *world.Bathymetry, x0, y0, x1, y1, depthFt float64) bool {
	if bathy == nil {
		return false
	}
	steps := int(dist2d(x0, y0, x1, y1)/200) + 1
	for s := 1; s < steps; s++ {
		t := float64(s) / float64(steps)
		x := x0 + (x1-x0)*t
		y := y0 + (y1-y0)*t
		if bathy.IsShoreBlocked(x, y) || !bathy.NavigableFor(x, y, world.KindSubmarine, depthFt) {
			return true
		}
	}
	return false
}

func dist2d(x0, y0, x1, y1 float64) float64 {
	dx, dy := x1-x0, y1-y0
	return dx*dx + dy*dy
}
