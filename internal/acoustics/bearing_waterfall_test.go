package acoustics

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestWaterfallFlowNoiseAtHighSpeed(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	player := testEntity("player", "los_angeles", world.KindSubmarine, 180, 21)
	sonar := NewSonarState()
	row := BearingWaterfallSlice(model, player, []*world.Entity{player}, &sonar, PassiveArrayHull, 0)
	peak := 0.0
	for _, v := range row.Bearings {
		if v > peak {
			peak = v
		}
	}
	if peak < 12 {
		t.Fatalf("high speed should wash waterfall with flow noise, peak=%.1f", peak)
	}
}

func TestWaterfallPeakDecreasesWithRange(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	player := testEntity("player", "los_angeles", world.KindSubmarine, 180, 8)
	enemy := testEntity("dd", "udaloy", world.KindSurfaceShip, 0, 14)
	sonar := NewSonarState()
	emitters := []*world.Entity{player, enemy}

	enemy.Y = 2500
	near := waterfallPeak(model, player, emitters, &sonar)
	enemy.Y = 6000
	mid := waterfallPeak(model, player, emitters, &sonar)
	enemy.Y = 11000
	far := waterfallPeak(model, player, emitters, &sonar)

	if mid >= near-4 || far >= mid-4 {
		t.Fatalf("waterfall peak should fade with range: near=%.1f mid=%.1f far=%.1f", near, mid, far)
	}
}

func waterfallPeak(model Model, player *world.Entity, emitters []*world.Entity, sonar *SonarState) float64 {
	row := BearingWaterfallSlice(model, player, emitters, sonar, PassiveArrayHull, 0)
	peak := 0.0
	for _, v := range row.Bearings {
		if v > peak {
			peak = v
		}
	}
	return peak
}

func TestBearingWaterfallSliceDetectsNearbyTarget(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	player := testEntity("player", "los_angeles", world.KindSubmarine, 320, 8)
	enemy := testEntity("dd", "udaloy", world.KindSurfaceShip, 0, 14)
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
	enemy := testEntity("dd", "udaloy", world.KindSurfaceShip, 0, 14)
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
	if aboveHalf < 4 {
		t.Fatalf("expected compact contact smear, bins above half=%d", aboveHalf)
	}
	if aboveHalf > 28 {
		t.Fatalf("contact smear too wide, bins above half=%d", aboveHalf)
	}
}

func TestWaterfallListenBandSuppressesOffBandTargets(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	player := testEntity("player", "los_angeles", world.KindSubmarine, 320, 8)
	sonar := NewSonarState()

	torp := &world.Entity{
		ID: "mk48-1", SignatureID: "mk48", Kind: world.KindTorpedo, Status: world.StatusActive,
		Y: 2200, DepthFt: 200, SpeedKts: 50, HeadingDeg: 180,
	}
	ship := testEntity("dd", "udaloy", world.KindSurfaceShip, 0, 14)
	ship.Y = 2200

	emitters := []*world.Entity{player, torp}
	sonar.ListenBand = ListenBroadband
	bbTorp := waterfallGlobalPeak(model, player, emitters, &sonar)
	sonar.ListenBand = ListenHF
	hfTorp := waterfallGlobalPeak(model, player, emitters, &sonar)
	if bbTorp >= 6 || hfTorp <= bbTorp+3 {
		t.Fatalf("broadband should suppress torpedo trace: bb=%.1f hf=%.1f", bbTorp, hfTorp)
	}

	emitters = []*world.Entity{player, ship}
	sonar.ListenBand = ListenBroadband
	bbShip := waterfallGlobalPeak(model, player, emitters, &sonar)
	sonar.ListenBand = ListenHF
	hfShip := waterfallGlobalPeak(model, player, emitters, &sonar)
	if hfShip >= 6 || bbShip <= hfShip+3 {
		t.Fatalf("HF band should suppress ship trace: bb=%.1f hf=%.1f", bbShip, hfShip)
	}
}

func waterfallGlobalPeak(model Model, player *world.Entity, emitters []*world.Entity, sonar *SonarState) float64 {
	row := BearingWaterfallSlice(model, player, emitters, sonar, PassiveArrayHull, 0)
	peak := 0.0
	for _, v := range row.Bearings {
		if v > peak {
			peak = v
		}
	}
	return peak
}

func TestTorpedoPingOnlyWhenFacingListener(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	player := testEntity("player", "los_angeles", world.KindSubmarine, 320, 8)
	torp := &world.Entity{
		ID: "mk48-1", SignatureID: "mk48", Kind: world.KindTorpedo, Status: world.StatusActive,
		X: 6000, Y: 0, DepthFt: 200, SpeedKts: 50, LastPingTime: 10,
	}
	sonar := NewSonarState()
	emitters := []*world.Entity{player, torp}

	torp.HeadingDeg = 270 // west — torpedo at +X points toward player at origin
	facing := BearingWaterfallSlice(model, player, emitters, &sonar, PassiveArrayHull, 10.2)
	torp.HeadingDeg = 90 // east — away from listener
	away := BearingWaterfallSlice(model, player, emitters, &sonar, PassiveArrayHull, 10.2)

	bin90 := BearingWaterfallBins / 4
	if facing.Bearings[bin90] < 20 {
		t.Fatalf("torpedo ping should flash when facing listener, bin90=%.1f", facing.Bearings[bin90])
	}
	if away.Bearings[bin90] > facing.Bearings[bin90]*0.35 {
		t.Fatalf("torpedo ping should be weak when not facing listener: facing90=%.1f away90=%.1f", facing.Bearings[bin90], away.Bearings[bin90])
	}
}

func TestShipPingVisibleInBothListenBands(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	player := testEntity("player", "los_angeles", world.KindSubmarine, 320, 8)
	enemy := testEntity("dd", "udaloy", world.KindSurfaceShip, 0, 14)
	enemy.X = 6000
	enemy.Y = 0
	enemy.LastPingTime = 10
	sonar := NewSonarState()
	emitters := []*world.Entity{player, enemy}

	sonar.ListenBand = ListenBroadband
	bb := BearingWaterfallSlice(model, player, emitters, &sonar, PassiveArrayHull, 10.2)
	sonar.ListenBand = ListenHF
	hf := BearingWaterfallSlice(model, player, emitters, &sonar, PassiveArrayHull, 10.2)

	bin90 := BearingWaterfallBins / 4
	if bb.Bearings[bin90] < 15 || hf.Bearings[bin90] < 15 {
		t.Fatalf("ship ping should be visible in both bands: bb90=%.1f hf90=%.1f", bb.Bearings[bin90], hf.Bearings[bin90])
	}
}

func TestEnemyActivePingAppearsOnWaterfall(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	player := testEntity("player", "los_angeles", world.KindSubmarine, 320, 8)
	enemy := testEntity("dd", "udaloy", world.KindSurfaceShip, 0, 14)
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
