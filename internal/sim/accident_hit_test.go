package sim

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestDebugAccidentHitSkipsDefcon(t *testing.T) {
	sc := campaign.DemoRuntime()
	eng := NewEngine(sc)
	var target *world.Entity
	for _, e := range sc.Entities {
		if e != nil && e.ID == "civ_merchant" {
			target = e
			break
		}
	}
	if target == nil {
		t.Fatal("missing civ_merchant")
	}
	before := map[string]int{}
	for _, e := range sc.Entities {
		if e != nil && e.Side == world.SideEnemy {
			before[e.ID] = e.Defcon
		}
	}
	targetBefore := target.Defcon

	if !eng.DebugAccidentHit(target.ID) {
		t.Fatal("DebugAccidentHit returned false")
	}
	if target.Defcon != targetBefore {
		t.Fatalf("civilian DEFCON %d → %d (want unchanged)", targetBefore, target.Defcon)
	}
	for _, e := range sc.Entities {
		if e == nil || e.Side != world.SideEnemy {
			continue
		}
		if e.Defcon != before[e.ID] {
			t.Fatalf("%s DEFCON %d → %d (want unchanged on accident)", e.ID, before[e.ID], e.Defcon)
		}
	}
	hit := target.HullFireUntil > eng.Clock.GameTime || target.Status == world.StatusSinking || !target.Alive()
	if !hit {
		t.Fatal("expected damage/fire/sinking after accident hit")
	}
}

func TestAccidentDetonationFlagSkipsNotify(t *testing.T) {
	sc := campaign.DemoRuntime()
	eng := NewEngine(sc)
	var grisha *world.Entity
	for _, e := range sc.Entities {
		if e != nil && e.ID == "enemy_grisha" {
			grisha = e
			break
		}
	}
	if grisha == nil {
		t.Fatal("missing grisha")
	}
	prev := grisha.Defcon
	eng.handleDetonation(&weapons.Detonation{
		X: grisha.X, Y: grisha.Y, Hit: grisha, Harpoon: true, Accident: true,
	}, eng.Clock.GameTime)
	if grisha.Defcon != prev {
		t.Fatalf("Grisha DEFCON raised on Accident detonation: %d → %d", prev, grisha.Defcon)
	}
}

func TestCivilianSinkingSkipsCookOff(t *testing.T) {
	sc := campaign.DemoRuntime()
	eng := NewEngine(sc)
	var civ *world.Entity
	for _, e := range sc.Entities {
		if e != nil && e.ID == "civ_merchant" {
			civ = e
			break
		}
	}
	if civ == nil {
		t.Fatal("missing civ_merchant")
	}
	eng.beginSinking(civ, eng.Clock.GameTime)
	if civ.CookOffLeft != 0 || civ.NextCookOffAt != 0 {
		t.Fatalf("civilian cook-off scheduled: left=%d next=%.1f", civ.CookOffLeft, civ.NextCookOffAt)
	}
	if civ.WreckNoiseUntil <= eng.Clock.GameTime {
		t.Fatal("civilian wreck should still radiate flooding noise")
	}
}

func TestWarshipSinkingSchedulesCookOff(t *testing.T) {
	sc := campaign.DemoRuntime()
	eng := NewEngine(sc)
	var war *world.Entity
	for _, e := range sc.Entities {
		if e != nil && e.ID == "enemy_grisha" {
			war = e
			break
		}
	}
	if war == nil {
		t.Fatal("missing enemy_grisha")
	}
	eng.beginSinking(war, eng.Clock.GameTime)
	if war.CookOffLeft <= 0 || war.NextCookOffAt <= eng.Clock.GameTime {
		t.Fatalf("warship should cook off: left=%d next=%.1f", war.CookOffLeft, war.NextCookOffAt)
	}
}
