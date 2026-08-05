package world

import (
	"math"
	"math/rand"
	"testing"
)

func TestFirstNewCriticalSystem(t *testing.T) {
	d := NewFullHealth()
	before := SnapshotCritical(&d)
	d.Eff[SysSteering] = RepairThresholdPct
	d.Eff[SysPropulsion] = RepairThresholdPct - 1
	sys := FirstNewCriticalSystem(before, &d)
	if sys != SysSteering && sys != SysPropulsion {
		t.Fatalf("expected new critical, got %d (%s)", sys, SystemName(sys))
	}
	before2 := SnapshotCritical(&d)
	if FirstNewCriticalSystem(before2, &d) != SysNone {
		t.Fatal("no new critical expected")
	}
}

func TestApplyTorpedoHitDamagesButOftenSurvives(t *testing.T) {
	const n = 40
	for i := 0; i < n; i++ {
		e := &Entity{
			ID: "t", Name: "Test", Kind: KindSubmarine, Side: SidePlayer,
			Status: StatusActive, SignatureID: "los_angeles", HeadingDeg: 90, Damage: NewFullHealth(),
		}
		rng := rand.New(rand.NewSource(int64(i + 1)))
		fatal, _ := ApplyTorpedoHit(e, rng, false)
		if e.Damage.EffOf(SysHull) >= 100 {
			t.Fatal("expected hull damage from heavy fish")
		}
		if fatal && e.Damage.EffOf(SysHull) > 0 {
			t.Fatalf("fatal with hull still %.0f", e.Damage.EffOf(SysHull))
		}
	}
	// Second hit should finish a 688 that survived the first.
	secondKills := 0
	for i := 0; i < n; i++ {
		e := &Entity{
			ID: "t", Name: "Test", Kind: KindSubmarine, Side: SidePlayer,
			Status: StatusActive, SignatureID: "los_angeles", Damage: NewFullHealth(),
		}
		rng := rand.New(rand.NewSource(int64(i + 101)))
		ApplyTorpedoHit(e, rng, false)
		if e.Damage.EffOf(SysHull) <= 0 {
			secondKills++ // already dead on first
			continue
		}
		fatal, _ := ApplyTorpedoHit(e, rng, false)
		if fatal {
			secondKills++
		}
	}
	if secondKills < n*3/4 {
		t.Fatalf("two heavy hits should usually kill a 688; kills=%d/%d", secondKills, n)
	}
}

func TestApplyTorpedoHitSinksGrishaInOne(t *testing.T) {
	survivors := 0
	const n = 30
	for i := 0; i < n; i++ {
		e := &Entity{
			ID: "g", Name: "Grisha", Kind: KindSurfaceShip, Side: SideEnemy,
			Status: StatusActive, SignatureID: "grisha", Damage: NewFullHealth(),
		}
		rng := rand.New(rand.NewSource(int64(i + 11)))
		fatal, _ := ApplyTorpedoHit(e, rng, false)
		if !fatal {
			survivors++
		}
	}
	if survivors > 2 {
		t.Fatalf("Mk48 should nearly always kill Grisha in one hit; survivors=%d/%d", survivors, n)
	}
}

func TestApplyLightTorpedoHitRarelyOneShots688(t *testing.T) {
	kills := 0
	const n = 40
	for i := 0; i < n; i++ {
		e := &Entity{
			ID: "p", Name: "688", Kind: KindSubmarine, Side: SidePlayer,
			Status: StatusActive, SignatureID: "los_angeles", Damage: NewFullHealth(),
		}
		rng := rand.New(rand.NewSource(int64(i + 3)))
		fatal, _ := ApplyTorpedoHit(e, rng, true)
		if fatal {
			kills++
		}
	}
	if kills > n/4 {
		t.Fatalf("light fish too lethal vs 688: %d/%d fatal", kills, n)
	}
}

func TestRepairThresholdAndRate(t *testing.T) {
	d := NewFullHealth()
	d.Eff[SysSteering] = 40
	ok, _ := d.StartRepair(SysSteering)
	if !ok {
		t.Fatal("should repair at 40%")
	}
	// 45 min from 25→100 ⇒ rate = 75/2700 %/s. From 40→100 needs 60 pts.
	need := (100.0 - 40.0) / ((100.0 - RepairThresholdPct) / RepairMinToFullSec)
	d.AdvanceRepair(need + 1)
	if d.Eff[SysSteering] < 99.9 {
		t.Fatalf("repair incomplete: %.2f after %.0fs", d.Eff[SysSteering], need)
	}
	if d.Repairing != SysNone {
		t.Fatal("repair should finish")
	}

	d.Eff[SysActive] = 20
	ok, _ = d.StartRepair(SysActive)
	if ok {
		t.Fatal("should not repair at 20%")
	}
}

func TestPropulsionCapsSpeed(t *testing.T) {
	e := &Entity{
		Kind: KindSubmarine, Status: StatusActive, SignatureID: "los_angeles",
		OrderedSpeed: 25, SpeedKts: 20, Damage: NewFullHealth(),
	}
	e.Damage.Eff[SysPropulsion] = 40
	e.Advance(1)
	if e.OrderedSpeed > e.MaxSpeedKts()+0.01 {
		t.Fatalf("ordered %.1f > max %.1f", e.OrderedSpeed, e.MaxSpeedKts())
	}
	want := 32.0 * 0.40
	if math.Abs(e.MaxSpeedKts()-want) > 0.01 {
		t.Fatalf("max speed %.1f want %.1f", e.MaxSpeedKts(), want)
	}
}
