package weapons

import "github.com/bubnov-mikhail/ssn688/internal/world"

// TorpedoDebugLabel is the short WEPS/PLOT debug tag for an in-water fish.
func TorpedoDebugLabel(t *Torpedo) string {
	if t == nil {
		return "TORP"
	}
	if t.OrdnanceType == OrdnanceMk48Exercise || t.TerminalMode == TerminalSignal {
		return "EX"
	}
	if t.Side == world.SidePlayer {
		if t.WireCut {
			return "MK48 AUTO"
		}
		return "MK48"
	}
	if t.Class == ClassUMGT1 {
		switch t.AcousticSig {
		case "set40":
			return "SET40"
		case "mk46":
			return "MK46"
		default:
			return "LW"
		}
	}
	return "TORP"
}
