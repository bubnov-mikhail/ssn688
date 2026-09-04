package ai

import (
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const (
	// Underwater HE is audible across much of a coastal OP AREA (gameplay band).
	blastHearYd        = 32000.0
	blastHearMinPeakDB = 58.0
	torpedoAimConeDeg  = 42.0
	harpoonLaunchHearYd  = 16000.0
	harpoonLaunchPeakDB  = 92.0
	harpoonLaunchMinRecv = 55.0
)

// DefconContext carries per-tick inputs for alert escalation.
type DefconContext struct {
	Entities []*world.Entity
	Player   *world.Entity
	Zones    []world.RestrictedZone
	Torps    []*weapons.Torpedo
	Harpoons []*weapons.HarpoonMissile
	Model    acoustics.Model
	Bathy    *world.Bathymetry
	Weather  world.Weather
	ESM      *acoustics.ESMState
	COMM     *acoustics.COMMState
	Peri     *acoustics.PeriscopeState
	GameTime float64
	Dt       float64
}

// UpdateDefcon raises enemy alert levels from acoustic / geometry triggers.
func UpdateDefcon(ctx DefconContext) {
	if ctx.Player == nil || !ctx.Player.Alive() {
		return
	}
	for _, z := range ctx.Zones {
		if z.PlayerInside(ctx.Player) {
			raiseAllEnemies(ctx.Entities, world.DefconWeaponsFree)
			break
		}
	}
	for _, ent := range ctx.Entities {
		if ent == nil || !ent.Alive() || ent.Side != world.SideEnemy {
			continue
		}
		applyDefconProximity(ent, ctx.Player)
		if heardPlayerActiveSonar(ctx.Model.Env, ent, ctx.Player, ctx.GameTime) {
			ent.RaiseDefcon(world.DefconHostile)
		}
		if acoustics.EnemyRadarDetectsMast(ent, ctx.Player, ctx.Weather, ctx.ESM, ctx.COMM, ctx.Peri, ctx.GameTime, ctx.Bathy) {
			ent.RaiseDefcon(world.DefconHostile)
		}
	}
	checkPlayerTorpedoThreats(ctx.Entities, ctx.Torps)
	checkPlayerTorpedoLaunches(ctx)
	checkPlayerHarpoonThreats(ctx)
}

// NotifyDefconDetonation raises DEFCON when combatants hear an underwater blast
// and steers enemy + allied AI toward the acoustic datum.
func NotifyDefconDetonation(entities []*world.Entity, player *world.Entity, env acoustics.Environment, det *weapons.Detonation, gameTime float64) {
	if det == nil || det.SelfKill || det.SignalOnly || det.Intercepted {
		return
	}
	hit := det.Hit
	for _, ent := range entities {
		if !blastReactor(ent, player) {
			continue
		}
		if hit != nil && ent.ID == hit.ID {
			continue // victim already knows
		}
		if !heardExplosion(env, ent, det.X, det.Y, det.DepthFt) {
			continue
		}
		raiseDefconForHeardBlast(ent, hit)
		steerTowardBlastDatum(ent, det.X, det.Y, det.DepthFt, gameTime)
	}
}

func blastReactor(ent, player *world.Entity) bool {
	if ent == nil || !ent.Alive() {
		return false
	}
	if world.IsOwnship(ent, player) {
		return false
	}
	if ent.Side == world.SideNeutral {
		return false
	}
	return ent.Kind == world.KindSubmarine || ent.Kind == world.KindSurfaceShip
}

func raiseDefconForHeardBlast(ent *world.Entity, hit *world.Entity) {
	if ent == nil {
		return
	}
	// Any combat detonation in earshot → at least Hostile; friendly/own-side kill → Weapons Free.
	ent.RaiseDefcon(world.DefconHostile)
	if hit == nil {
		return
	}
	sameSide := hit.Side == ent.Side
	if sameSide || hit.Side == world.SideNeutral {
		ent.RaiseDefcon(world.DefconWeaponsFree)
	}
	// Enemy hears allied/player hull hit → Weapons Free.
	if ent.Side == world.SideEnemy && (hit.Side == world.SidePlayer) {
		ent.RaiseDefcon(world.DefconWeaponsFree)
	}
	// Ally hears hostile hull hit → Weapons Free.
	if world.IsFriendly(ent) && hit.Side == world.SideEnemy {
		ent.RaiseDefcon(world.DefconWeaponsFree)
	}
}

// steerTowardBlastDatum seeds a weak crew track on the blast and commits AI to investigate.
func steerTowardBlastDatum(ent *world.Entity, x, y, depthFt, gameTime float64) {
	if ent == nil || !shouldSeekBlastDatum(ent, gameTime) {
		return
	}
	dist := math.Hypot(ent.X-x, ent.Y-y)
	s := ent.CrewSkill01()
	// Localization error grows with range; veterans stay closer to truth.
	sigma := (0.12 + 0.22*(1-s)) * dist
	if sigma < 200 {
		sigma = 200
	}
	if sigma > 4500 {
		sigma = 4500
	}
	n1 := pseudoNoise(ent.ID, gameTime, 21)
	n2 := pseudoNoise(ent.ID, gameTime, 22)
	estX := x + (n1*2-1)*sigma
	estY := y + (n2*2-1)*sigma
	estDepth := depthFt
	if estDepth < 40 {
		// Surface / shallow splash — keep a workable ASW search depth cue.
		if ent.Kind == world.KindSubmarine {
			estDepth = 120
		} else {
			estDepth = 60
		}
	}
	ent.Track = world.AITrack{
		Valid: true, X: estX, Y: estY, DepthFt: estDepth,
		CourseDeg: 0, SpeedKts: 0,
		ClassConf: 0.20 + 0.12*s, HoldSec: 2.0, UpdatedAt: gameTime,
	}
	ent.AIProsecuting = true
	ent.AILostContactSec = 0
	ent.AIEngageCooldownUntil = 0
	markRouteInterrupted(ent)
	ent.EnsureDamage()
	ent.AIState = "DATUM"
	brg := ent.Track.BearingDegFrom(ent.X, ent.Y)
	if !ent.Damage.Destroyed(world.SysSteering) {
		ent.OrderedHead = brg
	}
	spd := 10.0
	if ent.Kind == world.KindSurfaceShip {
		spd = 16.0
	} else if ent.SignatureID == "yasen_m" || ent.SignatureID == "victor_iii" {
		spd = 9.0
	}
	ent.OrderedSpeed = math.Min(spd, ent.MaxSpeedKts())
	if ent.Kind == world.KindSubmarine && !ent.Damage.Destroyed(world.SysDepth) {
		d := estDepth
		if d < 140 {
			d = 140
		}
		if d > 300 {
			d = 300
		}
		// ASCM boats may already be climbing for a shot — don't yank them deep.
		if weapons.EnemyASCMMagazineFor(ent.SignatureID) == 0 || ent.DepthFt > 100 {
			ent.OrderedDepth = d
		}
	}
}

func shouldSeekBlastDatum(ent *world.Entity, gameTime float64) bool {
	if ent == nil {
		return false
	}
	if !ent.AIProsecuting || !ent.Track.Valid {
		return true
	}
	// Solid live solution — don't abandon for a distant boom.
	if ent.Track.ClassConf >= 0.35 && ent.Track.HoldSec >= 4 && gameTime-ent.Track.UpdatedAt < 20 {
		return false
	}
	return true
}

func applyDefconProximity(enemy, player *world.Entity) {
	if enemy.RangeYardsTo(player) < world.DefconTorpedoRangeYd {
		enemy.RaiseDefcon(world.DefconHostile)
	}
}

func heardPlayerActiveSonar(env acoustics.Environment, listener, player *world.Entity, gameTime float64) bool {
	return acoustics.HeardPlayerPing(env, listener, player, gameTime)
}

func heardExplosion(env acoustics.Environment, listener *world.Entity, x, y, depthFt float64) bool {
	if listener == nil {
		return false
	}
	dist := math.Hypot(listener.X-x, listener.Y-y)
	if dist > blastHearYd {
		return false
	}
	// ~20·log10 spherical spreading from a loud HE peak, plus layer/column loss.
	peak := 112.0 - 18.0*math.Log10(math.Max(dist, 200)/200)
	layer := env.LayerCrossingLoss(depthFt, listener.DepthFt)
	column := env.ColumnAttenuationDB(depthFt, listener.DepthFt, 60) * 0.35
	recv := peak - layer - column
	return recv >= blastHearMinPeakDB
}

// checkPlayerTorpedoThreats raises DEFCON when a live player fish is an evasion threat
// (including CM deployment). Unlike launch detection, this applies to the threatened
// unit directly — e.g. a surface DD under attack while submerged radio is blocked.
func checkPlayerTorpedoThreats(entities []*world.Entity, torps []*weapons.Torpedo) {
	for _, ent := range entities {
		if ent == nil || !ent.Alive() || ent.Side != world.SideEnemy {
			continue
		}
		if mostThreateningTorpedo(ent, torps) != nil {
			ent.RaiseDefcon(world.DefconWeaponsFree)
		}
	}
}

func checkPlayerTorpedoLaunches(ctx DefconContext) {
	player := ctx.Player
	if player == nil {
		return
	}
	for _, t := range ctx.Torps {
		if t == nil || !t.Alive || !t.EscalatesDefcon() || t.Side != player.Side {
			continue
		}
		if t.Age > ctx.Dt*1.6 {
			continue
		}
		if !playerTorpedoThreatensHostile(t, player, ctx.Entities) {
			continue
		}
		// Torpedo aimed at a combatant → weapons free for all combatants.
		// (No "submerged radio silence" pacifism: a DD under Mk48 fire must shoot back.)
		for _, ent := range ctx.Entities {
			if ent == nil || !ent.Alive() || !world.IsCombatant(ent) {
				continue
			}
			ent.RaiseDefcon(world.DefconWeaponsFree)
		}
	}
}

func playerTorpedoThreatensHostile(t *weapons.Torpedo, player *world.Entity, entities []*world.Entity) bool {
	aim := t.GyroCourseDeg
	for _, h := range entities {
		if h == nil || !h.Alive() || !world.IsCombatant(h) {
			continue
		}
		brg := bearingDeg(player.X, player.Y, h.X, h.Y)
		if math.Abs(shortestRel(aim-brg)) > torpedoAimConeDeg {
			continue
		}
		if player.RangeYardsTo(h) > world.DefconTorpedoRangeYd {
			continue
		}
		return true
	}
	return false
}

func raiseAllEnemies(entities []*world.Entity, level int) {
	for _, ent := range entities {
		if ent != nil && ent.Alive() && ent.Side == world.SideEnemy {
			ent.RaiseDefcon(level)
		}
	}
}

// NotifyHarpoonLaunchAcoustic raises DEFCON when enemies hear a Sub-Harpoon capsule launch.
func NotifyHarpoonLaunchAcoustic(entities []*world.Entity, env acoustics.Environment, launcher *world.Entity, gameTime float64) {
	if launcher == nil {
		return
	}
	for _, ent := range entities {
		if ent == nil || !ent.Alive() || ent.Side != world.SideEnemy {
			continue
		}
		if !heardHarpoonLaunch(env, ent, launcher.X, launcher.Y, launcher.DepthFt) {
			continue
		}
		ent.RaiseDefcon(world.DefconWeaponsFree)
	}
}

// checkPlayerHarpoonThreats raises DEFCON for enemies that hear the underwater
// booster / capsule egress, or that have an inbound cruise missile aimed at them.
func checkPlayerHarpoonThreats(ctx DefconContext) {
	player := ctx.Player
	if player == nil {
		return
	}
	env := ctx.Model.Env
	for _, h := range ctx.Harpoons {
		if h == nil || !h.Alive || h.Side != player.Side {
			continue
		}
		// Underwater egress: loud booster — surface and sub combatants can hear it.
		if h.Phase == weapons.HarpoonUnderwater || h.Age <= ctx.Dt*2.5 {
			sx, sy := h.X, h.Y
			depth := player.DepthFt
			if h.Phase != weapons.HarpoonUnderwater {
				sx, sy = h.LaunchX, h.LaunchY
			}
			for _, ent := range ctx.Entities {
				if ent == nil || !ent.Alive() || !world.IsCombatant(ent) {
					continue
				}
				if heardHarpoonLaunch(env, ent, sx, sy, depth) {
					ent.RaiseDefcon(world.DefconWeaponsFree)
				}
			}
			continue
		}
		// Airborne cruise: raise if missile is inbound toward this combatant.
		for _, ent := range ctx.Entities {
			if ent == nil || !ent.Alive() || !world.IsCombatant(ent) {
				continue
			}
			if !harpoonThreatensEntity(h, ent) {
				continue
			}
			ent.RaiseDefcon(world.DefconWeaponsFree)
		}
	}
}

func harpoonThreatensEntity(h *weapons.HarpoonMissile, ent *world.Entity) bool {
	if h == nil || ent == nil {
		return false
	}
	brg := bearingDeg(h.X, h.Y, ent.X, ent.Y)
	if math.Abs(shortestRel(h.HeadingDeg-brg)) > torpedoAimConeDeg {
		return false
	}
	dist := math.Hypot(ent.X-h.X, ent.Y-h.Y)
	if dist > world.DefconTorpedoRangeYd*2 {
		return false
	}
	// Closing: missile should be getting closer along its course.
	return dist < h.DestructRangeYd
}

func heardHarpoonLaunch(env acoustics.Environment, listener *world.Entity, x, y, srcDepthFt float64) bool {
	if listener == nil {
		return false
	}
	dist := math.Hypot(listener.X-x, listener.Y-y)
	if dist > harpoonLaunchHearYd {
		return false
	}
	peak := harpoonLaunchPeakDB * (1 - dist/harpoonLaunchHearYd)
	layer := env.LayerCrossingLoss(srcDepthFt, listener.DepthFt)
	column := env.ColumnAttenuationDB(srcDepthFt, listener.DepthFt, 80) * 0.35
	recv := peak - layer - column
	return recv >= harpoonLaunchMinRecv
}
