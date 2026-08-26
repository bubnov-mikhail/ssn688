package world

import "testing"

func TestParseStartTimeHHMM(t *testing.T) {
	sec, err := ParseStartTimeHHMM("4:30")
	if err != nil || sec != 4*3600+30*60 {
		t.Fatalf("got %v %v", sec, err)
	}
	sec, err = ParseStartTimeHHMM("18:05")
	if err != nil || sec != 18*3600+5*60 {
		t.Fatalf("got %v %v", sec, err)
	}
	if _, err := ParseStartTimeHHMM("25:00"); err == nil {
		t.Fatal("expected error")
	}
}

func TestFormatMissionClock(t *testing.T) {
	start := float64(4*3600 + 30*60) // 04:30
	got := FormatMissionClock(start, 90)
	if got != "04:31:30" {
		t.Fatalf("got %q", got)
	}
	got = FormatMissionClock(float64(23*3600+59*60), 120) // wrap
	if got != "00:01:00" {
		t.Fatalf("wrap got %q", got)
	}
}
