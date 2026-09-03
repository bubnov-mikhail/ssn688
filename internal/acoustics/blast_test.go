package acoustics

import (
	"math"
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestBlastWashoutFadesSmoothly(t *testing.T) {
	sonar := NewSonarState()
	listener := &world.Entity{ID: "p", X: 0, Y: 0, HeadingDeg: 0, Status: world.StatusActive, SpeedKts: 5}
	ApplyDetonationDeaf(&sonar, listener, 800, 0, 10, &world.Entity{Kind: world.KindSurfaceShip, X: 800, Y: 0})

	arrive := sonar.LastBlastAt
	if arrive <= 10 {
		t.Fatalf("acoustic arrival should be after detonation: arrive=%.3f detonate=%.3f", arrive, sonar.LastBlastDetonateAt)
	}

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

	// Before sound arrives — no acoustic washout on waterfall.
	if p := atBearing(10.05); p > 15 {
		t.Fatalf("waterfall washout before travel delay: %.1f", p)
	}

	p0 := atBearing(arrive + 0.4)
	p1 := atBearing(arrive + sonar.LastBlastFlashSec*0.55)
	p2 := atBearing(arrive + sonar.LastBlastFlashSec*0.92)
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

func TestBlastAcousticDelayMatchesSoundSpeed(t *testing.T) {
	sonar := NewSonarState()
	listener := &world.Entity{ID: "p", X: 0, Y: 0, Status: world.StatusActive}
	const rangeYd = 3238.0 // exactly 2s at SoundSpeedYdPerSec
	ApplyDetonationDeaf(&sonar, listener, rangeYd, 0, 100, nil)
	if sonar.LastBlastDetonateAt != 100 {
		t.Fatalf("detonate at: got %.3f", sonar.LastBlastDetonateAt)
	}
	wantArrive := 100 + rangeYd/SoundSpeedYdPerSec
	if math.Abs(sonar.LastBlastAt-wantArrive) > 0.01 {
		t.Fatalf("arrive=%.3f want %.3f", sonar.LastBlastAt, wantArrive)
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
