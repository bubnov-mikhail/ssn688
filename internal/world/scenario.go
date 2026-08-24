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
	// Routes are coastal / transit lanes assigned to AI units by RouteID.
	Routes []*Route
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
	ID           string
	Description  string
	Complete     bool
	TargetID     string
	Primary      bool // true = primary task
	NeedIdentify bool // must successfully ID the target
	NeedDestroy  bool // must sink / destroy the target
	Identified   bool
}

func NewTrainingScenario() *Scenario {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	bathy := DefaultBathy

	// Coarse NW→SE PingPong lanes with lateral offsets (may cross near the island).
	type laneSpec struct {
		id     string
		offset float64
		numWP  int
	}
	lanes := []laneSpec{
		{"route_grisha", -3500, 60},
		{"route_merchant", -1200, 50},
		{"route_tanker", 800, 50},
		{"route_trawler", 2200, 60},
		{"route_foxtrot", 4200, 50},
	}
	routes := make([]*Route, 0, len(lanes))
	for _, ln := range lanes {
		if r := BuildNWSETransit(&bathy, ln.id, ln.offset, ln.numWP); r != nil {
			routes = append(routes, r)
		}
	}

	player := &Entity{
		ID: "player", Name: "USS Los Angeles", Kind: KindSubmarine, Side: SidePlayer,
		Status: StatusActive, SignatureID: "los_angeles",
		X: 0, Y: 0, DepthFt: 60, HeadingDeg: 45, SpeedKts: 0,
		OrderedSpeed: 0, OrderedDepth: 60, OrderedHead: 45,
		LengthFt: 360,
	}
	// Near SW corner, within 3000 yd of a transit lane (not sitting on it).
	const minRouteClearYd = 800.0
	const maxRouteClearYd = 3000.0
	if !PlaceNearChartCorner(player, &bathy, "SW", routes, minRouteClearYd, maxRouteClearYd) {
		placeOnWater(rng, player, 0, 4000, nil, &bathy)
	}
	clampSubToBottom(player, &bathy)

	enemyGrisha := &Entity{
		ID: "enemy_grisha", Name: "Hostile Corvette", Kind: KindSurfaceShip, Side: SideEnemy,
		Status: StatusActive, SignatureID: "grisha",
		DepthFt: 0, SpeedKts: 14, OrderedSpeed: 14, OrderedDepth: 0,
		LengthFt: 235, AIState: "PATROL", Defcon: DefconPassive,
		CrewSkill: RandomCrewSkill(60, 20, rng.Float64()),
	}
	civMerchant := &Entity{
		ID: "civ_merchant", Name: "MV Pacific Star", Kind: KindSurfaceShip, Side: SideNeutral,
		Status: StatusActive, SignatureID: "merchant",
		DepthFt: 0, SpeedKts: 11, OrderedSpeed: 11, LengthFt: 520, AIState: "CRUISE",
	}
	civTanker := &Entity{
		ID: "civ_tanker", Name: "MT Horizon", Kind: KindSurfaceShip, Side: SideNeutral,
		Status: StatusActive, SignatureID: "tanker",
		DepthFt: 0, SpeedKts: 9, OrderedSpeed: 9, LengthFt: 900, AIState: "CRUISE",
	}
	civTrawler := &Entity{
		ID: "civ_trawler", Name: "FV Northern Light", Kind: KindSurfaceShip, Side: SideNeutral,
		Status: StatusActive, SignatureID: "fishing",
		DepthFt: 0, SpeedKts: 7, OrderedSpeed: 7, LengthFt: 140, AIState: "CRUISE",
	}
	enemyFoxtrot := &Entity{
		ID: "enemy_foxtrot", Name: "Hostile SS Foxtrot", Kind: KindSubmarine, Side: SideEnemy,
		Status: StatusActive, SignatureID: "foxtrot",
		DepthFt: 100 + rng.Float64()*60, SpeedKts: 5, OrderedSpeed: 5,
		LengthFt: 300, AIState: "PATROL", Defcon: DefconPassive,
		CrewSkill: RandomCrewSkill(30, 10, rng.Float64()),
	}
	enemyFoxtrot.OrderedDepth = enemyFoxtrot.DepthFt

	allySpruance := &Entity{
		ID: "ally_spruance", Name: "USS Spruance", Kind: KindSurfaceShip, Side: SidePlayer,
		Status: StatusActive, SignatureID: "spruance",
		DepthFt: 0, SpeedKts: 12, OrderedSpeed: 12, OrderedDepth: 0,
		LengthFt: 563, AIState: "PATROL", Defcon: DefconHostile,
		CrewSkill: RandomCrewSkill(70, 15, rng.Float64()),
	}
	allySSN := &Entity{
		ID: "ally_688", Name: "USS Bremerton", Kind: KindSubmarine, Side: SidePlayer,
		Status: StatusActive, SignatureID: "los_angeles",
		DepthFt: 140 + rng.Float64()*40, SpeedKts: 5, OrderedSpeed: 5,
		LengthFt: 360, AIState: "PATROL", Defcon: DefconHostile,
		CrewSkill: RandomCrewSkill(75, 10, rng.Float64()),
	}
	allySSN.OrderedDepth = allySSN.DepthFt

	// Ally edge patrol: SE → SW along bottom, then up the west edge to NW.
	if allyRoute := BuildAllyEdgePatrol(&bathy, "route_ally_edge", 24); allyRoute != nil {
		routes = append(routes, allyRoute)
	}

	placeFrac := []struct {
		e    *Entity
		id   string
		frac float64
	}{
		{enemyGrisha, "route_grisha", 0.22},
		{civMerchant, "route_merchant", 0.38},
		{civTanker, "route_tanker", 0.55},
		{civTrawler, "route_trawler", 0.70},
		{enemyFoxtrot, "route_foxtrot", 0.45},
		// Start near SE (lower-right); stagger so they are not stacked.
		{allySpruance, "route_ally_edge", 0.02},
		{allySSN, "route_ally_edge", 0.08},
	}
	for _, p := range placeFrac {
		r := FindRoute(routes, p.id)
		if r == nil || !PlaceOnRouteFraction(p.e, r, p.frac, &bathy) {
			if p.e.Side == SidePlayer {
				if !PlaceNearChartCorner(p.e, &bathy, "SE", nil, 0, 0) {
					placeAwayFrom(rng, p.e, player, 2500, 9000, nil, &bathy)
				}
				if r != nil {
					AssignRoute(p.e, r)
				}
			} else {
				placeAwayFrom(rng, p.e, player, 2500, 9000, nil, &bathy)
				if r != nil {
					AssignRoute(p.e, r)
				}
			}
		}
		if p.e.Kind == KindSubmarine {
			clampSubToBottom(p.e, &bathy)
		}
	}

	hostiles := []*Entity{enemyGrisha, enemyFoxtrot}
	allies := []*Entity{allySpruance, allySSN}
	ents := append([]*Entity{}, hostiles...)
	ents = append(ents, allies...)
	ents = append(ents, civMerchant, civTanker, civTrawler)

	InitCombatantDamage(player)
	for _, h := range hostiles {
		InitCombatantDamage(h)
	}
	for _, a := range allies {
		InitCombatantDamage(a)
	}

	return &Scenario{
		Name:        "Santa Catalina Approaches",
		Description: "Find and sink the hostile diesel boat. Identify and sink the hostile surface combatant. Identify the tanker. Do not attack civilian shipping.",
		Player:      player,
		Entities:    ents,
		Bathy:       &DefaultBathy,
		Weather:     RandomWeather(rng),
		Routes:      routes,
		Objectives: []Objective{
			{
				ID: "obj_foxtrot", Primary: true, NeedDestroy: true, TargetID: "enemy_foxtrot",
				Description: "Locate and sink hostile diesel submarine",
			},
			{
				ID: "obj_grisha", NeedIdentify: true, NeedDestroy: true, TargetID: "enemy_grisha",
				Description: "Identify and sink hostile surface combatant",
			},
			{
				ID: "obj_tanker", NeedIdentify: true, TargetID: "civ_tanker",
				Description: "Locate and identify tanker (do not engage)",
			},
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
				"EXECUTE.\n" +
				"PRIMARY: LOCATE AND SINK HOSTILE DIESEL SUBMARINE IN YOUR OP AREA.\n" +
				"SECONDARY: LOCATE, POSITIVELY IDENTIFY, AND SINK HOSTILE SURFACE COMBATANT.\n" +
				"SECONDARY: LOCATE AND POSITIVELY IDENTIFY TANKER. DO NOT ENGAGE.\n" +
				"ID CRITERIA: VISUAL VIA PERISCOPE INSIDE 800 YARDS, OR ACOUSTIC FINGERPRINT — 80 PCT HARMONIC MATCH WITH LIBRARY ETALON HELD TWO MINUTES.\n" +
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

// placeAtBearing puts e near ref at bearingDeg °T and rangeYd, nudging if dry.
func placeAtBearing(rng *rand.Rand, e, ref *Entity, bearingDeg, rangeYd float64, bathy *Bathymetry) {
	if ref == nil {
		ref = &Entity{}
	}
	if rangeYd < 100 {
		rangeYd = 100
	}
	tryR := []float64{rangeYd, rangeYd * 0.85, rangeYd * 1.15, rangeYd * 0.7, rangeYd * 1.35}
	tryBrg := []float64{0, 15, -15, 30, -30, 45, -45}
	for _, dr := range tryR {
		for _, db := range tryBrg {
			a := (bearingDeg + db) * math.Pi / 180
			e.X = ref.X + math.Sin(a)*dr
			e.Y = ref.Y + math.Cos(a)*dr
			if bathy != nil && bathy.Valid() && !bathy.NavigableFor(e.X, e.Y, e.Kind, e.DepthFt) {
				continue
			}
			// Face roughly across the LOS so peri sees a beam-ish aspect.
			e.HeadingDeg = math.Mod(bearingDeg+90+rng.Float64()*40-20+360, 360)
			e.OrderedHead = e.HeadingDeg
			return
		}
	}
	// Fall back to random ring.
	placeAwayFrom(rng, e, ref, rangeYd*0.7, rangeYd*1.4, nil, bathy)
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
		destroyed := false
		for _, e := range s.Entities {
			if e.ID == obj.TargetID && e.Status != StatusActive {
				destroyed = true
				break
			}
		}
		idOK := !obj.NeedIdentify || obj.Identified
		killOK := !obj.NeedDestroy || destroyed
		obj.Complete = idOK && killOK
	}
}

// NoteIdentified marks matching objectives as identified (sticky).
func (s *Scenario) NoteIdentified(entityID string) {
	if s == nil || entityID == "" {
		return
	}
	for i := range s.Objectives {
		if s.Objectives[i].TargetID == entityID {
			s.Objectives[i].Identified = true
		}
	}
}

func (s *Scenario) PrimaryObjectivesComplete() bool {
	if s == nil {
		return false
	}
	s.CheckObjectives()
	for _, o := range s.Objectives {
		if !o.Primary {
			continue
		}
		if !o.Complete {
			return false
		}
	}
	return len(s.Objectives) > 0
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
		prio := "SEC"
		if o.Primary {
			prio = "PRI"
		}
		b.WriteString(fmt.Sprintf("  [%s] %s  %s", mark, prio, desc))
		if o.NeedIdentify {
			idMark := "NO"
			if o.Identified {
				idMark = "YES"
			}
			b.WriteString("  ID:" + idMark)
		}
		if o.NeedDestroy {
			kill := "NO"
			for _, e := range s.Entities {
				if e.ID == o.TargetID && e.Status != StatusActive {
					kill = "YES"
					break
				}
			}
			b.WriteString("  KILL:" + kill)
		}
		b.WriteString("\n")
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
