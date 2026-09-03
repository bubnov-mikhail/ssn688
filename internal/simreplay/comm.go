package simreplay

import (
	"fmt"
	"math/rand"
	"os"

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

// CommLines builds scrollable COMM lines visible at game time t.
func CommLines(msgs []CommSnap, startSec, t float64, lang string, maxW int) []render.MDLine {
	var lines []render.MDLine
	for _, msg := range msgs {
		if msg.TimeSec > t {
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

// CaptureCommTimeline runs the mission with COMM mast up and records inbox traffic.
func CaptureCommTimeline(opt RecordOptions) (msgs []CommSnap, startSec float64, err error) {
	eng, m, err := newMissionEngine(opt)
	if err != nil {
		return nil, 0, err
	}
	maxSec := opt.MaxSec
	if maxSec <= 0 {
		maxSec = DefaultMaxSec
	}
	eng.COMM.Extension = 1.0
	eng.COMM.Order = acoustics.COMMMastRaise

	prev := 0
	dt := 1.0 / sim.TickRate
	for eng.Clock.GameTime < maxSec {
		eng.Update(dt)
		prev = appendInboxDelta(eng, prev, &msgs)
	}
	return msgs, m.StartTimeSec, nil
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
