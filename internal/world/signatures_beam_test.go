package world

import "testing"

func TestSurfaceHullBeamRel(t *testing.T) {
	if SurfaceHullBeamRel(nil) != 0 {
		t.Fatal("nil entity")
	}
	sub := &Entity{Kind: KindSubmarine, SignatureID: "los_angeles"}
	if SurfaceHullBeamRel(sub) != 0 {
		t.Fatal("sub should have no bow-wash beam")
	}
	tanker := &Entity{Kind: KindSurfaceShip, SignatureID: "tanker"}
	fish := &Entity{Kind: KindSurfaceShip, SignatureID: "fishing"}
	if SurfaceHullBeamRel(tanker) <= SurfaceHullBeamRel(fish) {
		t.Fatalf("tanker beam %.2f should exceed fishing %.2f",
			SurfaceHullBeamRel(tanker), SurfaceHullBeamRel(fish))
	}
	if g := SurfaceHullBeamRel(&Entity{Kind: KindSurfaceShip, SignatureID: "grisha"}); g >= SurfaceHullBeamRel(tanker) {
		t.Fatalf("grisha %.2f should be narrower than tanker", g)
	}
}
