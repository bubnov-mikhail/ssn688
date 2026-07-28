package acoustics

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestBearingWaterfallSliceDetectsNearbyTarget(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	player := testEntity("player", "los_angeles", world.KindSubmarine, 320, 8)
	enemy := testEntity("dd", "spruance", world.KindSurfaceShip, 0, 14)
	enemy.X = 4200
	enemy.Y = 3200
	sonar := NewSonarState()

	row := BearingWaterfallSlice(model, player, []*world.Entity{player, enemy}, &sonar, PassiveArrayHull, 0)
	peak := 0.0
	for _, v := range row.Bearings {
		if v > peak {
			peak = v
		}
	}
	if peak < 5 {
		t.Fatalf("expected audible energy on bearing waterfall, peak=%.1f", peak)
	}
}

func TestHeadingToWaterfallX(t *testing.T) {
	if HeadingToWaterfallX(0, 360) != 0 {
		t.Fatal("0 deg should map to left edge")
	}
	if HeadingToWaterfallX(180, 360) != 180 {
		t.Fatal("180 deg should map to center")
	}
}

func TestHullAndTowedWaterfallDiffer(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	player := testEntity("player", "los_angeles", world.KindSubmarine, 320, 8)
	enemy := testEntity("dd", "spruance", world.KindSurfaceShip, 0, 14)
	enemy.X = 4200
	enemy.Y = 3200
	sonar := NewSonarState()
	sonar.TowedCablePct = 1
	emitters := []*world.Entity{player, enemy}

	hull := BearingWaterfallSlice(model, player, emitters, &sonar, PassiveArrayHull, 10)
	towed := BearingWaterfallSlice(model, player, emitters, &sonar, PassiveArrayTowed, 10)

	hullPeak, towedPeak := 0.0, 0.0
	diffSum := 0.0
	for i := range hull.Bearings {
		if hull.Bearings[i] > hullPeak {
			hullPeak = hull.Bearings[i]
		}
		if towed.Bearings[i] > towedPeak {
			towedPeak = towed.Bearings[i]
		}
		diffSum += abs(hull.Bearings[i] - towed.Bearings[i])
	}
	if hullPeak < 5 || towedPeak < 5 {
		t.Fatalf("expected audible returns hull=%.1f towed=%.1f", hullPeak, towedPeak)
	}
	if diffSum < 5 {
		t.Fatalf("hull and towed waterfalls should differ, diffSum=%.1f", diffSum)
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func TestContactSpreadIsGradual(t *testing.T) {
	bins := make([]float64, BearingWaterfallBins)
	spreadBearingEnergy(bins, 90, 20, PassiveArrayHull, 0, 1)
	peak := 0.0
	aboveHalf := 0
	for _, v := range bins {
		if v > peak {
			peak = v
		}
		if v > 10 {
			aboveHalf++
		}
	}
	// Compact contact blob: a few degrees of smear, not a wide wash.
	if aboveHalf < 6 {
		t.Fatalf("expected compact contact smear, bins above half=%d", aboveHalf)
	}
	if aboveHalf > 40 {
		t.Fatalf("contact smear too wide, bins above half=%d", aboveHalf)
	}
}

func TestEnemyActivePingAppearsOnWaterfall(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	player := testEntity("player", "los_angeles", world.KindSubmarine, 320, 8)
	enemy := testEntity("dd", "spruance", world.KindSurfaceShip, 0, 14)
	enemy.X = 6000
	enemy.Y = 0
	enemy.LastPingTime = 10
	sonar := NewSonarState()

	quiet := BearingWaterfallSlice(model, player, []*world.Entity{player, enemy}, &sonar, PassiveArrayHull, 0)
	loud := BearingWaterfallSlice(model, player, []*world.Entity{player, enemy}, &sonar, PassiveArrayHull, 10.2)

	quietPeak, loudPeak := 0.0, 0.0
	for i := range quiet.Bearings {
		if quiet.Bearings[i] > quietPeak {
			quietPeak = quiet.Bearings[i]
		}
		if loud.Bearings[i] > loudPeak {
			loudPeak = loud.Bearings[i]
		}
	}
	if loudPeak < quietPeak+6 {
		t.Fatalf("expected active ping flash on waterfall, quiet=%.1f loud=%.1f", quietPeak, loudPeak)
	}
	// Ping should peak near east (90°) for enemy at +X.
	bin90 := BearingWaterfallBins / 4
	if loud.Bearings[bin90] < quiet.Bearings[bin90]+5 {
		t.Fatalf("ping energy should rise on target bearing, quiet90=%.1f loud90=%.1f", quiet.Bearings[bin90], loud.Bearings[bin90])
	}
}
