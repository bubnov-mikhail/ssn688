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
	Bathy       *Bathymetry
}

type Objective struct {
	ID          string
	Description string
	Complete    bool
	TargetID    string
}

func NewTrainingScenario() *Scenario {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	bathy := DefaultBathy

	// Santa Catalina shelf drops quickly into basin water; keep initial depth conservative.
	player := &Entity{
		ID: "player", Name: "USS Los Angeles", Kind: KindSubmarine, Side: SidePlayer,
		Status: StatusActive, SignatureID: "los_angeles",
		X: 0, Y: 0, DepthFt: 180, HeadingDeg: 90, SpeedKts: 8,
		OrderedSpeed: 8, OrderedDepth: 180, OrderedHead: 90,
		LengthFt: 360,
	}
	placeOnWater(rng, player, 0, 2500, nil, &bathy)
	clampSubToBottom(player, &bathy)

	enemySurface := &Entity{
		ID: "enemy_surface", Name: "Hostile DD", Kind: KindSurfaceShip, Side: SideEnemy,
		Status: StatusActive, SignatureID: "spruance",
		DepthFt: 0, SpeedKts: 14, OrderedSpeed: 14, OrderedDepth: 0,
		LengthFt: 563, AIState: "SEARCH",
	}
	placeOnWater(rng, enemySurface, 4500, 9000, []*Entity{player}, &bathy)

	enemySub := &Entity{
		ID: "enemy_sub", Name: "Hostile SS", Kind: KindSubmarine, Side: SideEnemy,
		Status: StatusActive, SignatureID: "kilo",
		DepthFt: 120 + rng.Float64()*80, SpeedKts: 6, OrderedSpeed: 6,
		LengthFt: 240, AIState: "PATROL",
	}
	enemySub.OrderedDepth = enemySub.DepthFt
	placeOnWater(rng, enemySub, 4000, 8000, []*Entity{player, enemySurface}, &bathy)
	clampSubToBottom(enemySub, &bathy)

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
		placeOnWater(rng, c, 2500, 9000, placed, &bathy)
		placed = append(placed, c)
	}

	ents := []*Entity{enemySurface, enemySub}
	ents = append(ents, civilians...)

	return &Scenario{
		Name:        "Santa Catalina Approaches",
		Description: "Locate and destroy hostile units near Santa Catalina Island. Do not attack civilian shipping.",
		Player:      player,
		Entities:    ents,
		Bathy:       &DefaultBathy,
		Objectives: []Objective{
			{ID: "obj_surface", Description: "Destroy hostile surface combatant", TargetID: "enemy_surface"},
			{ID: "obj_sub", Description: "Destroy hostile submarine", TargetID: "enemy_sub"},
		},
	}
}

func clampSubToBottom(e *Entity, bathy *Bathymetry) {
	if e == nil || e.Kind != KindSubmarine || bathy == nil || !bathy.Valid() {
		return
	}
	bot := bathy.DepthAtFt(e.X, e.Y)
	maxDepth := bot - 60
	if maxDepth < 60 {
		maxDepth = 60
	}
	if e.DepthFt > maxDepth {
		e.DepthFt = maxDepth
		e.OrderedDepth = maxDepth
	}
}

func placeOnWater(rng *rand.Rand, e *Entity, minR, maxR float64, others []*Entity, bathy *Bathymetry) {
	for attempt := 0; attempt < 80; attempt++ {
		ang := rng.Float64() * 2 * math.Pi
		r := minR + rng.Float64()*(maxR-minR)
		if minR <= 0 && maxR > 0 && attempt < 20 {
			// Prefer near-origin for ownship first attempts.
			r = rng.Float64() * maxR
		}
		e.X = math.Sin(ang) * r
		e.Y = math.Cos(ang) * r
		e.HeadingDeg = rng.Float64() * 360
		e.OrderedHead = e.HeadingDeg
		if bathy != nil && bathy.Valid() && !bathy.NavigableFor(e.X, e.Y, e.Kind, e.DepthFt) {
			continue
		}
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
	// Last resort: walk a spiral from origin looking for water.
	if bathy != nil && bathy.Valid() {
		minX, minY, maxX, maxY := bathy.BoundsYards()
		maxR := math.Min(maxX-minX, maxY-minY) * 0.45
		if maxR < 12000 {
			maxR = 12000
		}
		for r := 500.0; r < maxR; r += 400 {
			for k := 0; k < 16; k++ {
				ang := float64(k) * (2 * math.Pi / 16)
				e.X = math.Sin(ang) * r
				e.Y = math.Cos(ang) * r
				if bathy.NavigableFor(e.X, e.Y, e.Kind, e.DepthFt) {
					e.HeadingDeg = rng.Float64() * 360
					e.OrderedHead = e.HeadingDeg
					return
				}
			}
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

// BottomDepthAt returns chart depth under a point, falling back to acoustic env default.
func (s *Scenario) BottomDepthAt(x, y float64) float64 {
	if s.Bathy != nil && s.Bathy.Valid() {
		d := s.Bathy.DepthAtFt(x, y)
		if d > 0 {
			return d
		}
	}
	return 2200
}
