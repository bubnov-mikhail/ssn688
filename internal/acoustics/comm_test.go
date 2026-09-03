package acoustics_test

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

func TestCOMMMastAutoRetractOnSpeed(t *testing.T) {
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
		Status: world.StatusActive, DepthFt: 60, SpeedKts: 6, Damage: world.NewFullHealth(),
	}
	var comm acoustics.COMMState
	comm.OrderRaiseCOMM(player)
	comm.Extension = 1
	player.SpeedKts = 14
	evs := acoustics.AutoProtectExtendedGear(player, nil, &comm, nil, nil)
	if len(evs) == 0 || comm.Order != acoustics.COMMMastStow {
		t.Fatalf("expected auto-retract, events=%v order=%v", evs, comm.Order)
	}
	if player.Damage.Destroyed(world.SysCOMM) {
		t.Fatal("SysCOMM should not be destroyed")
	}
}

func TestCOMMScheduleRequiresMastUp(t *testing.T) {
	sc := &world.Scenario{
		CommSchedule: []world.CommScheduledMessage{{
			ID: "m1", AtSec: 5, Text: i18n.T("EXECUTE.", "EXECUTE."),
		}},
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Status: world.StatusActive,
		DepthFt: 60, Damage: world.NewFullHealth(),
	}
	var comm acoustics.COMMState
	comm.SeedBriefing(i18n.T("BRIEF", "BRIEF"))
	acoustics.UpdateCOMM(&comm, sc, player, 30)
	if len(comm.Inbox) != 1 {
		t.Fatalf("mast down should not deliver schedule, inbox=%d", len(comm.Inbox))
	}
	comm.Order = acoustics.COMMMastRaise
	comm.Extension = 1
	acoustics.UpdateCOMM(&comm, sc, player, 30)
	if !comm.NotifyPending || len(comm.Inbox) != 2 {
		t.Fatalf("expected delivery, pending=%v inbox=%d", comm.NotifyPending, len(comm.Inbox))
	}
	if !comm.DeliveredIDs["m1"] {
		t.Fatal("expected delivered id")
	}
}

func TestCOMMTrafficWaitingWhileMastDown(t *testing.T) {
	sc := &world.Scenario{
		CommSchedule: []world.CommScheduledMessage{{
			ID: "m1", AtSec: 10, Text: i18n.T("EXECUTE.", "EXECUTE."),
		}},
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Status: world.StatusActive,
		DepthFt: 60, Damage: world.NewFullHealth(),
	}
	var comm acoustics.COMMState
	acoustics.UpdateCOMMTrafficWaiting(&comm, sc, player, 5)
	if comm.TrafficWaiting || comm.TrafficWaitingNotify {
		t.Fatal("nothing due yet")
	}
	acoustics.UpdateCOMMTrafficWaiting(&comm, sc, player, 10)
	if !comm.TrafficWaiting || !comm.TrafficWaitingNotify {
		t.Fatalf("expected waiting+notify, waiting=%v notify=%v", comm.TrafficWaiting, comm.TrafficWaitingNotify)
	}
	comm.TrafficWaitingNotify = false
	acoustics.UpdateCOMMTrafficWaiting(&comm, sc, player, 15)
	if !comm.TrafficWaiting || comm.TrafficWaitingNotify {
		t.Fatal("should stay waiting without re-notify")
	}
	comm.Order = acoustics.COMMMastRaise
	comm.Extension = 1
	acoustics.UpdateCOMM(&comm, sc, player, 15)
	acoustics.UpdateCOMMTrafficWaiting(&comm, sc, player, 15)
	if comm.TrafficWaiting {
		t.Fatal("waiting should clear after delivery")
	}
}

func TestCOMMCatchUpDeliversBacklog(t *testing.T) {
	sc := &world.Scenario{
		CommSchedule: []world.CommScheduledMessage{
			{ID: "early", AtSec: 10, Text: i18n.T("MSG A", "MSG A")},
			{ID: "mid", AtSec: 20, Text: i18n.T("MSG B", "MSG B")},
			{ID: "late", AtSec: 50, Text: i18n.T("MSG C", "MSG C")},
		},
	}
	player := &world.Entity{
		ID: "player", Kind: world.KindSubmarine, Status: world.StatusActive,
		DepthFt: 60, Damage: world.NewFullHealth(),
	}
	var comm acoustics.COMMState
	// Stay stowed past first two transmit times.
	for tsec := 0.0; tsec <= 35; tsec += 1 {
		acoustics.UpdateCOMM(&comm, sc, player, tsec)
	}
	if len(comm.Inbox) != 0 {
		t.Fatalf("expected empty inbox while stowed, got %d", len(comm.Inbox))
	}

	comm.Order = acoustics.COMMMastRaise
	comm.Extension = 0.92 // nearly up — should flush backlog
	acoustics.UpdateCOMM(&comm, sc, player, 35)
	if len(comm.Inbox) != 2 {
		t.Fatalf("expected 2 catch-up messages, got %d", len(comm.Inbox))
	}
	if comm.Inbox[0].TimeSec != 10 || comm.Inbox[1].TimeSec != 20 {
		t.Fatalf("stamps=%v %v want 10 and 20", comm.Inbox[0].TimeSec, comm.Inbox[1].TimeSec)
	}
	if !comm.NotifyPending {
		t.Fatal("expected notify on backlog flush")
	}
	if comm.DeliveredIDs["late"] {
		t.Fatal("future message must not deliver early")
	}

	// Raising further later still gets the third when due.
	acoustics.UpdateCOMM(&comm, sc, player, 50)
	if len(comm.Inbox) != 3 || !comm.DeliveredIDs["late"] {
		t.Fatalf("expected third message at t=50, inbox=%d", len(comm.Inbox))
	}
}

func TestTrainingBriefingPresent(t *testing.T) {
	sc := campaign.DemoRuntime()
	if sc == nil || sc.CommBriefing.GetText(i18n.LangEN) == "" {
		t.Fatal("missing briefing")
	}
	if len(sc.CommSchedule) == 0 || sc.CommSchedule[0].AtSec != 20 {
		t.Fatalf("expected 20s follow-on, got %#v", sc.CommSchedule)
	}
}
