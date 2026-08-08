package world

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
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
	Weather     Weather
	// RestrictedZones reserved for future missions (player entry → DEFCON 3).
	RestrictedZones []RestrictedZone
	// CommBriefing is shown in the COMM inbox at mission start (no mast required).
	CommBriefing string
	// CommSchedule delivers traffic when COMM mast is raised at/after AtSec.
	CommSchedule []CommScheduledMessage
}

// CommScheduledMessage is follow-on traffic gated by game time + raised COMM mast.
type CommScheduledMessage struct {
	ID    string
	AtSec float64
	Text  string
}

// CommInboxEntry is a received (or briefing) message in the COMM console.
type CommInboxEntry struct {
	TimeSec float64
	Text    string
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
		X: 0, Y: 0, DepthFt: 60, HeadingDeg: 90, SpeedKts: 0,
		OrderedSpeed: 0, OrderedDepth: 60, OrderedHead: 90,
		LengthFt: 360,
	}
	placeOnWater(rng, player, 0, 2500, nil, &bathy)
	clampSubToBottom(player, &bathy)

	placed := []*Entity{player}

	// Weakest surface + weakest sub only — keeps the starter mission readable.
	// Surface contact starts near ownship for MAST/ESM/periscope practice (DEFCON 0).
	enemyGrisha := &Entity{
		ID: "enemy_grisha", Name: "Hostile Corvette", Kind: KindSurfaceShip, Side: SideEnemy,
		Status: StatusActive, SignatureID: "grisha",
		DepthFt: 0, SpeedKts: 16, OrderedSpeed: 16, OrderedDepth: 0,
		LengthFt: 235, AIState: "SEARCH", Defcon: DefconPassive,
	}
	placeAwayFrom(rng, enemyGrisha, player, 1200, 2800, nil, &bathy)
	placed = append(placed, enemyGrisha)

	enemyFoxtrot := &Entity{
		ID: "enemy_foxtrot", Name: "Hostile SS Foxtrot", Kind: KindSubmarine, Side: SideEnemy,
		Status: StatusActive, SignatureID: "foxtrot",
		DepthFt: 100 + rng.Float64()*60, SpeedKts: 5, OrderedSpeed: 5,
		LengthFt: 300, AIState: "PATROL", Defcon: DefconPassive,
	}
	enemyFoxtrot.OrderedDepth = enemyFoxtrot.DepthFt
	placeAwayFrom(rng, enemyFoxtrot, player, 4500, 9000, placed[1:], &bathy)
	clampSubToBottom(enemyFoxtrot, &bathy)
	placed = append(placed, enemyFoxtrot)

	// Two merchants near ownship for visual/ESM contrast; trawler stays farther out.
	nearCivilians := []*Entity{
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
	}
	for _, c := range nearCivilians {
		placeAwayFrom(rng, c, player, 1400, 3200, placed[1:], &bathy)
		placed = append(placed, c)
	}
	farTrawler := &Entity{
		ID: "civ_trawler", Name: "FV Northern Light", Kind: KindSurfaceShip, Side: SideNeutral,
		Status: StatusActive, SignatureID: "fishing",
		DepthFt: 0, SpeedKts: 7, OrderedSpeed: 7, LengthFt: 140, AIState: "TRANSIT",
	}
	placeAwayFrom(rng, farTrawler, player, 4500, 9000, placed[1:], &bathy)
	placed = append(placed, farTrawler)

	hostiles := []*Entity{enemyGrisha, enemyFoxtrot}
	ents := append([]*Entity{}, hostiles...)
	ents = append(ents, nearCivilians...)
	ents = append(ents, farTrawler)

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
		Weather:     RandomWeather(rng),
		Objectives: []Objective{
			{ID: "obj_grisha", Description: "Destroy hostile Grisha corvette", TargetID: "enemy_grisha"},
			{ID: "obj_foxtrot", Description: "Destroy hostile Foxtrot SS", TargetID: "enemy_foxtrot"},
		},
		CommBriefing: "" +
			"TOP SECRET // FLASH\n" +
			"FROM: COMSUBPAC\n" +
			"TO: USS LOS ANGELES (SSN-688)\n" +
			"BT\n" +
			"PROCEED TO ASSIGNED OP AREA VICINITY SANTA CATALINA ISLAND.\n" +
			"CONDUCT COVERT PATROL. REMAIN UNDETECTED. DO NOT ENGAGE UNTIL DIRECTED.\n" +
			"COME TO COMMUNICATIONS DEPTH AND RAISE HF ANTENNA FOR FOLLOW-ON TASKING.\n" +
			"BT",
		CommSchedule: []CommScheduledMessage{{
			ID:    "tasking_engage",
			AtSec: 20,
			Text: "" +
				"TOP SECRET // IMMEDIATE\n" +
				"FROM: COMSUBPAC\n" +
				"TO: USS LOS ANGELES (SSN-688)\n" +
				"BT\n" +
				"EXECUTE. LOCATE AND SINK HOSTILE DIESEL SUBMARINE AND HOSTILE SURFACE COMBATANT IN YOUR OP AREA.\n" +
				"CIVILIAN SHIPPING IS NOT TO BE ENGAGED.\n" +
				"REPORT COMPLETION VIA THIS CHANNEL.\n" +
				"BT",
		}},
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
	return false
}

// MissionStatusReport formats current objective progress for COMM REPORT.
func (s *Scenario) MissionStatusReport() string {
	if s == nil {
		return "NO SCENARIO."
	}
	s.CheckObjectives()
	var b strings.Builder
	b.WriteString("OWN SHIP REPORT — MISSION STATUS\n")
	switch {
	case s.MissionFailed():
		reason := s.FailReason
		if reason == "" {
			reason = "MISSION FAILED"
		}
		b.WriteString("OVERALL: FAILED — " + reason + "\n")
	case s.MissionComplete():
		b.WriteString("OVERALL: COMPLETE — ALL OBJECTIVES MET\n")
	default:
		b.WriteString("OVERALL: IN PROGRESS\n")
	}
	if len(s.Objectives) == 0 {
		b.WriteString("NO OBJECTIVES ASSIGNED.")
		return b.String()
	}
	b.WriteString("OBJECTIVES:\n")
	for _, o := range s.Objectives {
		mark := "OPEN"
		if o.Complete {
			mark = "DONE"
		}
		desc := o.Description
		if desc == "" {
			desc = o.ID
		}
		b.WriteString(fmt.Sprintf("  [%s] %s\n", mark, desc))
	}
	return strings.TrimRight(b.String(), "\n")
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
