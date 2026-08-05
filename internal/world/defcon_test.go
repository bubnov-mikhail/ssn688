package world

import "testing"

func TestRaiseDefconOnlyIncreases(t *testing.T) {
	e := &Entity{Defcon: DefconHostile}
	e.RaiseDefcon(DefconAware)
	if e.Defcon != DefconHostile {
		t.Fatalf("got %d", e.Defcon)
	}
	e.RaiseDefcon(DefconWeaponsFree)
	if e.Defcon != DefconWeaponsFree {
		t.Fatalf("got %d", e.Defcon)
	}
}

func TestRestrictedZonePlayerInside(t *testing.T) {
	z := RestrictedZone{CenterX: 0, CenterY: 0, RadiusYd: 1000}
	p := &Entity{X: 500, Y: 0}
	if !z.PlayerInside(p) {
		t.Fatal("expected inside")
	}
	p.X = 2000
	if z.PlayerInside(p) {
		t.Fatal("expected outside")
	}
}
