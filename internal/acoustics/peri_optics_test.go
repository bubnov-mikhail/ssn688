package acoustics

import (
	"math"
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestEyeAboveWaterAtPeriscopeDepth(t *testing.T) {
	h := EyeAboveWaterFt(60, 1)
	if h < periMinEyeAboveWaterFt-0.01 {
		t.Fatalf("PD eye height %.2f, want >= %.0f ft floor", h, periMinEyeAboveWaterFt)
	}
	if EyeAboveWaterFt(60, 0) != 0 {
		t.Fatal("stowed should be 0")
	}
	half := EyeAboveWaterFt(60, 0.5)
	if math.Abs(half-h*0.5) > 0.01 {
		t.Fatalf("extension scale: %.2f vs %.2f", half, h*0.5)
	}
}

func TestGeometricHorizon(t *testing.T) {
	// 6 ft → ~2.87 nm → ~5800 yd
	h := GeometricHorizonYd(6)
	want := 1.17 * math.Sqrt(6) * world.YardsPerNM
	if math.Abs(h-want) > 1 {
		t.Fatalf("horizon %.0f want %.0f", h, want)
	}
}

func TestBearingToViewX(t *testing.T) {
	x, ok := BearingToViewX(90, 90, 32, 320)
	if !ok || x < 155 || x > 165 {
		t.Fatalf("center x=%d ok=%v", x, ok)
	}
	_, ok = BearingToViewX(90+20, 90, 32, 320)
	if ok {
		t.Fatal("20° off 32° FOV should be outside")
	}
	xL, ok := BearingToViewX(90-16, 90, 32, 320)
	if !ok || xL > 5 {
		t.Fatalf("left edge x=%d", xL)
	}
}

func TestBearingToViewXFSmooth(t *testing.T) {
	x0, ok := BearingToViewXF(90, 90, 32, 320)
	if !ok || math.Abs(x0-160) > 1 {
		t.Fatalf("center xf=%v ok=%v", x0, ok)
	}
	x1, _ := BearingToViewXF(90.05, 90, 32, 320)
	if x1 <= x0 {
		t.Fatalf("expected fractional advance: %v → %v", x0, x1)
	}
	if x1-x0 > 2 {
		t.Fatalf("0.05° step too large: %v", x1-x0)
	}
}


func TestShipBeamAspect(t *testing.T) {
	// Heading parallel to LOS: ship going away → stern toward eye.
	if ShipBeamAspect01(90, 90) > 0.05 {
		t.Fatal("end-on (going away) should be near 0 beam aspect")
	}
	if ShipBeamAspect01(90, 0) < 0.95 {
		t.Fatal("beam-on should be near 1")
	}
	// Ship north of eye (LOS 0°): heading 0° = going away → stern (180).
	if math.Abs(ShipAspectDeg(0, 0)-180) > 1 {
		t.Fatalf("going away want stern~180, got %v", ShipAspectDeg(0, 0))
	}
	// Heading 180° = coming toward eye → bow (0).
	if ShipAspectDeg(0, 180) > 1 {
		t.Fatalf("closing want bow~0, got %v", ShipAspectDeg(0, 180))
	}
	if math.Abs(ShipAspectDeg(0, 90)-90) > 1 {
		t.Fatalf("beam aspect want ~90, got %v", ShipAspectDeg(0, 90))
	}
	if ShipBeamAspect01(0, 180) > 0.05 {
		t.Fatal("bow-on beam aspect should be near 0")
	}
}

func TestProjectSurfaceShipInFOV(t *testing.T) {
	ship := &world.Entity{
		ID: "m", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 0, Y: 2000, HeadingDeg: 90, LengthFt: 520, SignatureID: "merchant",
	}
	const horizonY = 100
	// Look north toward ship at Y+.
	proj, ok := ProjectSurfaceShip(0, 0, 0, 32, 360, 200, horizonY, 8000, 6, ship)
	if !ok {
		t.Fatal("expected projection")
	}
	if proj.WidthPx < 2 {
		t.Fatalf("proj %#v", proj)
	}
	// At ~2000 yd / 6 ft HOE the waterline sits essentially on the horizon.
	if proj.WaterY > horizonY+4 {
		t.Fatalf("waterline too low for PD eye height: WaterY=%d horizon=%d", proj.WaterY, horizonY)
	}
	near := &world.Entity{
		ID: "n", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 0, Y: 120, HeadingDeg: 0, LengthFt: 520, SignatureID: "merchant",
	}
	nearProj, ok := ProjectSurfaceShip(0, 0, 0, 32, 360, 200, horizonY, 8000, 6, near)
	if !ok {
		t.Fatal("expected near projection")
	}
	if nearProj.WaterY <= proj.WaterY {
		t.Fatalf("near ship should sit lower than far: near=%d far=%d", nearProj.WaterY, proj.WaterY)
	}
	// Look east — ship is north, outside FOV.
	_, ok = ProjectSurfaceShip(0, 0, 90, 32, 360, 200, horizonY, 8000, 6, ship)
	if ok {
		t.Fatal("should be outside FOV")
	}
}

func TestSeaSurfacePixelYHugsHorizon(t *testing.T) {
	y := SeaSurfacePixelY(6, 2500, 360, 200, 90, 32)
	if y > 93 {
		t.Fatalf("2500 yd should hug horizon, got y=%d", y)
	}
}

func TestProjectSurfaceShipSinking(t *testing.T) {
	ship := &world.Entity{
		ID: "m", Kind: world.KindSurfaceShip, Status: world.StatusSinking,
		X: 0, Y: 800, HeadingDeg: 90, LengthFt: 520, SignatureID: "merchant",
		DepthFt: 0, SinkRateFPM: 25,
	}
	const horizonY = 100
	air := SurfaceShipAirDraftFt(520) // ~43.3 ft
	sec := SurfaceShipSubmergeSec(520, 25)
	wantSec := air / 25 * 60
	if math.Abs(sec-wantSec) > 0.01 {
		t.Fatalf("submerge sec %.1f want %.1f", sec, wantSec)
	}
	if sec < 90 || sec > 130 {
		t.Fatalf("merchant submerge ~%.0fs (air=%.0f ft @ 25 fpm), got %.0f", wantSec, air, sec)
	}

	proj, ok := ProjectSurfaceShip(0, 0, 0, 32, 360, 200, horizonY, 8000, 6, ship)
	if !ok || !proj.Sinking || proj.SinkFrac != 0 {
		t.Fatalf("just hit: ok=%v sinking=%v frac=%v", ok, proj.Sinking, proj.SinkFrac)
	}
	ship.DepthFt = air * 0.4
	proj, ok = ProjectSurfaceShip(0, 0, 0, 32, 360, 200, horizonY, 8000, 6, ship)
	if !ok || math.Abs(proj.SinkFrac-0.4) > 0.01 {
		t.Fatalf("partial: ok=%v frac=%v", ok, proj.SinkFrac)
	}
	ship.DepthFt = air
	_, ok = ProjectSurfaceShip(0, 0, 0, 32, 360, 200, horizonY, 8000, 6, ship)
	if ok {
		t.Fatal("fully submerged should not project")
	}
	ship.DepthFt = air + 50
	_, ok = ProjectSurfaceShip(0, 0, 0, 32, 360, 200, horizonY, 8000, 6, ship)
	if ok {
		t.Fatal("deep wreck should not project")
	}
}
