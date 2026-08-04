package weapons

import "testing"

func TestTubeAmmoStatus(t *testing.T) {
	loaded := Tube{State: TubeLoaded, TorpedoType: "Mk48"}
	if got := TubeAmmoStatus(loaded, 0); got != "Mk48" {
		t.Fatalf("loaded: got %q", got)
	}
	open := Tube{State: TubeDoorOpen, TorpedoType: "Mk48"}
	if got := TubeAmmoStatus(open, 0); got != "Mk48" {
		t.Fatalf("door open: got %q", got)
	}
	wired := Tube{State: TubeFired, TorpedoType: "Mk48", WireIntact: true}
	if got := TubeAmmoStatus(wired, 0); got != "Mk48, wired" {
		t.Fatalf("wired: got %q", got)
	}
	cut := Tube{State: TubeFired, TorpedoType: "Mk48", WireIntact: false}
	if got := TubeAmmoStatus(cut, 0); got != "Mk48" {
		t.Fatalf("wire cut: got %q", got)
	}
	empty := Tube{State: TubeEmpty}
	if got := TubeAmmoStatus(empty, 0); got != "EMPTY" {
		t.Fatalf("empty: got %q", got)
	}
	reloading := Tube{State: TubeReloading, TorpedoType: "Mk48"}
	if got := TubeAmmoStatus(reloading, 42.2); got != "RELOADING 42s" {
		t.Fatalf("reloading: got %q", got)
	}
}
