package acoustics

import (
	"fmt"
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const (
	esmBearingBins      = 360
	esmContactMergeDeg  = 8.0
	esmAutoClassifyConf = 0.60 // RF equipment ID lock for MAST CLASS column
	esmIlluminateSafe   = 0.28 // green
	esmIlluminateCaution = 0.55 // yellow above; red above this
	esmDetectSNR         = 0.10 // min relative strength to register a spike
	esmContactRetainSec  = 60.0 // MAST table / chip retention after last RF hit
)

// ESMMastOrder is the commanded mast position.
type ESMMastOrder int

const (
	ESMMastStow ESMMastOrder = iota
	ESMMastRaise
)

// ESMState is the player's ESM mast + intercept console.
type ESMState struct {
	Order           ESMMastOrder
	Extension       float64 // 0..1
	LastWarnAt      float64 // rate-limit voice/text warnings
	Sheared         bool    // permanent loss this session (also SysESM destroyed)
	BearingHeat     [esmBearingBins]float64
	MaxIllumination float64 // 0..1 — how hard enemy radars paint own mast
	RFConfidence    map[string]float64 // emitter entity ID → RF ID confidence
	RFClass         map[string]string  // emitter entity ID → radar equipment name (e.g. MR-302 Rubka)
	LastRFBearing   map[string]float64 // emitter entity ID → true bearing at last RF hit (frozen between paints)
	LastRFAt        map[string]float64 // emitter entity ID → gameTime of last RF hit
	ChirpPending    bool               // main-beam intercept this tick — UI plays FX
	LastChirpAt     float64
}

func (e *ESMState) ensure() {
	if e.RFConfidence == nil {
		e.RFConfidence = map[string]float64{}
	}
	if e.RFClass == nil {
		e.RFClass = map[string]string{}
	}
	if e.LastRFBearing == nil {
		e.LastRFBearing = map[string]float64{}
	}
	if e.LastRFAt == nil {
		e.LastRFAt = map[string]float64{}
	}
}

// HasRecentRF is true while an emitter still belongs on the MAST intercept table.
func (e *ESMState) HasRecentRF(sourceID string, gameTime float64) bool {
	if e == nil || sourceID == "" {
		return false
	}
	e.ensure()
	at, ok := e.LastRFAt[sourceID]
	return ok && gameTime-at <= esmContactRetainSec
}

// FrozenRFBearing returns the last intercepted true bearing for chips/table.
func (e *ESMState) FrozenRFBearing(sourceID string, fallback float64) float64 {
	if e == nil || sourceID == "" {
		return fallback
	}
	e.ensure()
	if brg, ok := e.LastRFBearing[sourceID]; ok {
		return brg
	}
	return fallback
}

// RFEquipmentClass returns the locked radar equipment name for an emitter, if any.
func (e *ESMState) RFEquipmentClass(sourceID string) string {
	if e == nil || sourceID == "" {
		return ""
	}
	e.ensure()
	return e.RFClass[sourceID]
}

// SecondsSinceRF is age of the last intercept (or a large sentinel if never).
func (e *ESMState) SecondsSinceRF(sourceID string, gameTime float64) float64 {
	if e == nil || sourceID == "" {
		return esmContactRetainSec + 1
	}
	e.ensure()
	at, ok := e.LastRFAt[sourceID]
	if !ok {
		return esmContactRetainSec + 1
	}
	return math.Max(0, gameTime-at)
}

// MastUp is true when the antenna is sufficiently extended to receive.
func (e *ESMState) MastUp() bool {
	return e != nil && !e.Sheared && e.Extension >= 0.95
}

// MastMoving reports raise/lower in progress.
func (e *ESMState) MastMoving() bool {
	if e == nil || e.Sheared {
		return false
	}
	if e.Order == ESMMastRaise && e.Extension < 1 {
		return true
	}
	if e.Order == ESMMastStow && e.Extension > 0 {
		return true
	}
	return false
}

// CanRaiseESM reports whether depth/speed allow mast raise.
func CanRaiseESM(player *world.Entity) (ok bool, reason string) {
	if player == nil {
		return false, "No ownship."
	}
	player.EnsureDamage()
	if player.Damage.Destroyed(world.SysESM) {
		return false, "ESM mast destroyed — beyond repair."
	}
	if player.DepthFt > world.ESMMastMaxDepthFt+0.5 {
		return false, fmt.Sprintf("Too deep — ESM mast requires ≤%.0f ft (periscope depth).", world.ESMMastMaxDepthFt)
	}
	spd := math.Abs(player.SpeedKts)
	if spd > world.ESMMastMaxSpeedKts+0.05 {
		return false, fmt.Sprintf("Too fast — ESM mast requires ≤%.0f kn.", world.ESMMastMaxSpeedKts)
	}
	return true, ""
}

// OrderRaiseESM begins raising if conditions allow.
func (e *ESMState) OrderRaiseESM(player *world.Entity) (ok bool, msg string) {
	e.ensure()
	if e.Sheared || (player != nil && player.Damage.Destroyed(world.SysESM)) {
		e.Sheared = true
		return false, "ESM mast destroyed — beyond repair."
	}
	if ok, reason := CanRaiseESM(player); !ok {
		return false, reason
	}
	e.Order = ESMMastRaise
	return true, "Raising ESM mast."
}

// OrderLowerESM begins stowing.
func (e *ESMState) OrderLowerESM() string {
	e.ensure()
	e.Order = ESMMastStow
	return "Lowering ESM mast."
}

// AdvanceMastMotion animates extension and enforces shear limits while up/moving.
// Returns events: warnings and shear messages.
func (e *ESMState) AdvanceMastMotion(dt, gameTime float64, player *world.Entity) (events []string, shearedNow bool) {
	e.ensure()
	if player == nil {
		return nil, false
	}
	player.EnsureDamage()
	if player.Damage.Destroyed(world.SysESM) {
		e.Sheared = true
		e.Order = ESMMastStow
		e.Extension = 0
		return nil, false
	}
	if e.Sheared {
		e.Extension = 0
		e.Order = ESMMastStow
		return nil, false
	}

	switch e.Order {
	case ESMMastRaise:
		if ok, _ := CanRaiseESM(player); !ok && e.Extension < 0.05 {
			e.Order = ESMMastStow
		} else {
			e.Extension = math.Min(1, e.Extension+dt/world.ESMMastRaiseSec)
		}
	default:
		e.Extension = math.Max(0, e.Extension-dt/world.ESMMastLowerSec)
	}
	return events, false
}

// UpdateESM processes search-radar intercepts while the mast is up.
// weather affects receive strength and own-mast illumination.
// dt is the sim tick length (needed to catch narrow beams that dwell < 1 tick).
func UpdateESM(sonar *SonarState, esm *ESMState, player *world.Entity, emitters []*world.Entity, weather world.Weather, gameTime, dt float64, bathy *world.Bathymetry) {
	if esm == nil || sonar == nil || player == nil {
		return
	}
	esm.ensure()
	esm.MaxIllumination = 0
	if dt <= 0 {
		dt = 0.1
	}

	if !esm.MastUp() || player.Damage.Destroyed(world.SysESM) {
		// Clear intercept heat once the mast is no longer receiving.
		for i := range esm.BearingHeat {
			esm.BearingHeat[i] = 0
		}
		pruneStaleESMTracks(sonar, esm, gameTime)
		return
	}

	for i := range esm.BearingHeat {
		esm.BearingHeat[i] *= 0.88 // decay heat map
	}

	rxMul := weather.ESMReceiveFactor()
	illMul := weather.MastDetectFactor()

	for _, em := range emitters {
		if em == nil || !em.Alive() || em.Kind != world.KindSurfaceShip {
			continue
		}
		prof, ok := world.RadarBySignature(em.SignatureID)
		if !ok {
			continue
		}
		rangeYd := player.RangeYardsTo(em)
		if rangeYd < 1 || rangeYd > prof.MaxRangeYd*1.2 {
			continue
		}
		if horizonBlocked(bathy, player, em) {
			continue
		}
		brgTrue := player.BearingDegTo(em)
		brgRel := normalizeBearing(brgTrue - player.HeadingDeg)
		illuminates := world.RadarBeamPassed(em, gameTime, dt, em.BearingDegTo(player))

		// Soft range falloff; sqrt power so commercial nav radars remain visible in-beam.
		rangeFac := clamp01(1.0 - rangeYd/(prof.MaxRangeYd*1.05))
		rangeFac = math.Pow(rangeFac, 1.35)
		powerFac := clamp01(0.22 + 0.78*math.Sqrt(prof.PeakPowerKW/90.0))

		recv := rangeFac * powerFac * rxMul
		if illuminates {
			recv *= 1.0
		} else {
			// Close-in sidelobes still tickle ESM; far ones do not.
			side := 0.10 + 0.18*clamp01(1.0-rangeYd/6000.0)
			recv *= side
		}
		if recv >= esmDetectSNR {
			bin := int(brgRel) % esmBearingBins
			if bin < 0 {
				bin += esmBearingBins
			}
			if recv > esm.BearingHeat[bin] {
				esm.BearingHeat[bin] = math.Min(1, recv)
			}
			for d := -2; d <= 2; d++ {
				b := (bin + d + esmBearingBins) % esmBearingBins
				fall := 1 - math.Abs(float64(d))/3
				v := recv * fall * 0.7
				if v > esm.BearingHeat[b] {
					esm.BearingHeat[b] = math.Min(1, v)
				}
			}
			// Contact / chip / LAST SGNL only refresh on main-beam paints.
			if illuminates {
				mergeESMContact(sonar, esm, em, &prof, brgTrue, rangeYd, recv, gameTime)
				if gameTime-esm.LastChirpAt >= 0.4 {
					esm.ChirpPending = true
					esm.LastChirpAt = gameTime
				}
			}
		}

		if illuminates {
			detectFrac := rangeYd / (prof.MastDetectYd * illMul)
			ill := clamp01(1.2 - detectFrac)
			ill *= clamp01(0.5 + powerFac*0.5)
			if ill > esm.MaxIllumination {
				esm.MaxIllumination = ill
			}
		}
	}
	pruneStaleESMTracks(sonar, esm, gameTime)
}

func pruneStaleESMTracks(sonar *SonarState, esm *ESMState, gameTime float64) {
	if esm == nil {
		return
	}
	esm.ensure()
	for id, at := range esm.LastRFAt {
		if gameTime-at > esmContactRetainSec {
			delete(esm.LastRFAt, id)
			delete(esm.LastRFBearing, id)
			delete(esm.RFConfidence, id)
			delete(esm.RFClass, id)
		}
	}
	if sonar == nil {
		return
	}
	dst := sonar.Contacts[:0]
	for i := range sonar.Contacts {
		c := sonar.Contacts[i]
		if !containsToken(c.DetectedBy, "esm") && c.DetectedBy != "esm" {
			dst = append(dst, c)
			continue
		}
		last, ok := esm.LastRFAt[c.SourceEntityID]
		age := gameTime - c.LastUpdate
		if ok {
			age = gameTime - last
		}
		if age > esmContactRetainSec {
			// Drop pure ESM tracks; keep fused acoustic tracks without the esm tag.
			if c.DetectedBy == "esm" {
				continue
			}
			c.DetectedBy = stripToken(c.DetectedBy, "esm")
		}
		dst = append(dst, c)
	}
	sonar.Contacts = dst
}

func stripToken(s, tok string) string {
	parts := splitPlus(s)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != tok {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return ""
	}
	res := out[0]
	for i := 1; i < len(out); i++ {
		res += "+" + out[i]
	}
	return res
}

func mergeESMContact(sonar *SonarState, esm *ESMState, em *world.Entity, prof *world.RadarProfile, brgTrue, rangeYd, recv, gameTime float64) {
	// Main-beam paints accumulate RF ID quickly (a few sweeps → lock).
	esm.RFConfidence[em.ID] = math.Min(1, esm.RFConfidence[em.ID]+math.Max(0.14, 0.25*recv))
	esm.LastRFBearing[em.ID] = brgTrue
	esm.LastRFAt[em.ID] = gameTime
	rfConf := esm.RFConfidence[em.ID]
	// Lock radar-equipment type only — never hull/ship class, and never ConfirmedClass.
	if rfConf >= esmAutoClassifyConf {
		esm.RFClass[em.ID] = prof.Name
	}

	// Prefer merging onto an existing sonar contact with same source or close bearing.
	idx := -1
	for i := range sonar.Contacts {
		c := &sonar.Contacts[i]
		if c.SourceEntityID == em.ID {
			idx = i
			break
		}
	}
	if idx < 0 {
		for i := range sonar.Contacts {
			c := &sonar.Contacts[i]
			if math.Abs(shortestAngle(c.BearingDeg, brgTrue)) <= esmContactMergeDeg {
				idx = i
				break
			}
		}
	}

	if idx >= 0 {
		c := &sonar.Contacts[idx]
		c.LastUpdate = gameTime
		c.ListenTime += 0.1
		// Freeze plotted bearing between RF hits — only refresh on this signal.
		c.BearingDeg = brgTrue
		c.UncBearingDeg = math.Max(1.5, c.UncBearingDeg*0.92)
		if c.DetectedBy == "" || c.DetectedBy == "passive" {
			c.DetectedBy = "passive+esm"
		} else if c.DetectedBy == "active" {
			c.DetectedBy = "active+esm"
		} else if c.DetectedBy != "esm" && !containsToken(c.DetectedBy, "esm") {
			c.DetectedBy = c.DetectedBy + "+esm"
		}
		if c.SourceEntityID == "" {
			c.SourceEntityID = em.ID
		}
		c.SNR = math.Max(c.SNR, recv*40)
		if rangeYd > 0 && (c.EstimatedRangeYd <= 0 || c.UncRangeYd > rangeYd*0.25) {
			if c.EstimatedRangeYd < 100 {
				c.EstimatedRangeYd = rangeYd * (0.7 + 0.3*(1-recv))
				c.UncRangeYd = rangeYd * 0.45
			}
		}
		return
	}

	sonar.contactSeq++
	c := Contact{
		ID:               fmt.Sprintf("E%02d", sonar.contactSeq),
		BearingDeg:       brgTrue,
		EstimatedRangeYd: 0, // bearing-only until fused
		UncBearingDeg:    6,
		UncRangeYd:       0,
		SNR:              recv * 40,
		SourceEntityID:   em.ID,
		Kind:             world.KindSurfaceShip,
		DetectedBy:       "esm",
		LastUpdate:       gameTime,
		FirstSeen:        gameTime,
		ListenTime:       0.1,
	}
	sonar.Contacts = append(sonar.Contacts, c)
}

func containsToken(s, tok string) bool {
	for _, p := range splitPlus(s) {
		if p == tok {
			return true
		}
	}
	return false
}

func splitPlus(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '+' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	return out
}

func normalizeBearing(d float64) float64 {
	for d < 0 {
		d += 360
	}
	for d >= 360 {
		d -= 360
	}
	return d
}

func shortestAngle(a, b float64) float64 {
	d := b - a
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}

// IlluminationBand returns 0=safe(green), 1=caution(yellow), 2=critical(red).
func IlluminationBand(v float64) int {
	switch {
	case v >= esmIlluminateCaution:
		return 2
	case v >= esmIlluminateSafe:
		return 1
	default:
		return 0
	}
}

// EnemyRadarDetectsMast is true when a ship's search radar currently has a solid
// paint on the player's raised ESM/COMM/periscope mast (used by AI).
func EnemyRadarDetectsMast(ship, player *world.Entity, weather world.Weather, esm *ESMState, comm *COMMState, peri *PeriscopeState, gameTime float64, bathy *world.Bathymetry) bool {
	if ship == nil || player == nil {
		return false
	}
	mastUp := (esm != nil && esm.MastUp()) || (comm != nil && comm.MastUp()) || (peri != nil && peri.MastUp())
	if !mastUp {
		return false
	}
	if !ship.Alive() || ship.Kind != world.KindSurfaceShip {
		return false
	}
	prof, ok := world.RadarBySignature(ship.SignatureID)
	if !ok {
		return false
	}
	if !world.RadarBeamPassed(ship, gameTime, 0.1, ship.BearingDegTo(player)) {
		return false
	}
	if horizonBlocked(bathy, ship, player) {
		return false
	}
	rangeYd := ship.RangeYardsTo(player)
	maxYd := prof.MastDetectYd * weather.MastDetectFactor()
	return rangeYd <= maxYd
}

// EnemyRadarDetectsSurface is true when ship search radar has a solid paint on a
// surface contact (hull, not a thin mast). Used by ally/enemy surface AI.
func EnemyRadarDetectsSurface(ship, target *world.Entity, gameTime, dt float64, bathy *world.Bathymetry) bool {
	if ship == nil || target == nil {
		return false
	}
	if !ship.Alive() || ship.Kind != world.KindSurfaceShip {
		return false
	}
	if !target.Alive() || target.Kind != world.KindSurfaceShip {
		return false
	}
	prof, ok := world.RadarBySignature(ship.SignatureID)
	if !ok {
		return false
	}
	if dt <= 0 {
		dt = 0.1
	}
	if !world.RadarBeamPassed(ship, gameTime, dt, ship.BearingDegTo(target)) {
		return false
	}
	if horizonBlocked(bathy, ship, target) {
		return false
	}
	return ship.RangeYardsTo(target) <= prof.MaxRangeYd
}

func horizonBlocked(bathy *world.Bathymetry, a, b *world.Entity) bool {
	if bathy == nil || !bathy.Valid() || a == nil || b == nil {
		return false
	}
	return bathy.HorizonBlocked(a.X, a.Y, b.X, b.Y)
}
