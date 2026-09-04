package world

import "testing"

func testCoastBathy(t *testing.T) Bathymetry {
	t.Helper()
	const w, h = 24, 12
	depths := make([]float32, w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if i < 8 {
				depths[j*w+i] = -10
			} else {
				depths[j*w+i] = 120
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

func TestDistanceToShoreYd_Coast(t *testing.T) {
	b := testCoastBathy(t)
	// Water column i=8 starts at x=800; ship at x=950 is ~150 yd from blocked water.
	d := b.DistanceToShoreYd(950, 600)
	if d < 80 || d > 220 {
		t.Fatalf("distance to shore = %.0f yd, want ~80–220", d)
	}
	if b.DistanceToShoreYd(400, 600) > 50 {
		t.Fatalf("on land should read near zero, got %.0f", b.DistanceToShoreYd(400, 600))
	}
}

func TestNearestShoreBearingDeg(t *testing.T) {
	b := testCoastBathy(t)
	brg := b.NearestShoreBearingDeg(950, 600)
	// Land is to the west (~270°).
	if brg < 240 || brg > 300 {
		t.Fatalf("nearest shore bearing %.0f, want ~270", brg)
	}
}

func openOceanBathy() Bathymetry {
	const w, h = 40, 40
	depths := make([]float32, w*h)
	for i := range depths {
		depths[i] = 2000
	}
	return Bathymetry{
		Width: w, Height: h,
		OriginX: 0, OriginY: 0,
		CellSize: 100,
		Depths:   depths,
	}
}

func TestDistanceToShoreYd_ChartEdge(t *testing.T) {
	b := openOceanBathy()
	// Near east edge (maxX=4000); no land — edge should read as shore.
	d := b.DistanceToShoreYd(3700, 2000)
	if d < 200 || d > 450 {
		t.Fatalf("distance to chart edge = %.0f yd, want ~300", d)
	}
	if !b.IsShoreBlocked(4100, 2000) {
		t.Fatal("off-chart must count as shore-blocked")
	}
	if b.IsShoreBlocked(2000, 2000) {
		t.Fatal("open ocean cell should not be shore-blocked")
	}
}

func TestNearestShoreBearingDeg_ChartEdge(t *testing.T) {
	b := openOceanBathy()
	brg := b.NearestShoreBearingDeg(3700, 2000)
	// East edge is ~90°.
	if brg < 60 || brg > 120 {
		t.Fatalf("nearest chart-edge bearing %.0f, want ~90", brg)
	}
}
