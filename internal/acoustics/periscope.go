package acoustics

import (
	"fmt"
	"math"

	"github.com/ssn688/sim/internal/world"
)

// PeriMastOrder is the commanded periscope position.
type PeriMastOrder int

const (
	PeriMastStow PeriMastOrder = iota
	PeriMastRaise
)

// Discrete optical zoom steps (approximate Type-18 / photonics FOV labels).
const (
	PeriZoomLow = iota
	PeriZoomMed
	PeriZoomHigh
	PeriZoomCount
)

const (
	periTrainStepDeg = 5.0
	periRaiseSec     = 5.0 // optical mast is a bit slower than ESM
	periLowerSec     = 3.5
)

// TrainStepDeg is the discrete train increment for the current zoom.
// High power uses 1° so fine aiming is possible in a narrow FOV.
func (p *PeriscopeState) TrainStepDeg() float64 {
	if p == nil {
		return periTrainStepDeg
	}
	switch clampPeriZoom(p.Zoom) {
	case PeriZoomHigh:
		return 1
	case PeriZoomMed:
		return 2
	default:
		return periTrainStepDeg
	}
}

// PeriZoomLabel returns a short FOV label for the UI.
func PeriZoomLabel(level int) string {
	switch clampPeriZoom(level) {
	case PeriZoomLow:
		return "1.5×"
	case PeriZoomMed:
		return "6×"
	default:
		return "12×"
	}
}

// PeriFOVDeg is the horizontal field of view at a zoom step (for the view reticule).
func PeriFOVDeg(level int) float64 {
	switch clampPeriZoom(level) {
	case PeriZoomLow:
		return 32
	case PeriZoomMed:
		return 12
	default:
		return 6
	}
}

func clampPeriZoom(level int) int {
	if level < 0 {
		return 0
	}
	if level >= PeriZoomCount {
		return PeriZoomCount - 1
	}
	return level
}

// PeriscopeState is the player's optical mast + view train/zoom.
type PeriscopeState struct {
	Order      PeriMastOrder
	Extension  float64 // 0..1
	LastWarnAt float64
	Sheared    bool
	// TrainRelDeg is look direction relative to ownship heading (0 = bow, +CW).
	TrainRelDeg float64
	Zoom        int // PeriZoom*
	// LockEntityID, when set, slews train to keep that platform in the crosshair
	// while acoustic, ESM, or visual contact remains.
	LockEntityID string
}

// MastUp is true when the optic is extended enough for a usable picture.
func (p *PeriscopeState) MastUp() bool {
	return p != nil && !p.Sheared && p.Extension >= 0.95
}

// MastMoving reports raise/lower in progress.
func (p *PeriscopeState) MastMoving() bool {
	if p == nil || p.Sheared {
		return false
	}
	if p.Order == PeriMastRaise && p.Extension < 1 {
		return true
	}
	if p.Order == PeriMastStow && p.Extension > 0 {
		return true
	}
	return false
}

// TrueBearingDeg is the absolute look bearing (0..360).
func (p *PeriscopeState) TrueBearingDeg(ownHeadingDeg float64) float64 {
	if p == nil {
		return normalizeDeg360(ownHeadingDeg)
	}
	return normalizeDeg360(ownHeadingDeg + p.TrainRelDeg)
}

// ZoomLabel is the current FOV label.
func (p *PeriscopeState) ZoomLabel() string {
	if p == nil {
		return PeriZoomLabel(PeriZoomLow)
	}
	return PeriZoomLabel(p.Zoom)
}

// FOVDeg is the current horizontal field of view.
func (p *PeriscopeState) FOVDeg() float64 {
	if p == nil {
		return PeriFOVDeg(PeriZoomLow)
	}
	return PeriFOVDeg(p.Zoom)
}

// CanRaisePeriscope reports whether depth/speed allow raise.
func CanRaisePeriscope(player *world.Entity) (ok bool, reason string) {
	if player == nil {
		return false, "No ownship."
	}
	player.EnsureDamage()
	if player.Damage.Destroyed(world.SysPeriscope) {
		return false, "Periscope destroyed — beyond repair."
	}
	if player.DepthFt > world.ESMMastMaxDepthFt+0.5 {
		return false, fmt.Sprintf("Too deep — periscope requires ≤%.0f ft.", world.ESMMastMaxDepthFt)
	}
	spd := math.Abs(player.SpeedKts)
	if spd > world.ESMMastMaxSpeedKts+0.05 {
		return false, fmt.Sprintf("Too fast — periscope requires ≤%.0f kn.", world.ESMMastMaxSpeedKts)
	}
	return true, ""
}

// OrderRaise begins raising if conditions allow.
func (p *PeriscopeState) OrderRaise(player *world.Entity) (ok bool, msg string) {
	if p.Sheared || (player != nil && player.Damage.Destroyed(world.SysPeriscope)) {
		p.Sheared = true
		return false, "Periscope destroyed — beyond repair."
	}
	if ok, reason := CanRaisePeriscope(player); !ok {
		return false, reason
	}
	p.Order = PeriMastRaise
	return true, "Raising periscope."
}

// OrderLower begins stowing.
func (p *PeriscopeState) OrderLower() string {
	p.Order = PeriMastStow
	return "Lowering periscope."
}

// TrainLeft steps the optic counterclockwise (port) relative to the hull.
func (p *PeriscopeState) TrainLeft() {
	if p == nil || p.Sheared {
		return
	}
	p.ClearLock()
	p.TrainRelDeg = normalizeRel180(p.TrainRelDeg - p.TrainStepDeg())
}

// TrainRight steps the optic clockwise (starboard) relative to the hull.
func (p *PeriscopeState) TrainRight() {
	if p == nil || p.Sheared {
		return
	}
	p.ClearLock()
	p.TrainRelDeg = normalizeRel180(p.TrainRelDeg + p.TrainStepDeg())
}

// ClearLock drops optical target lock (manual train / lost contact).
func (p *PeriscopeState) ClearLock() {
	if p != nil {
		p.LockEntityID = ""
	}
}

// Locked reports an active optical lock.
func (p *PeriscopeState) Locked() bool {
	return p != nil && p.LockEntityID != ""
}

// EngageLock snaps train onto trueBrg and begins tracking entityID.
func (p *PeriscopeState) EngageLock(entityID string, ownHeadingDeg, trueBrgDeg float64) {
	if p == nil || entityID == "" || p.Sheared {
		return
	}
	p.LockEntityID = entityID
	p.TrainRelDeg = normalizeRel180(AngleDiffSigned(trueBrgDeg, ownHeadingDeg))
}

const periLockSlewDegPerSec = 60.0

// UpdateLock slews the optic to keep LockEntityID centered while any track
// channel (passive/active contact, ESM RF, or visual) remains valid.
func (p *PeriscopeState) UpdateLock(dt float64, player *world.Entity, sonar *SonarState, esm *ESMState, entities []*world.Entity, weather world.Weather, gameTime float64) {
	if p == nil || !p.Locked() || player == nil || p.Sheared {
		return
	}
	brg, ok := PeriLockBearing(p.LockEntityID, player, sonar, esm, entities, p, weather, gameTime)
	if !ok {
		p.ClearLock()
		return
	}
	desired := normalizeRel180(AngleDiffSigned(brg, player.HeadingDeg))
	delta := AngleDiffSigned(desired, p.TrainRelDeg)
	maxStep := periLockSlewDegPerSec * dt
	if maxStep < 0.1 {
		maxStep = 0.1
	}
	if math.Abs(delta) <= maxStep {
		p.TrainRelDeg = desired
		return
	}
	if delta > 0 {
		p.TrainRelDeg = normalizeRel180(p.TrainRelDeg + maxStep)
	} else {
		p.TrainRelDeg = normalizeRel180(p.TrainRelDeg - maxStep)
	}
}

// PeriLockBearing returns the best true bearing to a locked platform while any
// channel still holds: sonar contact, recent ESM RF, or visual (raised optic +
// surface target inside optical horizon).
func PeriLockBearing(entityID string, player *world.Entity, sonar *SonarState, esm *ESMState, entities []*world.Entity, peri *PeriscopeState, weather world.Weather, gameTime float64) (brg float64, ok bool) {
	if entityID == "" || player == nil {
		return 0, false
	}
	var ent *world.Entity
	for _, e := range entities {
		if e != nil && e.ID == entityID {
			ent = e
			break
		}
	}
	if ent == nil || (!ent.Alive() && ent.Status != world.StatusSinking) {
		return 0, false
	}

	acoustic := false
	if sonar != nil {
		for i := range sonar.Contacts {
			c := &sonar.Contacts[i]
			if c.SourceEntityID == entityID && gameTime-c.LastUpdate < 60 {
				acoustic = true
				break
			}
		}
	}
	esmHit := esm != nil && esm.HasRecentRF(entityID, gameTime)
	visual := false
	if peri != nil && peri.MastUp() && ent.Kind == world.KindSurfaceShip {
		eye := EyeAboveWaterFt(player.DepthFt, peri.Extension)
		maxR := OpticalMaxRangeYd(eye, weather)
		if player.RangeYardsTo(ent) <= maxR {
			visual = true
		}
	}
	if !acoustic && !esmHit && !visual {
		return 0, false
	}
	return player.BearingDegTo(ent), true
}

// ZoomIn increases magnification (narrower FOV).
func (p *PeriscopeState) ZoomIn() {
	if p == nil || p.Sheared {
		return
	}
	if p.Zoom < PeriZoomCount-1 {
		p.Zoom++
	}
}

// ZoomOut decreases magnification (wider FOV).
func (p *PeriscopeState) ZoomOut() {
	if p == nil || p.Sheared {
		return
	}
	if p.Zoom > 0 {
		p.Zoom--
	}
}

// AdvanceMastMotion animates extension and enforces shear limits while exposed.
func (p *PeriscopeState) AdvanceMastMotion(dt, gameTime float64, player *world.Entity) (events []string, shearedNow bool) {
	if player == nil {
		return nil, false
	}
	player.EnsureDamage()
	if player.Damage.Destroyed(world.SysPeriscope) {
		p.Sheared = true
		p.Order = PeriMastStow
		p.Extension = 0
		p.ClearLock()
		return nil, false
	}
	if p.Sheared {
		p.Extension = 0
		p.Order = PeriMastStow
		p.ClearLock()
		return nil, false
	}

	exposed := p.Extension > 0.05
	depthOK := player.DepthFt <= world.ESMMastMaxDepthFt+0.5
	speedOK := math.Abs(player.SpeedKts) <= world.ESMMastMaxSpeedKts+0.05

	if exposed && (!depthOK || !speedOK) {
		critical := player.DepthFt > world.ESMMastMaxDepthFt+15 || math.Abs(player.SpeedKts) > world.ESMMastMaxSpeedKts+1.5
		if critical {
			p.shear(player)
			return []string{"PERISCOPE SHEARED — optic destroyed"}, true
		}
		if gameTime-p.LastWarnAt > 8 {
			p.LastWarnAt = gameTime
			if !depthOK {
				events = append(events, fmt.Sprintf("WARNING — periscope exposed below periscope depth (%.0f ft). Reduce depth or lower scope.", player.DepthFt))
			}
			if !speedOK {
				events = append(events, fmt.Sprintf("WARNING — periscope exposed above %.0f kn (%.0f kn). Reduce speed or lower scope.", world.ESMMastMaxSpeedKts, math.Abs(player.SpeedKts)))
			}
		}
	}

	switch p.Order {
	case PeriMastRaise:
		if ok, _ := CanRaisePeriscope(player); !ok && p.Extension < 0.05 {
			p.Order = PeriMastStow
		} else {
			p.Extension = math.Min(1, p.Extension+dt/periRaiseSec)
		}
	default:
		p.Extension = math.Max(0, p.Extension-dt/periLowerSec)
	}
	return events, false
}

func (p *PeriscopeState) shear(player *world.Entity) {
	p.Sheared = true
	p.Order = PeriMastStow
	p.Extension = 0
	p.ClearLock()
	if player != nil {
		player.EnsureDamage()
		player.Damage.Eff[world.SysPeriscope] = 0
		player.Damage.CancelRepair()
	}
}

func normalizeDeg360(d float64) float64 {
	d = math.Mod(d, 360)
	if d < 0 {
		d += 360
	}
	return d
}

func normalizeRel180(d float64) float64 {
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}
