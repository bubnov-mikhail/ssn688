package ai

import (
	"math"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

const (
	blastHearYd          = 12000.0
	blastHearMinPeakDB   = 72.0
	torpedoAimConeDeg    = 42.0
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
		if acoustics.EnemyRadarDetectsMast(ent, ctx.Player, ctx.Weather, ctx.ESM, ctx.COMM, ctx.Peri, ctx.GameTime) {
			ent.RaiseDefcon(world.DefconHostile)
		}
	}
	checkPlayerTorpedoThreats(ctx.Entities, ctx.Torps)
	checkPlayerTorpedoLaunches(ctx)
	checkPlayerHarpoonThreats(ctx)
}

// NotifyDefconDetonation raises DEFCON when enemies hear an underwater blast.
func NotifyDefconDetonation(entities []*world.Entity, env acoustics.Environment, det *weapons.Detonation, gameTime float64) {
	if det == nil || det.SelfKill {
		return
	}
	hit := det.Hit
	for _, ent := range entities {
		if ent == nil || !ent.Alive() || ent.Side != world.SideEnemy {
			continue
		}
		if !heardExplosion(env, ent, det.X, det.Y, det.DepthFt) {
			continue
		}
		if hit == nil {
			continue
		}
		switch hit.Side {
		case world.SideNeutral:
			ent.RaiseDefcon(world.DefconHostile)
		case world.SideEnemy:
			ent.RaiseDefcon(world.DefconWeaponsFree)
		}
	}
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
	peak := 95.0 * (1 - dist/blastHearYd)
	layer := env.LayerCrossingLoss(depthFt, listener.DepthFt)
	column := env.ColumnAttenuationDB(depthFt, listener.DepthFt, 60) * 0.4
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
	submerged := player.DepthFt > world.SubmergedDepthFt
	for _, t := range ctx.Torps {
		if t == nil || !t.Alive || t.Side != player.Side {
			continue
		}
		if t.Age > ctx.Dt*1.6 {
			continue
		}
		if !playerTorpedoThreatensHostile(t, player, ctx.Entities) {
			continue
		}
		for _, ent := range ctx.Entities {
			if ent == nil || !ent.Alive() || !world.IsCombatant(ent) {
				continue
			}
			if submerged && ent.Kind == world.KindSurfaceShip {
				continue // submerged launch — no radio warning to surface units
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
