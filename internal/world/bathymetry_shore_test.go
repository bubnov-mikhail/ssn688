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
