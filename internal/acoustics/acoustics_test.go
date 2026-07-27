package acoustics

import (
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func testEntity(id, sig string, kind world.EntityKind, depth, speed float64) *world.Entity {
	return &world.Entity{
		ID: id, SignatureID: sig, Kind: kind, Status: world.StatusActive,
		DepthFt: depth, SpeedKts: speed, X: 0, Y: 0,
	}
}

func TestThermoclineHidesSub(t *testing.T) {
	model := NewModel(DefaultEnvironment())

	surface := testEntity("dd", "spruance", world.KindSurfaceShip, 0, 12)
	subShallow := testEntity("sub1", "kilo", world.KindSubmarine, 150, 8)
	subDeep := testEntity("sub2", "kilo", world.KindSubmarine, 900, 8)
	subShallow.Y = 5000
	subDeep.Y = 5000

	shallowSNR := model.Detect(surface, subShallow, ModePassive, 0).PeakSNR
	deepSNR := model.Detect(surface, subDeep, ModePassive, 0).PeakSNR

	if deepSNR >= shallowSNR {
		t.Fatalf("sub under thermocline should be harder to detect: deep=%.1f shallow=%.1f", deepSNR, shallowSNR)
	}
}

func TestCavitationIncreasesSelfNoise(t *testing.T) {
	env := DefaultEnvironment()
	quiet := SelfNoiseSpectrum(testEntity("a", "los_angeles", world.KindSubmarine, 400, 6), env, PassiveArrayHull, 0)
	loud := SelfNoiseSpectrum(testEntity("b", "los_angeles", world.KindSubmarine, 80, 22), env, PassiveArrayHull, 0)
	if loud.Peak() <= quiet.Peak() {
		t.Fatalf("cavitating platform should be louder: quiet=%.1f loud=%.1f", quiet.Peak(), loud.Peak())
	}
}

func TestPassiveDetectionUnified(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := testEntity("sub", "los_angeles", world.KindSubmarine, 320, 5)
	emitter := testEntity("dd", "spruance", world.KindSurfaceShip, 0, 14)
	emitter.Y = 6000

	r := model.Detect(listener, emitter, ModePassive, 0)
	if !r.Detected {
		t.Fatalf("expected surface ship detected at 6 kyd, peak SNR=%.1f bands=%d", r.PeakSNR, r.BandsAbove)
	}
}

func TestClassificationTemplateSelfMatch(t *testing.T) {
	for _, id := range []string{"kilo", "spruance", "perry", "los_angeles"} {
		p := mustProfile(id)
		c := Classify(TemplateSpectrumForTest(p), 20, 3000)
		if c.ProfileID != id {
			t.Fatalf("template %s matched %s (%.2f)", id, c.ProfileID, c.Confidence)
		}
	}
}

func mustProfile(id string) world.SignatureProfile {
	p, ok := world.ProfileByID(id)
	if !ok {
		panic("profile " + id)
	}
	return p
}

func TestAIUsesSameDetector(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	enemy := testEntity("dd", "spruance", world.KindSurfaceShip, 0, 14)
	player := testEntity("player", "los_angeles", world.KindSubmarine, 180, 20)
	player.Y = 2000

	if !model.CanDetectPassive(enemy, player) {
		r := model.Detect(enemy, player, ModePassive, 0)
		t.Fatalf("enemy should detect fast player at 1800 yd: peak=%.1f bands=%d", r.PeakSNR, r.BandsAbove)
	}
}
