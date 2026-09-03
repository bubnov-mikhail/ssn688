package weapons

import (
	"fmt"
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// ASCMVariant identifies anti-ship cruise missiles sharing HarpoonMissile physics.
type ASCMVariant int

const (
	ASCMHarpoon ASCMVariant = iota
	ASCMKlub
	ASCMOniks
	ASCMKalibr
)

// Open-source inspired gameplay bands (subsonic Klub/Kalibr, supersonic Oniks).
const (
	KlubMaxRangeNm    = 120.0
	KlubCruiseKts     = 520.0
	KlubUnderwaterSec = 4.0
	KlubMaxLaunchFt   = 130.0 // ~40 m — Klub-S tube launch envelope

	OniksMaxRangeNm    = 40.0
	OniksCruiseKts     = 1200.0 // ~Mach 2 sea-skimmer
	OniksUnderwaterSec = 2.5
	OniksTurnRateMul   = 1.6

	KalibrMaxRangeNm    = 70.0
	KalibrCruiseKts     = 500.0
	KalibrUnderwaterSec = 5.0
)

// ASCMCruiseKts returns cruise speed for a variant (save/load helper).
func ASCMCruiseKts(v ASCMVariant) float64 {
	return v.cruiseKts()
}

func (v ASCMVariant) cruiseKts() float64 {
	switch v {
	case ASCMKlub:
		return KlubCruiseKts
	case ASCMOniks:
		return OniksCruiseKts
	case ASCMKalibr:
		return KalibrCruiseKts
	default:
		return HarpoonCruiseKts
	}
}

func (v ASCMVariant) destructYd() float64 {
	switch v {
	case ASCMKlub:
		return KlubMaxRangeNm * world.YardsPerNM
	case ASCMOniks:
		return OniksMaxRangeNm * world.YardsPerNM
	case ASCMKalibr:
		return KalibrMaxRangeNm * world.YardsPerNM
	default:
		return HarpoonMaxRangeYd
	}
}

func (v ASCMVariant) underwaterSec() float64 {
	switch v {
	case ASCMKlub:
		return KlubUnderwaterSec
	case ASCMOniks:
		return OniksUnderwaterSec
	case ASCMKalibr:
		return KalibrUnderwaterSec
	default:
		return HarpoonUnderwaterSec
	}
}

func (v ASCMVariant) turnRateDegPerSec() float64 {
	r := HarpoonTurnRateDegPerSec
	if v == ASCMOniks {
		r *= OniksTurnRateMul
	}
	return r
}

// ASCMDebugBaseLabel is the tactical DEBUG tag for a cruise missile variant.
func ASCMDebugBaseLabel(v ASCMVariant) string {
	switch v {
	case ASCMKlub:
		return "KLUB"
	case ASCMOniks:
		return "ONKS"
	case ASCMKalibr:
		return "KLBR"
	default:
		return "HSM"
	}
}

// ASCMDebugLabel adds phase hints for DEBUG / replay overlays.
func ASCMDebugLabel(v ASCMVariant, phase HarpoonPhase, locked, radarOn bool) string {
	base := ASCMDebugBaseLabel(v)
	switch {
	case phase == HarpoonUnderwater:
		return base + " UW"
	case locked:
		return base + " LCK"
	case radarOn:
		return base + " RDR"
	default:
		return base
	}
}

// EnemyASCMMagazineFor returns default Klub/Oniks/Kalibr stowage by submarine class.
func EnemyASCMMagazineFor(signatureID string) int {
	switch signatureID {
	case "kilo":
		return 4 // Klub-S from 533 mm tubes (export typical salvo)
	case "yasen_m":
		return 16 // mixed UKSK load (8 Oniks + 8 Kalibr gameplay split)
	default:
		return 0
	}
}

// EnemyASCMMaxLaunchDepthFt is the shallowest depth band allowed for tube/VLS launch.
func EnemyASCMMaxLaunchDepthFt(signatureID string) float64 {
	switch signatureID {
	case "kilo":
		return KlubMaxLaunchFt
	case "yasen_m":
		return 90 // raised UKSK / periscope-depth snapshot
	default:
		return 0
	}
}

func enemyASCMVariant(sub *world.Entity, seq int) ASCMVariant {
	if sub == nil {
		return ASCMKlub
	}
	switch sub.SignatureID {
	case "kilo":
		return ASCMKlub
	case "yasen_m":
		if seq%2 == 0 {
			return ASCMOniks
		}
		return ASCMKalibr
	default:
		return ASCMKlub
	}
}

func (fc *FireControl) enemyASCMAmmo(sub *world.Entity) int {
	if fc == nil || sub == nil {
		return 0
	}
	if fc.EnemyASCMMag == nil {
		fc.EnemyASCMMag = map[string]int{}
	}
	if v, ok := fc.EnemyASCMMag[sub.ID]; ok {
		return v
	}
	n := EnemyASCMMagazineFor(sub.SignatureID)
	fc.EnemyASCMMag[sub.ID] = n
	return n
}

// EnemyASCMLeft returns remaining cruise missiles for a hostile submarine.
func (fc *FireControl) EnemyASCMLeft(subID string) int {
	if fc == nil || fc.EnemyASCMMag == nil {
		return 0
	}
	return fc.EnemyASCMMag[subID]
}

// SpawnEnemyASCM launches Klub / Oniks / Kalibr at a surface target.
func (fc *FireControl) SpawnEnemyASCM(sub, target *world.Entity) *HarpoonMissile {
	if fc == nil || sub == nil || !sub.Alive() || target == nil || !target.Alive() {
		return nil
	}
	if EnemyASCMMagazineFor(sub.SignatureID) == 0 {
		return nil
	}
	left := fc.enemyASCMAmmo(sub)
	if left <= 0 {
		return nil
	}
	fc.EnemyASCMMag[sub.ID] = left - 1

	variant := enemyASCMVariant(sub, fc.torpedoSeq+1)
	return fc.spawnASCM(sub, target, variant, false)
}

func (fc *FireControl) spawnASCM(sub, target *world.Entity, variant ASCMVariant, visibleWEPS bool) *HarpoonMissile {
	aim := target
	if sub.Track.Valid {
		aim = sub.Track.GhostTarget(target.ID, target.Side)
	}
	cruise := variant.cruiseKts()
	gyro := sub.BearingDegTo(aim)
	if course, ok := InterceptCourseDeg(
		aim.X-sub.X, aim.Y-sub.Y,
		aim.HeadingDeg, aim.SpeedKts, cruise,
	); ok {
		gyro = course
	}
	skill := sub.CrewSkill01()
	gyro += (1-skill) * 10 * (pseudoNoise(sub.ID, aim.ID) - 0.5) * 2
	gyro = normalizeAngle(gyro)

	fc.torpedoSeq++
	rad := sub.HeadingDeg * math.Pi / 180
	offset := 40.0
	radarYd := HarpoonRadarRangeYd(HarpoonSRCHMedium)
	destructYd := variant.destructYd()
	if destructYd > HarpoonDestructRangeYd(HarpoonDSTRLong) && variant == ASCMHarpoon {
		destructYd = HarpoonDestructRangeYd(HarpoonDSTRLong)
	}

	idPrefix := "ASCM"
	switch variant {
	case ASCMHarpoon:
		idPrefix = "HSM"
	case ASCMKlub:
		idPrefix = "KLUB"
	case ASCMOniks:
		idPrefix = "ONKS"
	case ASCMKalibr:
		idPrefix = "KLBR"
	}

	h := &HarpoonMissile{
		ID:              fmt.Sprintf("%s-%d", idPrefix, fc.torpedoSeq),
		ParentSubID:     sub.ID,
		Side:            sub.Side,
		LaunchX:         sub.X + math.Sin(rad)*offset,
		LaunchY:         sub.Y + math.Cos(rad)*offset,
		X:               sub.X + math.Sin(rad)*offset,
		Y:               sub.Y + math.Cos(rad)*offset,
		HeadingDeg:      gyro,
		ProgrammedHead:  gyro,
		SpeedKts:        HarpoonUnderwaterKts,
		CruiseKts:       cruise,
		Phase:           HarpoonUnderwater,
		UnderwaterLeft:  variant.underwaterSec(),
		BeamHalfDeg:     HarpoonBeamHalfDeg(HarpoonBeamWide),
		RadarRangeYd:    radarYd,
		DestructRangeYd: destructYd,
		Alive:           true,
		VisibleOnWEPS:   visibleWEPS,
		Variant:         variant,
	}
	fc.ActiveHarpoons = append(fc.ActiveHarpoons, h)
	return h
}

// SpawnAIHarpoon launches a Sub-Harpoon from an allied submarine magazine.
func (fc *FireControl) SpawnAIHarpoon(sub, target *world.Entity) *HarpoonMissile {
	if fc == nil || sub == nil || !sub.Alive() || target == nil || !target.Alive() {
		return nil
	}
	left := fc.allyHarpoonAmmo(sub)
	if left <= 0 {
		return nil
	}
	fc.AllyHarpoonMag[sub.ID] = left - 1
	return fc.spawnASCM(sub, target, ASCMHarpoon, false)
}
