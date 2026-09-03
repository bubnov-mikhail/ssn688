package theaterpreview

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

type coastSegment struct {
	x0, y0, x1, y1 float64
}

type bathyViewKey struct {
	zoom                    float64
	centerX, centerY        float64
	mapX, mapY, mapW, mapH  int
}


func buildCoastSegments(bathy *world.Bathymetry) []coastSegment {
	if bathy == nil || !bathy.Valid() || bathy.Width < 2 || bathy.Height < 2 {
		return nil
	}
	segments := make([]coastSegment, 0, bathy.Width*bathy.Height/4)
	for j := 0; j < bathy.Height-1; j++ {
		y0 := bathy.OriginY + float64(j)*bathy.CellSize
		y1 := bathy.OriginY + float64(j+1)*bathy.CellSize
		for i := 0; i < bathy.Width-1; i++ {
			x0 := bathy.OriginX + float64(i)*bathy.CellSize
			x1 := bathy.OriginX + float64(i+1)*bathy.CellSize
			dBL := float64(bathy.Depths[j*bathy.Width+i])
			dBR := float64(bathy.Depths[j*bathy.Width+i+1])
			dTL := float64(bathy.Depths[(j+1)*bathy.Width+i])
			dTR := float64(bathy.Depths[(j+1)*bathy.Width+i+1])
			mask := 0
			if dBL > 0 {
				mask |= 1
			}
			if dBR > 0 {
				mask |= 2
			}
			if dTR > 0 {
				mask |= 4
			}
			if dTL > 0 {
				mask |= 8
			}
			if mask == 0 || mask == 15 {
				continue
			}
			bottomX, bottomY := interpZero(x0, y0, dBL, x1, y0, dBR)
			rightX, rightY := interpZero(x1, y0, dBR, x1, y1, dTR)
			topX, topY := interpZero(x0, y1, dTL, x1, y1, dTR)
			leftX, leftY := interpZero(x0, y0, dBL, x0, y1, dTL)
			switch mask {
			case 1, 14:
				segments = append(segments, coastSegment{leftX, leftY, bottomX, bottomY})
			case 2, 13:
				segments = append(segments, coastSegment{bottomX, bottomY, rightX, rightY})
			case 3, 12:
				segments = append(segments, coastSegment{leftX, leftY, rightX, rightY})
			case 4, 11:
				segments = append(segments, coastSegment{rightX, rightY, topX, topY})
			case 5:
				segments = append(segments,
					coastSegment{leftX, leftY, topX, topY},
					coastSegment{bottomX, bottomY, rightX, rightY},
				)
			case 6, 9:
				segments = append(segments, coastSegment{bottomX, bottomY, topX, topY})
			case 7, 8:
				segments = append(segments, coastSegment{leftX, leftY, topX, topY})
			case 10:
				segments = append(segments,
					coastSegment{leftX, leftY, bottomX, bottomY},
					coastSegment{rightX, rightY, topX, topY},
				)
			}
		}
	}
	return segments
}

func interpZero(x0, y0, d0, x1, y1, d1 float64) (float64, float64) {
	den := d0 - d1
	if math.Abs(den) < 1e-6 {
		return (x0 + x1) * 0.5, (y0 + y1) * 0.5
	}
	t := d0 / den
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return x0 + (x1-x0)*t, y0 + (y1-y0)*t
}

func (v *MapView) invalidateBathy() {
	v.bathyKey = bathyViewKey{}
}

func (v *MapView) ensureBakedRGBA() []byte {
	if v.bathy == nil || !v.bathy.Valid() {
		return nil
	}
	if v.bakedBathy == v.bathy && len(v.bakedRGBA) == v.bathy.Width*v.bathy.Height*4 {
		return v.bakedRGBA
	}
	v.bakedRGBA = render.BakeBathyRGBA(v.bathy.Width, v.bathy.Height, v.bathy.Depths)
	v.bakedBathy = v.bathy
	return v.bakedRGBA
}

func (v *MapView) ensureBathyImage() *ebiten.Image {
	if v.bathy == nil || !v.bathy.Valid() || v.zoom <= 0 || v.mapW <= 0 || v.mapH <= 0 {
		return nil
	}
	const step = 4
	q := float64(step) / v.zoom
	if q < 25 {
		q = 25
	}
	if q > 250 {
		q = 250
	}
	cx, cy := v.centerWorld()
	key := bathyViewKey{
		zoom:    v.zoom,
		centerX: math.Round(cx/q) * q,
		centerY: math.Round(cy/q) * q,
		mapX:    v.mapX,
		mapY:    v.mapY,
		mapW:    v.mapW,
		mapH:    v.mapH,
	}
	if v.bathyImg != nil && v.bathyKey == key {
		return v.bathyImg
	}
	w := (v.mapW + step - 1) / step
	h := (v.mapH + step - 1) / step
	if w < 1 || h < 1 {
		return nil
	}
	need := w * h * 4
	if len(v.bathyPix) != need {
		v.bathyPix = make([]byte, need)
	}
	if v.bathyImg == nil || v.bathyImg.Bounds().Dx() != w || v.bathyImg.Bounds().Dy() != h {
		v.bathyImg = ebiten.NewImage(w, h)
	}
	qCX, qCY := key.centerX, key.centerY
	const half = step / 2
	baked := v.ensureBakedRGBA()
	buf := v.bathyPix
	for py := 0; py < h; py++ {
		sy := v.mapY + py*step + half
		for px := 0; px < w; px++ {
			sx := v.mapX + px*step + half
			wx := qCX + (float64(sx)-float64(v.mapX+v.mapW/2))/v.zoom
			wy := qCY - (float64(sy)-float64(v.mapY+v.mapH/2))/v.zoom
			clr := render.SampleBathyChart(
				baked, v.bathy.Width, v.bathy.Height, v.bathy.Depths,
				v.bathy.OriginX, v.bathy.OriginY, v.bathy.CellSize, wx, wy,
			)
			off := (py*w + px) * 4
			buf[off] = clr.R
			buf[off+1] = clr.G
			buf[off+2] = clr.B
			buf[off+3] = clr.A
		}
	}
	v.bathyImg.WritePixels(buf)
	v.bathyKey = key
	return v.bathyImg
}

func (v *MapView) drawBathymetry(screen *ebiten.Image) {
	img := v.ensureBathyImage()
	if img == nil {
		return
	}
	const step = 4
	cx, cy := v.centerWorld()
	// Cached raster uses a quantized center; shift blit so vectors stay aligned while panning.
	offX := (v.bathyKey.centerX - cx) * v.zoom
	offY := (cy - v.bathyKey.centerY) * v.zoom
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(step, step)
	op.GeoM.Translate(float64(v.mapX)+offX, float64(v.mapY)+offY)
	screen.DrawImage(img, op)
}

func (v *MapView) drawCoastline(screen *ebiten.Image) {
	if v.bathy == nil || !v.bathy.Valid() {
		return
	}
	if v.coastBathy != v.bathy || v.coastSegments == nil {
		v.coastSegments = buildCoastSegments(v.bathy)
		v.coastBathy = v.bathy
	}
	left := float64(v.mapX)
	top := float64(v.mapY)
	right := float64(v.mapX + v.mapW - 1)
	bottom := float64(v.mapY + v.mapH - 1)
	shadow := color.RGBA{5, 10, 12, 220}
	shore := render.BathyLandColor
	for _, seg := range v.coastSegments {
		x0, y0, ok := v.WorldToScreen(seg.x0, seg.y0)
		if !ok {
			continue
		}
		x1, y1, ok := v.WorldToScreen(seg.x1, seg.y1)
		if !ok {
			continue
		}
		x0, y0, x1, y1, ok = clipLineToRect(x0, y0, x1, y1, left, top, right, bottom)
		if !ok {
			continue
		}
		render.DrawLine(screen, x0+1, y0+1, x1+1, y1+1, shadow)
		render.DrawLine(screen, x0, y0, x1, y1, shore)
	}
}

func (v *MapView) drawRoutes(screen *ebiten.Image) {
	if len(v.routes) == 0 {
		return
	}
	lineClr := render.ColorDebugRoute
	wpClr := render.ColorDebugRouteWP
	for _, r := range v.routes {
		if r == nil || len(r.Waypoints) < 2 {
			continue
		}
		n := r.UniqueCount()
		for i := 1; i < n; i++ {
			a0, a1 := r.Waypoints[i-1], r.Waypoints[i]
			x0, y0, ok0 := v.WorldToScreen(a0.X, a0.Y)
			x1, y1, ok1 := v.WorldToScreen(a1.X, a1.Y)
			if ok0 && ok1 {
				render.DrawLine(screen, x0, y0, x1, y1, lineClr)
			}
		}
		if r.Looped && !r.PingPong && n >= 2 {
			a0, a1 := r.Waypoints[n-1], r.Waypoints[0]
			x0, y0, ok0 := v.WorldToScreen(a0.X, a0.Y)
			x1, y1, ok1 := v.WorldToScreen(a1.X, a1.Y)
			if ok0 && ok1 {
				render.DrawLine(screen, x0, y0, x1, y1, lineClr)
			}
		}
		for i := 0; i < n; i++ {
			wp := r.Waypoints[i]
			sx, sy, ok := v.WorldToScreen(wp.X, wp.Y)
			if !ok || !v.ContainsScreen(int(sx), int(sy)) {
				continue
			}
			render.FillRect(screen, int(sx)-2, int(sy)-2, 5, 5, wpClr)
			if i == 0 || (r.PingPong && i == n-1) {
				label := r.ID
				if r.PingPong && i == n-1 && i != 0 {
					label = r.ID + "⇄"
				}
				render.DrawText(screen, label, int(sx)+6, int(sy)-4, lineClr, true)
			}
		}
	}
}

func clipLineToRect(x0, y0, x1, y1, left, top, right, bottom float64) (float64, float64, float64, float64, bool) {
	const (
		clipLeft   = 1
		clipRight  = 2
		clipBottom = 4
		clipTop    = 8
	)
	encode := func(x, y float64) int {
		c := 0
		if x < left {
			c |= clipLeft
		} else if x > right {
			c |= clipRight
		}
		if y < top {
			c |= clipTop
		} else if y > bottom {
			c |= clipBottom
		}
		return c
	}
	c0 := encode(x0, y0)
	c1 := encode(x1, y1)
	for {
		if c0 == 0 && c1 == 0 {
			return x0, y0, x1, y1, true
		}
		if c0&c1 != 0 {
			return 0, 0, 0, 0, false
		}
		out := c0
		if out == 0 {
			out = c1
		}
		var x, y float64
		switch {
		case out&clipTop != 0:
			x = x0 + (x1-x0)*(top-y0)/(y1-y0)
			y = top
		case out&clipBottom != 0:
			x = x0 + (x1-x0)*(bottom-y0)/(y1-y0)
			y = bottom
		case out&clipRight != 0:
			y = y0 + (y1-y0)*(right-x0)/(x1-x0)
			x = right
		case out&clipLeft != 0:
			y = y0 + (y1-y0)*(left-x0)/(x1-x0)
			x = left
		}
		if out == c0 {
			x0, y0 = x, y
			c0 = encode(x0, y0)
		} else {
			x1, y1 = x, y
			c1 = encode(x1, y1)
		}
	}
}
