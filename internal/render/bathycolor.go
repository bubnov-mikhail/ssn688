package render

import (
	"image/color"
	"math"
)

// BathyDeepThresholdFt is the depth at which the reference trench navy is reached.
const BathyDeepThresholdFt = 12000.0

// Reference bathymetry palette (Mariana-trench style chart).
var (
	// BathyLandColor is the chart land fill (light beige).
	BathyLandColor    = color.RGBA{228, 220, 198, 255}
	bathyShallowColor = color.RGBA{158, 208, 228, 255}
	bathyDeepColor    = color.RGBA{10, 28, 78, 255}
)

// BathyColor maps chart depth (feet) to raster color. Land <= 0; >=12000 ft uses trench navy.
func BathyColor(depthFt float64) color.RGBA {
	if depthFt <= 0 {
		return BathyLandColor
	}
	if depthFt >= BathyDeepThresholdFt {
		return bathyDeepColor
	}
	t := depthFt / BathyDeepThresholdFt
	// Log ramp keeps shelf detail before the deep basin plateaus.
	t = math.Log1p(t*9) / math.Log1p(10)
	return lerpBathyColor(bathyShallowColor, bathyDeepColor, t)
}

func lerpBathyColor(a, b color.RGBA, t float64) color.RGBA {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	return color.RGBA{
		R: uint8(float64(a.R) + t*(float64(b.R)-float64(a.R))),
		G: uint8(float64(a.G) + t*(float64(b.G)-float64(a.G))),
		B: uint8(float64(a.B) + t*(float64(b.B)-float64(a.B))),
		A: 255,
	}
}

// BakeBathyRGBA builds a Width×Height RGBA chart from cell depths (no world interpolation).
// Applies a one-time blur across land and water so the shore edge softens. Caller caches and reuses.
func BakeBathyRGBA(width, height int, depths []float32) []byte {
	if width < 1 || height < 1 || len(depths) != width*height {
		return nil
	}
	src := make([]byte, width*height*4)
	for i, d := range depths {
		c := BathyColor(float64(d))
		off := i * 4
		src[off] = c.R
		src[off+1] = c.G
		src[off+2] = c.B
		src[off+3] = 255
	}
	return blurBathyRGBA(src, width, height)
}

// SampleBathyRGBA bilinear-samples a baked RGBA chart at fractional cell coords (fx, fy).
// Off-chart samples return solid land beige.
func SampleBathyRGBA(pix []byte, width, height int, fx, fy float64) color.RGBA {
	if width < 1 || height < 1 || len(pix) < width*height*4 {
		return BathyLandColor
	}
	if fx < 0 || fy < 0 || fx >= float64(width-1) || fy >= float64(height-1) {
		return BathyLandColor
	}
	i0 := int(fx)
	j0 := int(fy)
	tx := fx - float64(i0)
	ty := fy - float64(j0)
	c00 := pixRGBAAt(pix, width, i0, j0)
	c10 := pixRGBAAt(pix, width, i0+1, j0)
	c01 := pixRGBAAt(pix, width, i0, j0+1)
	c11 := pixRGBAAt(pix, width, i0+1, j0+1)
	return color.RGBA{
		R: lerpByte(lerpByte(c00.R, c10.R, tx), lerpByte(c01.R, c11.R, tx), ty),
		G: lerpByte(lerpByte(c00.G, c10.G, tx), lerpByte(c01.G, c11.G, tx), ty),
		B: lerpByte(lerpByte(c00.B, c10.B, tx), lerpByte(c01.B, c11.B, tx), ty),
		A: 255,
	}
}

// SampleBathyChart returns the baked chart color at world yards (bilinear over the blurred texture).
func SampleBathyChart(pix []byte, width, height int, depths []float32, originX, originY, cellSize, worldX, worldY float64) color.RGBA {
	if width < 1 || height < 1 || cellSize <= 0 {
		return BathyLandColor
	}
	fx := (worldX - originX) / cellSize
	fy := (worldY - originY) / cellSize
	return SampleBathyRGBA(pix, width, height, fx, fy)
}

func pixRGBAAt(pix []byte, width, x, y int) color.RGBA {
	off := (y*width + x) * 4
	return color.RGBA{R: pix[off], G: pix[off+1], B: pix[off+2], A: pix[off+3]}
}

func lerpByte(a, b uint8, t float64) uint8 {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	return uint8(float64(a) + t*(float64(b)-float64(a)) + 0.5)
}

// blurBathyRGBA applies a 3×3 box blur (several passes) to soften cell edges across land and water.
func blurBathyRGBA(src []byte, w, h int) []byte {
	a := make([]byte, len(src))
	b := make([]byte, len(src))
	copy(a, src)
	cur, next := a, b
	const passes = 4
	for i := 0; i < passes; i++ {
		boxBlurRGBAPass(cur, next, w, h)
		cur, next = next, cur
	}
	return cur
}

func boxBlurRGBAPass(src, dst []byte, w, h int) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b, n int
			for dy := -1; dy <= 1; dy++ {
				yy := y + dy
				if yy < 0 {
					yy = 0
				} else if yy >= h {
					yy = h - 1
				}
				for dx := -1; dx <= 1; dx++ {
					xx := x + dx
					if xx < 0 {
						xx = 0
					} else if xx >= w {
						xx = w - 1
					}
					off := (yy*w + xx) * 4
					r += int(src[off])
					g += int(src[off+1])
					b += int(src[off+2])
					n++
				}
			}
			off := (y*w + x) * 4
			dst[off] = uint8(r / n)
			dst[off+1] = uint8(g / n)
			dst[off+2] = uint8(b / n)
			dst[off+3] = 255
		}
	}
}
