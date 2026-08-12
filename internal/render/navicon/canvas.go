package navicon

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// DesignSize is the coordinate space all icon geometry is authored in.
const DesignSize = 128

// Canvas rasterizes white line art on a transparent square bitmap.
type Canvas struct {
	*image.RGBA
	px int
	s  float64
}

func New(px int) *Canvas {
	if px < 1 {
		px = 1
	}
	m := image.NewRGBA(image.Rect(0, 0, px, px))
	draw.Draw(m, m.Bounds(), &image.Uniform{color.RGBA{}}, image.Point{}, draw.Src)
	return &Canvas{RGBA: m, px: px, s: float64(px) / DesignSize}
}

func (c *Canvas) S(v float64) float64 { return v * c.s }

func (c *Canvas) plot(x, y int, a float64) {
	if a <= 0 {
		return
	}
	if x < 0 || y < 0 || x >= c.px || y >= c.px {
		return
	}
	i := c.PixOffset(x, y)
	old := c.Pix[i+3]
	na := uint8(math.Min(255, float64(old)+a*255))
	c.Pix[i] = 255
	c.Pix[i+1] = 255
	c.Pix[i+2] = 255
	c.Pix[i+3] = na
}

func (c *Canvas) line(x0, y0, x1, y1, w float64) {
	c.linePx(c.S(x0), c.S(y0), c.S(x1), c.S(y1), c.S(w))
}

func (c *Canvas) linePx(x0, y0, x1, y1, w float64) {
	dx := x1 - x0
	dy := y1 - y0
	steps := int(math.Max(math.Abs(dx), math.Abs(dy))) + 1
	if steps < 1 {
		steps = 1
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		c.diskPx(x0+dx*t, y0+dy*t, w)
	}
}

func (c *Canvas) disk(cx, cy, r float64) {
	c.diskPx(c.S(cx), c.S(cy), c.S(r))
}

func (c *Canvas) diskPx(cx, cy, r float64) {
	ir := int(math.Ceil(r)) + 1
	ix, iy := int(math.Round(cx)), int(math.Round(cy))
	for y := iy - ir; y <= iy+ir; y++ {
		for x := ix - ir; x <= ix+ir; x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy)
			if d > r {
				continue
			}
			a := 1.0
			if d > r-1 {
				a = r - d
			}
			c.plot(x, y, a)
		}
	}
}

func (c *Canvas) circle(cx, cy, r, w float64) {
	cx, cy, r, w = c.S(cx), c.S(cy), c.S(r), c.S(w)
	const n = 96
	for i := 0; i < n; i++ {
		a0 := float64(i) * 2 * math.Pi / n
		a1 := float64(i+1) * 2 * math.Pi / n
		x0 := cx + math.Cos(a0)*r
		y0 := cy + math.Sin(a0)*r
		x1 := cx + math.Cos(a1)*r
		y1 := cy + math.Sin(a1)*r
		c.linePx(x0, y0, x1, y1, w)
	}
}

func (c *Canvas) rect(x0, y0, x1, y1, w float64) {
	c.line(x0, y0, x1, y0, w)
	c.line(x1, y0, x1, y1, w)
	c.line(x1, y1, x0, y1, w)
	c.line(x0, y1, x0, y0, w)
}

func (c *Canvas) fillRect(x0, y0, x1, y1 int) {
	x0 = int(c.S(float64(x0)))
	y0 = int(c.S(float64(y0)))
	x1 = int(c.S(float64(x1)))
	y1 = int(c.S(float64(y1)))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			c.plot(x, y, 1)
		}
	}
}

func (c *Canvas) arc(cx, cy, r, a0, a1, w float64) {
	cx, cy, r, w = c.S(cx), c.S(cy), c.S(r), c.S(w)
	steps := int(math.Abs(a1-a0)*r*1.2) + 4
	var px, py float64
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		a := a0 + (a1-a0)*t
		x := cx + math.Cos(a)*r
		y := cy + math.Sin(a)*r
		if i > 0 {
			c.linePx(px, py, x, y, w)
		}
		px, py = x, y
	}
}

func (c *Canvas) polygon(pts [][2]float64, w float64) {
	for i := range pts {
		j := (i + 1) % len(pts)
		c.line(pts[i][0], pts[i][1], pts[j][0], pts[j][1], w)
	}
}

func (c *Canvas) transform(x, y, cx, cy, cos, sin float64) (float64, float64) {
	return cx + x*cos - y*sin, cy + x*sin + y*cos
}

func (c *Canvas) drawTorpedo(cx, cy, angleDeg, length, width float64) {
	rad := angleDeg * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)
	hl := length / 2
	hw := width / 2

	body := [][2]float64{
		{-hl * 0.92, 0},
		{-hl * 0.78, -hw * 0.45},
		{-hl * 0.58, -hw},
		{-hl * 0.12, -hw},
		{hl * 0.52, -hw},
		{hl * 0.82, -hw * 0.55},
		{hl * 0.96, -hw * 0.18},
		{hl * 0.96, hw * 0.18},
		{hl * 0.82, hw * 0.55},
		{hl * 0.52, hw},
		{-hl * 0.12, hw},
		{-hl * 0.58, hw},
		{-hl * 0.78, hw * 0.45},
	}
	pts := make([][2]float64, len(body))
	for i, p := range body {
		pts[i][0], pts[i][1] = c.transform(p[0], p[1], cx, cy, cos, sin)
	}
	c.polygon(pts, 2)

	nx, ny := c.transform(hl*0.94, 0, cx, cy, cos, sin)
	c.disk(nx, ny, hw*0.55)

	for _, dy := range []float64{-hw * 1.05, hw * 1.05} {
		x0, y0 := c.transform(-hl*0.62, dy, cx, cy, cos, sin)
		x1, y1 := c.transform(-hl*0.44, dy, cx, cy, cos, sin)
		c.line(x0, y0, x1, y1, 2)
	}

	px, py := c.transform(-hl*0.92, 0, cx, cy, cos, sin)
	c.disk(px, py, hw*0.35)
	for _, ly := range []float64{-hw * 0.65, hw * 0.65, -hw * 0.35, hw * 0.35} {
		bx, by := c.transform(-hl*0.92, ly, cx, cy, cos, sin)
		c.line(px, py, bx, by, 1.5)
	}

	x0, y0 := c.transform(hl*0.35, -hw*0.55, cx, cy, cos, sin)
	x1, y1 := c.transform(hl*0.35, hw*0.55, cx, cy, cos, sin)
	c.line(x0, y0, x1, y1, 1.4)
}

func (c *Canvas) solidLine(x0, y0, x1, y1, w float64) {
	c.solidLinePx(c.S(x0), c.S(y0), c.S(x1), c.S(y1), c.S(w))
}

func (c *Canvas) solidLinePx(x0, y0, x1, y1, w float64) {
	dx := x1 - x0
	dy := y1 - y0
	steps := int(math.Max(math.Abs(dx), math.Abs(dy))) + 1
	if steps < 1 {
		steps = 1
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		c.solidDiskPx(x0+dx*t, y0+dy*t, w)
	}
}

func (c *Canvas) solidDisk(cx, cy, r float64) {
	c.solidDiskPx(c.S(cx), c.S(cy), c.S(r))
}

func (c *Canvas) solidDiskPx(cx, cy, r float64) {
	ir := int(math.Ceil(r)) + 1
	ix, iy := int(math.Round(cx)), int(math.Round(cy))
	for y := iy - ir; y <= iy+ir; y++ {
		for x := ix - ir; x <= ix+ir; x++ {
			if math.Hypot(float64(x)-cx, float64(y)-cy) <= r {
				c.plot(x, y, 1)
			}
		}
	}
}

func (c *Canvas) solidArc(cx, cy, r, a0, a1, w float64) {
	cx, cy, r, w = c.S(cx), c.S(cy), c.S(r), c.S(w)
	steps := int(math.Abs(a1-a0)*r*1.2) + 4
	var px, py float64
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		a := a0 + (a1-a0)*t
		x := cx + math.Cos(a)*r
		y := cy + math.Sin(a)*r
		if i > 0 {
			c.solidLinePx(px, py, x, y, w)
		}
		px, py = x, y
	}
}

func (c *Canvas) solidPolygon(pts [][2]float64, w float64) {
	scaled := make([][2]float64, len(pts))
	for i, p := range pts {
		scaled[i][0] = c.S(p[0])
		scaled[i][1] = c.S(p[1])
	}
	sw := c.S(w)
	for i := range scaled {
		j := (i + 1) % len(scaled)
		c.solidLinePx(scaled[i][0], scaled[i][1], scaled[j][0], scaled[j][1], sw)
	}
}

func (c *Canvas) solidRect(x0, y0, x1, y1, w float64) {
	c.solidLine(x0, y0, x1, y0, w)
	c.solidLine(x1, y0, x1, y1, w)
	c.solidLine(x1, y1, x0, y1, w)
	c.solidLine(x0, y1, x0, y0, w)
}

func pointInPolygon(x, y float64, pts [][2]float64) bool {
	n := len(pts)
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := pts[i][0], pts[i][1]
		xj, yj := pts[j][0], pts[j][1]
		if ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi+1e-9)+xi) {
			inside = !inside
		}
		j = i
	}
	return inside
}

func (c *Canvas) clearInsidePolygon(pts [][2]float64) {
	scaled := make([][2]float64, len(pts))
	minX, minY := c.px, c.px
	maxX, maxY := 0, 0
	for i, p := range pts {
		x := c.S(p[0])
		y := c.S(p[1])
		scaled[i][0], scaled[i][1] = x, y
		ix, iy := int(math.Floor(x)), int(math.Floor(y))
		if ix < minX {
			minX = ix
		}
		if iy < minY {
			minY = iy
		}
		if ix > maxX {
			maxX = ix
		}
		if iy > maxY {
			maxY = iy
		}
	}
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= c.px {
		maxX = c.px - 1
	}
	if maxY >= c.px {
		maxY = c.px - 1
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if !pointInPolygon(float64(x)+0.5, float64(y)+0.5, scaled) {
				continue
			}
			i := c.PixOffset(x, y)
			c.Pix[i] = 0
			c.Pix[i+1] = 0
			c.Pix[i+2] = 0
			c.Pix[i+3] = 0
		}
	}
}
