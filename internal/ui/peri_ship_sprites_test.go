package ui

import (
	"math"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
)

func TestPeriShipSpritesLoaded(t *testing.T) {
	ensurePeriShipSprites()
	if len(periShipSpriteMap) < 700 {
		t.Fatalf("expected 724 sprites (4×181 @1° 0..180), got %d", len(periShipSpriteMap))
	}
	for _, cls := range []periShipClass{periClassMerchant, periClassTanker, periClassFishing, periClassCombatant} {
		sp := pickPeriShipSprite(cls, 90)
		if sp == nil {
			t.Fatalf("missing beam sprite for %s", periShipClassName(cls))
		}
		if sp.x1 <= sp.x0 || sp.y1 <= sp.y0 {
			t.Fatalf("empty bbox for %s", periShipClassName(cls))
		}
		stern := pickPeriShipSprite(cls, 180)
		if stern == nil {
			t.Fatalf("missing stern sprite for %s", periShipClassName(cls))
		}
		q := pickPeriShipSprite(cls, 135)
		if q == nil {
			t.Fatalf("missing stern-quarter sprite for %s", periShipClassName(cls))
		}
	}
}

func TestBlitPeriShipSprite(t *testing.T) {
	sp := pickPeriShipSprite(periClassCombatant, 90)
	if sp == nil {
		t.Fatal("no sprite")
	}
	const fw, fh = 120, 80
	pix := make([]byte, fw*fh*4)
	// Mid-gray background — opaque blit must overwrite, not only brighten.
	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+1], pix[i+2], pix[i+3] = 90, 90, 90, 255
	}
	const waterY = 50
	const boxW, boxH = 40, 20
	fitW, fitH := periShipSpriteFitDest(sp, boxW, boxH)
	if fitW < 2 || fitH < 2 {
		t.Fatal("fit failed")
	}
	// Uniform scale: dest aspect matches cropped sprite aspect.
	sw := sp.x1 - sp.x0
	sh := sp.y1 - sp.y0
	srcAspect := float64(sw) / float64(sh)
	dstAspect := float64(fitW) / float64(fitH)
	if math.Abs(srcAspect-dstAspect)/srcAspect > 0.05 {
		t.Fatalf("aspect distorted: src=%.3f dst=%.3f (%dx%d → %dx%d)", srcAspect, dstAspect, sw, sh, fitW, fitH)
	}
	if fitW > boxW || fitH > boxH {
		t.Fatalf("fit exceeds box: %dx%d in %dx%d", fitW, fitH, boxW, boxH)
	}
	x0, y0, x1, y1, ok := blitPeriShipSprite(pix, nil, fw, fh, sp, 60, waterY, boxW, boxH, false, 1, 0, 1000)
	if !ok {
		t.Fatal("blit failed")
	}
	if x1 <= x0 || y1 <= y0 {
		t.Fatalf("bad aabb %d,%d-%d,%d", x0, y0, x1, y1)
	}
	if y1 != waterY {
		t.Fatalf("visible stub should end at waterline, y1=%d", y1)
	}
	fullH := y1 - y0
	if fullH < fitH/2 {
		t.Fatalf("floating blit too short: visible %d want~%d", fullH, fitH)
	}
	if x1-x0 != fitW {
		t.Fatalf("blit width %d != fit %d", x1-x0, fitW)
	}
	// Darker-than-bg hull pixels prove opaque overwrite (ghost path would leave 90).
	darker := 0
	lit := 0
	below := 0
	for y := 0; y < fh; y++ {
		for x := 0; x < fw; x++ {
			v := pix[(y*fw+x)*4]
			if v != 90 {
				lit++
				if v < 90 {
					darker++
				}
				if y >= waterY {
					below++
				}
			}
		}
	}
	if lit < 30 {
		t.Fatalf("expected lit ship pixels, got %d", lit)
	}
	if darker < 5 {
		t.Fatalf("expected opaque dark hull pixels over sky, got %d", darker)
	}
	if below > 0 {
		t.Fatalf("pixels below waterline: %d", below)
	}

	// Half-sunk: fewer pixels, still clipped at waterline.
	pix2 := make([]byte, fw*fh*4)
	_, _, _, _, ok = blitPeriShipSprite(pix2, nil, fw, fh, sp, 60, waterY, boxW, boxH, false, 1, 0.55, 1000)
	if !ok {
		t.Fatal("half-sunk blit failed")
	}
	lit2 := 0
	for i := 0; i < len(pix2); i += 4 {
		if pix2[i] > 20 {
			lit2++
		}
	}
	if lit2 >= lit {
		t.Fatalf("sinking should show less ship: lit %d vs floating %d", lit2, lit)
	}
	_, _, _, _, ok = blitPeriShipSprite(pix2, nil, fw, fh, sp, 60, waterY, boxW, boxH, false, 1, 1, 1000)
	if ok {
		t.Fatal("fully submerged blit should fail")
	}

	drawPeriShipSilhouette(pix, nil, fw, fh, acoustics.PeriShipProj{
		CenterX: 60, WaterY: 50, WidthPx: 40, HullHPx: 8, SuperHPx: 12, RangeYd: 1000,
		AspectDeg: 90, Brightness: 1, Signature: "udaloy", BowRight: true, SpeedKts: 12,
	})
}

func TestPeriShipSpriteOpaqueMatchesBlit(t *testing.T) {
	sp := pickPeriShipSprite(periClassMerchant, 45)
	if sp == nil {
		t.Fatal("no merchant sprite")
	}
	const fw, fh = 120, 80
	const cx, waterY, destW, destH = 60, 55, 48, 22
	pix := make([]byte, fw*fh*4)
	for i := 0; i < len(pix); i += 4 {
		pix[i+3] = 255
	}
	_, _, _, _, ok := blitPeriShipSprite(pix, nil, fw, fh, sp, cx, waterY, destW, destH, true, 1, 0, 1000)
	if !ok {
		t.Fatal("blit failed")
	}
	onShip, offSky := 0, 0
	for y := 0; y < fh; y++ {
		for x := 0; x < fw; x++ {
			opaque := periShipSpriteOpaqueAt(sp, cx, waterY, destW, destH, true, 0, x, y)
			lit := pix[(y*fw+x)*4] > 0
			if opaque {
				onShip++
				if !lit {
					t.Fatalf("mask claims ship at %d,%d but blit left black", x, y)
				}
			} else if lit {
				offSky++
			}
		}
	}
	if onShip < 40 {
		t.Fatalf("expected many ship mask pixels, got %d", onShip)
	}
	if offSky > 0 {
		t.Fatalf("blit lit %d pixels outside sprite mask", offSky)
	}
	// Sky above AABB must never be "on ship".
	if periShipSpriteOpaqueAt(sp, cx, waterY, destW, destH, true, 0, cx, 2) {
		t.Fatal("sky pixel marked as ship")
	}
}
