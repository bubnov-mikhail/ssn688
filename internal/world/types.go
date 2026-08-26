package world

import "github.com/ssn688/sim/internal/i18n"

type Side int

const (
	SidePlayer Side = iota
	SideEnemy
	SideNeutral
)

type EntityKind int

const (
	KindSubmarine EntityKind = iota
	KindSurfaceShip
	KindTorpedo
	KindCountermeasure // soft-kill ADC / Nixie (acoustic seduction only)
)

type Status int

const (
	StatusActive Status = iota
	StatusDestroyed
	StatusSunk
	StatusSinking // mortally hit; still radiates noise while settling
)

// NoiseBand describes broadband acoustic emission over a frequency range.
type NoiseBand struct {
	LowHz   float64
	HighHz  float64
	LevelDB float64
}

// TonalLine is a narrowband LOFAR/DEMON peak used for signature matching
// (Cold Waters style fingerprint lines on the spectrum analyzer).
type TonalLine struct {
	FreqHz   float64
	RelLevel float64 // 0..1 display / relative source strength
}

// SignatureProfile is a library entry for classification.
type SignatureProfile struct {
	ID           string
	Name         i18n.TranslatedText
	Class        i18n.TranslatedText
	Kind         EntityKind
	Bands        []NoiseBand
	Tonals       []TonalLine
	BladeRateHz  float64
	CavitationDB float64
}

// DisplayName is the profile name in the active UI language.
func (p SignatureProfile) DisplayName() string {
	return p.Name.GetText(i18n.CurrentLang())
}

// DisplayClass is the short class label in the active UI language.
func (p SignatureProfile) DisplayClass() string {
	return p.Class.GetText(i18n.CurrentLang())
}

// MatchesLabel reports whether s equals Name or Class in any supported language.
func (p SignatureProfile) MatchesLabel(s string) bool {
	if s == "" {
		return false
	}
	for _, lang := range i18n.SupportedLangs {
		if p.Name.GetText(lang) == s || p.Class.GetText(lang) == s {
			return true
		}
	}
	return false
}
