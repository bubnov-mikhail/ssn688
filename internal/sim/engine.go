package sim

import (
	"math"
	"math/rand"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/ai"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

const TickRate = 10.0

const (
	tubeOpenTransientFreqHz  = 180.0
	tubeCloseTransientFreqHz = 140.0
	tubeOpenTransientDB      = 22.0
	tubeCloseTransientDB     = 14.0
	tubeTransientDurSec      = 5.0
	tubeTransientHearYd      = 6000.0
	enemyTubeOpenLeadSec     = 3.0
)

// Engine runs the simulation at fixed timestep.
type Engine struct {
	Clock             Clock
	Scenario          *world.Scenario
	Acoustics         acoustics.Model
	Sonar             acoustics.SonarState
	FireControl       weapons.FireControl
	Accum             float64
	Events            []string
	entityScratch     []*world.Entity
	torpedoEntityPool []world.Entity
	torpedoEntityPtrs []*world.Entity
	bioRng            *rand.Rand
}

func NewEngine(scenario *world.Scenario) *Engine {
	return &Engine{
		Clock:       NewClock(),
		Scenario:    scenario,
		Acoustics:   acoustics.NewModel(acoustics.DefaultEnvironment()),
		Sonar:       acoustics.NewSonarState(),
		FireControl: weapons.NewFireControl(),
	}
}

func (e *Engine) Update(realDT float64) {
	if e.Clock.Paused {
		return
	}
	e.Accum += realDT * e.Clock.TimeScale
	step := 1.0 / TickRate
	const maxTicksPerFrame = 6
	ticks := 0
	for e.Accum >= step && ticks < maxTicksPerFrame {
		e.tick(step)
		e.Accum -= step
		ticks++
	}
}

// VisualGameTime returns sub-tick interpolated simulation time for smooth UI motion.
func (e *Engine) VisualGameTime() float64 {
	if e == nil {
		return 0
	}
	if e.Clock.Paused {
		return e.Clock.GameTime
	}
	return e.Clock.GameTime + e.Accum
}

func (e *Engine) tick(dt float64) {
	e.Clock.Advance(dt)
	t := e.Clock.GameTime
	player := e.Scenario.Player

	player.Advance(dt)
	for _, ent := range e.Scenario.Entities {
		if ent.InWater() {
			ent.Advance(dt)
		}
	}
	e.expireTransientNoise(t)
	e.clampToSeafloor()
	e.finalizeSunkWrecks(t)

	ai.UpdateAllAI(e.Scenario.Entities, player, t, e.Acoustics, e.FireControl.ActiveTorpedoes)
	e.guideEnemyTorpedoes(player, t)
	e.tryEnemyTorpedoShots(player, t)

	e.FireControl.UpdateTubes(t)

	emitters := e.AcousticEmitters()
	e.Sonar.UpdateTowed(dt)
	if e.bioRng == nil {
		e.bioRng = rand.New(rand.NewSource(0xB10C0DE ^ int64(t*1000)))
	}
	acoustics.UpdateBiology(&e.Sonar, t, e.bioRng)
	acoustics.UpdatePassive(e.Acoustics, player, emitters, &e.Sonar, t)
	acoustics.FireActivePing(e.Acoustics, player, emitters, &e.Sonar, t)
	acoustics.ProcessActiveEchoes(e.Acoustics, player, emitters, &e.Sonar, t)

	shipTargets := e.AllEntities()
	alive := e.FireControl.ActiveTorpedoes[:0]
	for _, torp := range e.FireControl.ActiveTorpedoes {
		if torp == nil || !torp.Alive {
			continue
		}
		if det := torp.Advance(dt, t, shipTargets, e.torpedoLayerAtten); det != nil {
			e.handleDetonation(det, t)
		}
		if torp.Alive {
			alive = append(alive, torp)
		} else {
			e.FireControl.UnlinkDead(torp)
		}
	}
	e.FireControl.ActiveTorpedoes = alive

	e.Scenario.CheckObjectives()
	e.Acoustics.Env.UpdateLayerSurvey(t)
	if e.Scenario.Bathy != nil && e.Scenario.Bathy.Valid() {
		if d := e.Scenario.Bathy.DepthAtFt(player.X, player.Y); d > 0 {
			e.Acoustics.Env.BottomDepthFt = d
		}
	}

	if player.DepthFt > 1200 && player.Alive() {
		player.Status = world.StatusSunk
	}
}

func (e *Engine) expireTransientNoise(gameTime float64) {
	clearOne := func(ent *world.Entity) {
		if ent == nil || ent.TransientUntil <= 0 || gameTime < ent.TransientUntil {
			return
		}
		ent.TransientUntil = 0
		ent.TransientFreqHz = 0
		ent.TransientLevelDB = 0
	}
	clearOne(e.Scenario.Player)
	for _, ent := range e.Scenario.Entities {
		clearOne(ent)
	}
}

func (e *Engine) EmitTubeTransient(src *world.Entity, gameTime float64, opening bool) {
	if e == nil || src == nil {
		return
	}
	freq := tubeCloseTransientFreqHz
	level := tubeCloseTransientDB
	peak := 10.0
	kind := "tube_close"
	if opening {
		freq = tubeOpenTransientFreqHz
		level = tubeOpenTransientDB
		peak = 14.0
		kind = "tube_open"
	}
	src.TransientUntil = gameTime + tubeTransientDurSec
	src.TransientFreqHz = freq
	src.TransientLevelDB = level

	player := e.Scenario.Player
	if player == nil || src.ID == player.ID {
		return
	}
	dist := src.RangeYardsTo(player)
	if dist > tubeTransientHearYd {
		return
	}
	bearing := player.BearingDegTo(src)
	peak *= 1 - dist/(tubeTransientHearYd*1.2)
	if peak < 2 {
		peak = 2
	}
	acoustics.AddPassiveTransient(&e.Sonar, bearing, peak, 4.0, kind, freq, gameTime)
}

func (e *Engine) handleDetonation(det *weapons.Detonation, gameTime float64) {
	if det == nil {
		return
	}
	if det.SelfKill {
		e.Events = append(e.Events, "Torpedo self-destructed")
		return
	}
	player := e.Scenario.Player
	acoustics.ApplyDetonationDeaf(&e.Sonar, player, det.X, det.Y, gameTime, det.Hit)
	e.emitBlastTransient(player, det, gameTime)
	if det.Hit != nil && det.Hit.Alive() {
		e.beginSinking(det.Hit, gameTime)
		e.FireControl.OnPlatformLost(det.Hit.ID)
		if player != nil && det.Hit.ID == player.ID {
			e.Events = append(e.Events, "PLAYER SUBMARINE HIT - SINKING")
		} else {
			e.Events = append(e.Events, "Target destroyed: "+det.Hit.Name)
		}
	} else {
		e.Events = append(e.Events, "Underwater explosion")
	}
}

func (e *Engine) emitBlastTransient(player *world.Entity, det *weapons.Detonation, gameTime float64) {
	if player == nil || det == nil {
		return
	}
	dist := math.Hypot(player.X-det.X, player.Y-det.Y)
	const hearYd = 12000.0
	if dist > hearYd {
		return
	}
	bearing := math.Atan2(det.X-player.X, det.Y-player.Y) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	peak := 95 * (1 - dist/hearYd)
	if det.Hit != nil && det.Hit.Kind == world.KindSurfaceShip {
		peak *= 1.25
	}
	if peak < 10 {
		peak = 10
	}
	acoustics.AddPassiveTransient(&e.Sonar, bearing, peak, 6.5, "blast", 60, gameTime)
}

func (e *Engine) beginSinking(ent *world.Entity, gameTime float64) {
	if ent == nil {
		return
	}
	ent.Status = world.StatusSinking
	ent.OrderedSpeed = 0
	ent.SinkRateFPM = 40
	if ent.Kind == world.KindSurfaceShip {
		ent.SinkRateFPM = 25
	}
	ent.WreckNoiseUntil = gameTime + 90
}

func (e *Engine) finalizeSunkWrecks(gameTime float64) {
	bathy := e.Scenario.Bathy
	check := func(ent *world.Entity) {
		if ent == nil || ent.Status != world.StatusSinking {
			return
		}
		bottom := 2000.0
		if bathy != nil && bathy.Valid() {
			if d := bathy.DepthAtFt(ent.X, ent.Y); d > 0 {
				bottom = d
			}
		}
		if ent.DepthFt >= bottom-10 {
			ent.DepthFt = bottom - 10
			ent.Status = world.StatusSunk
			ent.SinkRateFPM = 0
		}
		if gameTime > ent.WreckNoiseUntil && ent.WreckNoiseUntil > 0 {
			ent.WreckNoiseUntil = 0
		}
	}
	check(e.Scenario.Player)
	for _, ent := range e.Scenario.Entities {
		check(ent)
	}
}

// torpedoLayerAtten is seeker path loss across the water column (thermocline, etc.).
func (e *Engine) torpedoLayerAtten(srcDepthFt, dstDepthFt float64) float64 {
	if e == nil {
		return 0
	}
	env := e.Acoustics.Env
	// Seeker band ~ few hundred Hz: layer crossing + partial column scatter.
	return env.LayerCrossingLoss(srcDepthFt, dstDepthFt) +
		env.ColumnAttenuationDB(srcDepthFt, dstDepthFt, 350)*0.55
}

func (e *Engine) guideEnemyTorpedoes(player *world.Entity, gameTime float64) {
	if player == nil || !player.Alive() {
		return
	}
	for _, torp := range e.FireControl.ActiveTorpedoes {
		if torp == nil || !torp.Alive || torp.Side != world.SideEnemy {
			continue
		}
		if torp.WireCut || torp.Mode != weapons.ModeWire {
			continue
		}
		// Wire steer toward player with small lag / noise.
		brg := bearingDeg(torp.X, torp.Y, player.X, player.Y)
		diff := shortest(torp.OrderedHead, brg)
		torp.OrderedHead += clampAngle(diff*0.35, -8, 8)
		torp.RunDepthFt += (player.DepthFt - torp.RunDepthFt) * 0.2
		// Enable seeker mid-run.
		if torp.Age > 25 && !torp.SeekerOn {
			torp.SeekerOn = true
			torp.Mode = weapons.ModeSearch
			torp.WireCut = true
		}
	}
}

func bearingDeg(x1, y1, x2, y2 float64) float64 {
	deg := math.Atan2(x2-x1, y2-y1) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}

func shortest(from, to float64) float64 {
	d := to - from
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}

func clampAngle(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (e *Engine) PopEvents() []string {
	ev := e.Events
	e.Events = nil
	return ev
}

// AllEntities returns the reusable entity list for acoustics and weapons.
func (e *Engine) AllEntities() []*world.Entity {
	if e == nil || e.Scenario == nil {
		return nil
	}
	e.entityScratch = e.Scenario.AppendAllEntities(e.entityScratch[:0])
	return e.entityScratch
}

// AcousticEmitters is AllEntities plus live torpedoes as synthetic acoustic sources.
func (e *Engine) AcousticEmitters() []*world.Entity {
	ents := e.AllEntities()
	n := 0
	for _, t := range e.FireControl.ActiveTorpedoes {
		if t != nil && t.Alive {
			n++
		}
	}
	if n == 0 {
		return ents
	}
	if cap(e.torpedoEntityPool) < n {
		e.torpedoEntityPool = make([]world.Entity, n)
		e.torpedoEntityPtrs = make([]*world.Entity, n)
	} else {
		e.torpedoEntityPool = e.torpedoEntityPool[:n]
		e.torpedoEntityPtrs = e.torpedoEntityPtrs[:n]
	}
	i := 0
	for _, t := range e.FireControl.ActiveTorpedoes {
		if t == nil || !t.Alive {
			continue
		}
		e.torpedoEntityPtrs[i] = t.AcousticEntity(&e.torpedoEntityPool[i])
		ents = append(ents, e.torpedoEntityPtrs[i])
		i++
	}
	e.entityScratch = ents
	return ents
}

func (e *Engine) tryEnemyTorpedoShots(player *world.Entity, gameTime float64) {
	if player == nil || !player.Alive() {
		return
	}
	for _, ent := range e.Scenario.Entities {
		if ent == nil || !ent.Alive() || ent.Side != world.SideEnemy || ent.Kind != world.KindSubmarine {
			continue
		}
		if ent.AIState != "FIRING" && ent.AIState != "ATTACK" {
			continue
		}
		rangeYd := ent.RangeYardsTo(player)
		if rangeYd > 3500 || rangeYd < 400 {
			continue
		}
		if e.FireControl.EnemyTubeOpenAt != nil {
			if openAt, ok := e.FireControl.EnemyTubeOpenAt[ent.ID]; ok {
				if gameTime-openAt < enemyTubeOpenLeadSec {
					continue
				}
				delete(e.FireControl.EnemyTubeOpenAt, ent.ID)
				if e.FireControl.SpawnHostileTorpedo(ent, player) != nil {
					e.EmitTubeTransient(ent, gameTime, false)
					e.Events = append(e.Events, "Torpedo launch detected (hostile)")
					ent.AIState = "ATTACK"
				}
				continue
			}
		}
		if e.FireControl.HasRecentShotFrom(ent.ID, 45) {
			continue
		}
		if ent.AIState == "ATTACK" && int(gameTime*10)%110 != 0 {
			continue
		}
		if e.FireControl.EnemyTubeOpenAt == nil {
			e.FireControl.EnemyTubeOpenAt = map[string]float64{}
		}
		e.FireControl.EnemyTubeOpenAt[ent.ID] = gameTime
		e.EmitTubeTransient(ent, gameTime, true)
		ent.AIState = "FIRING"
	}
}

func (e *Engine) clampToSeafloor() {
	bathy := e.Scenario.Bathy
	if bathy == nil || !bathy.Valid() {
		return
	}
	clamp := func(ent *world.Entity) {
		if ent == nil || !ent.InWater() {
			return
		}
		if ent.Kind != world.KindSubmarine && ent.Status != world.StatusSinking {
			return
		}
		bot := bathy.DepthAtFt(ent.X, ent.Y)
		if bot <= 0 {
			return
		}
		maxDepth := bot - 50
		if maxDepth < 40 {
			maxDepth = 40
		}
		if ent.DepthFt > maxDepth {
			ent.DepthFt = maxDepth
		}
		if ent.Alive() && ent.OrderedDepth > maxDepth {
			ent.OrderedDepth = maxDepth
		}
	}
	clamp(e.Scenario.Player)
	for _, ent := range e.Scenario.Entities {
		clamp(ent)
	}
}

func (e *Engine) EnemyEmitters() []*world.Entity {
	var out []*world.Entity
	for _, ent := range e.Scenario.Entities {
		if ent.Alive() {
			out = append(out, ent)
		}
	}
	return out
}
