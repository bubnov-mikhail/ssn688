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
	// RestrictedZones reserved for future missions (player entry → DEFCON 3).
	RestrictedZones []RestrictedZone
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

	placed := []*Entity{player}

	// Weakest surface + weakest sub only — keeps the starter mission readable.
	enemyGrisha := &Entity{
		ID: "enemy_grisha", Name: "Hostile Corvette", Kind: KindSurfaceShip, Side: SideEnemy,
		Status: StatusActive, SignatureID: "grisha",
		DepthFt: 0, SpeedKts: 16, OrderedSpeed: 16, OrderedDepth: 0,
		LengthFt: 235, AIState: "SEARCH", Defcon: DefconHostile,
	}
	placeAwayFrom(rng, enemyGrisha, player, YardsPerNM, 4*YardsPerNM, nil, &bathy)
	placed = append(placed, enemyGrisha)

	enemyFoxtrot := &Entity{
		ID: "enemy_foxtrot", Name: "Hostile SS Foxtrot", Kind: KindSubmarine, Side: SideEnemy,
		Status: StatusActive, SignatureID: "foxtrot",
		DepthFt: 100 + rng.Float64()*60, SpeedKts: 5, OrderedSpeed: 5,
		LengthFt: 300, AIState: "PATROL", Defcon: DefconHostile,
	}
	enemyFoxtrot.OrderedDepth = enemyFoxtrot.DepthFt
	placeAwayFrom(rng, enemyFoxtrot, player, 4500, 9000, placed[1:], &bathy)
	clampSubToBottom(enemyFoxtrot, &bathy)
	placed = append(placed, enemyFoxtrot)

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
	for _, c := range civilians {
		placeOnWater(rng, c, 2500, 9000, placed, &bathy)
		placed = append(placed, c)
	}

	hostiles := []*Entity{enemyGrisha, enemyFoxtrot}
	ents := append([]*Entity{}, hostiles...)
	ents = append(ents, civilians...)

	InitCombatantDamage(player)
	for _, h := range hostiles {
		InitCombatantDamage(h)
	}

	return &Scenario{
		Name:        "Santa Catalina Approaches",
		Description: "Locate and destroy hostile units near Santa Catalina Island. Do not attack civilian shipping.",
		Player:      player,
		Entities:    ents,
		Bathy:       &DefaultBathy,
		Objectives: []Objective{
			{ID: "obj_grisha", Description: "Destroy hostile Grisha corvette", TargetID: "enemy_grisha"},
			{ID: "obj_foxtrot", Description: "Destroy hostile Foxtrot SS", TargetID: "enemy_foxtrot"},
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
	origin := &Entity{X: 0, Y: 0}
	placeAwayFrom(rng, e, origin, minR, maxR, others, bathy)
}

// placeAwayFrom positions e on navigable water at range [minR, maxR] from ref.
func placeAwayFrom(rng *rand.Rand, e, ref *Entity, minR, maxR float64, others []*Entity, bathy *Bathymetry) {
	if ref == nil {
		ref = &Entity{}
	}
	if maxR < minR {
		maxR = minR
	}
	minSepOthers := 1800.0
	for attempt := 0; attempt < 80; attempt++ {
		ang := rng.Float64() * 2 * math.Pi
		r := minR + rng.Float64()*(maxR-minR)
		if minR <= 0 && maxR > 0 && attempt < 20 {
			r = rng.Float64() * maxR
		}
		e.X = ref.X + math.Sin(ang)*r
		e.Y = ref.Y + math.Cos(ang)*r
		e.HeadingDeg = rng.Float64() * 360
		e.OrderedHead = e.HeadingDeg
		if bathy != nil && bathy.Valid() && !bathy.NavigableFor(e.X, e.Y, e.Kind, e.DepthFt) {
			continue
		}
		if minR > 0 && ref.RangeYardsTo(e) < minR-1 {
			continue
		}
		ok := true
		for _, o := range others {
			if o == nil {
				continue
			}
			if o.RangeYardsTo(e) < minSepOthers {
				ok = false
				break
			}
		}
		if ok {
			return
		}
	}
	// Last resort: ring around ref at minR+.
	if bathy != nil && bathy.Valid() {
		for r := minR; r < maxR+8000; r += 400 {
			if r < 500 {
				r = 500
			}
			for k := 0; k < 16; k++ {
				ang := float64(k) * (2 * math.Pi / 16)
				e.X = ref.X + math.Sin(ang)*r
				e.Y = ref.Y + math.Cos(ang)*r
				if !bathy.NavigableFor(e.X, e.Y, e.Kind, e.DepthFt) {
					continue
				}
				if minR > 0 && ref.RangeYardsTo(e) < minR {
					continue
				}
				e.HeadingDeg = rng.Float64() * 360
				e.OrderedHead = e.HeadingDeg
				return
			}
		}
	}
}

// AppendAllEntities appends player + mission entities into dst without allocating
// when cap(dst) is already large enough.
func (s *Scenario) AppendAllEntities(dst []*Entity) []*Entity {
	need := 1 + len(s.Entities)
	if cap(dst) < need {
		dst = make([]*Entity, 0, need)
	} else {
		dst = dst[:0]
	}
	dst = append(dst, s.Player)
	dst = append(dst, s.Entities...)
	return dst
}

func (s *Scenario) AllEntities() []*Entity {
	return s.AppendAllEntities(nil)
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
