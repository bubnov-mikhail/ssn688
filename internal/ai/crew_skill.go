package ai

import (
	"math"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/world"
)

// UpdateCrewTrack refreshes hunter.Track from truth when the player is sensed.
// Skill gates classification confidence, localization noise, and TMA convergence.
func UpdateCrewTrack(hunter, player *world.Entity, detected, active bool, peakSNR, gameTime, dt float64) {
	if hunter == nil || player == nil || !hunter.Alive() {
		return
	}
	tr := &hunter.Track
	s := hunter.CrewSkill01()

	if !detected {
		if hunter.AIProsecuting && tr.Valid {
			// Freeze DATUM; don't zero HoldSec — passive flicker was blocking ClassConf rise.
			tr.ClassConf *= math.Pow(0.995, math.Max(dt, 0.05)*10)
			if tr.ClassConf < 0.08 {
				tr.ClassConf = 0.08
			}
			return
		}
		tr.HoldSec = 0
		tr.ClassConf *= math.Pow(0.97, math.Max(dt, 0.05)*10) // fade ~3%/0.1s
		if tr.ClassConf < 0.05 {
			tr.ClassConf = 0
			tr.Valid = false
		}
		return
	}

	fresh := !tr.Valid || tr.HoldSec <= 0
	tr.HoldSec += dt
	tr.UpdatedAt = gameTime
	tr.Valid = true

	// --- Classification (harmonics / contact ID trust) ---
	// Green crews barely accumulate ClassConf; veterans lock quickly with SNR.
	snrFac := clamp01((peakSNR - 4) / 18)
	if active {
		snrFac = clamp01(snrFac + 0.35)
	}
	// Rise rate: ~0.01/s at skill 0 → ~0.35/s at skill 1 (with good SNR).
	rise := (0.012 + 0.34*s) * (0.25 + 0.75*snrFac) * dt
	// Green need long dwell before they trust the contact.
	dwellGate := 1.0
	needHold := 55*(1-s) + 4*s
	if tr.HoldSec < needHold {
		dwellGate = tr.HoldSec / math.Max(needHold, 1)
	}
	tr.ClassConf = clamp01(tr.ClassConf + rise*dwellGate)

	// --- Localization (position) ---
	trueR := hunter.RangeYardsTo(player)
	brgSigma := 38*(1-s) + 1.2*s // degrees 1σ
	rngFrac := 0.60*(1-s) + 0.035*s
	n1 := pseudoNoise(hunter.ID, gameTime, 1)
	n2 := pseudoNoise(hunter.ID, gameTime, 2)
	measBrg := hunter.BearingDegTo(player) + n1*brgSigma
	measRng := trueR * (1 + n2*rngFrac)
	if measRng < 200 {
		measRng = 200
	}
	rad := measBrg * math.Pi / 180
	measX := hunter.X + math.Sin(rad)*measRng
	measY := hunter.Y + math.Cos(rad)*measRng

	posAlpha := 0.06 + 0.55*s // veterans snap; green lags
	if fresh {
		tr.X, tr.Y = measX, measY
	} else {
		tr.X += (measX - tr.X) * posAlpha
		tr.Y += (measY - tr.Y) * posAlpha
	}

	// --- TMA (course / speed) ---
	tmaAlpha := (0.015 + 0.28*s) * clamp01(tr.ClassConf+0.15)
	if s < 0.2 {
		tmaAlpha *= 0.25 // near-zero TMA for green
	}
	crsNoise := (45*(1-s) + 2*s) * pseudoNoise(hunter.ID, gameTime, 3)
	spdNoise := (10*(1-s) + 0.4*s) * pseudoNoise(hunter.ID, gameTime, 4)
	measCrs := player.HeadingDeg + crsNoise
	measSpd := player.SpeedKts + spdNoise
	if measSpd < 0 {
		measSpd = 0
	}
	tr.CourseDeg = lerpAngleDeg(tr.CourseDeg, measCrs, tmaAlpha)
	tr.SpeedKts += (measSpd - tr.SpeedKts) * tmaAlpha

	// Depth estimate
	depthSigma := 120*(1-s) + 8*s
	measDepth := player.DepthFt + pseudoNoise(hunter.ID, gameTime, 5)*depthSigma
	if measDepth < 40 {
		measDepth = 40
	}
	if measDepth > 900 {
		measDepth = 900
	}
	depthAlpha := 0.05 + 0.4*s
	tr.DepthFt += (measDepth - tr.DepthFt) * depthAlpha
	if tr.DepthFt < 40 {
		tr.DepthFt = 40
	}
}

// TrackClassified reports whether the crew trusts the contact enough to prosecute.
func TrackClassified(hunter *world.Entity) bool {
	if hunter == nil || !hunter.Track.Valid {
		return false
	}
	s := hunter.CrewSkill01()
	// Green need ~0.58 confidence; veterans prosecute from ~0.22.
	gate := 0.58 - 0.36*s
	return hunter.Track.ClassConf >= gate
}

// TrackWeaponRelease is true when held contact quality allows shooting.
// At Weapons Free a solid acoustic track is enough; lower DEFCON needs ID confidence.
func TrackWeaponRelease(hunter *world.Entity) bool {
	if hunter == nil || !hunter.Track.Valid {
		return false
	}
	if TrackClassified(hunter) {
		return true
	}
	if !hunter.CanDefconAttack() {
		return false
	}
	return hunter.Track.HoldSec >= 4 && hunter.Track.ClassConf >= 0.10
}

// TrackAimEntity returns a ghost aim point from the crew track, or truth if no track.
func TrackAimEntity(hunter, truth *world.Entity) *world.Entity {
	if hunter != nil && hunter.Track.Valid && hunter.Track.ClassConf > 0.08 {
		side := world.SideEnemy
		if truth != nil {
			side = truth.Side
		}
		return hunter.Track.GhostTarget("ai-aim-"+hunter.ID, side)
	}
	return truth
}

// WireGuideGain returns mid-course wire steer strength (0..~0.55).
func WireGuideGain(skill01 float64) float64 {
	return 0.06 + 0.42*clamp01(skill01)
}

// WireGuideNoiseDeg is 1σ heading noise applied each steer tick.
func WireGuideNoiseDeg(skill01 float64) float64 {
	return 22*(1-clamp01(skill01)) + 0.8*clamp01(skill01)
}

// WireHandoffAgeSec — when to cut wire / enable seeker.
func WireHandoffAgeSec(skill01 float64) float64 {
	// Green cut early (lost solution) or hold poorly; veterans wire-guide longer.
	return 28 + 40*clamp01(skill01)
}

func PeakSNRForAI(model acoustics.Model, hunter, player *world.Entity, active bool) float64 {
	if hunter == nil || player == nil {
		return 0
	}
	if active {
		return model.PeakActiveSNR(hunter, player, 0.7)
	}
	return model.PeakPassiveSNR(hunter, player)
}

func pseudoNoise(id string, gameTime float64, salt int) float64 {
	// Deterministic hash → approx Uniform[-1,1] without a Rand.
	h := hashID(id)*73856093 ^ salt*19349663 ^ int(gameTime*37)
	if h < 0 {
		h = -h
	}
	u := float64(h%2000)/1000.0 - 1.0 // [-1,1)
	return u
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func lerpAngleDeg(from, to, a float64) float64 {
	if a <= 0 {
		return from
	}
	if a >= 1 {
		return normalizeHead(to)
	}
	d := shortestRel(to - from)
	return normalizeHead(from + d*a)
}
