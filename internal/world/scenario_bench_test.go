package world

import "testing"

func benchScenario() *Scenario {
	player := &Entity{ID: "player", Status: StatusActive}
	ents := []*Entity{
		{ID: "a", Status: StatusActive},
		{ID: "b", Status: StatusActive},
		{ID: "c", Status: StatusActive},
	}
	return &Scenario{Player: player, Entities: ents}
}

func BenchmarkAppendAllEntities(b *testing.B) {
	sc := benchScenario()
	dst := make([]*Entity, 0, 8)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dst = sc.AppendAllEntities(dst[:0])
	}
}

func BenchmarkAllEntities(b *testing.B) {
	sc := benchScenario()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = sc.AllEntities()
	}
}
