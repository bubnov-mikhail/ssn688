package render

import "testing"

func TestBathyColorDeepPlateau(t *testing.T) {
	deep := BathyColor(15000)
	at := BathyColor(12000)
	mid := BathyColor(2000)
	shallow := BathyColor(50)
	land := BathyColor(-5)
	if deep != at {
		t.Fatalf(">=12000ft should match deep navy: %v vs %v", deep, at)
	}
	if mid == deep {
		t.Fatalf("2000ft should still be on the ramp, not deep plateau: %v", mid)
	}
	if shallow == deep || shallow == land {
		t.Fatalf("shallow should differ from deep/land")
	}
	if land != BathyLandColor {
		t.Fatalf("land color=%v want %v", land, BathyLandColor)
	}
}

func TestBakeBathyRGBABlursLandAndWater(t *testing.T) {
	// 4x4: land left half, mid water right half.
	w, h := 4, 4
	depths := make([]float32, w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if i < 2 {
				depths[j*w+i] = -10
			} else {
				depths[j*w+i] = 500
			}
		}
	}
	pix := BakeBathyRGBA(w, h, depths)
	if len(pix) != w*h*4 {
		t.Fatalf("len=%d", len(pix))
	}
	// Shore-adjacent land should no longer be pure beige after full-chart blur.
	landOff := (1*w + 1) * 4
	if pix[landOff] == BathyLandColor.R && pix[landOff+1] == BathyLandColor.G && pix[landOff+2] == BathyLandColor.B {
		t.Fatalf("shore-adjacent land should be softened by blur")
	}
	// Shore-adjacent water should differ from unblurred water color.
	water := BathyColor(500)
	waterOff := (1*w + 2) * 4
	if pix[waterOff] == water.R && pix[waterOff+1] == water.G && pix[waterOff+2] == water.B {
		t.Fatalf("shore-adjacent water should be softened by blur")
	}
	samp := SampleBathyChart(pix, w, h, depths, 0, 0, 1, 1.5, 1.5)
	if samp.A != 255 {
		t.Fatalf("sample alpha=%d", samp.A)
	}
}
