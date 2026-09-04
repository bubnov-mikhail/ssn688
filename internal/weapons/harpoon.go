package weapons

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// UGM-84 Sub-Harpoon — gameplay ranges from published 75 nm max / programmable SRCH.
const (
	HarpoonMaxRangeNm   = 75.0
	HarpoonMaxRangeYd   = HarpoonMaxRangeNm * world.YardsPerNM // 136800 yd
	HarpoonCruiseKts    = 540.0                               // ~Mach 0.85
	HarpoonUnderwaterSec = 5.0
	HarpoonUnderwaterKts = 12.0
	HarpoonWideBeamDeg   = 30.0
	HarpoonNarrowBeamDeg = 15.0
	HarpoonHitRadiusYd   = 120.0
	// HarpoonSeekAcquireYd — active radar acquisition range once SRCH is reached.
	HarpoonSeekAcquireYd = 25000.0 // ~12 nm
	// HarpoonTurnRateDegPerSec — terminal / midcourse seeker turn authority.
	HarpoonTurnRateDegPerSec = 8.0
)

// Harpoon radar search range presets — PLOT-aligned rings plus 8 nm.
const (
	HarpoonRadarMinNm    = 1.0
	HarpoonRadarShortNm  = 2.0
	HarpoonRadarMediumNm = 4.0
	HarpoonRadarLongNm   = 6.0
	HarpoonRadarXLongNm  = 8.0
	HarpoonRadarMinYd    = HarpoonRadarMinNm * world.YardsPerNM
	HarpoonRadarShortYd  = HarpoonRadarShortNm * world.YardsPerNM
	HarpoonRadarMediumYd = HarpoonRadarMediumNm * world.YardsPerNM
	HarpoonRadarLongYd   = HarpoonRadarLongNm * world.YardsPerNM
	HarpoonRadarXLongYd  = HarpoonRadarXLongNm * world.YardsPerNM
	HarpoonDestructMedNm  = 40.0
	HarpoonDestructLongNm = 60.0
	HarpoonDestructMedYd  = HarpoonDestructMedNm * world.YardsPerNM
	HarpoonDestructLongYd = HarpoonDestructLongNm * world.YardsPerNM
)

// Harpoon setting tokens (fire-control prep / save).
const (
	HarpoonBeamWide   = "WIDE"
	HarpoonBeamNarrow = "NARROW"
	HarpoonSRCHMin    = "MIN" // 1 nm — earliest radar enable
	HarpoonSRCHShort  = "SHORT"
	HarpoonSRCHMedium = "MEDIUM"
	HarpoonSRCHLong   = "LONG"
	HarpoonSRCHXLong  = "XLONG" // 8 nm
	HarpoonDSTRMedium = "MEDIUM"
	HarpoonDSTRLong   = "LONG"
	HarpoonDSTRMax    = "MAX"
)

type HarpoonPhase int

const (
	HarpoonUnderwater HarpoonPhase = iota
	HarpoonCruise
)

// HarpoonMissile is an in-flight Sub-Harpoon (capsule egress + cruise).
type HarpoonMissile struct {
	ID              string
	ParentSubID     string
	TubeNumber      int
	TargetContactID string
	Side            world.Side
	LaunchX, LaunchY float64
	X, Y            float64
	HeadingDeg      float64
	ProgrammedHead  float64 // gyro at launch — WEPS assumed track never seeks
	SpeedKts        float64
	DistanceYd      float64
	AssumedDistanceYd float64 // straight-line WEPS plot along ProgrammedHead
	Phase           HarpoonPhase
	UnderwaterLeft  float64
	RadarOn         bool
	BeamHalfDeg     float64
	RadarRangeYd    float64
	DestructRangeYd float64
	Alive           bool
	Age             float64
	VisibleOnWEPS   bool
	ExpectedArrival float64 // game time when straight-line flight reaches programmed range
	LockedTargetID  string  // surface contact locked after radar search
	Variant         ASCMVariant
	CruiseKts       float64 // cruise-phase speed (set at launch)
}

func HarpoonRadarRangeYd(setting string) float64 {
	switch setting {
	case HarpoonSRCHMin:
		return HarpoonRadarMinYd
	case HarpoonSRCHShort:
		return HarpoonRadarShortYd
	case HarpoonSRCHLong:
		return HarpoonRadarLongYd
	case HarpoonSRCHXLong:
		return HarpoonRadarXLongYd
	default:
		return HarpoonRadarMediumYd
	}
}

func HarpoonDestructRangeYd(setting string) float64 {
	switch setting {
	case HarpoonDSTRMedium:
		return HarpoonDestructMedYd
	case HarpoonDSTRLong:
		return HarpoonDestructLongYd
	default:
		return HarpoonMaxRangeYd
	}
}

func HarpoonBeamHalfDeg(setting string) float64 {
	if setting == HarpoonBeamNarrow {
		return HarpoonNarrowBeamDeg
	}
	return HarpoonWideBeamDeg
}

// HarpoonRadarRangeLabel returns UI title text for SRCH buttons.
func HarpoonRadarRangeLabel(setting string) string {
	switch setting {
	case HarpoonSRCHMin:
		return fmt.Sprintf("SRCH %.0f nm", HarpoonRadarMinNm)
	case HarpoonSRCHShort:
		return fmt.Sprintf("SRCH %.0f nm", HarpoonRadarShortNm)
	case HarpoonSRCHLong:
		return fmt.Sprintf("SRCH %.0f nm", HarpoonRadarLongNm)
	case HarpoonSRCHXLong:
		return fmt.Sprintf("SRCH %.0f nm", HarpoonRadarXLongNm)
	default:
		return fmt.Sprintf("SRCH %.0f nm", HarpoonRadarMediumNm)
	}
}

// HarpoonDestructRangeLabel returns UI title text for DSTR buttons.
func HarpoonDestructRangeLabel(setting string) string {
	switch setting {
	case HarpoonDSTRMedium:
		return fmt.Sprintf("DSTR %.0f nm", HarpoonDestructMedNm)
	case HarpoonDSTRLong:
		return fmt.Sprintf("DSTR %.0f nm", HarpoonDestructLongNm)
	default:
		return fmt.Sprintf("DSTR %.0f nm", HarpoonMaxRangeNm)
	}
}

// EnsureHarpoonDestructValid bumps DSTR to the next preset when SRCH increases.
func EnsureHarpoonDestructValid(radarSetting, destructSetting string) string {
	radarYd := HarpoonRadarRangeYd(radarSetting)
	if HarpoonDestructRangeYd(destructSetting) >= radarYd {
		return destructSetting
	}
	order := []string{HarpoonDSTRMedium, HarpoonDSTRLong, HarpoonDSTRMax}
	for _, d := range order {
		if HarpoonDestructRangeYd(d) >= radarYd {
			return d
		}
	}
	return HarpoonDSTRMax
}

func nextHarpoonRadarSetting(cur string) string {
	switch cur {
	case HarpoonSRCHMin:
		return HarpoonSRCHShort
	case HarpoonSRCHShort:
		return HarpoonSRCHMedium
	case HarpoonSRCHMedium:
		return HarpoonSRCHLong
	case HarpoonSRCHLong:
		return HarpoonSRCHXLong
	default:
		return HarpoonSRCHMin
	}
}

func nextHarpoonDestructSetting(cur string) string {
	switch cur {
	case HarpoonDSTRMedium:
		return HarpoonDSTRLong
	case HarpoonDSTRLong:
		return HarpoonDSTRMax
	default:
		return HarpoonDSTRMedium
	}
}

func nextHarpoonBeamSetting(cur string) string {
	if cur == HarpoonBeamWide {
		return HarpoonBeamNarrow
	}
	return HarpoonBeamWide
}

// ShootHarpoon launches a Sub-Harpoon from an open tube loaded with Harpoon.
func (fc *FireControl) ShootHarpoon(sub *world.Entity, tubeNum int) *HarpoonMissile {
	t := fc.TubeByNumber(tubeNum)
	if t == nil {
		t = fc.Selected()
	}
	if t == nil || t.State != TubeDoorOpen || normalizeOrdnance(t.TorpedoType) != OrdnanceHarpoon {
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
	t.WireIntact = false
	t.TorpedoID = ""

	radarSetting := fc.HarpoonRadarRange
	destructSetting := EnsureHarpoonDestructValid(radarSetting, fc.HarpoonDestructRange)
	fc.HarpoonDestructRange = destructSetting

	radarYd := HarpoonRadarRangeYd(radarSetting)
	destructYd := HarpoonDestructRangeYd(destructSetting)
	ownHead := normalizeAngle(sub.HeadingDeg)
	gyro := normalizeAngle(fc.GyroAngleDeg)
	rad := ownHead * math.Pi / 180
	offset := 40.0

	h := &HarpoonMissile{
		ID:              fmt.Sprintf("HSM-%d", fc.torpedoSeq),
		ParentSubID:     sub.ID,
		TubeNumber:      t.Number,
		TargetContactID: t.TargetContactID,
		Side:            sub.Side,
		LaunchX:         sub.X + math.Sin(rad)*offset,
		LaunchY:         sub.Y + math.Cos(rad)*offset,
		X:               sub.X + math.Sin(rad)*offset,
		Y:               sub.Y + math.Cos(rad)*offset,
		HeadingDeg:      gyro,
		ProgrammedHead:  gyro,
		SpeedKts:        HarpoonUnderwaterKts,
		Phase:           HarpoonUnderwater,
		UnderwaterLeft:  HarpoonUnderwaterSec,
		BeamHalfDeg:     HarpoonBeamHalfDeg(fc.HarpoonRadarBeam),
		RadarRangeYd:    radarYd,
		DestructRangeYd: destructYd,
		Alive:           true,
		VisibleOnWEPS:   true,
		Variant:         ASCMHarpoon,
		CruiseKts:       HarpoonCruiseKts,
	}
	flightSec := destructYd / (HarpoonCruiseKts * world.KnotsToYPS)
	h.ExpectedArrival = flightSec // filled by engine with game time on launch

	fc.ActiveHarpoons = append(fc.ActiveHarpoons, h)
	return h
}

// AdvanceHarpoon moves one missile; returns detonation if it hits or self-destructs.
func (h *HarpoonMissile) Advance(dt float64, targets []*world.Entity) *Detonation {
	if h == nil {
		return nil
	}
	// Ghost WEPS track after soft-kill: keep assumed straight-line plot only.
	if !h.Alive {
		if h.VisibleOnWEPS {
			h.Age += dt
			h.advanceAssumed(dt)
			if h.AssumedDistanceYd >= h.DestructRangeYd {
				h.VisibleOnWEPS = false
			}
		}
		return nil
	}
	h.Age += dt
	step := h.SpeedKts * world.KnotsToYPS * dt

	if h.Phase == HarpoonUnderwater {
		h.UnderwaterLeft -= dt
		rad := h.HeadingDeg * math.Pi / 180
		h.X += math.Sin(rad) * step
		h.Y += math.Cos(rad) * step
		h.DistanceYd += step
		h.advanceAssumed(dt)
		if h.UnderwaterLeft <= 0 {
			h.Phase = HarpoonCruise
			if h.CruiseKts <= 0 {
				h.CruiseKts = h.Variant.cruiseKts()
			}
			h.SpeedKts = h.CruiseKts
		}
		return nil
	}

	if !h.RadarOn && h.DistanceYd >= h.RadarRangeYd {
		h.RadarOn = true
	}
	if h.RadarOn {
		h.updateSeeker(dt, targets)
	}

	rad := h.HeadingDeg * math.Pi / 180
	h.X += math.Sin(rad) * step
	h.Y += math.Cos(rad) * step
	h.DistanceYd += step
	h.advanceAssumed(dt)

	if h.RadarOn {
		if hit := h.checkImpact(targets); hit != nil {
			h.Alive = false
			h.VisibleOnWEPS = false
			return &Detonation{
				X: h.X, Y: h.Y, DepthFt: 0,
				Hit: hit, Harpoon: true, ShooterID: h.ParentSubID,
			}
		}
	}

	if h.DistanceYd >= h.DestructRangeYd {
		h.Alive = false
		h.VisibleOnWEPS = false
		return &Detonation{
			X: h.X, Y: h.Y, DepthFt: 0,
			SelfKill: true, ShooterID: h.ParentSubID,
		}
	}
	return nil
}

// advanceAssumed steps the WEPS-only straight-line track (no seeker turn).
func (h *HarpoonMissile) advanceAssumed(dt float64) {
	uwDist := HarpoonUnderwaterSec * HarpoonUnderwaterKts * world.KnotsToYPS
	spd := h.CruiseKts
	if spd <= 0 {
		spd = h.Variant.cruiseKts()
	}
	if h.AssumedDistanceYd < uwDist {
		spd = HarpoonUnderwaterKts
	}
	h.AssumedDistanceYd += spd * world.KnotsToYPS * dt
}

// AssumedXY is the WEPS plot position along the programmed gyro course.
func (h *HarpoonMissile) AssumedXY() (x, y float64) {
	if h == nil {
		return 0, 0
	}
	rad := h.ProgrammedHead * math.Pi / 180
	return h.LaunchX + math.Sin(rad)*h.AssumedDistanceYd, h.LaunchY + math.Cos(rad)*h.AssumedDistanceYd
}

func (h *HarpoonMissile) updateSeeker(dt float64, targets []*world.Entity) {
	lock := h.findLocked(targets)
	if lock == nil {
		lock = h.acquireTarget(targets)
		if lock != nil {
			h.LockedTargetID = lock.ID
		} else {
			h.LockedTargetID = ""
			return
		}
	}
	want := bearing(h.X, h.Y, lock.X, lock.Y)
	diff := shortestAngleDiff(h.HeadingDeg, want)
	maxTurn := h.Variant.turnRateDegPerSec() * dt
	h.HeadingDeg = normalizeAngle(h.HeadingDeg + clamp(diff, -maxTurn, maxTurn))
}

func (h *HarpoonMissile) findLocked(targets []*world.Entity) *world.Entity {
	if h.LockedTargetID == "" {
		return nil
	}
	for _, ent := range targets {
		if !h.validSeekerContact(ent) || ent.ID != h.LockedTargetID {
			continue
		}
		// Keep lock while target is roughly ahead; lose if far behind beam.
		brg := bearing(h.X, h.Y, ent.X, ent.Y)
		if math.Abs(shortestAngleDiff(h.HeadingDeg, brg)) > h.BeamHalfDeg*2.5 {
			return nil
		}
		return ent
	}
	return nil
}

func (h *HarpoonMissile) checkImpact(targets []*world.Entity) *world.Entity {
	for _, ent := range targets {
		if !h.validSeekerContact(ent) {
			continue
		}
		if h.LockedTargetID != "" && ent.ID != h.LockedTargetID {
			continue
		}
		if math.Hypot(ent.X-h.X, ent.Y-h.Y) <= HarpoonHitRadiusYd {
			return ent
		}
	}
	return nil
}

// validSeekerContact is true for surface/periscope contacts the seeker may paint.
// Same-side hulls are IFF-filtered so enemy Klub/Oniks/Kalibr do not retarget
// friendly surface screens between the shooter and the intended quarry.
func (h *HarpoonMissile) validSeekerContact(ent *world.Entity) bool {
	if h == nil || ent == nil || !ent.Alive() || ent.DepthFt > 5 {
		return false
	}
	if ent.Kind != world.KindSurfaceShip && ent.Kind != world.KindSubmarine {
		return false
	}
	if h.ParentSubID != "" && ent.ID == h.ParentSubID {
		return false
	}
	if ent.Side == h.Side {
		return false
	}
	return true
}

// acquireTarget picks the best surface contact in the radar search cone.
func (h *HarpoonMissile) acquireTarget(targets []*world.Entity) *world.Entity {
	if h == nil || !h.RadarOn {
		return nil
	}
	type cand struct {
		ent      *world.Entity
		priority int
		dist     float64
	}
	var best *cand
	aim := h.HeadingDeg
	for _, ent := range targets {
		if !h.validSeekerContact(ent) {
			continue
		}
		brg := bearing(h.X, h.Y, ent.X, ent.Y)
		if math.Abs(shortestAngleDiff(aim, brg)) > h.BeamHalfDeg {
			continue
		}
		dist := math.Hypot(ent.X-h.X, ent.Y-h.Y)
		if dist > HarpoonSeekAcquireYd {
			continue
		}
		pri := targetPriority(ent)
		c := &cand{ent: ent, priority: pri, dist: dist}
		if best == nil || c.priority > best.priority || (c.priority == best.priority && c.dist < best.dist) {
			best = c
		}
	}
	if best == nil {
		return nil
	}
	return best.ent
}

func targetPriority(ent *world.Entity) int {
	if ent == nil {
		return 0
	}
	if world.IsCombatant(ent) {
		return 3
	}
	if ent.Side == world.SideNeutral {
		switch ent.SignatureID {
		case "merchant", "fishing":
			return 1
		case "tanker":
			return 2
		default:
			return 1
		}
	}
	return 0
}

// AcousticEntity returns a brief underwater egress signature (torpedo-like).
func (h *HarpoonMissile) AcousticEntity(slot *world.Entity) *world.Entity {
	if h == nil || !h.Alive || h.Phase != HarpoonUnderwater {
		return nil
	}
	*slot = world.Entity{
		ID: "harpoon_egress_" + h.ID, Kind: world.KindTorpedo, Side: h.Side,
		X: h.X, Y: h.Y, DepthFt: 30, SpeedKts: h.SpeedKts, HeadingDeg: h.HeadingDeg,
		SignatureID: "mk48", Status: world.StatusActive,
	}
	return slot
}

// StraightLineEnd returns map endpoint for WEPS assumed course display.
func (h *HarpoonMissile) StraightLineEnd() (x, y float64) {
	if h == nil {
		return 0, 0
	}
	ax, ay := h.AssumedXY()
	remain := h.DestructRangeYd - h.AssumedDistanceYd
	if remain < 0 {
		remain = 0
	}
	rad := h.ProgrammedHead * math.Pi / 180
	return ax + math.Sin(rad)*remain, ay + math.Cos(rad)*remain
}

func (fc *FireControl) CycleHarpoonBeam() {
	fc.HarpoonRadarBeam = nextHarpoonBeamSetting(fc.HarpoonRadarBeam)
}

func (fc *FireControl) CycleHarpoonRadarRange() {
	fc.HarpoonRadarRange = nextHarpoonRadarSetting(fc.HarpoonRadarRange)
	fc.HarpoonDestructRange = EnsureHarpoonDestructValid(fc.HarpoonRadarRange, fc.HarpoonDestructRange)
}

func (fc *FireControl) CycleHarpoonDestructRange() {
	fc.HarpoonDestructRange = nextHarpoonDestructSetting(fc.HarpoonDestructRange)
	fc.HarpoonDestructRange = EnsureHarpoonDestructValid(fc.HarpoonRadarRange, fc.HarpoonDestructRange)
}

func (fc *FireControl) SetHarpoonRadarRange(setting string) {
	fc.HarpoonRadarRange = setting
	fc.HarpoonDestructRange = EnsureHarpoonDestructValid(fc.HarpoonRadarRange, fc.HarpoonDestructRange)
}

func (fc *FireControl) SetHarpoonDestructRange(setting string) {
	fc.HarpoonDestructRange = setting
	fc.HarpoonDestructRange = EnsureHarpoonDestructValid(fc.HarpoonRadarRange, fc.HarpoonDestructRange)
}

func (fc *FireControl) HarpoonByTube(tubeNum int) *HarpoonMissile {
	for _, h := range fc.ActiveHarpoons {
		if h != nil && h.VisibleOnWEPS && h.TubeNumber == tubeNum {
			return h
		}
	}
	return nil
}

func (fc *FireControl) AdvanceHarpoons(dt, gameTime float64, targets []*world.Entity, rng *rand.Rand) []*Detonation {
	if len(fc.ActiveHarpoons) == 0 {
		return nil
	}
	var dets []*Detonation
	keep := fc.ActiveHarpoons[:0]
	for _, h := range fc.ActiveHarpoons {
		if h == nil {
			continue
		}
		if h.Alive {
			if det := h.Advance(dt, targets); det != nil {
				dets = append(dets, det)
			} else if h.Alive {
				if det := fc.TryPointDefense(h, targets, gameTime, rng); det != nil {
					dets = append(dets, det)
				}
			}
		} else if h.VisibleOnWEPS {
			h.Advance(dt, nil) // assumed ghost track only
		}
		if h.Alive || h.VisibleOnWEPS {
			keep = append(keep, h)
		}
	}
	fc.ActiveHarpoons = keep
	return dets
}

// CheckHarpoonBlastHeard hides WEPS track when sonar hears a blast near the assumed track.
func (fc *FireControl) CheckHarpoonBlastHeard(gameTime, blastX, blastY, blastAt float64) {
	for _, h := range fc.ActiveHarpoons {
		if h == nil || !h.VisibleOnWEPS {
			continue
		}
		if gameTime-blastAt > 25 {
			continue
		}
		ax, ay := h.AssumedXY()
		dist := math.Hypot(ax-blastX, ay-blastY)
		if dist < 8000 {
			h.VisibleOnWEPS = false
		}
	}
}

func (fc *FireControl) allyHarpoonAmmo(sub *world.Entity) int {
	if fc == nil || sub == nil {
		return 0
	}
	if fc.AllyHarpoonMag == nil {
		fc.AllyHarpoonMag = map[string]int{}
	}
	if v, ok := fc.AllyHarpoonMag[sub.ID]; ok {
		return v
	}
	n := AllyHarpoonMagazineFor(sub.SignatureID)
	fc.AllyHarpoonMag[sub.ID] = n
	return n
}

// AllyHarpoonLeft returns remaining Sub-Harpoon rounds for an AI platform.
func (fc *FireControl) AllyHarpoonLeft(subID string) int {
	if fc == nil || fc.AllyHarpoonMag == nil {
		return 0
	}
	return fc.AllyHarpoonMag[subID]
}

func pseudoNoise(a, b string) float64 {
	h := 0
	for _, c := range a + b {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return float64(h%1000) / 1000.0
}
