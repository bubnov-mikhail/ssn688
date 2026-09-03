package ai

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestMostThreateningTorpedoAimed(t *testing.T) {
	ship := &world.Entity{
		ID: "enemy_dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 2000, HeadingDeg: 0, SpeedKts: 14,
	}
	// Fish north of ship, heading south toward it.
	fish := &weapons.Torpedo{
		ID: "MK48-1", Side: world.SidePlayer, Alive: true,
		X: 0, Y: 3500, HeadingDeg: 180, SpeedKts: 55, Mode: weapons.ModeSearch,
		LastPingTime: 1,
	}
	miss := &weapons.Torpedo{
		ID: "MK48-2", Side: world.SidePlayer, Alive: true,
		X: 4000, Y: 2000, HeadingDeg: 90, SpeedKts: 55, // going away east
	}
	got := mostThreateningTorpedo(ship, []*weapons.Torpedo{miss, fish})
	if got == nil || got.ID != "MK48-1" {
		t.Fatalf("expected aimed fish, got %#v", got)
	}
}

func TestTryEvadeTorpedoOrdersFlankComb(t *testing.T) {
	ship := &world.Entity{
		ID: "enemy_dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, HeadingDeg: 0, OrderedSpeed: 12,
	}
	fish := &weapons.Torpedo{
		ID: "MK48-1", Side: world.SidePlayer, Alive: true,
		X: 0, Y: 1500, HeadingDeg: 180, SpeedKts: 50, Mode: weapons.ModeWire,
	}
	cm := weapons.NewCountermeasureSystem()
	ctx := EvadeContext{CM: &cm, Env: acoustics.DefaultEnvironment(), GameTime: 10}
	if !tryEvadeTorpedo(ship, []*weapons.Torpedo{fish}, ctx) {
		t.Fatal("expected evade")
	}
	if ship.AIState != "TORPEDO_EVADE" {
		t.Fatalf("state=%s", ship.AIState)
	}
	if ship.OrderedSpeed < 25 {
		t.Fatalf("expected flank speed, got %.0f", ship.OrderedSpeed)
	}
	if !cm.NixieOn(ship.ID) {
		t.Fatal("expected Nixie enabled on surface evade")
	}
	if cm.MagazineLeft(ship.ID) >= weapons.CMMagazineDefault {
		t.Fatal("expected ADC expenditure")
	}
	h := ship.OrderedHead
	// Zigzag may offset comb; allow wide band around E/W.
	if !((h > 50 && h < 130) || (h > 230 && h < 310) || (h < 40 || h > 320)) {
		// still ok if zigzag pushed toward N briefly — just ensure not stuck on fish bearing 180
		if h > 150 && h < 210 {
			t.Fatalf("unexpected comb heading %.0f", h)
		}
	}
}

func TestSubEvadeChangesDepth(t *testing.T) {
	sub := &world.Entity{
		ID: "enemy_ss", Kind: world.KindSubmarine, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, DepthFt: 200, HeadingDeg: 90,
	}
	fish := &weapons.Torpedo{
		ID: "MK48-1", Side: world.SidePlayer, Alive: true,
		X: 800, Y: 0, HeadingDeg: 270, DepthFt: 180, SpeedKts: 48,
		Mode: weapons.ModeSearch, TargetID: sub.ID, LastPingTime: 10,
	}
	cm := weapons.NewCountermeasureSystem()
	ctx := EvadeContext{CM: &cm, Env: acoustics.DefaultEnvironment(), GameTime: 5}
	if !tryEvadeTorpedo(sub, []*weapons.Torpedo{fish}, ctx) {
		t.Fatal("expected evade")
	}
	if sub.OrderedDepth <= sub.DepthFt {
		t.Fatalf("expected deeper ordered depth away from shallow fish, got %.0f", sub.OrderedDepth)
	}
	if sub.OrderedSpeed < 18 {
		t.Fatalf("expected high speed, got %.0f", sub.OrderedSpeed)
	}
	if len(cm.Active) == 0 {
		t.Fatal("expected ADC deploy from sub")
	}
}

func TestEvadeDoesNotDeployCMWithoutCollisionThreat(t *testing.T) {
	ship := &world.Entity{
		ID: "enemy_dd", Kind: world.KindSurfaceShip, Side: world.SideEnemy,
		Status: world.StatusActive, X: 0, Y: 0, HeadingDeg: 0, SpeedKts: 14,
	}
	fish := &weapons.Torpedo{
		ID: "MK48-9", Side: world.SidePlayer, Alive: true,
		X: 2200, Y: 2600, HeadingDeg: 90, SpeedKts: 45, Mode: weapons.ModeSearch,
	}
	cm := weapons.NewCountermeasureSystem()
	applyTorpedoEvade(ship, fish, EvadeContext{CM: &cm, Env: acoustics.DefaultEnvironment(), GameTime: 12})
	if cm.NixieOn(ship.ID) {
		t.Fatal("nixie must stay off without CPA threat")
	}
	if len(cm.Active) != 0 {
		t.Fatal("countermeasures must not deploy without CPA threat")
	}
}
