package weapons

import (
	"math"
	"testing"
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
