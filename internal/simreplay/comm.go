package simreplay

import (
	"fmt"
	"math/rand"
	"os"
	"sort"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/sim"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// CommSnap is one inbox message the player would receive (mast up).
type CommSnap struct {
	TimeSec float64             `json:"time_sec"`
	ID      string              `json:"id,omitempty"`
	Body    i18n.TranslatedText `json:"body"`
}

// CommPlayhead dispatches COMM snaps relative to the replay playhead.
// Sync adds due messages (TimeSec <= t) and drops later ones when scrubbing back.
type CommPlayhead struct {
	all   []CommSnap
	inbox []CommSnap
}

// NewCommPlayhead builds a dispatcher over a timeline (copied + sorted by time).
func NewCommPlayhead(msgs []CommSnap) *CommPlayhead {
	all := append([]CommSnap(nil), msgs...)
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].TimeSec != all[j].TimeSec {
			return all[i].TimeSec < all[j].TimeSec
		}
		return all[i].ID < all[j].ID
	})
	return &CommPlayhead{all: all}
}

// Sync updates the received inbox for simulation time t.
func (p *CommPlayhead) Sync(t float64) {
	if p == nil {
		return
	}
	n := 0
	for n < len(p.all) && p.all[n].TimeSec <= t+1e-9 {
		n++
	}
	if n < len(p.inbox) {
		p.inbox = p.inbox[:n]
		return
	}
	for i := len(p.inbox); i < n; i++ {
		p.inbox = append(p.inbox, p.all[i])
	}
}

// Inbox returns messages received at the last Sync time.
func (p *CommPlayhead) Inbox() []CommSnap {
	if p == nil {
		return nil
	}
	return p.inbox
}

// CommLines builds scrollable COMM lines from a playhead inbox.
func CommLines(msgs []CommSnap, startSec, t float64, lang string, maxW int) []render.MDLine {
	var lines []render.MDLine
	for _, msg := range msgs {
		if msg.TimeSec > t+1e-9 {
			continue
		}
		if msg.Body.GetText(i18n.LangEN) == "" {
			continue
		}
		stamp := "[" + world.FormatMissionClock(startSec, msg.TimeSec) + "]"
		lines = append(lines, render.MarkdownLinesForCOMM(stamp, msg.Body.GetText(lang), maxW)...)
	}
	return lines
}

// CaptureCommTimeline runs the mission with COMM mast forced up and records inbox traffic.
func CaptureCommTimeline(opt RecordOptions) (msgs []CommSnap, startSec float64, err error) {
	eng, m, err := newMissionEngine(opt)
	if err != nil {
		return nil, 0, err
	}
	maxSec := opt.MaxSec
	if maxSec <= 0 {
		maxSec = DefaultMaxSec
	}
	forceCOMMCatchUp(eng)

	prev := 0
	dt := 1.0 / sim.TickRate
	for eng.Clock.GameTime < maxSec {
		eng.Update(dt)
		forceCOMMCatchUp(eng)
		prev = appendInboxDelta(eng, prev, &msgs)
	}
	sortCommSnaps(msgs)
	return msgs, m.StartTimeSec, nil
}

// forceCOMMCatchUp keeps the COMM mast raised during AFK capture and delivers any
// schedule traffic due at the current game time. Ownship often starts below the
// auto-retract depth, which would otherwise stow the mast and drop HQ messages.
// Also re-runs delivery after mission events append schedule entries mid-tick.
func forceCOMMCatchUp(eng *sim.Engine) {
	if eng == nil || eng.Scenario == nil || eng.Scenario.Player == nil {
		return
	}
	eng.COMM.Extension = 1.0
	eng.COMM.Order = acoustics.COMMMastRaise
	eng.COMM.Sheared = false
	acoustics.UpdateCOMM(&eng.COMM, eng.Scenario, eng.Scenario.Player, eng.Clock.GameTime)
}

func appendInboxDelta(eng *sim.Engine, prev int, out *[]CommSnap) int {
	if eng == nil {
		return prev
	}
	inbox := eng.COMM.Inbox
	for i := prev; i < len(inbox); i++ {
		e := inbox[i]
		*out = append(*out, CommSnap{
			TimeSec: e.TimeSec,
			ID:      e.SourceID,
			Body:    e.Body,
		})
	}
	return len(inbox)
}

func sortCommSnaps(msgs []CommSnap) {
	sort.SliceStable(msgs, func(i, j int) bool {
		if msgs[i].TimeSec != msgs[j].TimeSec {
			return msgs[i].TimeSec < msgs[j].TimeSec
		}
		return msgs[i].ID < msgs[j].ID
	})
}

func newMissionEngine(opt RecordOptions) (*sim.Engine, *campaign.MissionDef, error) {
	if opt.Seed == 0 {
		opt.Seed = 1
	}
	data, err := os.ReadFile(opt.ScenarioPath)
	if err != nil {
		return nil, nil, err
	}
	scDef, err := campaign.ParseScenarioJSON(data, opt.ScenarioPath)
	if err != nil {
		return nil, nil, err
	}
	m := campaign.FindMission(&scDef, campaign.MissionID(opt.MissionID))
	if m == nil {
		return nil, nil, fmt.Errorf("mission %q not found", opt.MissionID)
	}
	ctx := campaign.BuildContext{Vars: map[string]string{}}
	rng := rand.New(rand.NewSource(opt.Seed))
	rt := campaign.Instantiate(&scDef, m, ctx)
	if rt == nil {
		return nil, nil, fmt.Errorf("instantiate failed")
	}
	rollCrew(rng, m, rt)
	eng := sim.NewEngine(rt)
	campaign.ApplyUnitPayloads(&eng.FireControl, m, ctx.Vars)
	if opt.PlayerIdle && eng.Scenario.Player != nil {
		eng.Scenario.Player.OrderedSpeed = 0
		eng.Scenario.Player.SpeedKts = 0
	}
	return eng, m, nil
}
