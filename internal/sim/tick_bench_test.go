package sim_test

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/sim"
)

func BenchmarkEngineTick(b *testing.B) {
	eng := sim.NewEngine(campaign.DemoRuntime())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.Update(0.1) // one sim second at 1x would be 10 ticks; force catch-up
	}
}

func BenchmarkEngineTenTicks(b *testing.B) {
	eng := sim.NewEngine(campaign.DemoRuntime())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			eng.Update(0.016)
		}
	}
}
