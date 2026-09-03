package world

import "testing"

func testIslandBathy(t *testing.T) Bathymetry {
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
	return Bathymetry{
		Width: w, Height: h,
		OriginX: 0, OriginY: 0,
		CellSize: 100,
		Depths:   depths,
	}
}

func TestAcousticPathBlocked_CrossesIsland(t *testing.T) {
	b := testIslandBathy(t)
	if !b.AcousticPathBlocked(500, 500, 2500, 500) {
		t.Fatal("path across central island should be blocked")
	}
}

func TestAcousticPathBlocked_SameSideOpenWater(t *testing.T) {
	b := testIslandBathy(t)
	if b.AcousticPathBlocked(500, 500, 700, 500) {
		t.Fatal("path on west side should stay in water")
	}
	if b.AcousticPathBlocked(2200, 500, 2500, 500) {
		t.Fatal("path on east side should stay in water")
	}
}

func TestHorizonBlockedMatchesAcousticPath(t *testing.T) {
	b := testIslandBathy(t)
	if !b.HorizonBlocked(500, 500, 2500, 500) {
		t.Fatal("island should block horizon")
	}
	if b.HorizonBlocked(500, 500, 700, 500) {
		t.Fatal("open water should not block")
	}
}

func TestAcousticPathBlocked_NoChart(t *testing.T) {
	var b Bathymetry
	if b.AcousticPathBlocked(0, 0, 5000, 0) {
		t.Fatal("invalid chart must not block")
	}
}
