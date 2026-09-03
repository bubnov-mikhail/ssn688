//go:build ignore

package main

import (
	"fmt"
	"math"
	"os"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func main() {
	raw, _ := os.ReadFile("scenarios_generated/theater_bathy/taiwan_strait.bin")
	bathy, _ := world.LoadBathymetry(raw)
	b := &bathy
	cx, cy := 5.0, 166.0
	orbit := []struct{ dx, dy float64 }{
		{-4000, -2400}, {4000, -2400}, {4000, 2000}, {-4000, -1200},
	}
	for i, p := range orbit {
		x, y := p.dx+cx, p.dy+cy
		fmt.Printf("wp%d (%.0f,%.0f) dist=%.0f sub60=%v depth=%.0f\n", i+1, x, y,
			math.Hypot(x-cx, y-cy), b.NavigableFor(x, y, world.KindSubmarine, 60), b.DepthAtFt(x, y))
	}
}
