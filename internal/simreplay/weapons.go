package simreplay

import (
	"image/color"
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// SnapshotWeapons copies in-flight ordnance like the tactical DEBUG overlay.
func SnapshotWeapons(fc *weapons.FireControl, gameTime float64) ([]WeaponSnap, []FlashSnap) {
	if fc == nil {
		return nil, nil
	}
	out := make([]WeaponSnap, 0, 16)
	for _, t := range fc.ActiveTorpedoes {
		if t == nil || !t.Alive {
			continue
		}
		out = append(out, WeaponSnap{
			Kind:     WeaponTorpedo,
			Label:    torpedoDebugLabel(t),
			Side:     sideName(t.Side, t.Side == world.SidePlayer),
			X:        t.X,
			Y:        t.Y,
			Heading:  t.HeadingDeg,
			SpeedKts: t.SpeedKts,
			Alive:    true,
			WireCut:  t.WireCut,
		})
	}
	for _, h := range fc.ActiveHarpoons {
		if h == nil || !h.Alive {
			continue
		}
		ws := WeaponSnap{
			Kind:     WeaponHarpoon,
			Label:    weapons.ASCMDebugLabel(h.Variant, h.Phase, h.LockedTargetID != "", h.RadarOn),
			Side:     sideName(h.Side, h.Side == world.SidePlayer),
			X:        h.X,
			Y:        h.Y,
			Heading:  h.HeadingDeg,
			SpeedKts: h.SpeedKts,
			Alive:    true,
		}
		switch {
		case h.Phase == weapons.HarpoonUnderwater:
			ws.HarpoonUW = true
		case h.LockedTargetID != "":
			ws.HarpoonLock = true
		case h.RadarOn:
			ws.HarpoonRadar = true
		}
		out = append(out, ws)
	}
	for _, rbu := range fc.ActiveRBU {
		if rbu == nil || !rbu.Alive {
			continue
		}
		ax, ay := rbu.Pos(gameTime)
		out = append(out, WeaponSnap{
			Kind:    WeaponRBU,
			Label:   "RBU",
			Side:    "ENEMY",
			X:       ax,
			Y:       ay,
			X1:      rbu.X1,
			Y1:      rbu.Y1,
			Heading: bearingDeg(rbu.X0, rbu.Y0, rbu.X1, rbu.Y1),
			Alive:   true,
		})
	}
	for _, aroc := range fc.ActiveRastrub {
		if aroc == nil || !aroc.Alive {
			continue
		}
		ax, ay := aroc.Pos(gameTime)
		out = append(out, WeaponSnap{
			Kind:    WeaponRastrub,
			Label:   "RSTR",
			Side:    "ENEMY",
			X:       ax,
			Y:       ay,
			X1:      aroc.X1,
			Y1:      aroc.Y1,
			Heading: bearingDeg(aroc.X0, aroc.Y0, aroc.X1, aroc.Y1),
			Alive:   true,
		})
	}

	flashes := make([]FlashSnap, 0, 4)
	for _, f := range fc.DebugMapFlashes {
		if f.Until < gameTime {
			continue
		}
		flashes = append(flashes, FlashSnap{Label: f.Label, X: f.X, Y: f.Y})
	}
	return out, flashes
}

func torpedoDebugLabel(t *weapons.Torpedo) string {
	return weapons.TorpedoDebugLabel(t)
}

func sideName(s world.Side, isPlayer bool) string {
	if isPlayer {
		return "PLAYER"
	}
	switch s {
	case world.SideEnemy:
		return "ENEMY"
	case world.SideNeutral:
		return "NEUTRAL"
	default:
		return "ALLY"
	}
}

func bearingDeg(x0, y0, x1, y1 float64) float64 {
	dx := x1 - x0
	dy := y1 - y0
	if dx == 0 && dy == 0 {
		return 0
	}
	return math.Mod(math.Atan2(dx, dy)*180/math.Pi+360, 360)
}

// WeaponColor matches ui/minimap.go debug overlay.
func WeaponColor(w WeaponSnap) color.Color {
	switch w.Kind {
	case WeaponTorpedo:
		if w.Side == "PLAYER" {
			if w.WireCut {
				return color.RGBA{0, 180, 200, 255}
			}
			return render.ColorActive
		}
		return render.ColorDebugAttack
	case WeaponHarpoon:
		if w.Side == "PLAYER" {
			return color.RGBA{255, 140, 40, 255}
		}
		if w.HarpoonUW {
			return color.RGBA{200, 110, 40, 220}
		}
		return render.ColorDebugAttack
	case WeaponRBU:
		return color.RGBA{255, 90, 40, 255}
	case WeaponRastrub:
		return render.ColorAmber
	default:
		return render.ColorDebugAttack
	}
}

func WeaponTrailColor(w WeaponSnap) color.Color {
	switch w.Kind {
	case WeaponRBU:
		return color.RGBA{255, 120, 50, 140}
	case WeaponRastrub:
		return color.RGBA{255, 180, 40, 90}
	default:
		return color.RGBA{0, 0, 0, 0}
	}
}

// UnitDebugColor matches ui/minimap debugEntityColor.
func UnitDebugColor(u UnitSnap) color.Color {
	if !u.Alive {
		return render.ColorDebugInactive
	}
	switch u.Side {
	case "NEUTRAL":
		return color.RGBA{180, 180, 200, 255}
	case "PLAYER", "ALLY":
		switch u.AIState {
		case "INTERCEPT", "ATTACK", "FIRING", "SHADOW", "CLOSING", "OPENING", "TORPEDO_EVADE", "RBU", "RASTRUB", "SHIP_TUBE", "DATUM":
			return color.RGBA{80, 200, 255, 255}
		default:
			if u.Side == "PLAYER" {
				return render.ColorDebugPlayer
			}
			return color.RGBA{40, 160, 255, 255}
		}
	case "ENEMY":
		switch u.AIState {
		case "INTERCEPT", "ATTACK", "FIRING", "SHADOW", "CLOSING", "OPENING", "TORPEDO_EVADE", "RBU", "RASTRUB", "SHIP_TUBE":
			return render.ColorDebugAttack
		case "SEARCH", "PINGING", "ACTIVE_SEARCH", "PING_ALERT", "TRACKING", "RADAR_TRACK", "DATUM":
			return render.ColorDebugSearch
		default:
			return render.ColorDebugCalm
		}
	default:
		return render.ColorDebugCalm
	}
}
