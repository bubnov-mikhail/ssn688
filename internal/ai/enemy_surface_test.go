package ai

import (
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

func seedVeteranTrack(ship, player *world.Entity) {
	ship.CrewSkill = 90
	ship.Track = world.AITrack{
		Valid: true, ClassConf: 0.85, HoldSec: 60,
		X: player.X, Y: player.Y, DepthFt: player.DepthFt,
		CourseDeg: player.HeadingDeg, SpeedKts: player.SpeedKts,
	}
}

func TestSurfaceAIHoldsRastrubStandoff(t *testing.T) {
	ship := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 16,
		Damage: world.NewFullHealth(), LastPingTime: -100, Defcon: world.DefconHostile,
		SignatureID: "udaloy",
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 5500, DepthFt: 200, SpeedKts: 6,
	}
	seedVeteranTrack(ship, player)
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	updateSurfaceAI(ship, player, 100, 0.1, model, nil, EvadeContext{}, nil)
	if ship.AIState != "RASTRUB" {
		t.Fatalf("expected RASTRUB, got %s", ship.AIState)
	}
	if ship.OrderedSpeed > 19 {
		t.Fatalf("standoff speed too high: %.1f", ship.OrderedSpeed)
	}
}

func TestSurfaceAIShipTubeBand(t *testing.T) {
	ship := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0,
		Damage: world.NewFullHealth(), LastPingTime: -100, Defcon: world.DefconHostile,
		SignatureID: "udaloy",
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 1600, DepthFt: 180,
	}
	seedVeteranTrack(ship, player)
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	updateSurfaceAI(ship, player, 50, 0.1, model, nil, EvadeContext{}, nil)
	if ship.AIState != "SHIP_TUBE" {
		t.Fatalf("expected SHIP_TUBE, got %s", ship.AIState)
	}
	_ = weapons.ShipTubeMinRangeYd
}

func TestSurfaceAIGreenStaysTrackingUntilClassified(t *testing.T) {
	ship := &world.Entity{
		ID: "grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, SpeedKts: 14,
		Damage: world.NewFullHealth(), LastPingTime: -100, Defcon: world.DefconHostile,
		SignatureID: "grisha", CrewSkill: 5,
		Track: world.AITrack{Valid: true, ClassConf: 0.2, HoldSec: 5, X: 0, Y: 1400, DepthFt: 180},
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 1400, DepthFt: 180,
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	updateSurfaceAI(ship, player, 50, 0.1, model, nil, EvadeContext{}, nil)
	if ship.AIState == "RBU" || ship.AIState == "SHIP_TUBE" {
		t.Fatalf("green unclassified crew should not weaponize, got %s", ship.AIState)
	}
}

func TestSurfaceStickyHoldsDatumOnLostContact(t *testing.T) {
	ship := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 16,
		Damage: world.NewFullHealth(), LastPingTime: 0, Defcon: world.DefconHostile,
		SignatureID: "udaloy", CrewSkill: 80,
		AIProsecuting: true,
		RouteID:       "r1",
		RouteWP:       1,
		Track: world.AITrack{
			Valid: true, ClassConf: 0.5, HoldSec: 0,
			X: 0, Y: 3500, DepthFt: 200,
		},
	}
	// Player far / silent so passive detect fails — sticky must not snap to PATROL.
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 40000, DepthFt: 600, SpeedKts: 0,
	}
	route := &world.Route{
		ID: "r1", PingPong: true,
		Waypoints: []world.Waypoint{{X: 8000, Y: 0}, {X: 12000, Y: 0}},
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	updateSurfaceAI(ship, player, 200, 0.1, model, nil, EvadeContext{}, []*world.Route{route})
	if ship.AIState == "PATROL" {
		t.Fatal("lost contact while prosecuting should hold DATUM, not PATROL")
	}
	if ship.AIState != "DATUM" {
		t.Fatalf("expected DATUM, got %s", ship.AIState)
	}
	if !ship.AIProsecuting {
		t.Fatal("should still be prosecuting")
	}
	if !ship.Track.Valid {
		t.Fatal("track should stay frozen as DATUM")
	}
	// Steer toward last-known, not the patrol waypoint.
	if ship.OrderedHead < 350 && ship.OrderedHead > 10 {
		// bearing to (0,3500) from (0,0) is ~0°
		t.Fatalf("ordered head %.0f should aim near north DATUM", ship.OrderedHead)
	}
}

func TestSurfaceStickyBreaksToPatrolAfterTimeout(t *testing.T) {
	ship := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, SpeedKts: 14,
		Damage: world.NewFullHealth(), LastPingTime: 0, Defcon: world.DefconHostile,
		SignatureID: "udaloy", CrewSkill: 50,
		AIProsecuting: true,
		RouteID:       "r1",
		Track:         world.AITrack{Valid: true, ClassConf: 0.4, X: 0, Y: 3000, DepthFt: 200},
	}
	// Force timeout on next tick.
	ship.AILostContactSec = surfaceDatumHoldSec(ship) - 0.05
	player := &world.Entity{
		ID: "player", Status: world.StatusActive, X: 0, Y: 40000, DepthFt: 600, SpeedKts: 0,
	}
	route := &world.Route{
		ID: "r1", PingPong: true,
		Waypoints: []world.Waypoint{{X: 5000, Y: 0}, {X: 9000, Y: 0}},
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	updateSurfaceAI(ship, player, 300, 0.1, model, nil, EvadeContext{}, []*world.Route{route})
	if ship.AIProsecuting {
		t.Fatal("should break prosecute after DATUM timeout")
	}
	if ship.AIState != "PATROL" {
		t.Fatalf("expected PATROL, got %s", ship.AIState)
	}
	if ship.AIEngageCooldownUntil <= 300 {
		t.Fatal("expected engage cooldown after break")
	}
	if ship.Track.Valid {
		t.Fatal("track should clear on break")
	}
}

func TestSurfaceEngageCooldownBlocksReentry(t *testing.T) {
	ship := &world.Entity{
		ID: "dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, SpeedKts: 14,
		Damage: world.NewFullHealth(), LastPingTime: -100, Defcon: world.DefconHostile,
		SignatureID: "udaloy", CrewSkill: 90,
		AIEngageCooldownUntil: 500,
		RouteID:               "r1",
		RouteWP:               0,
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, X: 0, Y: 5500, DepthFt: 200, SpeedKts: 8,
	}
	seedVeteranTrack(ship, player)
	route := &world.Route{
		ID: "r1", PingPong: true,
		Waypoints: []world.Waypoint{{X: 8000, Y: 0}, {X: 12000, Y: 0}},
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	updateSurfaceAI(ship, player, 400, 0.1, model, nil, EvadeContext{}, []*world.Route{route})
	if ship.AIProsecuting || ship.AIState == "CLOSING" || ship.AIState == "RASTRUB" {
		t.Fatalf("cooldown should block re-engage, got state=%s prosecuting=%v", ship.AIState, ship.AIProsecuting)
	}
	if ship.AIState != "PATROL" {
		t.Fatalf("expected PATROL during cooldown, got %s", ship.AIState)
	}
}

func TestSpruanceRadarProsecutesSurface(t *testing.T) {
	spruance := &world.Entity{
		ID: "ally_spruance", Kind: world.KindSurfaceShip, Side: world.SidePlayer,
		Status: world.StatusActive, SignatureID: "spruance",
		X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 14,
		Defcon: world.DefconWeaponsFree, CrewSkill: 90,
		Damage: world.NewFullHealth(), LastPingTime: 0,
	}
	grisha := &world.Entity{
		ID: "enemy_grisha", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, SignatureID: "grisha",
		X: 0, Y: 5500, SpeedKts: 12,
		Damage: world.NewFullHealth(),
	}
	model := acoustics.NewModel(acoustics.DefaultEnvironment())
	// Sweep enough ticks for rotating radar beam + class conf to rise.
	for step := 0; step < 120; step++ {
		gt := 10 + float64(step)*0.1
		updateSurfaceAI(spruance, grisha, gt, 0.1, model, nil, EvadeContext{}, nil)
		if spruance.AIState == "RASTRUB" || spruance.AIState == "RADAR_TRACK" {
			return
		}
	}
	t.Fatalf("expected RASTRUB/RADAR_TRACK after surface radar cue, got %s conf=%.2f",
		spruance.AIState, spruance.Track.ClassConf)
}

func TestCrewTrackFreezesDuringProsecute(t *testing.T) {
	hunter := &world.Entity{
		ID: "dd", Status: world.StatusActive, CrewSkill: 70, AIProsecuting: true,
		Track: world.AITrack{Valid: true, ClassConf: 0.55, X: 100, Y: 2000, DepthFt: 180},
	}
	player := &world.Entity{ID: "p", Status: world.StatusActive, X: 0, Y: 5000, DepthFt: 200}
	UpdateCrewTrack(hunter, player, false, false, 0, 10, 0.1)
	if !hunter.Track.Valid || hunter.Track.X != 100 || hunter.Track.Y != 2000 {
		t.Fatalf("DATUM should freeze: valid=%v pos=%.0f,%.0f", hunter.Track.Valid, hunter.Track.X, hunter.Track.Y)
	}
	if hunter.Track.ClassConf < 0.08 {
		t.Fatalf("ClassConf should not drop below floor while prosecuting: %.3f", hunter.Track.ClassConf)
	}
}
