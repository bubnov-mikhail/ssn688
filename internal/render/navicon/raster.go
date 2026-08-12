package navicon

import "image"

// Names lists icon slugs in the same order as render.Icon* constants.
var Names = []string{
	"passive", "active", "spectrum", "library", "weapons",
	"maneuver", "tactical", "damage", "mast",
}

var drawers = map[string]func(*Canvas){
	"passive":  drawPassive,
	"active":   drawActive,
	"spectrum": drawSpectrum,
	"library":  drawLibrary,
	"weapons":  drawWeapons,
	"maneuver": drawManeuver,
	"mast":     drawMast,
	"damage":   drawDamage,
	"tactical": drawTactical,
}

// Raster draws a nav icon at px×px (white on transparent), normalized footprint.
func Raster(name string, px int) *image.RGBA {
	fn, ok := drawers[name]
	if !ok {
		return New(px).RGBA
	}
	c := New(DesignSize)
	fn(c)
	return normalizeIcon(c.RGBA, px)
}

// RasterKind draws by index matching render.Icon* constants.
func RasterKind(kind, px int) *image.RGBA {
	if kind < 0 || kind >= len(Names) {
		return New(px).RGBA
	}
	return Raster(Names[kind], px)
}

// HasOpaquePixels reports whether the icon has any visible pixels.
func HasOpaquePixels(name string, px int) bool {
	img := Raster(name, px)
	for y := 0; y < px; y++ {
		for x := 0; x < px; x++ {
			if img.Pix[img.PixOffset(x, y)+3] > 0 {
				return true
			}
		}
	}
	return false
}
