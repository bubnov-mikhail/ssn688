package weapons

import (
	"fmt"
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// Spec-derived / gameplay constants (Mk48 ADCAP + 688-class doctrine).
const (
	// SeekAcquireRangeYd is published Mk48 weapon acquisition range (~1600 yd).
	SeekAcquireRangeYd = 1600.0
	// SeekConeHalfAngleDeg is the forward search cone in ModeSearch.
	SeekConeHalfAngleDeg = 35.0
	// ProximityKillYd — influence/proximity fuse for serious damage from ~650 lb HE.
	ProximityKillYd = 80.0
	// ProximityDepthFt vertical window for proximity kill.
	ProximityDepthFt = 90.0
	// BlastDeafRadiusYd — temporary sonar washout from underwater detonation.
	BlastDeafRadiusYd = 2500.0
	// BlastDeafDurationSec how long listeners stay acoustically blinded.
	BlastDeafDurationSec = 50.0
	// TubeReloadSec — compressed from published 10–20 min tube reload (~15 min).
	TubeReloadSec = 180.0
	// PlayerMagazineCapacity — Los Angeles-class weapons-space order of magnitude (Mk48-focused).
	PlayerMagazineCapacity = 22
	// EnemySubMagazine — diesel/SSK typical loadout order of magnitude.
	EnemySubMagazine = 14
	// TubeClearYd — straight run along ownship heading before gyro turn is enabled.
	TubeClearYd = 180.0
	// SearchArmMinDistYd — minimum slant range from launcher before deferred seeker arms.
	SearchArmMinDistYd = 450.0
	// TorpedoActivePingIntervalSec is the search-mode seeker ping cadence.
	TorpedoActivePingIntervalSec = 3.0
	// TorpedoActivePingPower is quieter than a ship sonar but very audible nearby.
	TorpedoActivePingPower = 0.38
	// SeekLayerMinRangeFactor — floor on acquire range when layers heavily attenuate
	// (thermocline makes lock harder, not impossible at short range).
	SeekLayerMinRangeFactor = 0.30
)

// LayerAttenFunc returns one-way acoustic loss in dB between two depths (0 = same layer).
// Passed from the sim so weapons need not import acoustics (avoids an import cycle).
type LayerAttenFunc func(srcDepthFt, dstDepthFt float64) float64

type TubeState int

const (
	TubeEmpty TubeState = iota
	TubeLoaded
	TubeDoorOpen
	TubeFired
	TubeReloading
)

type Tube struct {
	Number          int
	State           TubeState
	TorpedoType     string
	WireIntact      bool
	TorpedoID       string  // fish launched from this tube (wire link)
	ReloadEnds      float64 // GameTime when reload finishes; 0 = idle
	ReloadOrdnance  string  // type being loaded during TubeReloading
	LastOrdnance    string  // type in tube before last reload cycle (default after fire)
	TargetContactID string  // sonar contact ID (e.g. S3) assigned on WEPS
}

type TorpedoMode int

const (
	ModeWire TorpedoMode = iota
	ModeSearch
)

type TorpedoTerminalMode int

const (
	TerminalExplode TorpedoTerminalMode = iota
	TerminalSignal
	TerminalSilent
)

// Torpedo is a weapon in the water.
type Torpedo struct {
	ID           string
	ParentSubID  string
	TubeNumber   int
	TargetID     string
	Side         world.Side
	X, Y         float64
	DepthFt      float64
	HeadingDeg   float64
	SpeedKts     float64
	RunDepthFt   float64
	OrderedHead  float64 // wire-guidance ordered course (after tube clear)
	SeekerOn     bool
	WireCut      bool
	Armed        bool
	Mode         TorpedoMode
	Alive        bool
	Age          float64
	LastPingTime float64
	CruiseKts    float64 // ordered run speed; SpeedKts ramps toward this

	// Tube-exit: fish runs LaunchHeadDeg until ClearDistYd >= TubeClearYd, then
	// steers to GyroCourseDeg (and may enable deferred ModeSearch).
	LaunchHeadDeg          float64
	GyroCourseDeg          float64
	ClearDistYd            float64
	EnableSearchAfterClear bool
	gyroEnabled            bool // true once tube-clear steering has been applied

	// Anti-CM: temporary blacklist of seduced decoys + lock clock for verification.
	RejectedUntil map[string]float64
	CMLockID      string
	CMLockSince   float64

	Class       WeaponClass // heavy vs UMGT-1 lightweight
	AcousticSig string      // optional override (e.g. set40 vs umgt1)
	OrdnanceType string
	TerminalMode TorpedoTerminalMode
	DisableSearch bool
}

// EscalatesDefcon reports whether this fish should raise combat alert levels.
// Exercise / signal-only and silent decoy rounds are audible but must not
// trigger weapons-free ROE.
func (t *Torpedo) EscalatesDefcon() bool {
	if t == nil {
		return false
	}
	return t.TerminalMode == TerminalExplode
}

// Detonation describes a warhead event for the sim (blast, deaf, sinking).
type Detonation struct {
	X, Y         float64
	DepthFt      float64
	Hit          *world.Entity // may be nil if scuttled without contact
	SelfKill     bool          // intentional self-destruct — no blast/deaf
	Grounded     bool          // struck coastline / seafloor (warhead cooks off)
	Harpoon      bool          // anti-ship missile warhead (surface)
	Intercepted  bool          // SAM/CIWS killed the missile
	Debris       bool          // close-in intercept — fragment damage to Hit
	RBU          bool          // RBU pattern splash — light ASW shock damage
	LightWarhead bool          // UMGT-1 / SET-40 — reduced hull damage vs heavy fish
	ShooterID    string
	// Accident: onboard explosion (debug) — full blast FX/damage, no DEFCON escalate.
	Accident bool
	SignalOnly   bool
	SignalLevel  float64
	SignalFreqHz float64
	SignalDurSec float64
	SignalLabel  string
}

// FireControl manages 688-style torpedo firing.
type FireControl struct {
	Tubes            [4]Tube
	SelectedTube     int
	GyroAngleDeg     float64 // absolute ordered launch course, 0–360° true
	RunDepthFt       float64
	SpeedSetting     string // "LOW", "HIGH"
	SeekerEnabled    bool
	MagazineLeft     int // Mk48 rounds in weapons space (not in tubes)
	HarpoonMagLeft   int
	HarpoonRadarBeam    string // WIDE / NARROW
	HarpoonRadarRange   string // MIN / SHORT / MEDIUM / LONG
	HarpoonDestructRange string // MEDIUM / LONG / MAX
	EnemyMagazine    map[string]int // hostile sub heavy fish
	EnemyASCMMag     map[string]int // hostile sub Klub/Oniks/Kalibr
	EnemyASCMLastAt  map[string]float64 // gameTime of last hostile ASCM launch (cooldown)
	AllyHarpoonMag   map[string]int // allied 688 Sub-Harpoon rounds
	EnemyRastrub     map[string]int // URPK-5 Rastrub ASW rockets
	EnemyShipTube    map[string]int // ship torpedo tubes (UMGT-1 / SET-40)
	EnemyExerciseTube map[string]int // exercise signal fish (practice hulks)
	EnemyRBU         map[string]int // RBU-6000 salvos (Grisha)
	EnemySAM         map[string]int     // Kinzhal / Osa-M rounds
	EnemyCIWS        map[string]int     // AK-630 burst magazine
	EnemyPDEngageAt  map[string]float64 // last point-defense engagement time
	EnemyTubeOpenAt  map[string]float64
	ActiveTorpedoes  []*Torpedo
	ActiveHarpoons   []*HarpoonMissile
	ActiveRastrub    []*RastrubFlight
	ActiveRBU        []*RBUSalvo
	DebugMapFlashes  []DebugMapFlash // short-lived WEPS debug letters (SAM/CIWS, …)
	torpedoSeq       int
}

// DebugMapFlash is a brief letter marker for the WEPS debug overlay.
type DebugMapFlash struct {
	X, Y  float64
	Label string
	Until float64 // game time when the marker expires
}

const DebugMapFlashSec = 10.0

func (fc *FireControl) PushDebugMapFlash(x, y float64, label string, gameTime float64) {
	if fc == nil || label == "" {
		return
	}
	fc.DebugMapFlashes = append(fc.DebugMapFlashes, DebugMapFlash{
		X: x, Y: y, Label: label, Until: gameTime + DebugMapFlashSec,
	})
}

func (fc *FireControl) PruneDebugMapFlashes(gameTime float64) {
	if fc == nil || len(fc.DebugMapFlashes) == 0 {
		return
	}
	dst := fc.DebugMapFlashes[:0]
	for _, f := range fc.DebugMapFlashes {
		if f.Until > gameTime {
			dst = append(dst, f)
		}
	}
	fc.DebugMapFlashes = dst
}

func NewFireControl() FireControl {
	fc := FireControl{
		SelectedTube:         1,
		GyroAngleDeg:         0,
		RunDepthFt:           400,
		SpeedSetting:         "HIGH",
		SeekerEnabled:        false,
		MagazineLeft:         PlayerMagazineCapacity - 2, // tubes 1–2 Mk48
		HarpoonMagLeft:       PlayerHarpoonMagazine - 2,  // tubes 3–4 Harpoon
		HarpoonRadarBeam:     HarpoonBeamWide,
		HarpoonRadarRange:    HarpoonSRCHMedium,
		HarpoonDestructRange: HarpoonDSTRLong,
		EnemyMagazine:        map[string]int{},
		EnemyASCMMag:         map[string]int{},
		EnemyASCMLastAt:      map[string]float64{},
		AllyHarpoonMag:       map[string]int{},
		EnemyRastrub:         map[string]int{},
		EnemyShipTube:        map[string]int{},
		EnemyExerciseTube:    map[string]int{},
		EnemyRBU:             map[string]int{},
		EnemySAM:             map[string]int{},
		EnemyCIWS:            map[string]int{},
		EnemyPDEngageAt:      map[string]float64{},
		EnemyTubeOpenAt:      map[string]float64{},
	}
	for i := range fc.Tubes {
		ord := OrdnanceMk48
		if i >= 2 {
			ord = OrdnanceHarpoon
		}
		fc.Tubes[i] = Tube{
			Number:      i + 1,
			State:       TubeLoaded,
			TorpedoType: ord,
		}
	}
	return fc
}

func (fc *FireControl) SelectTube(n int) {
	if n >= 1 && n <= 4 {
		fc.SelectedTube = n
	}
}

func (fc *FireControl) Selected() *Tube {
	return &fc.Tubes[fc.SelectedTube-1]
}

func (fc *FireControl) TubeByNumber(n int) *Tube {
	if n < 1 || n > 4 {
		return nil
	}
	return &fc.Tubes[n-1]
}

// SetTubeTargetContact assigns a sonar contact ID to a tube's target (WEPS).
func (fc *FireControl) SetTubeTargetContact(tubeNum int, contactID string) {
	t := fc.TubeByNumber(tubeNum)
	if t != nil {
		t.TargetContactID = contactID
	}
}

// OpenOuterDoor opens the selected (or numbered) tube. Cannot open while reloading.
func (fc *FireControl) OpenOuterDoor(tubeNum int) bool {
	t := fc.TubeByNumber(tubeNum)
	if t == nil {
		t = fc.Selected()
	}
	if t.State != TubeLoaded {
		return false
	}
	t.State = TubeDoorOpen
	return true
}

// CloseOuterDoor closes the tube. After a shot: cuts wire and starts autoload.
// If the door was open but unfired, returns to loaded (no magazine consume).
func (fc *FireControl) CloseOuterDoor(tubeNum int, gameTime float64) bool {
	t := fc.TubeByNumber(tubeNum)
	if t == nil {
		t = fc.Selected()
	}
	switch t.State {
	case TubeDoorOpen:
		// Unfired — just shut the door; weapon remains chambered.
		t.State = TubeLoaded
		return true
	case TubeFired:
		// Forced wire cut when shutting the outer door after launch.
		if t.TorpedoID != "" {
			if fish := fc.TorpedoByID(t.TorpedoID); fish != nil {
				fc.cutWireTorpedo(fish)
			}
		}
		t.WireIntact = false
		t.TorpedoID = ""
		fc.beginReload(t, gameTime)
		return true
	default:
		return false
	}
}

func (fc *FireControl) ordnanceMagLeft(ordnance string) int {
	switch {
	case OrdnanceUsesHarpoonMagazine(ordnance):
		return fc.HarpoonMagLeft
	default:
		return fc.MagazineLeft
	}
}

func (fc *FireControl) returnOrdnanceToMag(ordnance string) {
	switch {
	case OrdnanceUsesHarpoonMagazine(ordnance):
		fc.HarpoonMagLeft++
	default:
		fc.MagazineLeft++
	}
}

func (fc *FireControl) consumeOrdnance(ordnance string) bool {
	switch {
	case OrdnanceUsesHarpoonMagazine(ordnance):
		if fc.HarpoonMagLeft <= 0 {
			return false
		}
		fc.HarpoonMagLeft--
		return true
	default:
		if fc.MagazineLeft <= 0 {
			return false
		}
		fc.MagazineLeft--
		return true
	}
}

func (fc *FireControl) startOrdnanceReload(t *Tube, ordnance string, gameTime float64) {
	ordnance = normalizeOrdnance(ordnance)
	if !fc.consumeOrdnance(ordnance) {
		t.State = TubeEmpty
		t.ReloadEnds = 0
		t.TorpedoType = ""
		t.ReloadOrdnance = ""
		return
	}
	t.State = TubeReloading
	t.ReloadEnds = gameTime + TubeReloadSec
	t.ReloadOrdnance = ordnance
	t.TorpedoType = ""
	t.WireIntact = false
	t.TorpedoID = ""
}

// RequestOrdnanceReload starts or retargets a tube reload. Returns false if blocked or no-op.
func (fc *FireControl) RequestOrdnanceReload(tubeNum int, ordnance string, gameTime float64) bool {
	t := fc.TubeByNumber(tubeNum)
	if t == nil {
		return false
	}
	if t.State == TubeDoorOpen || t.State == TubeFired {
		return false
	}
	ordnance = normalizeOrdnance(ordnance)

	switch t.State {
	case TubeLoaded:
		cur := normalizeOrdnance(t.TorpedoType)
		if cur == ordnance {
			return false
		}
		t.LastOrdnance = cur
		fc.returnOrdnanceToMag(cur)
		fc.startOrdnanceReload(t, ordnance, gameTime)
		return true
	case TubeReloading:
		if normalizeOrdnance(t.ReloadOrdnance) == ordnance {
			return false
		}
		fc.returnOrdnanceToMag(t.ReloadOrdnance)
		fc.startOrdnanceReload(t, ordnance, gameTime)
		return true
	case TubeEmpty:
		if fc.ordnanceMagLeft(ordnance) <= 0 {
			return false
		}
		if t.TorpedoType != "" {
			t.LastOrdnance = normalizeOrdnance(t.TorpedoType)
		} else if t.LastOrdnance == "" {
			t.LastOrdnance = OrdnanceMk48
		}
		fc.startOrdnanceReload(t, ordnance, gameTime)
		return true
	default:
		return false
	}
}

func (fc *FireControl) beginReload(t *Tube, gameTime float64) {
	reloadType := normalizeOrdnance(t.TorpedoType)
	if reloadType == "" {
		reloadType = OrdnanceMk48
	}
	t.LastOrdnance = reloadType
	fc.startOrdnanceReload(t, reloadType, gameTime)
}

// UpdateTubes advances reload timers.
func (fc *FireControl) UpdateTubes(gameTime float64) {
	for i := range fc.Tubes {
		t := &fc.Tubes[i]
		if t.State == TubeReloading && t.ReloadEnds > 0 && gameTime >= t.ReloadEnds {
			t.State = TubeLoaded
			t.TorpedoType = normalizeOrdnance(t.ReloadOrdnance)
			if t.TorpedoType == "" {
				t.TorpedoType = OrdnanceMk48
			}
			t.ReloadOrdnance = ""
			t.ReloadEnds = 0
		}
	}
}

func (fc *FireControl) ReloadRemaining(t Tube, gameTime float64) float64 {
	if t.State != TubeReloading || t.ReloadEnds <= gameTime {
		return 0
	}
	return t.ReloadEnds - gameTime
}

// Shoot requires outer door open. Cannot fire while reloading/empty/closed.
func (fc *FireControl) Shoot(sub *world.Entity, tubeNum int) *Torpedo {
	t := fc.TubeByNumber(tubeNum)
	if t == nil {
		t = fc.Selected()
	}
	if t.State != TubeDoorOpen {
		return nil
	}
	ordnance := normalizeOrdnance(t.TorpedoType)
	if !OrdnanceIsTorpedo(ordnance) {
		return nil
	}
	if sub != nil {
		sub.EnsureDamage()
		sys := world.TubeSys(t.Number)
		if sys != world.SysNone && !sub.Damage.Operational(sys) {
			return nil
		}
	}
	fc.torpedoSeq++
	t.State = TubeFired
	t.WireIntact = true

	// Exit along ownship heading; gyro course is applied only after TubeClearYd.
	ownHead := normalizeAngle(sub.HeadingDeg)
	gyro := normalizeAngle(fc.GyroAngleDeg)
	rad := ownHead * math.Pi / 180
	offset := 40.0
	// Tube exit speed, then accelerate to cruise (Mk48 reaches speed in ~10–20 s).
	exitKts := 18.0
	cruise := speedKts(fc.SpeedSetting)
	idPrefix := "MK48"
	terminal := TerminalExplode
	if ordnance == OrdnanceMk48Exercise {
		idPrefix = "MK48X"
		terminal = TerminalSignal
	}
	torp := &Torpedo{
		ID:                     fmt.Sprintf("%s-%d", idPrefix, fc.torpedoSeq),
		ParentSubID:            sub.ID,
		TubeNumber:             t.Number,
		Side:                   sub.Side,
		X:                      sub.X + math.Sin(rad)*offset,
		Y:                      sub.Y + math.Cos(rad)*offset,
		DepthFt:                sub.DepthFt,
		HeadingDeg:             ownHead,
		OrderedHead:            ownHead,
		LaunchHeadDeg:          ownHead,
		GyroCourseDeg:          gyro,
		SpeedKts:               exitKts,
		CruiseKts:              cruise,
		RunDepthFt:             fc.RunDepthFt,
		SeekerOn:               false,
		Armed:                  true,
		Mode:                   ModeWire,
		Alive:                  true,
		LastPingTime:           -1,
		EnableSearchAfterClear: fc.SeekerEnabled,
		OrdnanceType:           ordnance,
		TerminalMode:           terminal,
	}
	t.TorpedoID = torp.ID
	fc.ActiveTorpedoes = append(fc.ActiveTorpedoes, torp)
	return torp
}

// SpawnHostileTorpedo launches an AI weapon (wire initially, then search).
// Gyro / depth come from the firing crew's Track estimate (skill-dependent).
func (fc *FireControl) SpawnHostileTorpedo(sub, target *world.Entity) *Torpedo {
	if sub == nil || !sub.Alive() || target == nil {
		return nil
	}
	left := fc.enemyAmmo(sub)
	if left <= 0 {
		return nil
	}
	fc.EnemyMagazine[sub.ID] = left - 1
	fc.torpedoSeq++
	ownHead := normalizeAngle(sub.HeadingDeg)
	aim := target
	if sub.Track.Valid {
		aim = sub.Track.GhostTarget(target.ID, target.Side)
	}
	skill := sub.CrewSkill01()
	cruise := HostileTorpedoCruiseKts(sub.SignatureID)
	gyro := sub.BearingDegTo(aim)
	if course, ok := TorpedoInterceptGyro(sub.X, sub.Y, ownHead, aim.X, aim.Y, aim.HeadingDeg, aim.SpeedKts, cruise); ok {
		gyro = course
	}
	// Green fire-control: smear gyro and depth badly.
	gyro += (1-skill) * 28 * (hashUnit(sub.ID+aim.ID) - 0.5) * 2
	gyro = normalizeAngle(gyro)
	runDepth := math.Max(80, aim.DepthFt)
	runDepth += (1 - skill) * 90 * (hashUnit(sub.ID+"depth") - 0.5) * 2
	if runDepth < 60 {
		runDepth = 60
	}
	rad := ownHead * math.Pi / 180
	torp := &Torpedo{
		ID:                     fmt.Sprintf("ETORP-%d", fc.torpedoSeq),
		ParentSubID:            sub.ID,
		TargetID:               target.ID,
		Side:                   sub.Side,
		X:                      sub.X + math.Sin(rad)*60,
		Y:                      sub.Y + math.Cos(rad)*60,
		DepthFt:                sub.DepthFt,
		HeadingDeg:             ownHead,
		OrderedHead:            ownHead,
		LaunchHeadDeg:          ownHead,
		GyroCourseDeg:          gyro,
		SpeedKts:               18,
		CruiseKts:              cruise,
		RunDepthFt:             runDepth,
		SeekerOn:               false,
		Armed:                  true,
		Mode:                   ModeWire,
		Alive:                  true,
		LastPingTime:           -1,
		EnableSearchAfterClear: false, // wire-guide to target after clear; seek later
		OrdnanceType:           OrdnanceMk48,
		TerminalMode:           TerminalExplode,
	}
	if sub.TorpedoVariant == EnemyOrdnanceSSN688Decoy {
		torp.ID = fmt.Sprintf("EDECOY-%d", fc.torpedoSeq)
		torp.OrdnanceType = EnemyOrdnanceSSN688Decoy
		torp.TerminalMode = TerminalSilent
		torp.DisableSearch = true
		torp.Armed = false
		torp.AcousticSig = "ssn688_decoy"
	}
	fc.ActiveTorpedoes = append(fc.ActiveTorpedoes, torp)
	return torp
}

func hashUnit(s string) float64 {
	h := 0
	for i := 0; i < len(s); i++ {
		h = h*31 + int(s[i])
	}
	if h < 0 {
		h = -h
	}
	return float64(h%1000) / 1000.0
}

func (fc *FireControl) enemyAmmo(sub *world.Entity) int {
	if sub == nil {
		return 0
	}
	if fc.EnemyMagazine == nil {
		fc.EnemyMagazine = map[string]int{}
	}
	if v, ok := fc.EnemyMagazine[sub.ID]; ok {
		return v
	}
	n := EnemySubMagazineFor(sub.SignatureID)
	fc.EnemyMagazine[sub.ID] = n
	return n
}

func (fc *FireControl) HasRecentShotFrom(subID string, maxAge float64) bool {
	for _, t := range fc.ActiveTorpedoes {
		if t != nil && t.Alive && t.ParentSubID == subID && t.Age < maxAge {
			return true
		}
	}
	return false
}

func (fc *FireControl) SetTorpedoSeq(n int) {
	if n > fc.torpedoSeq {
		fc.torpedoSeq = n
	}
}

func (fc *FireControl) TorpedoSeq() int {
	return fc.torpedoSeq
}

// MarkGyroEnabled restores post-tube-clear steering state after a save load.
func (t *Torpedo) MarkGyroEnabled(enabled bool) {
	if t == nil {
		return
	}
	t.gyroEnabled = enabled
}

func (t *Torpedo) GyroEnabled() bool {
	return t != nil && t.gyroEnabled
}

func (fc *FireControl) TorpedoByID(id string) *Torpedo {
	for _, t := range fc.ActiveTorpedoes {
		if t != nil && t.ID == id {
			return t
		}
	}
	return nil
}

func (fc *FireControl) TorpedoForTube(tubeNum int) *Torpedo {
	t := fc.TubeByNumber(tubeNum)
	if t == nil || t.TorpedoID == "" {
		return nil
	}
	fish := fc.TorpedoByID(t.TorpedoID)
	if fish == nil || !fish.Alive {
		return nil
	}
	return fish
}

func (fc *FireControl) cutWireTorpedo(torp *Torpedo) {
	if torp == nil {
		return
	}
	// Already autonomous (manual CUT, door close, or age expiry). Closing the
	// tube calls this again — must not reset ModeSearch / OrderedHead.
	if torp.WireCut {
		for i := range fc.Tubes {
			if fc.Tubes[i].TorpedoID == torp.ID {
				fc.Tubes[i].WireIntact = false
			}
		}
		return
	}
	torp.WireCut = true
	if torp.TubeCleared() {
		torp.EnableSearchAfterClear = true
		torp.Mode = ModeWire
		torp.SeekerOn = false
		if !torp.gyroEnabled {
			torp.enableGyroAfterClear()
		} else {
			torp.OrderedHead = torp.GyroCourseDeg
		}
	} else {
		// Finish tube-clear run, then go autonomous.
		torp.EnableSearchAfterClear = true
		torp.Mode = ModeWire
		torp.SeekerOn = false
		torp.OrderedHead = torp.LaunchHeadDeg
	}
	for i := range fc.Tubes {
		if fc.Tubes[i].TorpedoID == torp.ID {
			fc.Tubes[i].WireIntact = false
		}
	}
}

func (fc *FireControl) CutWire(torp *Torpedo) {
	fc.cutWireTorpedo(torp)
}

// SelfDestruct scuttles a fish without blast/deaf (safe abort).
// Requires an intact wire — after CUT the fish is autonomous and cannot be aborted remotely.
func (fc *FireControl) SelfDestruct(torp *Torpedo) *Detonation {
	if torp == nil || !torp.Alive || torp.WireCut {
		return nil
	}
	d := &Detonation{
		X: torp.X, Y: torp.Y, DepthFt: torp.DepthFt,
		SelfKill: true, ShooterID: torp.ParentSubID,
	}
	torp.Alive = false
	fc.unlinkTube(torp)
	return d
}

func (fc *FireControl) unlinkTube(torp *Torpedo) {
	for i := range fc.Tubes {
		if fc.Tubes[i].TorpedoID == torp.ID {
			fc.Tubes[i].TorpedoID = ""
			fc.Tubes[i].WireIntact = false
		}
	}
}

func (fc *FireControl) UnlinkDead(torp *Torpedo) {
	if torp == nil {
		return
	}
	fc.unlinkTube(torp)
}

// OnPlatformLost cuts wires for all fish from a destroyed launcher.
func (fc *FireControl) OnPlatformLost(subID string) {
	for _, t := range fc.ActiveTorpedoes {
		if t != nil && t.Alive && t.ParentSubID == subID && !t.WireCut {
			fc.cutWireTorpedo(t)
		}
	}
}

func (fc *FireControl) EnableSeeker(torp *Torpedo) {
	if torp == nil {
		return
	}
	if !torp.TubeCleared() {
		torp.EnableSearchAfterClear = true
		return
	}
	torp.EnableSearchAfterClear = true
}

// ToggleSeeker turns ModeSearch on/off while the wire is intact.
// With a cut wire the seeker can only be enabled (autonomous fish).
// Before tube-clear, only the deferred EnableSearchAfterClear flag is toggled.
func (fc *FireControl) ToggleSeeker(torp *Torpedo) {
	if torp == nil || !torp.Alive {
		return
	}
	if !torp.TubeCleared() {
		torp.EnableSearchAfterClear = !torp.EnableSearchAfterClear
		return
	}
	if torp.Mode == ModeSearch || torp.SeekerOn || torp.EnableSearchAfterClear {
		if torp.WireCut && (torp.Mode == ModeSearch || torp.SeekerOn) {
			return // autonomous — cannot cancel search without a wire
		}
		torp.SeekerOn = false
		torp.EnableSearchAfterClear = false
		torp.Mode = ModeWire
		torp.OrderedHead = torp.HeadingDeg
		torp.GyroCourseDeg = torp.HeadingDeg
		return
	}
	torp.EnableSearchAfterClear = true
	torp.Mode = ModeSearch
	torp.SeekerOn = true
}

func (fc *FireControl) WireSteer(torp *Torpedo, deltaHead, deltaDepth float64) {
	if torp == nil || !torp.Alive || torp.WireCut {
		return
	}
	// Steering while seeker is on drops back to wire guidance.
	if torp.Mode == ModeSearch || torp.SeekerOn {
		torp.SeekerOn = false
		torp.EnableSearchAfterClear = false
		torp.Mode = ModeWire
	}
	if !torp.TubeCleared() {
		// Gyro order updates, but fish keeps running tube-exit heading until clear.
		torp.GyroCourseDeg = normalizeAngle(torp.GyroCourseDeg + deltaHead)
	} else {
		torp.OrderedHead = normalizeAngle(torp.OrderedHead + deltaHead)
		torp.GyroCourseDeg = torp.OrderedHead
	}
	torp.RunDepthFt = clamp(torp.RunDepthFt+deltaDepth, 40, 1500)
}

// WireSetCourse sets an absolute gyro / ordered course while wire guidance is active.
// No-op if the fish is in ModeSearch (caller should leave seeker alone).
func (fc *FireControl) WireSetCourse(torp *Torpedo, courseDeg float64) {
	if torp == nil || !torp.Alive || torp.WireCut {
		return
	}
	if torp.Mode == ModeSearch || torp.SeekerOn {
		return
	}
	courseDeg = normalizeAngle(courseDeg)
	torp.GyroCourseDeg = courseDeg
	if torp.TubeCleared() {
		torp.OrderedHead = courseDeg
	}
}

// TubeCleared reports whether the fish has run far enough to enable gyro turn.
func (t *Torpedo) TubeCleared() bool {
	if t == nil {
		return true
	}
	need := TubeClearYd
	if t.Class == ClassUMGT1 {
		need = UMGT1TubeClearYd
	}
	return t.ClearDistYd >= need
}

func (t *Torpedo) enableGyroAfterClear() {
	if t == nil || t.gyroEnabled {
		return
	}
	t.gyroEnabled = true
	t.OrderedHead = t.GyroCourseDeg
}

func (t *Torpedo) pendingSearchArm() bool {
	return t != nil && t.EnableSearchAfterClear && !t.SeekerOn && t.Mode != ModeSearch
}

func (t *Torpedo) distToParent(targets []*world.Entity) float64 {
	if t == nil || t.ParentSubID == "" {
		return -1
	}
	for _, tgt := range targets {
		if tgt != nil && tgt.ID == t.ParentSubID && tgt.Alive() {
			return math.Hypot(tgt.X-t.X, tgt.Y-t.Y)
		}
	}
	return -1
}

func (t *Torpedo) tryArmSearch(targets []*world.Entity) {
	if t == nil || t.DisableSearch {
		return
	}
	if !t.pendingSearchArm() || !t.TubeCleared() {
		return
	}
	if !t.gyroEnabled {
		t.enableGyroAfterClear()
	}
	if d := t.distToParent(targets); d >= 0 && d < SearchArmMinDistYd {
		return
	}
	t.Mode = ModeSearch
	t.SeekerOn = true
}

// AcousticEntity fills dst with a world.Entity view for sonar detection.
func (t *Torpedo) AcousticEntity(dst *world.Entity) *world.Entity {
	if t == nil || dst == nil {
		return nil
	}
	sig := "mk48"
	switch {
	case t.AcousticSig != "":
		sig = t.AcousticSig
	case t.Class == ClassUMGT1:
		sig = "umgt1"
	case t.Side == world.SideEnemy:
		sig = "type53"
	}
	*dst = world.Entity{
		ID: t.ID, Name: t.ID, Kind: world.KindTorpedo, Side: t.Side,
		Status: world.StatusActive, SignatureID: sig,
		X: t.X, Y: t.Y, DepthFt: t.DepthFt, HeadingDeg: t.HeadingDeg,
		SpeedKts: t.SpeedKts, OrderedSpeed: t.CruiseKts,
		OrderedDepth: t.RunDepthFt, OrderedHead: t.OrderedHead, LengthFt: 19,
		LastPingTime: t.LastPingTime, LastPingPower: TorpedoActivePingPower,
	}
	return dst
}

func speedKts(s string) float64 {
	if s == "LOW" {
		return 28
	}
	return 55
}

// Advance moves the torpedo. Returns a detonation when the warhead goes off.
// layerAtten (optional) models thermocline / layer loss for active seeker acquisition.
// bathy (optional) triggers grounding detonation on coastline / insufficient depth.
func (t *Torpedo) Advance(dt, gameTime float64, targets []*world.Entity, layerAtten LayerAttenFunc, bathy *world.Bathymetry) *Detonation {
	if !t.Alive {
		return nil
	}
	t.Age += dt
	t.DepthFt += (t.RunDepthFt - t.DepthFt) * dt * 0.5
	if t.CruiseKts > 0 {
		// Ramp to cruise; ~6 kts/s keeps tube-exit → ordered speed under ~20 s.
		const accel = 6.0
		ds := t.CruiseKts - t.SpeedKts
		t.SpeedKts += clamp(ds, -accel*dt, accel*dt)
	}

	wasCleared := t.TubeCleared()
	t.ClearDistYd += t.SpeedKts * world.KnotsToYPS * dt
	if !wasCleared && t.TubeCleared() {
		t.enableGyroAfterClear()
	}

	if t.Mode == ModeWire && !t.WireCut && t.Age > 120 {
		t.WireCut = true
		t.EnableSearchAfterClear = !t.DisableSearch
	}

	// Tube-exit: hold launch heading until clear distance is reached.
	if !t.TubeCleared() {
		t.OrderedHead = t.LaunchHeadDeg
		t.Mode = ModeWire
		t.SeekerOn = false
	}

	// Wire guidance: follow OrderedHead while wire intact, or while running out on gyro before seeker arms.
	if t.Mode == ModeWire && (!t.WireCut || t.pendingSearchArm()) {
		diff := shortestAngleDiff(t.HeadingDeg, t.OrderedHead)
		t.HeadingDeg += clamp(diff*dt*1.2, -dt*10, dt*10)
		t.HeadingDeg = normalizeAngle(t.HeadingDeg)
	} else if t.Mode == ModeWire && t.WireCut && !t.TubeCleared() {
		// Wire already cut but still clearing the tube — keep going straight.
		diff := shortestAngleDiff(t.HeadingDeg, t.LaunchHeadDeg)
		t.HeadingDeg += clamp(diff*dt*1.2, -dt*10, dt*10)
		t.HeadingDeg = normalizeAngle(t.HeadingDeg)
	}

	t.tryArmSearch(targets)

	// Active search: acquire ships/subs OR soft-kill decoys; anti-CM can reject decoys.
	if t.Mode == ModeSearch {
		if t.LastPingTime < 0 || gameTime-t.LastPingTime >= TorpedoActivePingIntervalSec {
			t.LastPingTime = gameTime
		}
		if best := t.acquireInCone(targets, layerAtten, gameTime); best != nil {
			desired := bearing(t.X, t.Y, best.X, best.Y)
			diff := shortestAngleDiff(t.HeadingDeg, desired)
			t.HeadingDeg += clamp(diff*dt*0.9, -dt*10, dt*10)
			t.HeadingDeg = normalizeAngle(t.HeadingDeg)
			if best.Kind == world.KindCountermeasure {
				if t.CMLockID != best.ID {
					t.CMLockID = best.ID
					t.CMLockSince = gameTime
				}
			} else {
				t.CMLockID = ""
				t.CMLockSince = 0
			}
			t.TargetID = best.ID
			t.RunDepthFt += (best.DepthFt - t.RunDepthFt) * dt * 0.35
		} else {
			t.CMLockID = ""
			t.CMLockSince = 0
		}
	}

	rad := t.HeadingDeg * math.Pi / 180
	yps := t.SpeedKts * world.KnotsToYPS
	prevX, prevY := t.X, t.Y
	t.X += math.Sin(rad) * yps * dt
	t.Y += math.Cos(rad) * yps * dt

	if det := t.checkGrounding(bathy, prevX, prevY); det != nil {
		return det
	}

	// Influence fuse: warshots while searching; once tube-cleared, also on wire
	// (blue-on-blue / exercise steer-backs). Enemy fish only fuse on player side.
	fuseActive := t.Armed && t.Age > 2 && t.Mode == ModeSearch
	if t.Armed && t.Age > 2 && t.TubeCleared() && t.TerminalMode != TerminalSilent {
		fuseActive = true
	}
	if fuseActive {
		proxYd := ProximityKillYd
		if t.Class == ClassUMGT1 {
			proxYd = UMGT1ProximityYd
		}
		for _, tgt := range targets {
			if !t.validFuseTarget(tgt) {
				continue
			}
			d := math.Hypot(tgt.X-t.X, tgt.Y-t.Y)
			if d > proxYd {
				continue
			}
			depthDiff := math.Abs(tgt.DepthFt - t.DepthFt)
			// Surface ships: under-keel influence if fish is shallow enough.
			hitOK := depthDiff <= ProximityDepthFt
			if tgt.Kind == world.KindSurfaceShip && t.DepthFt <= 200 {
				hitOK = true
			}
			if hitOK {
				t.Alive = false
				return t.terminalDetonation(tgt)
			}
		}
	}
	maxAge := 600.0
	if t.Class == ClassUMGT1 {
		maxAge = UMGT1MaxAgeSec
	}
	if t.Age > maxAge {
		t.Alive = false
	}
	return nil
}

// checkGrounding detonates if the move segment crosses land or the fish depth
// meets/exceeds local seafloor (runs into the shelf / beach).
func (t *Torpedo) checkGrounding(bathy *world.Bathymetry, prevX, prevY float64) *Detonation {
	if t == nil || bathy == nil || !bathy.Valid() {
		return nil
	}
	dx, dy := t.X-prevX, t.Y-prevY
	dist := math.Hypot(dx, dy)
	const sampleYd = 40.0
	steps := int(dist/sampleYd) + 1
	for i := 1; i <= steps; i++ {
		f := float64(i) / float64(steps)
		x := prevX + dx*f
		y := prevY + dy*f
		bottom := bathy.DepthAtFt(x, y)
		hitLand := bottom <= 0
		hitBottom := bottom > 0 && t.DepthFt >= bottom
		if !hitLand && !hitBottom {
			continue
		}
		t.X, t.Y = x, y
		t.Alive = false
		d := t.terminalDetonation(nil)
		d.Grounded = true
		return d
	}
	return nil
}

// validFuseTarget reports whether proximity fuse may terminate on tgt.
// Combat/exercise fish may hit ships/subs (including cross-side blue-on-blue)
// except: launching platform, and own-side surface hulls for lightweight ASW
// fish (tube overrun / wake suicide — SET-40/UMGT-1 are not anti-ship).
func (t *Torpedo) validFuseTarget(tgt *world.Entity) bool {
	if t == nil || tgt == nil || !tgt.Alive() {
		return false
	}
	if t.ParentSubID != "" && tgt.ID == t.ParentSubID {
		return false
	}
	if t.Class == ClassUMGT1 && tgt.Kind == world.KindSurfaceShip && tgt.Side == t.Side {
		return false
	}
	return tgt.Kind == world.KindSubmarine || tgt.Kind == world.KindSurfaceShip
}

// validSeekShip reports whether active search may lock a surface/sub target.
// Countermeasures are handled separately. No IFF — acoustic lock on any hull
// except the launching platform.
func (t *Torpedo) validSeekShip(tgt *world.Entity) bool {
	return t.validFuseTarget(tgt)
}

func (t *Torpedo) terminalDetonation(hit *world.Entity) *Detonation {
	if t == nil {
		return nil
	}
	d := &Detonation{
		X: t.X, Y: t.Y, DepthFt: t.DepthFt,
		Hit: hit, ShooterID: t.ParentSubID,
		LightWarhead: t.Class == ClassUMGT1,
	}
	switch t.TerminalMode {
	case TerminalSignal:
		d.Hit = nil
		d.SignalOnly = true
		d.SignalLevel = 58
		d.SignalFreqHz = 1150
		d.SignalDurSec = 5.0
		d.SignalLabel = "exercise_torpedo"
	case TerminalSilent:
		d.Hit = nil
		d.SelfKill = true
	default:
	}
	return d
}

func (t *Torpedo) acquireInCone(targets []*world.Entity, layerAtten LayerAttenFunc, gameTime float64) *world.Entity {
	var best *world.Entity
	bestScore := -1.0
	gullible := t != nil && t.Side == world.SideEnemy
	verifySec := AntiCMVerifySec
	if gullible {
		verifySec = EnemyAntiCMVerifySec
	}

	// Periodic anti-CM: if locked on a decoy long enough, maybe reject it.
	if t.CMLockID != "" && t.CMLockSince > 0 && gameTime-t.CMLockSince >= verifySec {
		t.maybeRejectCM(targets, gameTime)
	}

	for _, tgt := range targets {
		if tgt == nil || !tgt.Alive() {
			continue
		}
		isCM := tgt.Kind == world.KindCountermeasure
		isShip := tgt.Kind == world.KindSubmarine || tgt.Kind == world.KindSurfaceShip
		if !isCM && !isShip {
			continue
		}
		if isShip && !t.validSeekShip(tgt) {
			continue
		}
		// Wire-run never seduces on decoys — only ModeSearch (caller).
		if isCM && t.Mode != ModeSearch {
			continue
		}
		if t.RejectedUntil != nil {
			if until, ok := t.RejectedUntil[tgt.ID]; ok && gameTime < until {
				continue
			}
		}
		d := math.Hypot(tgt.X-t.X, tgt.Y-t.Y)
		maxR, coneHalf := t.seekAcquireLimits(tgt.DepthFt, layerAtten)
		if isCM {
			// Decoys are a bit easier to "hear" at the edge of the cone, but not beyond range.
			maxR *= 1.08
			coneHalf *= 1.05
			if gullible {
				maxR *= 1.12
				coneHalf *= 1.12
			}
		}
		if d < 1 || d > maxR {
			continue
		}
		brg := bearing(t.X, t.Y, tgt.X, tgt.Y)
		if math.Abs(shortestAngleDiff(t.HeadingDeg, brg)) > coneHalf {
			continue
		}
		// Closer = better; decoys get an attractiveness bonus (soft-kill seduction).
		score := (maxR - d) / maxR
		if isCM {
			if tgt.SignatureID == "jitter" {
				if gullible {
					score *= EnemySeekJitterAttractMul
				} else {
					score *= SeekJitterAttractMul
				}
			} else if gullible {
				score *= EnemySeekCMAttractMul
			} else {
				score *= SeekCMAttractMul
			}
			// Hovering ADC looks loud but Doppler-poor — slightly less sticky than Nixie.
			if tgt.SpeedKts < 2 && tgt.SignatureID != "jitter" {
				if gullible {
					score *= 0.98
				} else {
					score *= 0.92
				}
			}
		} else {
			// Prefer real platforms once within half-range (anti-CM bias toward truth).
			if d < maxR*0.45 {
				if gullible {
					score *= EnemyShipCloseBias
				} else {
					score *= 1.25
				}
			}
			// Hostile seekers: prefer the intended quarry / player-side hulls when
			// several contacts sit in the cone (AI aims at friendlies; fuse is still IFF-blind).
			if gullible {
				if t.TargetID != "" && tgt.ID == t.TargetID {
					score *= 1.55
				} else if world.IsFriendly(tgt) {
					score *= 1.35
				} else if tgt.Side == world.SideEnemy {
					score *= 0.55 // other hostiles — possible, but not preferred
				} else {
					score *= 0.75 // neutrals less sticky than friendlies
				}
			}
			// Prior lock on this ship sticks a little.
			if t.TargetID == tgt.ID && t.CMLockID == "" {
				if gullible {
					score *= EnemyPriorTargetBias
				} else {
					score *= 1.12
				}
			}
			// Broadband jammer in cone muddies ship lock quality.
			if gullible {
				score *= EnemyJitterConfuseFactor(targets, t.X, t.Y, t.HeadingDeg, coneHalf, maxR)
			} else {
				score *= JitterConfuseFactor(targets, t.X, t.Y, t.HeadingDeg, coneHalf, maxR)
			}
		}
		if score > bestScore {
			bestScore = score
			best = tgt
		}
	}
	return best
}

func (t *Torpedo) maybeRejectCM(targets []*world.Entity, gameTime float64) {
	if t == nil || t.CMLockID == "" {
		return
	}
	gullible := t.Side == world.SideEnemy
	verifySec := AntiCMVerifySec
	rejectHold := AntiCMRejectHoldSec
	if gullible {
		verifySec = EnemyAntiCMVerifySec
		rejectHold = EnemyAntiCMRejectHoldSec
	}
	var decoy *world.Entity
	var realInCone int
	var decoyDist, closestReal float64
	decoyDist, closestReal = 1e12, 1e12
	for _, tgt := range targets {
		if tgt == nil || !tgt.Alive() {
			continue
		}
		if tgt.ID == t.CMLockID {
			decoy = tgt
			decoyDist = math.Hypot(tgt.X-t.X, tgt.Y-t.Y)
		}
		if tgt.Kind != world.KindSubmarine && tgt.Kind != world.KindSurfaceShip {
			continue
		}
		d := math.Hypot(tgt.X-t.X, tgt.Y-t.Y)
		if d > SeekAcquireRangeYd {
			continue
		}
		brg := bearing(t.X, t.Y, tgt.X, tgt.Y)
		if math.Abs(shortestAngleDiff(t.HeadingDeg, brg)) <= SeekConeHalfAngleDeg {
			realInCone++
			if d < closestReal {
				closestReal = d
			}
		}
	}
	if decoy == nil || decoy.Kind != world.KindCountermeasure {
		t.CMLockID = ""
		t.CMLockSince = 0
		return
	}
	reject := false
	if gullible {
		// Hostile fish: stay seduced unless the real hull is clearly closer, or lock is very old.
		lockAge := gameTime - t.CMLockSince
		if realInCone > 0 && closestReal < decoyDist*0.55 {
			reject = true
		}
		if decoy.SignatureID == "adc" && decoy.SpeedKts < 1.2 && lockAge >= verifySec*1.6 {
			reject = true
		}
		if decoy.SignatureID == "nixie" && realInCone == 0 {
			reject = false
			if lockAge >= verifySec*2.8 {
				reject = true
			}
		}
		if decoy.SignatureID == "jitter" {
			// Jitter keeps confusing; only dump after a long hold with a much closer hull.
			if realInCone > 0 && closestReal < decoyDist*0.4 && lockAge >= verifySec*1.3 {
				reject = true
			} else if lockAge >= verifySec*2.0 {
				reject = true
			} else {
				reject = false
			}
		}
		if !reject && lockAge >= verifySec*2.4 {
			reject = true
		}
	} else {
		// Player Mk48: sharper anti-CM discrimination.
		if decoy.SpeedKts < 2.5 {
			reject = true
		}
		if realInCone > 0 {
			reject = true
		}
		if decoy.SignatureID == "nixie" && realInCone == 0 {
			reject = false
			if gameTime-t.CMLockSince >= verifySec*2.2 {
				reject = true
			}
		}
		if decoy.SignatureID == "jitter" && realInCone > 0 {
			reject = true
		}
	}
	if !reject {
		// Keep verifying later.
		t.CMLockSince = gameTime - verifySec*0.35
		return
	}
	if t.RejectedUntil == nil {
		t.RejectedUntil = map[string]float64{}
	}
	t.RejectedUntil[decoy.ID] = gameTime + rejectHold
	t.CMLockID = ""
	t.CMLockSince = 0
	if t.TargetID == decoy.ID {
		t.TargetID = ""
	}
}

// seekAcquireLimits shrinks seeker range/cone when acoustic layers separate fish and target.
func seekAcquireLimits(torpDepthFt, tgtDepthFt float64, layerAtten LayerAttenFunc) (maxRangeYd, coneHalfDeg float64) {
	return seekAcquireLimitsFor(ClassHeavy, torpDepthFt, tgtDepthFt, layerAtten)
}

func (t *Torpedo) seekAcquireLimits(tgtDepthFt float64, layerAtten LayerAttenFunc) (maxRangeYd, coneHalfDeg float64) {
	class := ClassHeavy
	if t != nil {
		class = t.Class
	}
	depth := 0.0
	if t != nil {
		depth = t.DepthFt
	}
	return seekAcquireLimitsFor(class, depth, tgtDepthFt, layerAtten)
}

func seekAcquireLimitsFor(class WeaponClass, torpDepthFt, tgtDepthFt float64, layerAtten LayerAttenFunc) (maxRangeYd, coneHalfDeg float64) {
	maxRangeYd = SeekAcquireRangeYd
	coneHalfDeg = SeekConeHalfAngleDeg
	if class == ClassUMGT1 {
		maxRangeYd = UMGT1SeekRangeYd
		coneHalfDeg = 40
	}
	if layerAtten == nil {
		return maxRangeYd, coneHalfDeg
	}
	loss := layerAtten(torpDepthFt, tgtDepthFt)
	if loss <= 0.5 {
		return maxRangeYd, coneHalfDeg
	}
	// ~16 dB thermocline crossing → ~factor 0.45; heavy column → approaches floor.
	factor := math.Pow(10, -loss/40)
	if factor < SeekLayerMinRangeFactor {
		factor = SeekLayerMinRangeFactor
	}
	maxRangeYd *= factor
	// Narrow the effective cone through the layer (multipath / refraction).
	coneScale := 1 - 0.35*math.Min(1, loss/22)
	if coneScale < 0.55 {
		coneScale = 0.55
	}
	coneHalfDeg *= coneScale
	return maxRangeYd, coneHalfDeg
}

func bearing(x1, y1, x2, y2 float64) float64 {
	deg := math.Atan2(x2-x1, y2-y1) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}

func normalizeAngle(a float64) float64 {
	for a >= 360 {
		a -= 360
	}
	for a < 0 {
		a += 360
	}
	return a
}

func shortestAngleDiff(from, to float64) float64 {
	d := to - from
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// TubeAmmoStatus is the WEPS tube load line: weapon type, with ", wired" while
// guidance wire from that tube is still intact.
func TubeAmmoStatus(t Tube, reloadRemainSec float64) string {
	switch t.State {
	case TubeEmpty:
		return "EMPTY"
	case TubeReloading:
		if reloadRemainSec > 0 {
			name := t.ReloadOrdnance
			if name == "" {
				name = "?"
			}
			return fmt.Sprintf("RELOAD %s %ds", name, int(reloadRemainSec+0.5))
		}
		return "RELOADING"
	default:
		name := t.TorpedoType
		if name == "" {
			name = "Mk48"
		}
		if t.State == TubeFired && t.WireIntact {
			return name + ", wired"
		}
		return name
	}
}
