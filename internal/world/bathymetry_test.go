package world

import (
	"testing"

	"github.com/ssn688/sim/assets"
)

func TestBathymetryLoadAndCenterWater(t *testing.T) {
	b, err := LoadBathymetry(assets.BathyChart)
	if err != nil {
		t.Fatal(err)
	}
	SetDefaultBathymetry(b)
	if b.DepthAtFt(0, 0) <= 0 {
		t.Fatalf("chart center should be water, depth=%.0f", b.DepthAtFt(0, 0))
	}
	if !b.NavigableFor(0, 0, KindSubmarine, 180) {
		t.Fatalf("center should be navigable for SSN at 180 ft, depth=%.0f", b.DepthAtFt(0, 0))
	}
	// Coast should appear somewhere on the mission chart.
	land := 0
	for _, d := range b.Depths {
		if d <= 0 {
			land++
		}
	}
	if land < 1000 {
		t.Fatalf("expected coastline land cells, got %d", land)
	}
	sc := NewTrainingScenario()
	if sc.Bathy == nil || !sc.Bathy.Valid() {
		t.Fatal("scenario missing bathymetry")
	}
	for _, e := range sc.AllEntities() {
		if !sc.Bathy.NavigableFor(e.X, e.Y, e.Kind, e.DepthFt) {
			t.Fatalf("%s spawned in invalid water at (%.0f,%.0f) depth chart=%.0f", e.ID, e.X, e.Y, sc.Bathy.DepthAtFt(e.X, e.Y))
		}
	}
}
