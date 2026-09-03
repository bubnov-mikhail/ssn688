package ai

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestFollowAssignedRouteAdvances(t *testing.T) {
	r := &world.Route{
		ID: "r1",
		Waypoints: []world.Waypoint{
			{X: 0, Y: 0},
			{X: 2000, Y: 0},
			{X: 4000, Y: 0},
		},
		PingPong: true,
	}
	ship := &world.Entity{
		ID: "s", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 100, Y: 0, RouteID: "r1", RouteWP: 0, RouteDir: 1,
		Damage: world.NewFullHealth(), SignatureID: "merchant",
	}
	if !followAssignedRoute(ship, []*world.Route{r}, "CRUISE", 11) {
		t.Fatal("follow failed")
	}
	if ship.AIState != "CRUISE" {
		t.Fatalf("state=%s", ship.AIState)
	}
	if ship.RouteWP != 1 {
		t.Fatalf("wp=%d want 1 after visit", ship.RouteWP)
	}
	if ship.OrderedHead < 80 || ship.OrderedHead > 100 {
		t.Fatalf("heading=%.0f want ~90 toward +X", ship.OrderedHead)
	}
}

func TestFollowResumeAfterInterrupt(t *testing.T) {
	r := &world.Route{
		ID: "r1",
		Waypoints: []world.Waypoint{
			{X: 0, Y: 0},
			{X: 2000, Y: 0},
			{X: 4000, Y: 0},
		},
		PingPong: true,
	}
	ship := &world.Entity{
		ID: "s", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 3200, Y: 50, RouteID: "r1", RouteWP: 0, RouteNeedResume: true,
		Damage: world.NewFullHealth(), SignatureID: "grisha",
	}
	followAssignedRoute(ship, []*world.Route{r}, "PATROL", 14)
	if ship.RouteNeedResume {
		t.Fatal("resume flag should clear")
	}
	if ship.RouteWP != 2 {
		t.Fatalf("resume wp=%d want 2 (near end)", ship.RouteWP)
	}
	if ship.RouteDir != -1 {
		t.Fatalf("dir=%d want -1 at terminus", ship.RouteDir)
	}
}

func TestFollowPingPongReversesAtEnd(t *testing.T) {
	r := &world.Route{
		ID: "r1",
		Waypoints: []world.Waypoint{
			{X: 0, Y: 0},
			{X: 2000, Y: 0},
			{X: 4000, Y: 0},
		},
		PingPong: true,
	}
	ship := &world.Entity{
		ID: "s", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 4000, Y: 0, RouteID: "r1", RouteWP: 2, RouteDir: 1,
		Damage: world.NewFullHealth(), SignatureID: "merchant",
	}
	followAssignedRoute(ship, []*world.Route{r}, "CRUISE", 11)
	if ship.RouteDir != -1 {
		t.Fatalf("dir=%d want -1", ship.RouteDir)
	}
	if ship.RouteWP != 1 {
		t.Fatalf("wp=%d want 1 after reverse", ship.RouteWP)
	}
}

func TestFollowOpenRouteResumeForward(t *testing.T) {
	r := &world.Route{
		ID: "r1",
		Waypoints: []world.Waypoint{
			{X: 0, Y: 0},
			{X: 2000, Y: 0},
			{X: 4000, Y: 0},
			{X: 6500, Y: -500},
		},
	}
	ship := &world.Entity{
		ID: "s", Kind: world.KindSurfaceShip, Side: world.SideNeutral, Status: world.StatusActive,
		X: 3200, Y: 40, RouteID: "r1", RouteWP: 2, RouteNeedResume: true,
		Damage: world.NewFullHealth(), SignatureID: "fishing",
	}
	followAssignedRoute(ship, []*world.Route{r}, "CRUISE", 7)
	if ship.RouteWP != 2 {
		t.Fatalf("open resume wp=%d want 2 (no backtrack to wp0)", ship.RouteWP)
	}
}
