package acoustics

import (
	"testing"

	"github.com/ssn688/sim/internal/campaign"
	"github.com/ssn688/sim/internal/world"
)

func benchScenario() (*world.Scenario, []float64) {
	sc := campaign.DemoRuntime()
	dst := make([]float64, NumBands)
	return sc, dst
}

func BenchmarkUpdatePassive(b *testing.B) {
	sc, _ := benchScenario()
	emitters := sc.AppendAllEntities(nil)
	model := NewModel(DefaultEnvironment())
	sonar := NewSonarState()
	player := sc.Player
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		UpdatePassive(model, player, emitters, &sonar, float64(i)*0.1)
	}
}

func BenchmarkBearingWaterfallInto(b *testing.B) {
	sc, _ := benchScenario()
	emitters := sc.AppendAllEntities(nil)
	model := NewModel(DefaultEnvironment())
	sonar := NewSonarState()
	player := sc.Player
	dst := make([]float64, BearingWaterfallBins)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		BearingWaterfallInto(dst, model, player, emitters, &sonar, PassiveArrayHull, float64(i)*0.15)
	}
}

func BenchmarkSpectrumAtBearingInto(b *testing.B) {
	sc, dst := benchScenario()
	emitters := sc.AppendAllEntities(nil)
	model := NewModel(DefaultEnvironment())
	sonar := NewSonarState()
	player := sc.Player
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		SpectrumAtBearingInto(dst, model, player, emitters, &sonar, float64(i%360), 0)
	}
}

func BenchmarkSpectrumAtBearing(b *testing.B) {
	sc, _ := benchScenario()
	emitters := sc.AppendAllEntities(nil)
	model := NewModel(DefaultEnvironment())
	sonar := NewSonarState()
	player := sc.Player
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = SpectrumAtBearing(model, player, emitters, &sonar, float64(i%360), 0)
	}
}
