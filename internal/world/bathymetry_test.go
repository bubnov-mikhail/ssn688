package world

import (
	"encoding/binary"
	"math"
	"testing"
)

func encodeTestBathy(w, h int, originX, originY, cell float64, depth float32) []byte {
	buf := make([]byte, 40+w*h*4)
	copy(buf[0:4], "BATH")
	binary.LittleEndian.PutUint32(buf[4:8], 1)
	binary.LittleEndian.PutUint32(buf[8:12], uint32(w))
	binary.LittleEndian.PutUint32(buf[12:16], uint32(h))
	binary.LittleEndian.PutUint64(buf[16:24], math.Float64bits(originX))
	binary.LittleEndian.PutUint64(buf[24:32], math.Float64bits(originY))
	binary.LittleEndian.PutUint64(buf[32:40], math.Float64bits(cell))
	for i := 0; i < w*h; i++ {
		off := 40 + i*4
		d := depth
		// Land strip along left edge for coastline tests.
		if i%w == 0 {
			d = -10
		}
		binary.LittleEndian.PutUint32(buf[off:off+4], math.Float32bits(d))
	}
	return buf
}

func TestBathymetryLoadAndCenterWater(t *testing.T) {
	raw := encodeTestBathy(8, 8, -1000, -1000, 250, 2000)
	b, err := LoadBathymetry(raw)
	if err != nil {
		t.Fatal(err)
	}
	if b.DepthAtFt(0, 0) <= 0 {
		t.Fatalf("chart center should be water, depth=%.0f", b.DepthAtFt(0, 0))
	}
	if !b.NavigableFor(0, 0, KindSubmarine, 180) {
		t.Fatalf("center should be navigable for SSN at 180 ft, depth=%.0f", b.DepthAtFt(0, 0))
	}
	land := 0
	for _, d := range b.Depths {
		if d <= 0 {
			land++
		}
	}
	if land < 1 {
		t.Fatalf("expected coastline land cells, got %d", land)
	}
}
