package weapons

import (
	"math"
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestInterceptCourseStationaryTarget(t *testing.T) {
	// Target due east of shooter — intercept heading should be ~90°.
	course, ok := InterceptCourseDeg(4000, 0, 0, 0, 55)
	if !ok {
		t.Fatal("expected intercept")
	}
	if math.Abs(course-90) > 0.5 {
		t.Fatalf("course=%.1f want 90", course)
	}
}

func TestInterceptCourseLeadsMovingTarget(t *testing.T) {
	// Target north of shooter, steaming east — gyro should lead east of north.
	course, ok := InterceptCourseDeg(0, 5000, 90, 20, 55)
	if !ok {
		t.Fatal("expected intercept")
	}
	if course < 5 || course > 45 {
		t.Fatalf("expected lead east of north, got %.1f", course)
	}
}

func TestInterceptFailsWhenWeaponTooSlow(t *testing.T) {
	// Target fleeing north faster than weapon.
	_, ok := InterceptCourseDeg(0, 2000, 0, 40, 28)
	if ok {
		t.Fatal("expected no intercept against faster fleeing target")
	}
}

func TestTorpedoInterceptGyroAccountsForTubeClear(t *testing.T) {
	// Ownship heading east; target north. Gyro after clear should still exist.
	course, ok := TorpedoInterceptGyro(0, 0, 90, 0, 6000, 90, 12, 55)
	if !ok {
		t.Fatal("expected intercept")
	}
	if course < 0 || course >= 360 {
		t.Fatalf("bad course %.1f", course)
	}
}

func TestDeferredSearchStaysWireUntilSafeDistance(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive,
		X: 0, Y: 0, DepthFt: 300,
	}
	target := &world.Entity{
		ID: "enemy", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive,
		X: 8000, Y: 0, DepthFt: 250,
	}
	targets := []*world.Entity{player, target}

	fish := &Torpedo{
		ID: "MK48-1", ParentSubID: "player", Side: world.SidePlayer,
		X: 50, Y: 0, DepthFt: 300, HeadingDeg: 90, OrderedHead: 90,
		LaunchHeadDeg: 90, GyroCourseDeg: 90,
		SpeedKts: 40, CruiseKts: 55, RunDepthFt: 300,
		Mode: ModeWire, Alive: true, Armed: true,
		ClearDistYd: TubeClearYd, EnableSearchAfterClear: true,
	}
	fish.MarkGyroEnabled(true)

	for i := 0; i < 200; i++ {
		fish.Advance(0.1, float64(i)*0.1, targets, nil, nil)
		if fish.Mode == ModeSearch {
			d := math.Hypot(fish.X-player.X, fish.Y-player.Y)
			if d < SearchArmMinDistYd-20 {
				t.Fatalf("search armed too close to launcher: %.0f yd at tick %d", d, i)
			}
			return
		}
	}
	t.Fatal("seeker never armed after run-out")
}

func TestShootWithPrearmedSeekerDefersSearch(t *testing.T) {
	fc := NewFireControl()
	fc.SeekerEnabled = true
	fc.GyroAngleDeg = 45
	sub := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		X: 0, Y: 0, DepthFt: 300, HeadingDeg: 0,
	}
	tube := fc.TubeByNumber(1)
	tube.State = TubeDoorOpen
	tube.TorpedoType = OrdnanceMk48

	fish := fc.Shoot(sub, 1)
	if fish == nil {
		t.Fatal("shoot failed")
	}
	if fish.Mode != ModeWire || fish.SeekerOn {
		t.Fatalf("launch state mode=%d seek=%v", fish.Mode, fish.SeekerOn)
	}
	if !fish.EnableSearchAfterClear {
		t.Fatal("expected deferred search flag")
	}
}
