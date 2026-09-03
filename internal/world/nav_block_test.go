package world

import "testing"

func TestRevertMoveIfBlocked(t *testing.T) {
	b := testIslandBathy(t)
	sub := &Entity{ID: "s1", Kind: KindSubmarine, DepthFt: 160, OrderedDepth: 160, X: 500, Y: 500}
	prevX, prevY := sub.X, sub.Y
	sub.X = 1500 // island cell
	RevertMoveIfBlocked(sub, prevX, prevY, &b)
	if sub.X != prevX || sub.Y != prevY {
		t.Fatalf("expected revert, got %.0f,%.0f", sub.X, sub.Y)
	}
}
