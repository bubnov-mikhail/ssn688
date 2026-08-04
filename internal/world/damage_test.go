package world

import (
	"math"
	"math/rand"
	"testing"
)

func TestApplyTorpedoHitDamagesButOftenSurvives(t *testing.T) {
	kills := 0
	const n = 40
	for i := 0; i < n; i++ {
		e := &Entity{
			ID: "t", Name: "Test", Kind: KindSubmarine, Side: SidePlayer,
			Status: StatusActive, HeadingDeg: 90, Damage: NewFullHealth(),
		}
		rng := rand.New(rand.NewSource(int64(i + 1)))
		fatal, _ := ApplyTorpedoHit(e, rng)
		if fatal {
			kills++
			if e.Damage.EffOf(SysHull) > 0 {
				t.Fatalf("fatal with hull still %.0f", e.Damage.EffOf(SysHull))
			}
		}
		// At least one system should have dropped.
		any := false
		for s := 0; s < SysCount; s++ {
			if e.Damage.EffOf(s) < 100 {
				any = true
				break
			}
		}
		if !any {
			t.Fatal("expected some damage")
		}
	}
	if kills == n {
		t.Fatal("every hit was fatal — damage model too lethal")
	}
	if kills == 0 {
		// Unlikely but ok if hull shock never reaches 0 in this sample.
		t.Log("no fatal hits in sample (acceptable)")
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
		Kind: KindSubmarine, Status: StatusActive,
		OrderedSpeed: 25, SpeedKts: 20, Damage: NewFullHealth(),
	}
	e.Damage.Eff[SysPropulsion] = 40
	e.Advance(1)
	if e.OrderedSpeed > e.MaxSpeedKts()+0.01 {
		t.Fatalf("ordered %.1f > max %.1f", e.OrderedSpeed, e.MaxSpeedKts())
	}
	if math.Abs(e.MaxSpeedKts()-12) > 0.01 {
		t.Fatalf("max speed %.1f", e.MaxSpeedKts())
	}
}
