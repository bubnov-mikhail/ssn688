package acoustics

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestHarmonicMatchFractionTemplateVsNoise(t *testing.T) {
	p, ok := world.ProfileByID("tanker")
	if !ok {
		t.Fatal("missing tanker profile")
	}
	tmpl := TemplateSpectrumForTest(p)
	if frac := HarmonicMatchFraction(tmpl, p); frac < HarmonicIdentifyMatchFrac {
		t.Fatalf("template match %.2f want ≥ %.2f", frac, HarmonicIdentifyMatchFrac)
	}
	var noise Spectrum
	if frac := HarmonicMatchFraction(noise, p); frac >= HarmonicIdentifyMatchFrac {
		t.Fatalf("empty spectrum should not match, got %.2f", frac)
	}
}

func TestAcousticIdentifyRequiresTwoMinutes(t *testing.T) {
	em := &world.Entity{ID: "civ_tanker", SignatureID: "tanker", Kind: world.KindSurfaceShip, Status: world.StatusActive}
	p, _ := world.ProfileByID("tanker")
	sig := TemplateSpectrumForTest(p)
	c := &Contact{ID: "C01", SourceEntityID: em.ID}

	gt := 0.0
	for gt < HarmonicIdentifyHoldSec+2 {
		gt += 0.1
		tryAcousticIdentify(c, sig, em, 0.1, gt)
		if gt < HarmonicIdentifyHoldSec-0.2 && c.Identified {
			t.Fatalf("identified too early at t=%.1f hold=%.1f", gt, c.HarmonicHoldSec)
		}
	}
	if !c.Identified || c.IdentifiedBy != IdentifiedByAcoustic {
		t.Fatalf("expected acoustic ID, got identified=%v by=%q hold=%.1f", c.Identified, c.IdentifiedBy, c.HarmonicHoldSec)
	}
	if c.ConfirmedID != "tanker" {
		t.Fatalf("ConfirmedID=%q", c.ConfirmedID)
	}
}

func TestAcousticIdentifyResetsOnLostFingerprint(t *testing.T) {
	em := &world.Entity{ID: "enemy_grisha", SignatureID: "grisha", Kind: world.KindSurfaceShip, Status: world.StatusActive}
	p, _ := world.ProfileByID("grisha")
	sig := TemplateSpectrumForTest(p)
	c := &Contact{ID: "C02", SourceEntityID: em.ID}
	for i := 0; i < 800; i++ {
		tryAcousticIdentify(c, sig, em, 0.1, float64(i)*0.1)
	}
	if c.HarmonicHoldSec < 70 {
		t.Fatalf("hold %.1f", c.HarmonicHoldSec)
	}
	var noise Spectrum
	tryAcousticIdentify(c, noise, em, 1, 81)
	if c.HarmonicHoldSec != 0 {
		t.Fatalf("hold should reset on lost fingerprint, got %.1f", c.HarmonicHoldSec)
	}
	if c.Identified {
		t.Fatal("must not identify after reset")
	}
}

func TestVisualIdentifyInside3000(t *testing.T) {
	em := &world.Entity{ID: "enemy_grisha", Name: "Grisha", SignatureID: "grisha", Kind: world.KindSurfaceShip, Status: world.StatusActive}
	c := &Contact{ID: "C03", SourceEntityID: em.ID}
	tryVisualIdentify(c, em, 3200, 10)
	if c.Identified {
		t.Fatal("3200 yd must not visual-ID")
	}
	tryVisualIdentify(c, em, 3000, 11)
	if c.Identified {
		t.Fatal("3000 yd is not inside <3000")
	}
	tryVisualIdentify(c, em, 2999, 12)
	if !c.Identified || c.IdentifiedBy != IdentifiedByVisual {
		t.Fatalf("2999 yd should visual-ID, got %+v", c)
	}
}

func TestIdentifyContactIdempotent(t *testing.T) {
	em := &world.Entity{ID: "x", SignatureID: "tanker", Kind: world.KindSurfaceShip}
	c := &Contact{ID: "C04"}
	if !IdentifyContact(c, em, IdentifiedByVisual, 5) {
		t.Fatal("first ID should return true")
	}
	if IdentifyContact(c, em, IdentifiedByAcoustic, 6) {
		t.Fatal("second ID should return false")
	}
	if c.IdentifiedBy != IdentifiedByVisual {
		t.Fatalf("by=%q", c.IdentifiedBy)
	}
}
