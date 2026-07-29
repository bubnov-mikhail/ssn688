package weapons

import (
	"fmt"
	"math"

	"github.com/ssn688/sim/internal/world"
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
	Number      int
	State       TubeState
	TorpedoType string
	WireIntact  bool
	TorpedoID   string  // fish launched from this tube (wire link)
	ReloadEnds  float64 // GameTime when reload finishes; 0 = idle
}

type TorpedoMode int

const (
	ModeWire TorpedoMode = iota
	ModeSearch
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
}

// Detonation describes a warhead event for the sim (blast, deaf, sinking).
type Detonation struct {
	X, Y      float64
	DepthFt   float64
	Hit       *world.Entity // may be nil if scuttled without contact
	SelfKill  bool          // intentional self-destruct — no blast/deaf
	ShooterID string
}

// FireControl manages 688-style torpedo firing.
type FireControl struct {
	Tubes            [4]Tube
	SelectedTube     int
	GyroAngleDeg     float64 // absolute ordered launch course, 0–360° true
	RunDepthFt       float64
	SpeedSetting     string // "LOW", "HIGH"
	SeekerEnabled    bool
	MagazineLeft     int
	EnemyMagazine    map[string]int
	EnemyTubeOpenAt  map[string]float64
	ActiveTorpedoes  []*Torpedo
	torpedoSeq       int
}

func NewFireControl() FireControl {
	fc := FireControl{
		SelectedTube:  1,
		GyroAngleDeg:  0,
		RunDepthFt:    400,
		SpeedSetting:  "HIGH",
		SeekerEnabled: false,
		MagazineLeft:  PlayerMagazineCapacity - 4, // 4 already in tubes
		EnemyMagazine: map[string]int{},
		EnemyTubeOpenAt: map[string]float64{},
	}
	for i := range fc.Tubes {
		fc.Tubes[i] = Tube{
			Number:      i + 1,
			State:       TubeLoaded,
			TorpedoType: "Mk48",
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

func (fc *FireControl) beginReload(t *Tube, gameTime float64) {
	if fc.MagazineLeft <= 0 {
		t.State = TubeEmpty
		t.ReloadEnds = 0
		t.TorpedoType = ""
		return
	}
	fc.MagazineLeft--
	t.State = TubeReloading
	t.ReloadEnds = gameTime + TubeReloadSec
	t.TorpedoType = "Mk48"
	t.WireIntact = false
	t.TorpedoID = ""
}

// UpdateTubes advances reload timers.
func (fc *FireControl) UpdateTubes(gameTime float64) {
	for i := range fc.Tubes {
		t := &fc.Tubes[i]
		if t.State == TubeReloading && t.ReloadEnds > 0 && gameTime >= t.ReloadEnds {
			t.State = TubeLoaded
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
	torp := &Torpedo{
		ID:                     fmt.Sprintf("MK48-%d", fc.torpedoSeq),
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
	}
	t.TorpedoID = torp.ID
	fc.ActiveTorpedoes = append(fc.ActiveTorpedoes, torp)
	return torp
}

// SpawnHostileTorpedo launches an AI weapon (wire initially, then search).
func (fc *FireControl) SpawnHostileTorpedo(sub, target *world.Entity) *Torpedo {
	if sub == nil || !sub.Alive() || target == nil {
		return nil
	}
	left := fc.enemyAmmo(sub.ID)
	if left <= 0 {
		return nil
	}
	fc.EnemyMagazine[sub.ID] = left - 1
	fc.torpedoSeq++
	ownHead := normalizeAngle(sub.HeadingDeg)
	brg := sub.BearingDegTo(target)
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
		GyroCourseDeg:          brg,
		SpeedKts:               18,
		CruiseKts:              48,
		RunDepthFt:             math.Max(80, target.DepthFt),
		SeekerOn:               false,
		Armed:                  true,
		Mode:                   ModeWire,
		Alive:                  true,
		LastPingTime:           -1,
		EnableSearchAfterClear: false, // wire-guide to target after clear; seek later
	}
	fc.ActiveTorpedoes = append(fc.ActiveTorpedoes, torp)
	return torp
}

func (fc *FireControl) enemyAmmo(subID string) int {
	if fc.EnemyMagazine == nil {
		fc.EnemyMagazine = map[string]int{}
	}
	if v, ok := fc.EnemyMagazine[subID]; ok {
		return v
	}
	fc.EnemyMagazine[subID] = EnemySubMagazine
	return EnemySubMagazine
}

func (fc *FireControl) HasRecentShotFrom(subID string, maxAge float64) bool {
	for _, t := range fc.ActiveTorpedoes {
		if t != nil && t.Alive && t.ParentSubID == subID && t.Age < maxAge {
			return true
		}
	}
	return false
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
	torp.WireCut = true
	if torp.TubeCleared() {
		torp.Mode = ModeSearch
		torp.SeekerOn = true
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
	torp.SeekerOn = true
	torp.Mode = ModeSearch
}

// ToggleSeeker turns ModeSearch on/off while the wire is intact.
// With a cut wire the seeker can only be enabled (autonomous fish).
func (fc *FireControl) ToggleSeeker(torp *Torpedo) {
	if torp == nil || !torp.Alive {
		return
	}
	if !torp.TubeCleared() {
		torp.EnableSearchAfterClear = !torp.EnableSearchAfterClear
		return
	}
	if torp.Mode == ModeSearch || torp.SeekerOn {
		if torp.WireCut {
			return // autonomous — cannot cancel search without a wire
		}
		torp.SeekerOn = false
		torp.EnableSearchAfterClear = false
		torp.Mode = ModeWire
		torp.OrderedHead = torp.HeadingDeg
		torp.GyroCourseDeg = torp.HeadingDeg
		return
	}
	torp.SeekerOn = true
	torp.Mode = ModeSearch
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

// TubeCleared reports whether the fish has run far enough to enable gyro turn.
func (t *Torpedo) TubeCleared() bool {
	if t == nil {
		return true
	}
	return t.ClearDistYd >= TubeClearYd
}

func (t *Torpedo) enableGyroAfterClear() {
	if t == nil || t.gyroEnabled {
		return
	}
	t.gyroEnabled = true
	t.OrderedHead = t.GyroCourseDeg
	if t.EnableSearchAfterClear || t.WireCut {
		t.Mode = ModeSearch
		t.SeekerOn = true
	}
}

// AcousticEntity fills dst with a world.Entity view for sonar detection.
func (t *Torpedo) AcousticEntity(dst *world.Entity) *world.Entity {
	if t == nil || dst == nil {
		return nil
	}
	sig := "mk48"
	if t.Side == world.SideEnemy {
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
func (t *Torpedo) Advance(dt, gameTime float64, targets []*world.Entity, layerAtten LayerAttenFunc) *Detonation {
	if !t.Alive {
		return nil
	}
	t.Age += dt
	t.DepthFt += (t.RunDepthFt - t.DepthFt) * dt * 0.5
	if t.CruiseKts > 0 {
		// ~4 kts/s — order of published Mk48 time-to-speed.
		const accel = 4.0
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
		t.EnableSearchAfterClear = true
		if t.TubeCleared() {
			t.Mode = ModeSearch
			t.SeekerOn = true
		}
	}

	// Tube-exit: hold launch heading until clear distance is reached.
	if !t.TubeCleared() {
		t.OrderedHead = t.LaunchHeadDeg
		t.Mode = ModeWire
		t.SeekerOn = false
	}

	// Wire guidance: follow OrderedHead while wire intact and not searching.
	if t.Mode == ModeWire && !t.WireCut {
		diff := shortestAngleDiff(t.HeadingDeg, t.OrderedHead)
		t.HeadingDeg += clamp(diff*dt*1.2, -dt*10, dt*10)
		t.HeadingDeg = normalizeAngle(t.HeadingDeg)
	} else if t.Mode == ModeWire && t.WireCut && !t.TubeCleared() {
		// Wire already cut but still clearing the tube — keep going straight.
		diff := shortestAngleDiff(t.HeadingDeg, t.LaunchHeadDeg)
		t.HeadingDeg += clamp(diff*dt*1.2, -dt*10, dt*10)
		t.HeadingDeg = normalizeAngle(t.HeadingDeg)
	}

	// Active search: acquire ANY ship/sub in forward cone (friendly fire possible).
	if t.Mode == ModeSearch {
		if t.LastPingTime < 0 || gameTime-t.LastPingTime >= TorpedoActivePingIntervalSec {
			t.LastPingTime = gameTime
		}
		if best := t.acquireInCone(targets, layerAtten); best != nil {
			desired := bearing(t.X, t.Y, best.X, best.Y)
			diff := shortestAngleDiff(t.HeadingDeg, desired)
			t.HeadingDeg += clamp(diff*dt*0.9, -dt*10, dt*10)
			t.HeadingDeg = normalizeAngle(t.HeadingDeg)
			t.TargetID = best.ID
			t.RunDepthFt += (best.DepthFt - t.RunDepthFt) * dt * 0.35
		}
	}

	rad := t.HeadingDeg * math.Pi / 180
	yps := t.SpeedKts * world.KnotsToYPS
	t.X += math.Sin(rad) * yps * dt
	t.Y += math.Cos(rad) * yps * dt

	// Influence fuse only while actively searching (not wire-run / transit).
	if t.Armed && t.Age > 2 && t.Mode == ModeSearch {
		for _, tgt := range targets {
			if tgt == nil || !tgt.Alive() {
				continue
			}
			if tgt.Kind != world.KindSubmarine && tgt.Kind != world.KindSurfaceShip {
				continue
			}
			d := math.Hypot(tgt.X-t.X, tgt.Y-t.Y)
			if d > ProximityKillYd {
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
				return &Detonation{
					X: t.X, Y: t.Y, DepthFt: t.DepthFt,
					Hit: tgt, ShooterID: t.ParentSubID,
				}
			}
		}
	}
	if t.Age > 600 {
		t.Alive = false
	}
	return nil
}

func (t *Torpedo) acquireInCone(targets []*world.Entity, layerAtten LayerAttenFunc) *world.Entity {
	var best *world.Entity
	bestDist := 1e9
	for _, tgt := range targets {
		if tgt == nil || !tgt.Alive() {
			continue
		}
		if tgt.Kind != world.KindSubmarine && tgt.Kind != world.KindSurfaceShip {
			continue
		}
		// Do not lock own launcher while still wire-connected.
		if tgt.ID == t.ParentSubID && !t.WireCut && t.Mode == ModeWire {
			continue
		}
		d := math.Hypot(tgt.X-t.X, tgt.Y-t.Y)
		maxR, coneHalf := seekAcquireLimits(t.DepthFt, tgt.DepthFt, layerAtten)
		if d > maxR || d >= bestDist {
			continue
		}
		brg := bearing(t.X, t.Y, tgt.X, tgt.Y)
		if math.Abs(shortestAngleDiff(t.HeadingDeg, brg)) > coneHalf {
			continue
		}
		bestDist = d
		best = tgt
	}
	return best
}

// seekAcquireLimits shrinks seeker range/cone when acoustic layers separate fish and target.
func seekAcquireLimits(torpDepthFt, tgtDepthFt float64, layerAtten LayerAttenFunc) (maxRangeYd, coneHalfDeg float64) {
	maxRangeYd = SeekAcquireRangeYd
	coneHalfDeg = SeekConeHalfAngleDeg
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
	maxRangeYd = SeekAcquireRangeYd * factor
	// Narrow the effective cone through the layer (multipath / refraction).
	coneScale := 1 - 0.35*math.Min(1, loss/22)
	if coneScale < 0.55 {
		coneScale = 0.55
	}
	coneHalfDeg = SeekConeHalfAngleDeg * coneScale
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

// TubeStateName is a UI helper.
func TubeStateName(s TubeState) string {
	switch s {
	case TubeEmpty:
		return "EMPTY"
	case TubeLoaded:
		return "LOADED"
	case TubeDoorOpen:
		return "DOOR OPEN"
	case TubeFired:
		return "FIRED"
	case TubeReloading:
		return "RELOADING"
	default:
		return "?"
	}
}
