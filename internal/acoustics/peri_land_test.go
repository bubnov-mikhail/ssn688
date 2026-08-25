package acoustics

import (
	"math"
	"testing"

	"github.com/ssn688/sim/internal/campaign"
	"github.com/ssn688/sim/internal/world"
)

// TestLandRayHitsCatalinaNearOrigin verifies the demo scenario chart has dry land
// within a few miles of the chart origin (needed for peri IR coast columns).
func TestLandRayHitsCatalinaNearOrigin(t *testing.T) {
	campaign.ReloadScenarios()
	bathy := campaign.ResolveMissionBathy(campaign.DemoScenarioID, campaign.DemoMissionTraining)
	if bathy == nil || !bathy.Valid() {
		t.Fatal("demo scenario missing bathymetry")
	}
	b := *bathy
	found := false
	maxR := 8 * world.YardsPerNM
	for brg := 0; brg < 360; brg += 5 {
		rad := float64(brg) * math.Pi / 180
		sx, sy := math.Sin(rad), math.Cos(rad)
		for r := 200.0; r <= maxR; r += 150 {
			if b.IsLand(sx*r, sy*r) {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("expected land near Catalina chart origin")
	}
}
