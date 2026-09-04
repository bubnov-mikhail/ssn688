package simreplay

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/sim"
	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// RecordOptions configures an AFK headless capture.
type RecordOptions struct {
	ScenarioPath string
	MissionID    string
	Seed         int64
	MaxSec       float64
	SampleSec    float64
	PlayerIdle   bool
	Progress     ProgressFunc
}

// RecordMission runs the sim and writes sampled frames.
func RecordMission(opt RecordOptions) (*Replay, error) {
	if opt.SampleSec <= 0 {
		opt.SampleSec = 1.0
	}
	if opt.MaxSec <= 0 {
		opt.MaxSec = DefaultMaxSec
	}
	if opt.Seed == 0 {
		opt.Seed = 1
	}
	data, err := os.ReadFile(opt.ScenarioPath)
	if err != nil {
		return nil, err
	}
	scDef, err := campaign.ParseScenarioJSON(data, opt.ScenarioPath)
	if err != nil {
		return nil, err
	}
	m := campaign.FindMission(&scDef, campaign.MissionID(opt.MissionID))
	if m == nil {
		return nil, fmt.Errorf("mission %q not found", opt.MissionID)
	}

	ctx := campaign.BuildContext{Vars: map[string]string{}}
	rng := rand.New(rand.NewSource(opt.Seed))
	rt := campaign.Instantiate(&scDef, m, ctx)
	if rt == nil {
		return nil, fmt.Errorf("instantiate failed")
	}
	rollCrew(rng, m, rt)

	eng := sim.NewEngine(rt)
	campaign.ApplyUnitPayloads(&eng.FireControl, m, ctx.Vars)
	player := eng.Scenario.Player
	if opt.PlayerIdle && player != nil {
		player.OrderedSpeed = 0
		player.SpeedKts = 0
	}
	eng.COMM.Extension = 1.0
	eng.COMM.Order = acoustics.COMMMastRaise
	forceCOMMCatchUp(eng)

	title := m.Title.GetText("en")
	if title == "" {
		title = string(m.ID)
	}

	rep := &Replay{
		FormatVersion:   FormatVersion,
		ScenarioID:      string(scDef.ID),
		MissionID:       string(m.ID),
		MissionTitle:    title,
		TheaterID:       string(m.TheaterID),
		Seed:            opt.Seed,
		DurationSec:     opt.MaxSec,
		SampleSec:       opt.SampleSec,
		MissionStartSec: m.StartTimeSec,
		Frames:          make([]Frame, 0, int(opt.MaxSec/opt.SampleSec)+2),
	}

	dt := 1.0 / sim.TickRate
	nextSample := 0.0
	lastPct := -1
	inboxPrev := appendInboxDelta(eng, 0, &rep.Comm)
	reportProgress := func(t float64) {
		if opt.Progress == nil || opt.MaxSec <= 0 {
			return
		}
		pct := int(t/opt.MaxSec*100 + 0.5)
		if pct == lastPct {
			return
		}
		lastPct = pct
		opt.Progress(t, opt.MaxSec)
	}

	for eng.Clock.GameTime < opt.MaxSec {
		eng.Update(dt)
		forceCOMMCatchUp(eng)
		inboxPrev = appendInboxDelta(eng, inboxPrev, &rep.Comm)
		reportProgress(eng.Clock.GameTime)
		if eng.Clock.GameTime+1e-9 >= nextSample {
			rep.Frames = append(rep.Frames, snapshotFrame(eng, player))
			nextSample += opt.SampleSec
		}
	}
	if len(rep.Frames) == 0 || rep.Frames[len(rep.Frames)-1].Time < eng.Clock.GameTime {
		rep.Frames = append(rep.Frames, snapshotFrame(eng, player))
	}
	rep.DurationSec = eng.Clock.GameTime
	sortCommSnaps(rep.Comm)
	if opt.Progress != nil {
		opt.Progress(rep.DurationSec, opt.MaxSec)
	}
	return rep, nil
}

func snapshotFrame(eng *sim.Engine, player *world.Entity) Frame {
	playerID := ""
	if player != nil {
		playerID = player.ID
	}
	units := make([]UnitSnap, 0, len(eng.Scenario.Entities)+1)
	if player != nil {
		units = append(units, entitySnap(player, playerID))
	}
	for _, e := range eng.Scenario.Entities {
		if e == nil || (player != nil && e.ID == player.ID) {
			continue
		}
		units = append(units, entitySnap(e, playerID))
	}
	wpn, flashes := SnapshotWeapons(&eng.FireControl, eng.Clock.GameTime)
	markers := make([]MarkerSnap, 0, len(eng.PlotMarkers))
	for _, m := range eng.PlotMarkers {
		markers = append(markers, MarkerSnap{
			ID: m.ID, Name: m.DisplayLabel("en"), X: m.X, Y: m.Y,
		})
	}
	return Frame{
		Time:    eng.Clock.GameTime,
		Units:   units,
		Weapons: wpn,
		Flashes: flashes,
		Markers: markers,
	}
}

func entitySnap(e *world.Entity, playerID string) UnitSnap {
	return UnitSnap{
		ID:       e.ID,
		Name:     e.Name,
		Side:     SideLabel(e, playerID),
		Status:   StatusLabel(e),
		AIState:  e.AIState,
		Defcon:   e.Defcon,
		X:        e.X,
		Y:        e.Y,
		Heading:  e.HeadingDeg,
		SpeedKts: e.SpeedKts,
		Alive:    e.Alive(),
	}
}

func rollCrew(rng *rand.Rand, m *campaign.MissionDef, rt *world.Scenario) {
	for _, e := range rt.AllEntities() {
		if e == nil {
			continue
		}
		spec := findUnitSpec(m, e.ID)
		if spec == nil && m.Player.ID == e.ID {
			spec = &m.Player
		}
		if spec == nil || (spec.CrewSkill <= 0 && spec.CrewJitter <= 0) {
			continue
		}
		e.CrewSkill = world.RandomCrewSkill(spec.CrewSkill, spec.CrewJitter, rng.Float64())
	}
}

func findUnitSpec(m *campaign.MissionDef, id string) *campaign.UnitSpec {
	for i := range m.Units {
		if m.Units[i].ID == id {
			return &m.Units[i]
		}
	}
	return nil
}
