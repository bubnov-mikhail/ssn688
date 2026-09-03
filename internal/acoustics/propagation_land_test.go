package acoustics

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func testIslandBathyModel(t *testing.T) (Model, world.Bathymetry) {
	t.Helper()
	const w, h = 30, 10
	depths := make([]float32, w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if i >= 10 && i <= 18 {
				depths[j*w+i] = -10
			} else {
				depths[j*w+i] = 200
			}
		}
	}
	bathy := world.Bathymetry{
		Width: w, Height: h,
		OriginX: 0, OriginY: 0,
		CellSize: 100,
		Depths:   depths,
	}
	m := NewModel(DefaultEnvironment())
	m.Bathy = &bathy
	return m, bathy
}

func TestDetectBlockedThroughLand(t *testing.T) {
	model, _ := testIslandBathyModel(t)
	listener := &world.Entity{
		ID: "player", SignatureID: "los_angeles", Kind: world.KindSubmarine, Status: world.StatusActive,
		X: 500, Y: 500, DepthFt: 240, SpeedKts: 5,
	}
	target := &world.Entity{
		ID: "far", SignatureID: "udaloy", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 2500, Y: 500, SpeedKts: 14,
	}
	blocked := model.Detect(listener, target, ModePassive, 0)
	if blocked.PeakSNR > -30 {
		t.Fatalf("land-blocked path should be silent, PeakSNR=%.1f", blocked.PeakSNR)
	}
	if blocked.Detected {
		t.Fatal("must not detect through island")
	}

	target.X = 700
	clear := model.Detect(listener, target, ModePassive, 0)
	if clear.PeakSNR <= blocked.PeakSNR+10 {
		t.Fatalf("open-water path should be louder: clear=%.1f blocked=%.1f", clear.PeakSNR, blocked.PeakSNR)
	}
}

func TestDetectBlockedThroughCatalinaLand(t *testing.T) {
	campaign.ReloadScenarios()
	bathy := campaign.ResolveMissionBathy(campaign.DemoScenarioID, campaign.DemoMissionTraining)
	if bathy == nil || !bathy.Valid() {
		t.Fatal("demo bathy missing")
	}
	model := NewModel(DefaultEnvironment())
	model.Bathy = bathy

	// West of Catalina island (water) listening to a contact east of the island.
	listener := &world.Entity{
		ID: "player", SignatureID: "los_angeles", Kind: world.KindSubmarine, Status: world.StatusActive,
		X: -8000, Y: 0, DepthFt: 240, SpeedKts: 5,
	}
	target := &world.Entity{
		ID: "east", SignatureID: "tanker", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 12000, Y: 0, SpeedKts: 12,
	}
	if !bathy.AcousticPathBlocked(listener.X, listener.Y, target.X, target.Y) {
		t.Fatal("Catalina demo path should cross land")
	}
	r := model.Detect(listener, target, ModePassive, 0)
	if r.Detected {
		t.Fatalf("must not detect through Catalina, PeakSNR=%.1f", r.PeakSNR)
	}
}
