package world

import "testing"

func BenchmarkAppendAllEntities(b *testing.B) {
	sc := NewTrainingScenario()
	dst := make([]*Entity, 0, 8)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dst = sc.AppendAllEntities(dst[:0])
	}
}

func BenchmarkAllEntities(b *testing.B) {
	sc := NewTrainingScenario()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = sc.AllEntities()
	}
}
