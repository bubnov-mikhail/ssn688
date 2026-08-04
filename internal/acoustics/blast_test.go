package acoustics

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestBlastWashoutFadesSmoothly(t *testing.T) {
	sonar := NewSonarState()
	listener := &world.Entity{ID: "p", X: 0, Y: 0, HeadingDeg: 0, Status: world.StatusActive, SpeedKts: 5}
	ApplyDetonationDeaf(&sonar, listener, 800, 0, 10, &world.Entity{Kind: world.KindSurfaceShip})

	atBearing := func(gameTime float64) float64 {
		bins := make([]float64, BearingWaterfallBins)
		BearingWaterfallInto(bins, NewModel(DefaultEnvironment()), listener, nil, &sonar, PassiveArrayHull, gameTime)
		// Blast is due east → ~90°.
		bin := BearingWaterfallBins / 4
		peak := bins[bin]
		for i := bin - 8; i <= bin+8; i++ {
			bi := i
			for bi < 0 {
				bi += BearingWaterfallBins
			}
			for bi >= BearingWaterfallBins {
				bi -= BearingWaterfallBins
			}
			if bins[bi] > peak {
				peak = bins[bi]
			}
		}
		return peak
	}

	p0 := atBearing(10.4)
	p1 := atBearing(10 + sonar.LastBlastFlashSec*0.55)
	p2 := atBearing(10 + sonar.LastBlastFlashSec*0.92)
	if p0 < 25 {
		t.Fatalf("early washout too weak: %.1f", p0)
	}
	if p1 >= p0*0.75 {
		t.Fatalf("washout should decay: early=%.1f mid=%.1f", p0, p1)
	}
	// Late in the window energy should be near ambient (no hard cliff from a still-bright flash).
	if p2 > p0*0.35 {
		t.Fatalf("late washout should be faint vs early: early=%.1f late=%.1f", p0, p2)
	}
}

func TestCookOffDeafIsWeakerThanPrimary(t *testing.T) {
	listener := &world.Entity{ID: "p", Status: world.StatusActive}
	ship := &world.Entity{Kind: world.KindSurfaceShip}
	primary := NewSonarState()
	cook := NewSonarState()
	ApplyDetonationDeaf(&primary, listener, 500, 0, 0, ship)
	ApplyCookOffDeaf(&cook, listener, 500, 0, 0, ship)
	if cook.LastBlastRangeYd >= primary.LastBlastRangeYd {
		t.Fatalf("cook-off should be shorter range: cook=%.0f primary=%.0f", cook.LastBlastRangeYd, primary.LastBlastRangeYd)
	}
	if cook.LastBlastFlashSec >= primary.LastBlastFlashSec {
		t.Fatalf("cook-off flash should be shorter: cook=%.0f primary=%.0f", cook.LastBlastFlashSec, primary.LastBlastFlashSec)
	}
}
