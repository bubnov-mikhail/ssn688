package world

import (
	"math"
	"math/rand"
	"time"
)

// Scenario holds mission state.
type Scenario struct {
	Name        string
	Description string
	Player      *Entity
	Entities    []*Entity
	Objectives  []Objective
	FailReason  string
}

type Objective struct {
	ID          string
	Description string
	Complete    bool
	TargetID    string
}

func NewTrainingScenario() *Scenario {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	player := &Entity{
		ID: "player", Name: "USS Los Angeles", Kind: KindSubmarine, Side: SidePlayer,
		Status: StatusActive, SignatureID: "los_angeles",
		X: 0, Y: 0, DepthFt: 320, HeadingDeg: 90, SpeedKts: 8,
		OrderedSpeed: 8, OrderedDepth: 320, OrderedHead: 90,
		LengthFt: 360,
	}

	enemySurface := &Entity{
		ID: "enemy_surface", Name: "Hostile DD", Kind: KindSurfaceShip, Side: SideEnemy,
		Status: StatusActive, SignatureID: "spruance",
		DepthFt: 0, SpeedKts: 14, OrderedSpeed: 14, OrderedDepth: 0,
		LengthFt: 563, AIState: "SEARCH",
	}
	placeAwayFrom(rng, enemySurface, 3500, 6500, []*Entity{player})

	enemySub := &Entity{
		ID: "enemy_sub", Name: "Hostile SS", Kind: KindSubmarine, Side: SideEnemy,
		Status: StatusActive, SignatureID: "kilo",
		DepthFt: 220 + rng.Float64()*120, SpeedKts: 6, OrderedSpeed: 6,
		LengthFt: 240, AIState: "PATROL",
	}
	enemySub.OrderedDepth = enemySub.DepthFt
	placeAwayFrom(rng, enemySub, 3000, 7000, []*Entity{player, enemySurface})

	civilians := []*Entity{
		{
			ID: "civ_merchant", Name: "MV Pacific Star", Kind: KindSurfaceShip, Side: SideNeutral,
			Status: StatusActive, SignatureID: "merchant",
			DepthFt: 0, SpeedKts: 11, OrderedSpeed: 11, LengthFt: 520, AIState: "TRANSIT",
		},
		{
			ID: "civ_tanker", Name: "MT Horizon", Kind: KindSurfaceShip, Side: SideNeutral,
			Status: StatusActive, SignatureID: "tanker",
			DepthFt: 0, SpeedKts: 9, OrderedSpeed: 9, LengthFt: 900, AIState: "TRANSIT",
		},
		{
			ID: "civ_trawler", Name: "FV Northern Light", Kind: KindSurfaceShip, Side: SideNeutral,
			Status: StatusActive, SignatureID: "fishing",
			DepthFt: 0, SpeedKts: 7, OrderedSpeed: 7, LengthFt: 140, AIState: "TRANSIT",
		},
	}
	placed := []*Entity{player, enemySurface, enemySub}
	for _, c := range civilians {
		placeAwayFrom(rng, c, 2500, 9000, placed)
		placed = append(placed, c)
	}

	ents := []*Entity{enemySurface, enemySub}
	ents = append(ents, civilians...)

	return &Scenario{
		Name:        "Training Engagement",
		Description: "Locate and destroy hostile units. Do not attack civilian shipping.",
		Player:      player,
		Entities:    ents,
		Objectives: []Objective{
			{ID: "obj_surface", Description: "Destroy hostile surface combatant", TargetID: "enemy_surface"},
			{ID: "obj_sub", Description: "Destroy hostile submarine", TargetID: "enemy_sub"},
		},
	}
}

func placeAwayFrom(rng *rand.Rand, e *Entity, minR, maxR float64, others []*Entity) {
	for attempt := 0; attempt < 40; attempt++ {
		ang := rng.Float64() * 2 * math.Pi
		r := minR + rng.Float64()*(maxR-minR)
		e.X = math.Sin(ang) * r
		e.Y = math.Cos(ang) * r
		e.HeadingDeg = rng.Float64() * 360
		e.OrderedHead = e.HeadingDeg
		ok := true
		for _, o := range others {
			if o.RangeYardsTo(e) < 1800 {
				ok = false
				break
			}
		}
		if ok {
			return
		}
	}
}

func (s *Scenario) AllEntities() []*Entity {
	out := make([]*Entity, 0, len(s.Entities)+1)
	out = append(out, s.Player)
	out = append(out, s.Entities...)
	return out
}

func (s *Scenario) CheckObjectives() {
	for i := range s.Objectives {
		obj := &s.Objectives[i]
		for _, e := range s.Entities {
			if e.ID == obj.TargetID && e.Status != StatusActive {
				obj.Complete = true
			}
		}
	}
	for _, e := range s.Entities {
		if e.Side == SideNeutral && e.Status != StatusActive {
			s.FailReason = "Civilian vessel destroyed: " + e.Name
		}
	}
}

func (s *Scenario) MissionComplete() bool {
	if s.MissionFailed() {
		return false
	}
	for _, o := range s.Objectives {
		if !o.Complete {
			return false
		}
	}
	return true
}

func (s *Scenario) MissionFailed() bool {
	if s.Player.Status != StatusActive {
		if s.FailReason == "" {
			s.FailReason = "Ownship lost."
		}
		return true
	}
	if s.FailReason != "" {
		return true
	}
	for _, e := range s.Entities {
		if e.Side == SideNeutral && e.Status != StatusActive {
			s.FailReason = "Civilian vessel destroyed: " + e.Name
			return true
		}
	}
	return false
}
