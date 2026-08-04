package acoustics

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestTryAutoClassifyTorpedo(t *testing.T) {
	c := &Contact{ID: "C01", Confidence: 0.4}
	class := Classification{ProfileID: "type53", ProfileName: "Type 53 Torpedo", Confidence: 0.62, BladeMatch: 0.55}
	if !TryAutoClassifyTorpedo(c, class) {
		t.Fatal("expected new TORP classify")
	}
	if c.ConfirmedClass != "TORP" || c.Kind != world.KindTorpedo {
		t.Fatalf("got class=%q kind=%v", c.ConfirmedClass, c.Kind)
	}
	if TryAutoClassifyTorpedo(c, class) {
		t.Fatal("second call should not re-fire")
	}
}

func TestTryAutoClassifyTorpedoRejectsWeak(t *testing.T) {
	c := &Contact{}
	class := Classification{ProfileID: "type53", Confidence: 0.40, BladeMatch: 0.20}
	if TryAutoClassifyTorpedo(c, class) {
		t.Fatal("weak match must not auto-classify")
	}
	ship := Classification{ProfileID: "kilo", Confidence: 0.9, BladeMatch: 0.8}
	if TryAutoClassifyTorpedo(c, ship) {
		t.Fatal("ship must not auto-classify as TORP")
	}
}

func TestKindFromMatchUnknownUntilConfident(t *testing.T) {
	if KindFromMatch(Classification{ProfileID: "type53", Confidence: 0.3}) >= 0 {
		t.Fatal("low confidence should be unknown kind")
	}
	if KindFromMatch(Classification{ProfileID: "type53", Confidence: 0.6}) != world.KindTorpedo {
		t.Fatal("expected torpedo kind")
	}
}
