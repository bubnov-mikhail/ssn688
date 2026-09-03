package ai

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestColregsNeutralDoesNotInterruptRoute(t *testing.T) {
	r := &world.Route{
		ID: "r1",
		Waypoints: []world.Waypoint{
			{X: 0, Y: 0},
			{X: 2000, Y: 0},
			{X: 4000, Y: 0},
		},
	}
	self := &world.Entity{
		ID: "civ", Kind: world.KindSurfaceShip, Side: world.SideNeutral,
		Status: world.StatusActive, X: 1000, Y: 0, HeadingDeg: 90, SpeedKts: 7,
		RouteID: "r1", RouteWP: 1, AIState: "CRUISE",
		Damage: world.NewFullHealth(), SignatureID: "fishing",
	}
	other := &world.Entity{
		ID: "other", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 1800, Y: 200, HeadingDeg: 270, SpeedKts: 14,
		Damage: world.NewFullHealth(), SignatureID: "grisha",
	}
	applyColregsTraffic(self, []*world.Entity{self, other})
	if self.RouteNeedResume {
		t.Fatal("neutral COLREGS must not interrupt scripted route")
	}
	if self.AIState == "AVOID" {
		t.Fatalf("neutral should stay on cruise, state=%s", self.AIState)
	}
	_ = followAssignedRoute(self, []*world.Route{r}, "CRUISE", 7)
	if self.RouteWP < 1 {
		t.Fatalf("route wp=%d", self.RouteWP)
	}
}
