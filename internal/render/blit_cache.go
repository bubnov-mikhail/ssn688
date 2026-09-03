package render

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

type blitSlotKey struct {
	key        string
	dstW, dstH int
}

const blitSlotMax = 8

var (
	blitSlotMu    sync.Mutex
	blitSlotCache = map[blitSlotKey]*ebiten.Image{}
)

// ClearBlitCache drops cached scaled blit targets (session / screen teardown).
func ClearBlitCache() {
	blitSlotMu.Lock()
	defer blitSlotMu.Unlock()
	for k, img := range blitSlotCache {
		if img != nil {
			img.Dispose()
		}
		delete(blitSlotCache, k)
	}
}

// DrawImageAt blits src at 1:1 into screen at (x, y).
func DrawImageAt(screen *ebiten.Image, src *ebiten.Image, x, y int) {
	if screen == nil || src == nil {
		return
	}
	var opts ebiten.DrawImageOptions
	opts.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(src, &opts)
}

// DrawImageSlot draws src scaled to dstW×dstH using a cached offscreen slot.
// Scale/filter is baked once per cacheKey+size — avoids per-frame Metal leaks on macOS.
func DrawImageSlot(screen *ebiten.Image, cacheKey string, src *ebiten.Image, dstX, dstY, dstW, dstH int) {
	if screen == nil || src == nil || cacheKey == "" || dstW < 1 || dstH < 1 {
		return
	}
	slot := ensureBlitSlot(cacheKey, src, dstW, dstH, false)
	if slot == nil {
		return
	}
	var opts ebiten.DrawImageOptions
	opts.GeoM.Translate(float64(dstX), float64(dstY))
	screen.DrawImage(slot, &opts)
}

// RefreshBlitSlot rebuilds the cached scaled image for cacheKey (same dst size).
// Use when src pixels changed but the cache key must stay stable (PLOT pan/zoom).
func RefreshBlitSlot(cacheKey string, src *ebiten.Image, dstW, dstH int) {
	if src == nil || cacheKey == "" || dstW < 1 || dstH < 1 {
		return
	}
	ensureBlitSlot(cacheKey, src, dstW, dstH, true)
}

func ensureBlitSlot(cacheKey string, src *ebiten.Image, dstW, dstH int, refresh bool) *ebiten.Image {
	sk := blitSlotKey{key: cacheKey, dstW: dstW, dstH: dstH}
	blitSlotMu.Lock()
	defer blitSlotMu.Unlock()
	if !refresh {
		if slot, ok := blitSlotCache[sk]; ok && slot != nil {
			return slot
		}
	}
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw < 1 || sh < 1 {
		return nil
	}
	slot := blitSlotCache[sk]
	if slot == nil || slot.Bounds().Dx() != dstW || slot.Bounds().Dy() != dstH {
		if slot != nil {
			slot.Dispose()
		}
		slot = ebiten.NewImage(dstW, dstH)
		blitSlotCache[sk] = slot
	} else {
		slot.Clear()
	}
	scaleX := float64(dstW) / float64(sw)
	scaleY := float64(dstH) / float64(sh)
	var opts ebiten.DrawImageOptions
	opts.GeoM.Scale(scaleX, scaleY)
	opts.Filter = ebiten.FilterNearest
	slot.DrawImage(src, &opts)
	evictBlitSlotsLocked(sk)
	return slot
}

func evictBlitSlotsLocked(keep blitSlotKey) {
	for len(blitSlotCache) > blitSlotMax {
		removed := false
		for k, img := range blitSlotCache {
			if k == keep {
				continue
			}
			if img != nil {
				img.Dispose()
			}
			delete(blitSlotCache, k)
			removed = true
			break
		}
		if !removed {
			return
		}
	}
}
