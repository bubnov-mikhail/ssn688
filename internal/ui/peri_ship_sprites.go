package ui

import (
	"fmt"
	"image"
	"image/png"
	"sync"

	"github.com/ssn688/sim/assets"
)

const (
	periSpriteAspectStep = 5
	periSpriteAspectMax  = 90
	periSpriteAlphaMin   = uint8(10) // below this = empty background
)

type periShipSprite struct {
	pix    []uint8
	w, h   int
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
		periShipSpriteMap = make(map[string]*periShipSprite, 80)
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
	if aspectDeg > 90 {
		aspectDeg = 90
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
// Returns the destination AABB of the *visible* stub (for bloom/fire).
func blitPeriShipSprite(pix []byte, frameW, frameH int, sp *periShipSprite, centerX, waterY, boxW, boxH int, flipX bool, brightness, sinkFrac float64) (dstX0, dstY0, dstX1, dstY1 int, ok bool) {
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
	dstX0 = centerX - destW/2
	dstX1 = dstX0 + destW
	// Full sprite maps to [waterY-destH+sinkPx, waterY+sinkPx); clip at waterY.
	dstY0 = waterY - destH + sinkPx
	dstY1 = waterY
	if brightness < 0.05 {
		brightness = 0.05
	}
	if brightness > 1.4 {
		brightness = 1.4
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
			sxOff := dx
			if flipX {
				sxOff = destW - 1 - dx
			}
			sx := sp.x0 + sxOff*sw/destW
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
			// Opaque write — periBrighten only lifts brighter-than-bg pixels and
			// made Workbench hulls look like ghosts against the IR sky/sea.
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
	return dstX0, visTop, dstX1, dstY1, true
}

// periShipSpriteOpaqueAt mirrors blitPeriShipSprite's dest→source mapping:
// true only for opaque ship pixels of this aspect sprite (not sky/sea gaps).
// boxW/boxH are the same optic bounds passed to blit (uniform fit applied here).
func periShipSpriteOpaqueAt(sp *periShipSprite, centerX, waterY, boxW, boxH int, flipX bool, sinkFrac float64, xx, yy int) bool {
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
	dstX0 := centerX - destW/2
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
	sxOff := dx
	if flipX {
		sxOff = destW - 1 - dx
	}
	sx := sp.x0 + sxOff*sw/destW
	sy := sp.y0 + dy*sh/destH
	if sx >= sp.x1 {
		sx = sp.x1 - 1
	}
	if sy >= sp.y1 {
		sy = sp.y1 - 1
	}
	return sp.pix[sy*sp.w+sx] >= periSpriteAlphaMin
}
