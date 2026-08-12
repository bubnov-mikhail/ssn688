package render

import (
	"testing"

	"github.com/ssn688/sim/internal/render/navicon"
)

func TestNavIconsRasterized(t *testing.T) {
	kinds := []int{
		IconPassive, IconActive, IconSpectrum, IconLibrary, IconWeapons,
		IconManeuver, IconTactical, IconDamage, IconMast,
	}
	for _, k := range kinds {
		w, h := NavIconBounds(k)
		if w != navicon.DesignSize || h != navicon.DesignSize {
			t.Fatalf("icon %d size %dx%d, want %dx%d", k, w, h, navicon.DesignSize, navicon.DesignSize)
		}
		if !NavIconOpaque(k) {
			t.Fatalf("icon %d missing pixels", k)
		}
	}
}

func TestNavBarIconSize(t *testing.T) {
	img := NavIconAtSize(IconPassive, NavBarIconSize)
	if img.Bounds().Dx() != NavBarIconSize || img.Bounds().Dy() != NavBarIconSize {
		t.Fatalf("navbar icon size %dx%d, want %dx%d",
			img.Bounds().Dx(), img.Bounds().Dy(), NavBarIconSize, NavBarIconSize)
	}
}
