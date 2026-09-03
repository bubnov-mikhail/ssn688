package ui

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"sync"

	"github.com/bubnov-mikhail/ssn688/assets"
)

const (
	periSpriteAspectStep = 1
	periSpriteAspectMax  = 180
	periSpriteAlphaMin   = uint8(10) // below this = empty background
)

type periShipSprite struct {
	pix  []uint8
	w, h int
	// Content bbox inside the PNG (tight crop).
	x0, y0, x1, y1 int
}

var (
	periShipSpriteOnce sync.Once
	periShipSpriteMap  map[string]*periShipSprite // "merchant_090"
)

func periShipClassName(c periShipClass) string {
	switch c {
	case periClassTanker:
		return "tanker"
	case periClassFishing:
		return "fishing"
	case periClassCombatant:
		return "combatant"
	default:
		return "merchant"
	}
}

func ensurePeriShipSprites() {
	periShipSpriteOnce.Do(func() {
		periShipSpriteMap = make(map[string]*periShipSprite, 750)
		classes := []string{"merchant", "tanker", "fishing", "combatant"}
		for _, cls := range classes {
			for aspect := 0; aspect <= periSpriteAspectMax; aspect += periSpriteAspectStep {
				key := fmt.Sprintf("%s_%03d", cls, aspect)
				path := "peri_ships/" + key + ".png"
				f, err := assets.PeriShipSprites.Open(path)
				if err != nil {
					continue
				}
				img, err := png.Decode(f)
				_ = f.Close()
				if err != nil {
					continue
				}
				sp := graySpriteFromImage(img)
				if sp != nil {
					periShipSpriteMap[key] = sp
				}
			}
		}
	})
}

func graySpriteFromImage(img image.Image) *periShipSprite {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 2 || h < 2 {
		return nil
	}
	pix := make([]uint8, w*h)
	x0, y0, x1, y1 := w, h, 0, 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			// 16-bit channels → 8-bit luma
			v := uint8(((r + g + bl) / 3) >> 8)
			pix[y*w+x] = v
			if v >= periSpriteAlphaMin {
				if x < x0 {
					x0 = x
				}
				if y < y0 {
					y0 = y
				}
				if x >= x1 {
					x1 = x + 1
				}
				if y >= y1 {
					y1 = y + 1
				}
			}
		}
	}
	if x1 <= x0 || y1 <= y0 {
		return &periShipSprite{pix: pix, w: w, h: h, x0: 0, y0: 0, x1: w, y1: h}
	}
	// Pad 1px so bloom edges aren't clipped hard.
	if x0 > 0 {
		x0--
	}
	if y0 > 0 {
		y0--
	}
	if x1 < w {
		x1++
	}
	if y1 < h {
		y1++
	}
	return &periShipSprite{pix: pix, w: w, h: h, x0: x0, y0: y0, x1: x1, y1: y1}
}

func pickPeriShipSprite(class periShipClass, aspectDeg float64) *periShipSprite {
	ensurePeriShipSprites()
	if aspectDeg < 0 {
		aspectDeg = 0
	}
	if aspectDeg > float64(periSpriteAspectMax) {
		aspectDeg = float64(periSpriteAspectMax)
	}
	bin := int(aspectDeg/periSpriteAspectStep+0.5) * periSpriteAspectStep
	if bin > periSpriteAspectMax {
		bin = periSpriteAspectMax
	}
	key := fmt.Sprintf("%s_%03d", periShipClassName(class), bin)
	if sp := periShipSpriteMap[key]; sp != nil {
		return sp
	}
	// Fallback nearest available.
	for d := periSpriteAspectStep; d <= periSpriteAspectMax; d += periSpriteAspectStep {
		for _, a := range []int{bin - d, bin + d} {
			if a < 0 || a > periSpriteAspectMax {
				continue
			}
			k := fmt.Sprintf("%s_%03d", periShipClassName(class), a)
			if sp := periShipSpriteMap[k]; sp != nil {
				return sp
			}
		}
	}
	return nil
}

// periShipSpriteFitDest picks a uniform X/Y scale so the cropped sprite fits
// inside the optic box (projected length × air draft) without distorting aspect.
func periShipSpriteFitDest(sp *periShipSprite, boxW, boxH int) (destW, destH int) {
	if sp == nil {
		return 0, 0
	}
	sw := sp.x1 - sp.x0
	sh := sp.y1 - sp.y0
	if sw < 1 || sh < 1 {
		return 0, 0
	}
	if boxW < 2 {
		boxW = 2
	}
	if boxH < 2 {
		boxH = 2
	}
	scaleW := float64(boxW) / float64(sw)
	scaleH := float64(boxH) / float64(sh)
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}
	destW = int(float64(sw)*scale + 0.5)
	destH = int(float64(sh)*scale + 0.5)
	if destW < 2 {
		destW = 2
	}
	if destH < 2 {
		destH = 2
	}
	return destW, destH
}

// blitPeriShipSprite scales the cropped sprite into the optic frame with uniform
// X/Y scale (sprite content aspect preserved). boxW×boxH is the optic bound
// from projected length × air draft; the blit may be smaller on one axis.
// sinkFrac (0..1) lowers the sprite into the sea and clips pixels at waterY.
// Sprites are authored with bow toward −X (left); flipX puts bow toward +X.
// depth is a per-pixel closest-range buffer (0 = empty); nearer RangeYd wins.
// Returns the destination AABB of the *visible* stub (for bloom/fire).
func blitPeriShipSprite(pix []byte, depth []float32, frameW, frameH int, sp *periShipSprite, centerX float64, waterY, boxW, boxH int, flipX bool, brightness, sinkFrac, rangeYd float64) (dstX0, dstY0, dstX1, dstY1 int, ok bool) {
	if sp == nil || boxW < 2 {
		return 0, 0, 0, 0, false
	}
	sw := sp.x1 - sp.x0
	sh := sp.y1 - sp.y0
	if sw < 1 || sh < 1 {
		return 0, 0, 0, 0, false
	}
	if sinkFrac < 0 {
		sinkFrac = 0
	}
	if sinkFrac >= 1 {
		return 0, 0, 0, 0, false
	}
	destW, destH := periShipSpriteFitDest(sp, boxW, boxH)
	if destW < 2 || destH < 2 {
		return 0, 0, 0, 0, false
	}
	sinkPx := int(float64(destH)*sinkFrac + 0.5)
	if sinkPx >= destH {
		return 0, 0, 0, 0, false
	}
	// Fractional center: subpixel shift in source sampling softens crawl between IR columns.
	dstX0f := centerX - float64(destW)/2
	dstX0 = int(math.Floor(dstX0f))
	dstX1 = dstX0 + destW + 1
	subX := dstX0f - float64(dstX0) // [0,1)
	// Full sprite maps to [waterY-destH+sinkPx, waterY+sinkPx); clip at waterY.
	dstY0 = waterY - destH + sinkPx
	dstY1 = waterY
	if brightness < 0.05 {
		brightness = 0.05
	}
	if brightness > 1.4 {
		brightness = 1.4
	}
	rangeF := float32(rangeYd)
	if rangeF < 1 {
		rangeF = 1
	}

	visTop := waterY
	lit := false
	for dy := 0; dy < destH; dy++ {
		yy := dstY0 + dy
		if yy < 0 || yy >= frameH || yy >= waterY {
			continue
		}
		sy := sp.y0 + dy*sh/destH
		if sy >= sp.y1 {
			sy = sp.y1 - 1
		}
		for dx := 0; dx < destW; dx++ {
			// Sample with horizontal subpixel offset.
			sxOffF := float64(dx) + subX
			if flipX {
				sxOffF = float64(destW-1) - (float64(dx) + subX)
			}
			if sxOffF < 0 {
				sxOffF = 0
			}
			if sxOffF > float64(destW-1) {
				sxOffF = float64(destW - 1)
			}
			sx := sp.x0 + int(sxOffF*float64(sw)/float64(destW))
			if sx >= sp.x1 {
				sx = sp.x1 - 1
			}
			v := sp.pix[sy*sp.w+sx]
			if v < periSpriteAlphaMin {
				continue
			}
			xx := dstX0 + dx
			if xx < 0 || xx >= frameW {
				continue
			}
			if !periDepthTry(depth, frameW, xx, yy, rangeF) {
				continue
			}
			g := int(float64(v) * brightness)
			periSetGray(pix, frameW, xx, yy, uint8(min255(g)))
			lit = true
			if yy < visTop {
				visTop = yy
			}
		}
	}
	if !lit {
		return 0, 0, 0, 0, false
	}
	return dstX0, visTop, dstX0 + destW, dstY1, true
}

// periShipSpriteOpaqueAt mirrors blitPeriShipSprite's dest→source mapping:
// true only for opaque ship pixels of this aspect sprite (not sky/sea gaps).
// boxW/boxH are the same optic bounds passed to blit (uniform fit applied here).
func periShipSpriteOpaqueAt(sp *periShipSprite, centerX float64, waterY, boxW, boxH int, flipX bool, sinkFrac float64, xx, yy int) bool {
	if sp == nil || boxW < 2 {
		return false
	}
	destW, destH := periShipSpriteFitDest(sp, boxW, boxH)
	if destW < 2 || destH < 2 {
		return false
	}
	if sinkFrac < 0 {
		sinkFrac = 0
	}
	if sinkFrac >= 1 {
		return false
	}
	sinkPx := int(float64(destH)*sinkFrac + 0.5)
	if sinkPx >= destH {
		return false
	}
	if yy >= waterY {
		return false
	}
	dstX0f := centerX - float64(destW)/2
	dstX0 := int(math.Floor(dstX0f))
	subX := dstX0f - float64(dstX0)
	dstY0 := waterY - destH + sinkPx
	dx := xx - dstX0
	dy := yy - dstY0
	if dx < 0 || dx >= destW || dy < 0 || dy >= destH {
		return false
	}
	sw := sp.x1 - sp.x0
	sh := sp.y1 - sp.y0
	if sw < 1 || sh < 1 {
		return false
	}
	sxOffF := float64(dx) + subX
	if flipX {
		sxOffF = float64(destW-1) - (float64(dx) + subX)
	}
	if sxOffF < 0 {
		sxOffF = 0
	}
	if sxOffF > float64(destW-1) {
		sxOffF = float64(destW - 1)
	}
	sx := sp.x0 + int(sxOffF*float64(sw)/float64(destW))
	sy := sp.y0 + dy*sh/destH
	if sx >= sp.x1 {
		sx = sp.x1 - 1
	}
	if sy >= sp.y1 {
		sy = sp.y1 - 1
	}
	return sp.pix[sy*sp.w+sx] >= periSpriteAlphaMin
}
