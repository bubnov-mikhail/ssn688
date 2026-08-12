package ai

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestColregsHeadOnTurnsStarboard(t *testing.T) {
	self := &world.Entity{
		ID: "a", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 10, OrderedSpeed: 10, OrderedHead: 0,
		AIState: "CRUISE", RouteID: "r",
	}
	other := &world.Entity{
		ID: "b", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 0, Y: 1500, HeadingDeg: 180, SpeedKts: 10, OrderedSpeed: 10, OrderedHead: 180,
		AIState: "CRUISE",
	}
	applyColregsTraffic(self, []*world.Entity{self, other})
	if self.AIState != "AVOID" {
		t.Fatalf("state=%s want AVOID", self.AIState)
	}
	if !self.RouteNeedResume {
		t.Fatal("route should be interrupted")
	}
	// Starboard from heading 0 is toward 40–90°.
	if self.OrderedHead < 20 || self.OrderedHead > 120 {
		t.Fatalf("head-on starboard turn got heading %.0f", self.OrderedHead)
	}
}

func TestColregsStandOnNoAvoid(t *testing.T) {
	self := &world.Entity{
		ID: "a", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 10, OrderedSpeed: 10, OrderedHead: 0,
		AIState: "CRUISE", RouteID: "r",
	}
	// Crossing from port — we are stand-on if other is give-way (other has us on starboard).
	other := &world.Entity{
		ID: "b", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: -2000, Y: 2000, HeadingDeg: 90, SpeedKts: 10, OrderedSpeed: 10, OrderedHead: 90,
		AIState: "CRUISE",
	}
	applyColregsTraffic(self, []*world.Entity{self, other})
	if self.AIState == "AVOID" {
		t.Fatalf("stand-on should not avoid (got AVOID head=%.0f)", self.OrderedHead)
	}
}
