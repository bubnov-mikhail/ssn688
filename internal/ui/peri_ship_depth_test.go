package ui

import (
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
)

func TestPeriShipDepthCloserWins(t *testing.T) {
	sp := pickPeriShipSprite(periClassCombatant, 90)
	if sp == nil {
		t.Fatal("no sprite")
	}
	const fw, fh = 80, 60
	const waterY = 45
	pix := make([]byte, fw*fh*4)
	depth := make([]float32, fw*fh)
	_, _, _, _, ok := blitPeriShipSprite(pix, depth, fw, fh, sp, 40, waterY, 36, 18, false, 1.0, 0, 4000)
	if !ok {
		t.Fatal("far blit failed")
	}
	farOwned := 0
	for _, d := range depth {
		if d == 4000 {
			farOwned++
		}
	}
	if farOwned < 20 {
		t.Fatalf("far ship should own pixels, got %d", farOwned)
	}
	_, _, _, _, ok = blitPeriShipSprite(pix, depth, fw, fh, sp, 40, waterY, 36, 18, false, 1.0, 0, 800)
	if !ok {
		t.Fatal("near blit failed")
	}
	nearOwned, farLeft := 0, 0
	for _, d := range depth {
		switch d {
		case 800:
			nearOwned++
		case 4000:
			farLeft++
		}
	}
	if nearOwned < 20 {
		t.Fatalf("near ship should claim pixels, got %d", nearOwned)
	}
	if nearOwned <= farLeft {
		t.Fatalf("near should dominate overlap: near=%d farLeft=%d", nearOwned, farLeft)
	}
}

func TestPeriShipDepthRejectsFartherOverwrite(t *testing.T) {
	sp := pickPeriShipSprite(periClassMerchant, 90)
	if sp == nil {
		t.Fatal("no sprite")
	}
	const fw, fh = 80, 60
	const waterY = 45
	pix := make([]byte, fw*fh*4)
	depth := make([]float32, fw*fh)
	_, _, _, _, ok := blitPeriShipSprite(pix, depth, fw, fh, sp, 40, waterY, 40, 20, false, 0.6, 0, 900)
	if !ok {
		t.Fatal("near blit failed")
	}
	var cx, cy int
	var nearTone uint8
	for y := 0; y < waterY; y++ {
		for x := 0; x < fw; x++ {
			if v := pix[(y*fw+x)*4]; v > 20 {
				cx, cy, nearTone = x, y, v
				break
			}
		}
		if nearTone > 20 {
			break
		}
	}
	if nearTone < 20 {
		t.Fatal("no near pixels")
	}
	// Far brighter attempt — must NOT replace nearer pixel.
	blitPeriShipSprite(pix, depth, fw, fh, sp, float64(cx), waterY, 40, 20, false, 1.35, 0, 5000)
	got := pix[(cy*fw+cx)*4]
	if got != nearTone {
		t.Fatalf("far ship overwrote nearer hull at %d,%d: was %d now %d", cx, cy, nearTone, got)
	}
}

func TestProjectSortWaterYFarFirst(t *testing.T) {
	// Sanity: closer ship has lower waterline (larger WaterY) than a distant one.
	near := acoustics.PeriShipProj{RangeYd: 800, WaterY: 120}
	far := acoustics.PeriShipProj{RangeYd: 3500, WaterY: 70}
	if !(far.WaterY < near.WaterY && far.RangeYd > near.RangeYd) {
		t.Fatal("expected far nearer horizon / larger range")
	}
}
