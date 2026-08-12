package navicon

import "testing"

func TestIconsNormalizedFootprint(t *testing.T) {
	const wantFill = 1 - 2*normPad
	const tol = 0.04
	for _, name := range Names {
		img := Raster(name, DesignSize)
		minX, minY, maxX, maxY := opaqueBounds(img)
		if maxX < minX {
			t.Fatalf("%s: no opaque pixels", name)
		}
		px := img.Bounds().Dx()
		fillW := float64(maxX-minX+1) / float64(px)
		fillH := float64(maxY-minY+1) / float64(px)
		fillMax := fillW
		if fillH > fillMax {
			fillMax = fillH
		}
		if diff := abs(fillMax - wantFill); diff > tol {
			t.Fatalf("%s max fill %.2f, want ~%.2f (w=%.2f h=%.2f)", name, fillMax, wantFill, fillW, fillH)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
