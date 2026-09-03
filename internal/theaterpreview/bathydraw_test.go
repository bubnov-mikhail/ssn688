package theaterpreview

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestCoastSegmentsAlignWithDepthAtFt(t *testing.T) {
	b := &world.Bathymetry{
		Width: 4, Height: 2,
		OriginX: 0, OriginY: 0,
		CellSize: 100,
		Depths: []float32{
			200, 200, -10, -10,
			200, 200, -10, -10,
		},
	}
	segs := buildCoastSegments(b)
	if len(segs) == 0 {
		t.Fatal("expected coast segments")
	}
	for _, s := range segs {
		mx := (s.x0 + s.x1) * 0.5
		my := (s.y0 + s.y1) * 0.5
		west := b.DepthAtFt(mx-1, my)
		east := b.DepthAtFt(mx+1, my)
		if west <= 0 || east >= 0 {
			t.Fatalf("segment (%.1f,%.1f)-(%.1f,%.1f) not on zero contour: depth west=%.2f east=%.2f",
				s.x0, s.y0, s.x1, s.y1, west, east)
		}
	}
}
