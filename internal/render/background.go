package render

import (
	"bytes"
	"image"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ssn688/sim/assets"
)

var (
	menuBGOnce      sync.Once
	menuBGImage     *ebiten.Image
	menuBGErr       error
	menuOverlayOnce sync.Once
	menuOverlay     *ebiten.Image
)

// ScenarioCover returns a cached cover image from assets/scenarios/.
func ScenarioCover(name string) (*ebiten.Image, error) {
	scenarioCoverMu.Lock()
	defer scenarioCoverMu.Unlock()
	if img, ok := scenarioCoverCache[name]; ok {
		return img, nil
	}
	data, err := assets.ScenarioCovers.ReadFile(name)
	if err != nil {
		return nil, err
	}
	dec, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	img := ebiten.NewImageFromImage(dec)
	scenarioCoverCache[name] = img
	return img, nil
}

// DrawScenarioCover draws a cover image letterboxed into the given rect.
func DrawScenarioCover(screen *ebiten.Image, name string, x, y, w, h int) {
	img, err := ScenarioCover(name)
	if err != nil || img == nil {
		FillRect(screen, x, y, w, h, ColorPanelInset)
		DrawText(screen, "NO ART", x+w/2-28, y+h/2, ColorDim, true)
		return
	}
	bw, bh := img.Bounds().Dx(), img.Bounds().Dy()
	if bw <= 0 || bh <= 0 {
		return
	}
	FillRect(screen, x, y, w, h, ColorPanelInset)
	scale := mathMin(float64(w)/float64(bw), float64(h)/float64(bh))
	dw := float64(bw) * scale
	dh := float64(bh) * scale
	ox := float64(x) + (float64(w)-dw)/2
	oy := float64(y) + (float64(h)-dh)/2
	var opts ebiten.DrawImageOptions
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(ox, oy)
	opts.Filter = ebiten.FilterLinear
	screen.DrawImage(img, &opts)
	// Frame around the slot — not over the photo (avoids gray stripe on cover-scale bleed).
	drawRectOutline(screen, x, y, w, h, ColorPanelStroke)
}

func drawRectOutline(screen *ebiten.Image, x, y, w, h int, clr color.Color) {
	if w < 2 || h < 2 {
		return
	}
	DrawLine(screen, float64(x), float64(y), float64(x+w-1), float64(y), clr)
	DrawLine(screen, float64(x), float64(y+h-1), float64(x+w-1), float64(y+h-1), clr)
	DrawLine(screen, float64(x), float64(y), float64(x), float64(y+h-1), clr)
	DrawLine(screen, float64(x+w-1), float64(y), float64(x+w-1), float64(y+h-1), clr)
}

var (
	scenarioCoverMu    sync.Mutex
	scenarioCoverCache = map[string]*ebiten.Image{}
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

func mathMin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
