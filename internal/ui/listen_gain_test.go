package ui

import "testing"

func TestListenGainFromContactSNR(t *testing.T) {
	faint := listenGainFromContactSNR(9)
	hot := listenGainFromContactSNR(24)
	if faint >= hot {
		t.Fatalf("faint SNR gain %.3f should be quieter than hot %.3f", faint, hot)
	}
	if faint < 0.05 || faint > 0.35 {
		t.Fatalf("faint gain out of expected whisper band: %.3f", faint)
	}
	if hot < 0.7 {
		t.Fatalf("hot gain too quiet: %.3f", hot)
	}
	if g := listenGainFromContactSNR(0); g < 0.05 || g > 0.12 {
		t.Fatalf("undetectable floor unexpected: %.3f", g)
	}
}
