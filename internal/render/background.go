package render

import (
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ssn688/sim/assets"
)

var (
	menuBGOnce    sync.Once
	menuBGImage   *ebiten.Image
	menuBGErr     error
	menuOverlayOnce sync.Once
	menuOverlay   *ebiten.Image
)

// MenuBackground returns the cached title-screen background image.
func MenuBackground() (*ebiten.Image, error) {
	menuBGOnce.Do(func() {
		img, _, err := image.Decode(bytes.NewReader(assets.MenuBG))
		if err != nil {
			menuBGErr = err
			return
		}
		menuBGImage = ebiten.NewImageFromImage(img)
	})
	return menuBGImage, menuBGErr
}

func menuDarkOverlay() *ebiten.Image {
	menuOverlayOnce.Do(func() {
		menuOverlay = ebiten.NewImage(ScreenW, ScreenH)
		menuOverlay.Fill(color.RGBA{4, 14, 22, 255})
	})
	return menuOverlay
}

// DrawMenuBackground scales and draws the submarine photo with a dark overlay.
func DrawMenuBackground(screen *ebiten.Image) {
	bg, err := MenuBackground()
	if err != nil || bg == nil {
		screen.Fill(ColorBG)
		return
	}

	bw, bh := bg.Bounds().Dx(), bg.Bounds().Dy()
	scale := mathMax(float64(ScreenW)/float64(bw), float64(ScreenH)/float64(bh))
	dw := float64(bw) * scale
	dh := float64(bh) * scale
	ox := (float64(ScreenW) - dw) / 2
	oy := (float64(ScreenH) - dh) / 2

	var opts ebiten.DrawImageOptions
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(ox, oy)
	opts.Filter = ebiten.FilterLinear
	screen.DrawImage(bg, &opts)

	var tint ebiten.DrawImageOptions
	tint.ColorScale.Scale(1, 1, 1, 0.48)
	screen.DrawImage(menuDarkOverlay(), &tint)
}

func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
