package sim_test

import (
	"testing"

	"github.com/ssn688/sim/internal/sim"
	"github.com/ssn688/sim/internal/world"
)

func BenchmarkEngineTick(b *testing.B) {
	eng := sim.NewEngine(world.NewTrainingScenario())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.Update(0.1) // one sim second at 1x would be 10 ticks; force catch-up
	}
}

func BenchmarkEngineTenTicks(b *testing.B) {
	eng := sim.NewEngine(world.NewTrainingScenario())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			eng.Update(0.016)
		}
	}
}
