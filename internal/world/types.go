package world

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
	Name         string
	Class        string
	Kind         EntityKind
	Bands        []NoiseBand
	Tonals       []TonalLine
	BladeRateHz  float64
	CavitationDB float64
}
