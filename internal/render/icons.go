package render

import (
	"image"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/bubnov-mikhail/ssn688/internal/render/navicon"
)

const (
	IconPassive = iota
	IconActive
	IconSpectrum
	IconLibrary
	IconWeapons
	IconManeuver
	IconTactical
	IconDamage
	IconMast

	// NavBarIconSize matches the icon slot in ui/navbar.go (left of label).
	NavBarIconSize = 40
)

var (
	navIconCache sync.Map // key: kind<<16|size → *ebiten.Image
)

func navIconKey(kind, size int) int {
	return kind<<16 | size
}

func navIconEbit(kind, size int) *ebiten.Image {
	if kind < 0 || kind >= len(navicon.Names) {
		return nil
	}
	key := navIconKey(kind, size)
	if v, ok := navIconCache.Load(key); ok {
		return v.(*ebiten.Image)
	}
	img := navicon.RasterKind(kind, size)
	ebit := ebiten.NewImageFromImage(img)
	navIconCache.Store(key, ebit)
	return ebit
}

// DrawScreenIcon draws a station icon centered at (cx, cy).
func DrawScreenIcon(screen *ebiten.Image, kind int, cx, cy, size int, clr color.Color) {
	src := navIconEbit(kind, size)
	if src == nil {
		return
	}
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw < 1 || sh < 1 {
		return
	}
	r, g, b, a := clr.RGBA()
	var opts ebiten.DrawImageOptions
	opts.GeoM.Translate(float64(cx)-float64(sw)/2, float64(cy)-float64(sh)/2)
	opts.ColorScale.Scale(float32(r)/65535, float32(g)/65535, float32(b)/65535, float32(a)/65535)
	screen.DrawImage(src, &opts)
}

// NavIconBounds returns the raster size of a nav icon at DesignSize (for tests).
func NavIconBounds(kind int) (int, int) {
	img := navicon.RasterKind(kind, navicon.DesignSize)
	return img.Bounds().Dx(), img.Bounds().Dy()
}

// NavIconOpaque reports whether an icon has any non-transparent pixels.
func NavIconOpaque(kind int) bool {
	if kind < 0 || kind >= len(navicon.Names) {
		return false
	}
	return navicon.HasOpaquePixels(navicon.Names[kind], navicon.DesignSize)
}

// NavIconAtSize returns the cached raster for tests.
func NavIconAtSize(kind, size int) image.Image {
	return navicon.RasterKind(kind, size)
}
