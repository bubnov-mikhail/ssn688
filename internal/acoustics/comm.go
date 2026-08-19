package acoustics

import (
	"fmt"
	"math"

	"github.com/ssn688/sim/internal/world"
)

// COMMMastOrder is the commanded HF/VLF comm mast position.
type COMMMastOrder int

const (
	COMMMastStow COMMMastOrder = iota
	COMMMastRaise
)

// COMMState is the player's communication mast + message inbox.
type COMMState struct {
	Order        COMMMastOrder
	Extension    float64
	LastWarnAt   float64
	Sheared      bool
	Inbox        []world.CommInboxEntry
	DeliveredIDs map[string]bool
	NotifyPending bool // UI: flash + voice for a newly delivered scheduled message
}

func (c *COMMState) ensure() {
	if c.DeliveredIDs == nil {
		c.DeliveredIDs = map[string]bool{}
	}
}

// MastUp is true when the antenna is extended enough to receive traffic.
func (c *COMMState) MastUp() bool {
	return c != nil && !c.Sheared && c.Extension >= 0.95
}

// MastMoving reports raise/lower in progress.
func (c *COMMState) MastMoving() bool {
	if c == nil || c.Sheared {
		return false
	}
	if c.Order == COMMMastRaise && c.Extension < 1 {
		return true
	}
	if c.Order == COMMMastStow && c.Extension > 0 {
		return true
	}
	return false
}

// CanRaiseCOMM reports whether depth/speed allow mast raise (same limits as ESM).
func CanRaiseCOMM(player *world.Entity) (ok bool, reason string) {
	if player == nil {
		return false, "No ownship."
	}
	player.EnsureDamage()
	if player.Damage.Destroyed(world.SysCOMM) {
		return false, "COMM mast destroyed — beyond repair."
	}
	if player.DepthFt > world.ESMMastMaxDepthFt+0.5 {
		return false, fmt.Sprintf("Too deep — COMM mast requires ≤%.0f ft (periscope depth).", world.ESMMastMaxDepthFt)
	}
	spd := math.Abs(player.SpeedKts)
	if spd > world.ESMMastMaxSpeedKts+0.05 {
		return false, fmt.Sprintf("Too fast — COMM mast requires ≤%.0f kn.", world.ESMMastMaxSpeedKts)
	}
	return true, ""
}

// OrderRaiseCOMM begins raising if conditions allow.
func (c *COMMState) OrderRaiseCOMM(player *world.Entity) (ok bool, msg string) {
	c.ensure()
	if c.Sheared || (player != nil && player.Damage.Destroyed(world.SysCOMM)) {
		c.Sheared = true
		return false, "COMM mast destroyed — beyond repair."
	}
	if ok, reason := CanRaiseCOMM(player); !ok {
		return false, reason
	}
	c.Order = COMMMastRaise
	return true, "Raising COMM mast."
}

// OrderLowerCOMM begins stowing.
func (c *COMMState) OrderLowerCOMM() string {
	c.ensure()
	c.Order = COMMMastStow
	return "Lowering COMM mast."
}

// SeedBriefing places the mission opening traffic in the inbox (no mast required).
func (c *COMMState) SeedBriefing(text string) {
	c.ensure()
	if text == "" {
		return
	}
	if len(c.Inbox) > 0 {
		return
	}
	c.Inbox = append(c.Inbox, world.CommInboxEntry{TimeSec: 0, Text: text})
}

// AppendLocalTraffic adds an ownship-generated line (e.g. REPORT) to the inbox.
func (c *COMMState) AppendLocalTraffic(gameTime float64, text string) {
	c.ensure()
	if text == "" {
		return
	}
	if gameTime < 0 {
		gameTime = 0
	}
	c.Inbox = append(c.Inbox, world.CommInboxEntry{TimeSec: gameTime, Text: text})
}

// AdvanceMastMotion animates extension and enforces shear limits while exposed.
func (c *COMMState) AdvanceMastMotion(dt, gameTime float64, player *world.Entity) (events []string, shearedNow bool) {
	c.ensure()
	if player == nil {
		return nil, false
	}
	player.EnsureDamage()
	if player.Damage.Destroyed(world.SysCOMM) {
		c.Sheared = true
		c.Order = COMMMastStow
		c.Extension = 0
		return nil, false
	}
	if c.Sheared {
		c.Extension = 0
		c.Order = COMMMastStow
		return nil, false
	}

	switch c.Order {
	case COMMMastRaise:
		if ok, _ := CanRaiseCOMM(player); !ok && c.Extension < 0.05 {
			c.Order = COMMMastStow
		} else {
			c.Extension = math.Min(1, c.Extension+dt/world.ESMMastRaiseSec)
		}
	default:
		c.Extension = math.Max(0, c.Extension-dt/world.ESMMastLowerSec)
	}
	return events, false
}

// UpdateCOMM delivers all scheduled traffic that is due once the mast is up.
// Raising late still receives the full backlog (no need to be up at AtSec).
func UpdateCOMM(comm *COMMState, scenario *world.Scenario, player *world.Entity, gameTime float64) {
	if comm == nil || scenario == nil || player == nil {
		return
	}
	comm.ensure()
	player.EnsureDamage()
	if player.Damage.Destroyed(world.SysCOMM) || comm.Sheared {
		return
	}
	// Accept traffic once the mast is mostly extended — including catch-up after AtSec.
	if comm.Extension < 0.90 {
		return
	}
	deliveredNow := 0
	for i := range scenario.CommSchedule {
		msg := &scenario.CommSchedule[i]
		if msg.ID == "" || msg.Text == "" {
			continue
		}
		if gameTime < msg.AtSec {
			continue
		}
		if comm.DeliveredIDs[msg.ID] {
			continue
		}
		comm.DeliveredIDs[msg.ID] = true
		// Stamp with transmit time so late pickup still shows when HQ sent it.
		stamp := msg.AtSec
		if stamp < 0 {
			stamp = 0
		}
		comm.Inbox = append(comm.Inbox, world.CommInboxEntry{
			TimeSec: stamp,
			Text:    msg.Text,
		})
		deliveredNow++
	}
	if deliveredNow > 0 {
		comm.NotifyPending = true
	}
}
