package ui

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ssn688/sim/internal/render"
)

type contactTableScrollState struct {
	passive  int
	active   int
	spectrum int
	weps     int
	library  int
}

func clampContactTableScroll(offset, total, visible int) int {
	if visible < 1 {
		visible = 1
	}
	maxOffset := total - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset < 0 {
		return 0
	}
	if offset > maxOffset {
		return maxOffset
	}
	return offset
}

func contactTableWindow(total, offset, visible int) (start, end int) {
	offset = clampContactTableScroll(offset, total, visible)
	end = offset + visible
	if end > total {
		end = total
	}
	return offset, end
}

func scrollContactTableWheel(mx, my, x, y, w, h, total, visible int, offset *int) bool {
	if offset == nil || total <= visible || !inRect(mx, my, x, y, w, h) {
		return false
	}
	_, wheelY := ebiten.Wheel()
	if math.Abs(wheelY) < 0.01 {
		return false
	}
	step := int(math.Ceil(math.Abs(wheelY)))
	if wheelY < 0 {
		*offset += step
	} else {
		*offset -= step
	}
	*offset = clampContactTableScroll(*offset, total, visible)
	return true
}

func drawContactTableScrollbar(screen *ebiten.Image, x, y, h, total, visible, offset int) {
	if total <= visible || visible < 1 || h <= 0 {
		return
	}
	trackW := 4
	render.FillRect(screen, x, y, trackW, h, color.RGBA{42, 46, 52, 255})
	thumbH := int(float64(h) * float64(visible) / float64(total))
	if thumbH < 10 {
		thumbH = 10
	}
	maxOffset := total - visible
	top := y
	if maxOffset > 0 && h-thumbH > 0 {
		top += int(float64(h-thumbH) * float64(offset) / float64(maxOffset))
	}
	render.FillRect(screen, x, top, trackW, thumbH, color.RGBA{115, 120, 128, 255})
}
