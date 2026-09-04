package appicon

import "testing"

func TestDecodeIcon(t *testing.T) {
	img, err := Decode()
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() < 64 || b.Dy() < 64 {
		t.Fatalf("icon too small: %dx%d", b.Dx(), b.Dy())
	}
	imgs := WindowImages()
	if len(imgs) != len(windowIconSizes) {
		t.Fatalf("got %d sizes, want %d", len(imgs), len(windowIconSizes))
	}
}
