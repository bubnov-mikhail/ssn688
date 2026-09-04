package appicon

import (
	"bytes"
	"image"
	"image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/bubnov-mikhail/ssn688/assets"
	"golang.org/x/image/draw"

	_ "image/jpeg"
)

// sizes for taskbar / window chrome (GLFW picks closest).
var windowIconSizes = []int{16, 32, 48, 128, 256}

// Decode returns the master icon image.
func Decode() (image.Image, error) {
	return png.Decode(bytes.NewReader(assets.AppIconPNG))
}

// WindowImages returns resized candidates for ebiten.SetWindowIcon.
func WindowImages() []image.Image {
	src, err := Decode()
	if err != nil {
		log.Printf("appicon: decode: %v", err)
		return nil
	}
	out := make([]image.Image, 0, len(windowIconSizes))
	for _, sz := range windowIconSizes {
		out = append(out, resizeNRGBA(src, sz, sz))
	}
	return out
}

// ApplyWindowIcon sets the OS window / taskbar icon (no-op effect on macOS).
func ApplyWindowIcon() {
	imgs := WindowImages()
	if len(imgs) == 0 {
		return
	}
	ebiten.SetWindowIcon(imgs)
}

func resizeNRGBA(src image.Image, w, h int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}
