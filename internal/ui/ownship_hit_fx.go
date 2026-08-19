package ui

import (
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ssn688/sim/internal/render"
)

const (
	ownshipHitVignetteDur = 850 * time.Millisecond
	ownshipHitShakeDur    = 800 * time.Millisecond
	ownshipHitShakeAmpPx  = 30.0 // vertical kick peak (up/down)
	ownshipHitShakeKicks  = 9
	ownshipHitVignetteMax = 72  // peak edge alpha (was too opaque)
	dcTabBlinkPeriod      = 400 * time.Millisecond
)

// isOwnshipDamageFXEvent reports events that should flash/shake and ping the DC tab.
func isOwnshipDamageFXEvent(ev string) bool {
	switch {
	case strings.HasPrefix(ev, "OWN SHIP HIT"),
		strings.HasPrefix(ev, "OWN SHIP CRITICAL"),
		strings.HasPrefix(ev, "PLAYER SUBMARINE FATAL"),
		strings.HasPrefix(ev, "PLAYER SUBMARINE LOST"):
		return true
	default:
		return false
	}
}

func (a *App) triggerOwnshipHitFX() {
	now := time.Now()
	a.hitVignetteAt = now
	a.hitShakeAt = now
	if a.CurrentScreen != ScreenDamage {
		a.dcTabAlert = true
	}
}

func (a *App) clearDCTabAlertIfOnDamage() {
	if a.CurrentScreen == ScreenDamage {
		a.dcTabAlert = false
	}
}

func (a *App) hitShakeOffset() (dx, dy int) {
	if a.hitShakeAt.IsZero() {
		return 0, 0
	}
	elapsed := time.Since(a.hitShakeAt)
	if elapsed >= ownshipHitShakeDur || elapsed < 0 {
		return 0, 0
	}
	t := float64(elapsed) / float64(ownshipHitShakeDur) // 0→1
	prog := t * float64(ownshipHitShakeKicks)
	kick := int(prog)
	if kick >= ownshipHitShakeKicks {
		kick = ownshipHitShakeKicks - 1
	}
	local := prog - float64(kick) // 0→1 within kick
	pulse := math.Sin(local * math.Pi)
	amp := ownshipHitShakeAmpPx * pulse
	// Vertical only: alternate up / down at full ±30 px each kick.
	if kick%2 == 0 {
		dy = int(math.Round(amp))
	} else {
		dy = int(math.Round(-amp))
	}
	return 0, dy
}

func (a *App) hitVignetteAlpha() uint8 {
	if a.hitVignetteAt.IsZero() {
		return 0
	}
	elapsed := time.Since(a.hitVignetteAt)
	if elapsed >= ownshipHitVignetteDur || elapsed < 0 {
		return 0
	}
	t := float64(elapsed) / float64(ownshipHitVignetteDur)
	// Peak immediately on impact, then ease out.
	strength := 1 - t
	if strength < 0 {
		strength = 0
	}
	return uint8(math.Round(float64(ownshipHitVignetteMax) * strength * strength))
}

func (a *App) drawOwnshipHitVignette(screen *ebiten.Image) {
	alpha := a.hitVignetteAlpha()
	if alpha == 0 {
		return
	}
	w, h := render.ScreenW, render.ScreenH
	// Soft falloff via thin strips — no blur/shader (blur would cost extra full-frame passes).
	const strips = 14
	const stripPx = 7 // ~98 px from each edge
	for i := 0; i < strips; i++ {
		// Outer strips stronger; ease toward the center.
		u := float64(i) / float64(strips-1) // 0 at outer edge → 1 inward
		fall := (1 - u) * (1 - u)
		aBand := uint8(math.Round(float64(alpha) * fall))
		if aBand == 0 {
			continue
		}
		// Muted blood/CRT tint, not candy-red.
		clr := color.RGBA{R: 160, G: 36, B: 32, A: aBand}
		y0 := i * stripPx
		x0 := i * stripPx
		th := stripPx
		// Top / bottom full width; left / right only the inset span so corners don't double-stack harshly.
		render.FillRect(screen, 0, y0, w, th, clr)
		render.FillRect(screen, 0, h-y0-th, w, th, clr)
		innerH := h - 2*(y0+th)
		if innerH > 0 {
			render.FillRect(screen, x0, y0+th, th, innerH, clr)
			render.FillRect(screen, w-x0-th, y0+th, th, innerH, clr)
		}
	}
}

func (a *App) drawGameWithHitFX(screen *ebiten.Image) {
	ox, oy := a.hitShakeOffset()
	if ox == 0 && oy == 0 {
		a.drawGame(screen)
		a.drawOwnshipHitVignette(screen)
		return
	}
	// Offscreen blit only while shaking (~800 ms) — no steady-state cost.
	if a.hitShakeBuf == nil {
		a.hitShakeBuf = ebiten.NewImage(render.ScreenW, render.ScreenH)
	}
	a.hitShakeBuf.Fill(render.ColorBG)
	a.drawGame(a.hitShakeBuf)
	screen.Fill(render.ColorBG)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(ox), float64(oy))
	screen.DrawImage(a.hitShakeBuf, op)
	a.drawOwnshipHitVignette(screen)
}

func (a *App) dcTabBlinkOn() bool {
	if !a.dcTabAlert {
		return false
	}
	return (time.Now().UnixNano()/int64(dcTabBlinkPeriod))%2 == 0
}
