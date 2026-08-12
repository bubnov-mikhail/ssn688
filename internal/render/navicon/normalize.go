package navicon

import (
	"image"

	xdraw "golang.org/x/image/draw"
)

// normPad is inset on each side after scaling content to a uniform footprint.
const normPad = 0.07

func opaqueBounds(img *image.RGBA) (minX, minY, maxX, maxY int) {
	minX, minY = img.Bounds().Dx(), img.Bounds().Dy()
	maxX, maxY = -1, -1
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.Pix[img.PixOffset(x, y)+3] < 12 {
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	return
}

func normalizeIcon(src *image.RGBA, outPx int) *image.RGBA {
	minX, minY, maxX, maxY := opaqueBounds(src)
	dst := image.NewRGBA(image.Rect(0, 0, outPx, outPx))
	if maxX < minX {
		return dst
	}

	bw := maxX - minX + 1
	bh := maxY - minY + 1
	if bw < 1 || bh < 1 {
		return dst
	}

	pad := float64(outPx) * normPad
	target := float64(outPx) - 2*pad
	scale := target / float64(max(bw, bh))
	dw := int(float64(bw)*scale + 0.5)
	dh := int(float64(bh)*scale + 0.5)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}

	dx := (outPx - dw) / 2
	dy := (outPx - dh) / 2
	crop := src.SubImage(image.Rect(minX, minY, maxX+1, maxY+1))
	xdraw.CatmullRom.Scale(dst, image.Rect(dx, dy, dx+dw, dy+dh), crop, crop.Bounds(), xdraw.Over, nil)
	return dst
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
