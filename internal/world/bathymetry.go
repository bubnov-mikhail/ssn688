package world

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Bathymetry is a fixed geographic depth grid (feet). Depth <= 0 is land.
type Bathymetry struct {
	Width, Height int
	OriginX       float64 // SW corner, yards east
	OriginY       float64 // SW corner, yards north
	CellSize      float64 // yards
	Depths        []float32
}

// LoadBathymetry parses a BATH binary chart.
func LoadBathymetry(data []byte) (Bathymetry, error) {
	if len(data) < 36 || string(data[0:4]) != "BATH" {
		return Bathymetry{}, fmt.Errorf("invalid bathymetry header")
	}
	ver := binary.LittleEndian.Uint32(data[4:8])
	if ver != 1 {
		return Bathymetry{}, fmt.Errorf("unsupported bathymetry version %d", ver)
	}
	w := int(binary.LittleEndian.Uint32(data[8:12]))
	h := int(binary.LittleEndian.Uint32(data[12:16]))
	ox := math.Float64frombits(binary.LittleEndian.Uint64(data[16:24]))
	oy := math.Float64frombits(binary.LittleEndian.Uint64(data[24:32]))
	cell := math.Float64frombits(binary.LittleEndian.Uint64(data[32:40]))
	need := 40 + w*h*4
	if len(data) < need {
		return Bathymetry{}, fmt.Errorf("bathymetry truncated: have %d need %d", len(data), need)
	}
	depths := make([]float32, w*h)
	for i := range depths {
		off := 40 + i*4
		depths[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[off : off+4]))
	}
	return Bathymetry{
		Width: w, Height: h,
		OriginX: ox, OriginY: oy,
		CellSize: cell,
		Depths:   depths,
	}, nil
}

func (b Bathymetry) Valid() bool {
	return b.Width > 0 && b.Height > 0 && len(b.Depths) == b.Width*b.Height
}

func (b Bathymetry) index(i, j int) int {
	return j*b.Width + i
}

// DepthAtFt returns seafloor depth in feet at world (x,y) yards. Land is <= 0.
func (b Bathymetry) DepthAtFt(x, y float64) float64 {
	if !b.Valid() {
		return 2200
	}
	fx := (x - b.OriginX) / b.CellSize
	fy := (y - b.OriginY) / b.CellSize
	if fx < 0 || fy < 0 || fx >= float64(b.Width-1) || fy >= float64(b.Height-1) {
		return -50 // off-chart treated as land/blocked
	}
	i0 := int(fx)
	j0 := int(fy)
	tx := fx - float64(i0)
	ty := fy - float64(j0)
	d00 := float64(b.Depths[b.index(i0, j0)])
	d10 := float64(b.Depths[b.index(i0+1, j0)])
	d01 := float64(b.Depths[b.index(i0, j0+1)])
	d11 := float64(b.Depths[b.index(i0+1, j0+1)])
	return d00*(1-tx)*(1-ty) + d10*tx*(1-ty) + d01*(1-tx)*ty + d11*tx*ty
}

// IsLand reports dry ground / chart edge.
func (b Bathymetry) IsLand(x, y float64) bool {
	return b.DepthAtFt(x, y) <= 0
}

// NavigableFor reports whether a unit of the given kind/depth can occupy (x,y).
func (b Bathymetry) NavigableFor(x, y float64, kind EntityKind, depthFt float64) bool {
	bottom := b.DepthAtFt(x, y)
	if bottom <= 0 {
		return false
	}
	switch kind {
	case KindSurfaceShip:
		return bottom >= 40
	case KindSubmarine:
		keel := depthFt + 40
		return bottom >= keel
	default:
		return bottom >= 40
	}
}

// BoundsYards returns world-space extent of the chart.
func (b Bathymetry) BoundsYards() (minX, minY, maxX, maxY float64) {
	minX, minY = b.OriginX, b.OriginY
	maxX = b.OriginX + float64(b.Width)*b.CellSize
	maxY = b.OriginY + float64(b.Height)*b.CellSize
	return
}
