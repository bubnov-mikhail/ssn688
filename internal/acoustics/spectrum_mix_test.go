package acoustics

import (
	"math"
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func bandNear(freqHz float64) int {
	best, bestD := 0, 1e9
	for i := 0; i < NumBands; i++ {
		d := math.Abs(BandCenterHz(i) - freqHz)
		if d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

func TestSpectrumBeamWeight(t *testing.T) {
	if SpectrumBeamWeight(0, 6) < 0.99 {
		t.Fatal("on-beam weight should be ~1")
	}
	if SpectrumBeamWeight(6, 6) > 0.7 || SpectrumBeamWeight(6, 6) < 0.4 {
		t.Fatalf("1σ weight unexpected: %.3f", SpectrumBeamWeight(6, 6))
	}
	if SpectrumBeamWeight(25, 6) > 0.01 {
		t.Fatal("far off-beam should be ~0")
	}
}

func TestSpectrumMixesCloseBearings(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := &world.Entity{
		ID: "player", SignatureID: "los_angeles", Kind: world.KindSubmarine, Status: world.StatusActive,
		DepthFt: 240, SpeedKts: 5, HeadingDeg: 0,
	}
	// Quiet diesel on look bearing.
	kilo := &world.Entity{
		ID: "kilo", SignatureID: "kilo", Kind: world.KindSubmarine, Status: world.StatusActive,
		Y: 4500, DepthFt: 200, SpeedKts: 6,
	}
	// Loud DD a few degrees off — within soft beam.
	ang := 4.0 * math.Pi / 180
	rng := 4500.0
	dd := &world.Entity{
		ID: "dd", SignatureID: "udaloy", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: rng * math.Sin(ang), Y: rng * math.Cos(ang), SpeedKts: 14,
	}
	sonar := NewSonarState()
	sonar.PassiveArray = PassiveArrayHull

	solo := SpectrumAtBearing(model, listener, []*world.Entity{listener, kilo}, &sonar, 0, 0)
	mixed := SpectrumAtBearing(model, listener, []*world.Entity{listener, kilo, dd}, &sonar, 0, 0)

	// Udaloy tonal cluster around 160 Hz should lift the mixed trace vs kilo alone.
	ddBand := bandNear(160)
	if mixed[ddBand] <= solo[ddBand]+1.5 {
		t.Fatalf("close DD harmonics should raise mixed spectrum at ~160Hz: solo=%.1f mixed=%.1f",
			solo[ddBand], mixed[ddBand])
	}
	// Kilo line should still contribute (not fully replaced by max-of-DD).
	kiloBand := bandNear(62)
	far := SpectrumAtBearing(model, listener, []*world.Entity{listener, kilo, dd}, &sonar, 40, 0)
	if mixed[kiloBand] < far[kiloBand] {
		t.Fatalf("on-beam mix should keep kilo energy vs far look: on=%.1f far=%.1f",
			mixed[kiloBand], far[kiloBand])
	}
}

func TestContaminateClassifySignalLowersMatch(t *testing.T) {
	model := NewModel(DefaultEnvironment())
	listener := &world.Entity{
		ID: "player", SignatureID: "los_angeles", Kind: world.KindSubmarine, Status: world.StatusActive,
		DepthFt: 240, SpeedKts: 5, HeadingDeg: 0,
	}
	kilo := &world.Entity{
		ID: "kilo", SignatureID: "kilo", Kind: world.KindSubmarine, Status: world.StatusActive,
		Y: 3500, DepthFt: 180, SpeedKts: 7,
	}
	ang := 3.5 * math.Pi / 180
	dd := &world.Entity{
		ID: "dd", SignatureID: "udaloy", Kind: world.KindSurfaceShip, Status: world.StatusActive,
		X: 3500 * math.Sin(ang), Y: 3500 * math.Cos(ang), SpeedKts: 16,
	}
	sonar := NewSonarState()
	clean := model.Detect(listener, kilo, ModePassive, 0).SignalForClassify
	dirty := ContaminateClassifySignal(clean, model, listener, kilo.ID, []*world.Entity{listener, kilo, dd}, 0, &sonar)

	cleanClass := Classify(clean, 22, 3500)
	dirtyClass := Classify(dirty, 22, 3500)
	// Contamination should not improve confidence toward the true quiet target.
	if dirtyClass.Confidence > cleanClass.Confidence+0.05 {
		t.Fatalf("mixed classify should not raise confidence: clean=%.2f dirty=%.2f",
			cleanClass.Confidence, dirtyClass.Confidence)
	}
	kiloProf, _ := world.ProfileByID("kilo")
	tmpl := templateSpectrum(kiloProf).NormalizeShape()
	cleanCorr := correlateShape(clean.NormalizeShape(), tmpl)
	dirtyCorr := correlateShape(dirty.NormalizeShape(), tmpl)
	if dirtyCorr >= cleanCorr-0.02 {
		t.Fatalf("neighbor mix should reduce shape match to kilo: clean=%.3f dirty=%.3f",
			cleanCorr, dirtyCorr)
	}
}
