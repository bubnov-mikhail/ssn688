package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	IconPassive = iota
	IconActive
	IconSpectrum
	IconLibrary
	IconWeapons
	IconManeuver
	IconTactical
	IconDamage
	IconMast
)

// DrawScreenIcon draws a 688-style station icon centered at (cx, cy).
func DrawScreenIcon(screen *ebiten.Image, kind int, cx, cy, size int, clr color.Color) {
	s := float64(size)
	switch kind {
	case IconPassive:
		// Towed array / hydrophone arc
		for deg := -60.0; deg <= 60; deg += 30 {
			rad := deg * math.Pi / 180
			DrawLine(screen, float64(cx), float64(cy), float64(cx)+math.Sin(rad)*s*0.55, float64(cy)-math.Cos(rad)*s*0.55, clr)
		}
		DrawLine(screen, float64(cx)-s*0.45, float64(cy)+s*0.15, float64(cx)+s*0.45, float64(cy)+s*0.15, clr)
		FillRect(screen, cx-3, cy-3, 7, 7, clr)
	case IconActive:
		// Active ping rings
		for r := 0.2; r <= 0.55; r += 0.175 {
			drawIconCircle(screen, float64(cx), float64(cy), s*r, clr)
		}
		FillRect(screen, cx-2, cy-2, 5, 5, clr)
	case IconSpectrum:
		// Spectrum bars
		for i := 0; i < 5; i++ {
			h := int(s * (0.25 + float64(i%3)*0.18))
			x := cx - int(s*0.35) + i*int(s*0.17)
			FillRect(screen, x, cy+h/2-int(s*0.35), int(s*0.1), h, clr)
		}
	case IconLibrary:
		// Filing cabinet / signature library
		FillRect(screen, cx-int(s*0.3), cy-int(s*0.35), int(s*0.6), int(s*0.7), clr)
		FillRect(screen, cx-int(s*0.22), cy-int(s*0.28), int(s*0.44), int(s*0.56), ColorPanelInset)
		for i := 0; i < 4; i++ {
			y := cy - int(s*0.2) + i*int(s*0.12)
			DrawLine(screen, float64(cx)-s*0.15, float64(y), float64(cx)+s*0.15, float64(y), clr)
		}
	case IconWeapons:
		// Torpedo shape
		DrawLine(screen, float64(cx), float64(cy)-s*0.4, float64(cx), float64(cy)+s*0.35, clr)
		DrawLine(screen, float64(cx)-s*0.12, float64(cy)+s*0.2, float64(cx)+s*0.12, float64(cy)+s*0.2, clr)
		DrawLine(screen, float64(cx)-s*0.08, float64(cy)-s*0.35, float64(cx), float64(cy)-s*0.45, clr)
		DrawLine(screen, float64(cx)+s*0.08, float64(cy)-s*0.35, float64(cx), float64(cy)-s*0.45, clr)
	case IconManeuver:
		// Ship's wheel
		drawIconCircle(screen, float64(cx), float64(cy), s*0.42, clr)
		for deg := 0; deg < 360; deg += 45 {
			rad := float64(deg) * math.Pi / 180
			DrawLine(screen, float64(cx), float64(cy),
				float64(cx)+math.Sin(rad)*s*0.38, float64(cy)-math.Cos(rad)*s*0.38, clr)
		}
		FillRect(screen, cx-3, cy-3, 7, 7, clr)
	case IconTactical:
		// Chart plot with contacts
		FillRect(screen, cx-int(s*0.35), cy-int(s*0.35), int(s*0.7), int(s*0.7), ColorPanelInset)
		DrawLine(screen, float64(cx)-s*0.3, float64(cy), float64(cx)+s*0.3, float64(cy), ColorGrid)
		DrawLine(screen, float64(cx), float64(cy)-s*0.3, float64(cx), float64(cy)+s*0.3, ColorGrid)
		FillRect(screen, cx-2, cy-2, 5, 5, clr)
		FillRect(screen, cx+int(s*0.15)-2, cy-int(s*0.1)-2, 5, 5, ColorDanger)
		FillRect(screen, cx-int(s*0.2)-2, cy+int(s*0.12)-2, 5, 5, ColorWarn)
	case IconDamage:
		// Damage control: hull outline + wrench cross
		drawIconCircle(screen, float64(cx), float64(cy), s*0.38, clr)
		DrawLine(screen, float64(cx)-s*0.22, float64(cy)+s*0.08, float64(cx)+s*0.22, float64(cy)-s*0.18, clr)
		DrawLine(screen, float64(cx)-s*0.18, float64(cy)-s*0.18, float64(cx)+s*0.18, float64(cy)+s*0.18, clr)
		FillRect(screen, cx-int(s*0.08), cy-int(s*0.28), int(s*0.16), int(s*0.14), clr)
	case IconMast:
		// Raised mast / periscope stalk with sensor head
		DrawLine(screen, float64(cx), float64(cy)+s*0.4, float64(cx), float64(cy)-s*0.25, clr)
		FillRect(screen, cx-int(s*0.18), cy-int(s*0.38), int(s*0.36), int(s*0.16), clr)
		DrawLine(screen, float64(cx)+s*0.18, float64(cy)-s*0.3, float64(cx)+s*0.42, float64(cy)-s*0.42, clr)
		FillRect(screen, cx-int(s*0.28), cy+int(s*0.32), int(s*0.56), int(s*0.1), clr)
	}
}

func drawIconCircle(screen *ebiten.Image, cx, cy, r float64, clr color.Color) {
	const n = 24
	for i := 0; i < n; i++ {
		a0 := float64(i) * 2 * math.Pi / n
		a1 := float64(i+1) * 2 * math.Pi / n
		DrawLine(screen, cx+math.Cos(a0)*r, cy+math.Sin(a0)*r, cx+math.Cos(a1)*r, cy+math.Sin(a1)*r, clr)
	}
}
