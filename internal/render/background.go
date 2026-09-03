package render

import (
	"bytes"
	"image"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/bubnov-mikhail/ssn688/assets"
)

var (
	menuBGOnce      sync.Once
	menuBGImage     *ebiten.Image
	menuBGErr       error
	menuOverlayOnce sync.Once
	menuOverlay     *ebiten.Image
	menuBGFrameOnce sync.Once
	menuBGFrame     *ebiten.Image
)

type coverSlotKey struct {
	srcKey string
	w, h   int
}

// DrawScenarioCoverBytes draws cover image from raw bytes (JSON scenarios).
func DrawScenarioCoverBytes(screen *ebiten.Image, cacheKey string, data []byte, x, y, w, h int) {
	EnsureScenarioCoverImage(cacheKey, data)
	DrawScenarioCoverImage(screen, cacheKey, x, y, w, h)
}

// EnsureScenarioCoverImage decodes and caches a scenario/mission image once.
func EnsureScenarioCoverImage(cacheKey string, data []byte) {
	if cacheKey == "" || len(data) == 0 {
		return
	}
	scenarioCoverMu.Lock()
	defer scenarioCoverMu.Unlock()
	if _, ok := scenarioCoverCache[cacheKey]; ok {
		return
	}
	dec, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return
	}
	scenarioCoverCache[cacheKey] = ebiten.NewImageFromImage(dec)
	invalidateCoverSlotsLocked(cacheKey)
}

// DrawScenarioCoverImage draws a previously cached scenario/mission image.
func DrawScenarioCoverImage(screen *ebiten.Image, cacheKey string, x, y, w, h int) {
	if cacheKey == "" {
		FillRect(screen, x, y, w, h, ColorPanelInset)
		DrawText(screen, "NO ART", x+w/2-28, y+h/2, ColorDim, true)
		return
	}
	scenarioCoverMu.Lock()
	img, ok := scenarioCoverCache[cacheKey]
	scenarioCoverMu.Unlock()
	if !ok || img == nil {
		FillRect(screen, x, y, w, h, ColorPanelInset)
		DrawText(screen, "NO ART", x+w/2-28, y+h/2, ColorDim, true)
		return
	}
	drawCoverImage(screen, img, cacheKey, x, y, w, h)
}

// ClearScenarioCoverCache drops decoded scenario/mission textures (after scenario reload).
func ClearScenarioCoverCache() {
	scenarioCoverMu.Lock()
	defer scenarioCoverMu.Unlock()
	for k, img := range scenarioCoverCache {
		if img != nil {
			img.Dispose()
		}
		delete(scenarioCoverCache, k)
	}
	clearCoverSlotsLocked()
}

func drawCoverImage(screen *ebiten.Image, img *ebiten.Image, srcKey string, x, y, w, h int) {
	slot := ensureCoverSlot(img, srcKey, w, h)
	if slot == nil {
		return
	}
	var opts ebiten.DrawImageOptions
	opts.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(slot, &opts)
	drawRectOutline(screen, x, y, w, h, ColorPanelStroke)
}

func ensureCoverSlot(src *ebiten.Image, srcKey string, w, h int) *ebiten.Image {
	if src == nil || srcKey == "" || w < 1 || h < 1 {
		return nil
	}
	sk := coverSlotKey{srcKey: srcKey, w: w, h: h}
	scenarioCoverMu.Lock()
	defer scenarioCoverMu.Unlock()
	if slot, ok := coverSlotCache[sk]; ok && slot != nil {
		return slot
	}
	slot := ebiten.NewImage(w, h)
	slot.Fill(ColorPanelInset)
	bw, bh := src.Bounds().Dx(), src.Bounds().Dy()
	if bw > 0 && bh > 0 {
		scale := mathMin(float64(w)/float64(bw), float64(h)/float64(bh))
		dw := float64(bw) * scale
		dh := float64(bh) * scale
		ox := (float64(w) - dw) / 2
		oy := (float64(h) - dh) / 2
		var opts ebiten.DrawImageOptions
		opts.GeoM.Scale(scale, scale)
		opts.GeoM.Translate(ox, oy)
		opts.Filter = ebiten.FilterLinear
		slot.DrawImage(src, &opts)
	}
	if old, ok := coverSlotCache[sk]; ok && old != nil {
		old.Dispose()
	}
	coverSlotCache[sk] = slot
	return slot
}

func invalidateCoverSlotsLocked(srcKey string) {
	for k, img := range coverSlotCache {
		if k.srcKey != srcKey {
			continue
		}
		if img != nil {
			img.Dispose()
		}
		delete(coverSlotCache, k)
	}
}

func clearCoverSlotsLocked() {
	for k, img := range coverSlotCache {
		if img != nil {
			img.Dispose()
		}
		delete(coverSlotCache, k)
	}
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
	coverSlotCache     = map[coverSlotKey]*ebiten.Image{}
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

func menuBackgroundFrame() *ebiten.Image {
	menuBGFrameOnce.Do(func() {
		bg, err := MenuBackground()
		if err != nil || bg == nil {
			return
		}
		frame := ebiten.NewImage(ScreenW, ScreenH)
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
		frame.DrawImage(bg, &opts)
		var tint ebiten.DrawImageOptions
		tint.ColorScale.Scale(1, 1, 1, 0.48)
		frame.DrawImage(menuDarkOverlay(), &tint)
		menuBGFrame = frame
	})
	return menuBGFrame
}

// DrawMenuBackground scales and draws the submarine photo with a dark overlay.
func DrawMenuBackground(screen *ebiten.Image) {
	frame := menuBackgroundFrame()
	if frame == nil {
		screen.Fill(ColorBG)
		return
	}
	screen.DrawImage(frame, nil)
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
