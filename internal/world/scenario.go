package world

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/ssn688/sim/internal/i18n"
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
	CommBriefing i18n.TranslatedText
	// CommSchedule delivers traffic when COMM mast is raised at/after AtSec.
	CommSchedule []CommScheduledMessage
	// StartTimeSec is seconds from midnight for wall-clock UI/COMM (mission start_time).
	StartTimeSec float64
	// Routes are coastal / transit lanes assigned to AI units by RouteID.
	Routes []*Route
	// MissionEvents are declarative when/then hooks (reveal objective, late COMM, …).
	MissionEvents []MissionEvent
	// FiredEventIDs tracks one-shot mission event dispatch.
	FiredEventIDs map[string]bool
}

// MissionEvent is a runtime when/then rule copied from campaign JSON.
type MissionEvent struct {
	ID          string
	WhenType    string
	ObjectiveID string
	UnitID      string
	Actions     []MissionEventAction
}

// MissionEventAction is one side-effect of a MissionEvent.
type MissionEventAction struct {
	Type        string
	ID          string
	Text        i18n.TranslatedText
	AtSec       float64
	UnitID      string
	Defcon      int
	AIState     string
	Var         string
	Value       string
	ObjectiveID string
}

// CommScheduledMessage is follow-on traffic gated by game time + raised COMM mast.
type CommScheduledMessage struct {
	ID    string
	AtSec float64
	Text  i18n.TranslatedText
}

// CommInboxEntry is a received (or briefing) message in the COMM console.
type CommInboxEntry struct {
	TimeSec float64
	Body    i18n.TranslatedText
}

// DisplayText returns the message body for lang (falls back via TranslatedText).
func (e CommInboxEntry) DisplayText(lang string) string {
	return e.Body.GetText(lang)
}

type Objective struct {
	ID           string
	Description  i18n.TranslatedText
	Complete     bool
	TargetID     string
	Primary      bool // true = primary task
	NeedIdentify bool // must successfully ID the target
	NeedDestroy  bool // must sink / destroy the target
	Identified   bool
	// Hidden tasks stay out of COMM REPORT / UI until RevealObjective.
	Hidden bool
}

// ClampSubToBottom keeps a sub above the charted seafloor with a 60 ft keel margin.
func ClampSubToBottom(e *Entity, bathy *Bathymetry) {
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
	PlaceAwayFrom(rng, e, origin, minR, maxR, others, bathy)
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
	PlaceAwayFrom(rng, e, ref, rangeYd*0.7, rangeYd*1.4, nil, bathy)
}

// PlaceAwayFrom positions e on navigable water at range [minR, maxR] from ref.
func PlaceAwayFrom(rng *rand.Rand, e, ref *Entity, minR, maxR float64, others []*Entity, bathy *Bathymetry) {
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

// RevealObjective clears Hidden on the named task (COMM / event follow-on orders).
func (s *Scenario) RevealObjective(id string) {
	if s == nil || id == "" {
		return
	}
	for i := range s.Objectives {
		if s.Objectives[i].ID == id {
			s.Objectives[i].Hidden = false
		}
	}
}

// MissionStatusReport formats current objective progress for COMM REPORT.
// Hidden objectives are omitted until revealed.
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
	visible := 0
	for _, o := range s.Objectives {
		if !o.Hidden {
			visible++
		}
	}
	if visible == 0 {
		b.WriteString("NO OBJECTIVES ASSIGNED.")
		return b.String()
	}
	b.WriteString("OBJECTIVES:\n")
	for _, o := range s.Objectives {
		if o.Hidden {
			continue
		}
		mark := "OPEN"
		if o.Complete {
			mark = "DONE"
		}
		desc := o.Description.GetText(i18n.CurrentLang())
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
