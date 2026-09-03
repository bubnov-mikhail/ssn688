package theaterpreview

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const (
	zoomMin      = 0.004
	zoomMax      = 0.35
	zoomStep     = 1.25
	mapFitMargin = 0.92
)

// MapView pans and zooms a mission map like the in-game PLOT panel.
type MapView struct {
	bathy  *world.Bathymetry
	routes []*world.Route
	minX   float64
	minY   float64
	maxX   float64
	maxY   float64

	mapX, mapY, mapW, mapH int
	centerX, centerY       float64
	panX, panY             float64
	zoom                   float64

	bathyImg      *ebiten.Image
	bathyPix      []byte
	bathyKey      bathyViewKey
	bakedBathy    *world.Bathymetry
	bakedRGBA     []byte
	coastBathy    *world.Bathymetry
	coastSegments []coastSegment
}

// NewMapView wraps a loaded mission map.
func NewMapView(m *MissionMap) *MapView {
	if m == nil || m.Bathy == nil {
		return &MapView{}
	}
	minX, minY, maxX, maxY := m.Bathy.BoundsYards()
	return &MapView{
		bathy:    m.Bathy,
		routes:   m.Routes,
		minX:     minX,
		minY:     minY,
		maxX:     maxX,
		maxY:     maxY,
		centerX:  (minX + maxX) / 2,
		centerY:  (minY + maxY) / 2,
	}
}

// SetRect updates the on-screen map panel and fits on first layout.
func (v *MapView) SetRect(x, y, w, h int) {
	if v.mapW != w || v.mapH != h || v.mapX != x || v.mapY != y {
		v.invalidateBathy()
	}
	v.mapX, v.mapY, v.mapW, v.mapH = x, y, w, h
	if v.zoom == 0 {
		v.FitAll()
	}
}

// FitAll resets pan/zoom to show the full theater chart.
func (v *MapView) FitAll() {
	if v.bathy == nil || v.mapW <= 0 || v.mapH <= 0 {
		return
	}
	spanX := v.maxX - v.minX
	spanY := v.maxY - v.minY
	if spanX <= 0 || spanY <= 0 {
		return
	}
	v.centerX = (v.minX + v.maxX) / 2
	v.centerY = (v.minY + v.maxY) / 2
	v.panX, v.panY = 0, 0
	v.zoom = math.Min(float64(v.mapW)/spanX, float64(v.mapH)/spanY) * mapFitMargin
	v.invalidateBathy()
}

func (v *MapView) centerWorld() (cx, cy float64) {
	return v.centerX + v.panX, v.centerY + v.panY
}

// WorldToScreen maps yards to screen pixels.
func (v *MapView) WorldToScreen(wx, wy float64) (sx, sy float64, ok bool) {
	if v.bathy == nil || v.zoom <= 0 {
		return 0, 0, false
	}
	cx, cy := v.centerWorld()
	sx = float64(v.mapX+v.mapW/2) + (wx-cx)*v.zoom
	sy = float64(v.mapY+v.mapH/2) - (wy-cy)*v.zoom
	return sx, sy, true
}

// ScreenToWorld inverts WorldToScreen.
func (v *MapView) ScreenToWorld(sx, sy int) (wx, wy float64) {
	cx, cy := v.centerWorld()
	wx = cx + (float64(sx)-float64(v.mapX+v.mapW/2))/v.zoom
	wy = cy - (float64(sy)-float64(v.mapY+v.mapH/2))/v.zoom
	return wx, wy
}

// ContainsScreen reports whether a point is inside the map panel.
func (v *MapView) ContainsScreen(px, py int) bool {
	return px >= v.mapX && px < v.mapX+v.mapW && py >= v.mapY && py < v.mapY+v.mapH
}

// ZoomAt adjusts zoom keeping the world point under the cursor fixed.
func (v *MapView) ZoomAt(mx, my int, zoomIn bool) {
	if v.bathy == nil {
		return
	}
	wx, wy := v.ScreenToWorld(mx, my)
	if zoomIn {
		v.zoom = math.Min(zoomMax, v.zoom*zoomStep)
	} else {
		v.zoom = math.Max(zoomMin, v.zoom/zoomStep)
	}
	v.panX = wx - v.centerX - (float64(mx)-float64(v.mapX+v.mapW/2))/v.zoom
	v.panY = wy - v.centerY + (float64(my)-float64(v.mapY+v.mapH/2))/v.zoom
	v.invalidateBathy()
}

// PanByScreenDelta pans by mouse movement in pixels.
func (v *MapView) PanByScreenDelta(dx, dy int) {
	if v.zoom <= 0 {
		return
	}
	v.panX -= float64(dx) / v.zoom
	v.panY += float64(dy) / v.zoom
}

// Draw renders bathy raster, coastline, and routes with the current view transform.
func (v *MapView) Draw(screen *ebiten.Image) {
	if v.bathy == nil || v.zoom <= 0 {
		return
	}
	render.FillPlotBackground(screen, v.mapX, v.mapY, v.mapW, v.mapH)
	v.drawBathymetry(screen)
	v.drawRoutes(screen)
}
