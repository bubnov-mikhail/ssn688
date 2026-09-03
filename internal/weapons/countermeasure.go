package weapons

import (
	"fmt"
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// CMKind identifies soft-kill countermeasure types.
type CMKind int

const (
	CMExpendableADC CMKind = iota // acoustic decoy / target emulator
	CMTowedNixie                  // surface towed acoustic decoy
	CMExpendableJitter            // broadband jammer / confusion
)

const (
	CMMagazineDefault       = 6
	CMJitterMagazineDefault = 6
	CMDeployCooldownSec     = 8.0
	CMADCLifeSec            = 90.0
	CMJitterLifeSec         = 75.0
	CMNixieTrailYd          = 220.0

	// Seeker anti-CM / seduction (player Mk48 — relatively sharp).
	AntiCMVerifySec      = 30.0 // ADC / jitter hold lock before reject check
	AntiCMRejectHoldSec  = 28.0
	SeekCMAttractMul     = 1.38 // ADC / Nixie seduction bonus
	SeekJitterAttractMul = 0.88

	// Hostile seekers are intentionally gullible so soft-kill remains useful.
	EnemyAntiCMVerifySec      = 55.0
	EnemyAntiCMRejectHoldSec  = 45.0
	EnemySeekCMAttractMul     = 1.95
	EnemySeekJitterAttractMul = 1.15
	EnemyShipCloseBias        = 1.06 // vs 1.25 for player fish
	EnemyPriorTargetBias      = 1.04 // vs 1.12
)

// Countermeasure is a soft-kill acoustic source in the water.
type Countermeasure struct {
	ID           string
	ParentID     string
	Side         world.Side
	Kind         CMKind
	X, Y         float64
	DepthFt      float64
	HeadingDeg   float64
	SpeedKts     float64
	Alive        bool
	Age          float64
	TTL          float64
	NoiseBoostDB float64
}

// CountermeasureSystem tracks magazines, Nixie, and live expendables.
type CountermeasureSystem struct {
	Active         []*Countermeasure
	Magazine       map[string]int // decoy (ADC) counts
	JitterMagazine map[string]int
	NixieEnabled   map[string]bool
	LastDeployAt   map[string]float64 // decoy cooldown clock
	LastJitterAt   map[string]float64
	nextID         int
	entityPool     []world.Entity
	entityPtrs     []*world.Entity
}

func NewCountermeasureSystem() CountermeasureSystem {
	return CountermeasureSystem{
		Magazine:       map[string]int{},
		JitterMagazine: map[string]int{},
		NixieEnabled:   map[string]bool{},
		LastDeployAt:   map[string]float64{},
		LastJitterAt:   map[string]float64{},
	}
}

func (s *CountermeasureSystem) EnsureMagazine(entityID string) {
	if s.Magazine == nil {
		s.Magazine = map[string]int{}
	}
	if s.JitterMagazine == nil {
		s.JitterMagazine = map[string]int{}
	}
	if _, ok := s.Magazine[entityID]; !ok {
		s.Magazine[entityID] = CMMagazineDefault
	}
	if _, ok := s.JitterMagazine[entityID]; !ok {
		s.JitterMagazine[entityID] = CMJitterMagazineDefault
	}
	if s.LastDeployAt == nil {
		s.LastDeployAt = map[string]float64{}
	}
	if s.LastJitterAt == nil {
		s.LastJitterAt = map[string]float64{}
	}
	if s.NixieEnabled == nil {
		s.NixieEnabled = map[string]bool{}
	}
}

func (s *CountermeasureSystem) SetMagazine(entityID string, n int) {
	s.EnsureMagazine(entityID)
	if n < 0 {
		n = 0
	}
	s.Magazine[entityID] = n
}

func (s *CountermeasureSystem) SetJitterMagazine(entityID string, n int) {
	s.EnsureMagazine(entityID)
	if n < 0 {
		n = 0
	}
	s.JitterMagazine[entityID] = n
}

// MagazineLeft returns remaining decoy (ADC) rounds.
func (s *CountermeasureSystem) MagazineLeft(entityID string) int {
	s.EnsureMagazine(entityID)
	return s.Magazine[entityID]
}

func (s *CountermeasureSystem) DecoyLeft(entityID string) int {
	return s.MagazineLeft(entityID)
}

func (s *CountermeasureSystem) JitterLeft(entityID string) int {
	s.EnsureMagazine(entityID)
	return s.JitterMagazine[entityID]
}

func (s *CountermeasureSystem) SetNixie(entityID string, on bool) {
	s.EnsureMagazine(entityID)
	s.NixieEnabled[entityID] = on
}

func (s *CountermeasureSystem) NixieOn(entityID string) bool {
	if s == nil || s.NixieEnabled == nil {
		return false
	}
	return s.NixieEnabled[entityID]
}

// DeployADC launches an expendable acoustic decoy toward (tx, ty).
func (s *CountermeasureSystem) DeployADC(owner *world.Entity, tx, ty, gameTime float64) *Countermeasure {
	return s.deployExpendable(owner, tx, ty, gameTime, CMExpendableADC)
}

// DeployJitter launches a broadband jammer toward (tx, ty).
func (s *CountermeasureSystem) DeployJitter(owner *world.Entity, tx, ty, gameTime float64) *Countermeasure {
	return s.deployExpendable(owner, tx, ty, gameTime, CMExpendableJitter)
}

func (s *CountermeasureSystem) deployExpendable(owner *world.Entity, tx, ty, gameTime float64, kind CMKind) *Countermeasure {
	if s == nil || owner == nil || !owner.Alive() {
		return nil
	}
	s.EnsureMagazine(owner.ID)

	var mag map[string]int
	var last map[string]float64
	ttl := CMADCLifeSec
	noise := 8.0
	spd := math.Max(2.5, owner.SpeedKts*0.4)
	if kind == CMExpendableJitter {
		mag = s.JitterMagazine
		last = s.LastJitterAt
		ttl = CMJitterLifeSec
		noise = 14.0
		spd = math.Max(1.5, owner.SpeedKts*0.25)
	} else {
		mag = s.Magazine
		last = s.LastDeployAt
	}
	if mag[owner.ID] <= 0 {
		return nil
	}
	if lastAt, ok := last[owner.ID]; ok && gameTime-lastAt < CMDeployCooldownSec {
		return nil
	}

	brg := math.Atan2(tx-owner.X, ty-owner.Y) * 180 / math.Pi
	if brg < 0 {
		brg += 360
	}
	// Drop slightly off the threat line so the fish sees a competing contact.
	rad := brg * math.Pi / 180
	offset := 120.0
	s.nextID++
	prefix := "adc"
	if kind == CMExpendableJitter {
		prefix = "jit"
	}
	cm := &Countermeasure{
		ID:           fmt.Sprintf("%s-%s-%d", prefix, owner.ID, s.nextID),
		ParentID:     owner.ID,
		Side:         owner.Side,
		Kind:         kind,
		X:            owner.X + math.Sin(rad)*offset,
		Y:            owner.Y + math.Cos(rad)*offset,
		DepthFt:      owner.DepthFt + 15,
		HeadingDeg:   brg,
		SpeedKts:     spd,
		Alive:        true,
		Age:          0,
		TTL:          ttl,
		NoiseBoostDB: noise,
	}
	if owner.Kind == world.KindSurfaceShip {
		cm.DepthFt = 40
	}
	mag[owner.ID]--
	last[owner.ID] = gameTime
	s.Active = append(s.Active, cm)
	return cm
}

// Advance ages expendables, moves them, and syncs towed Nixie bodies.
func (s *CountermeasureSystem) Advance(dt, gameTime float64, entities []*world.Entity) {
	if s == nil {
		return
	}
	out := s.Active[:0]
	for _, cm := range s.Active {
		if cm == nil || !cm.Alive {
			continue
		}
		if cm.Kind == CMTowedNixie {
			continue // rebuilt below from NixieEnabled
		}
		cm.Age += dt
		if cm.Age >= cm.TTL {
			cm.Alive = false
			continue
		}
		rad := cm.HeadingDeg * math.Pi / 180
		dist := cm.SpeedKts * world.KnotsToYPS * dt
		cm.X += math.Sin(rad) * dist
		cm.Y += math.Cos(rad) * dist
		cm.SpeedKts = math.Max(1.0, cm.SpeedKts-0.35*dt)
		out = append(out, cm)
	}
	s.Active = out

	// Rebuild towed Nixie markers from enabled parents.
	byParent := map[string]*world.Entity{}
	for _, e := range entities {
		if e != nil {
			byParent[e.ID] = e
		}
	}
	for id, on := range s.NixieEnabled {
		if !on {
			continue
		}
		parent := byParent[id]
		if parent == nil || !parent.Alive() || parent.Kind != world.KindSurfaceShip {
			continue
		}
		rad := parent.HeadingDeg * math.Pi / 180
		s.nextID++
		s.Active = append(s.Active, &Countermeasure{
			ID:           fmt.Sprintf("nixie-%s", id),
			ParentID:     id,
			Side:         parent.Side,
			Kind:         CMTowedNixie,
			X:            parent.X - math.Sin(rad)*CMNixieTrailYd,
			Y:            parent.Y - math.Cos(rad)*CMNixieTrailYd,
			DepthFt:      35,
			HeadingDeg:   parent.HeadingDeg,
			SpeedKts:     parent.SpeedKts,
			Alive:        true,
			Age:          0,
			TTL:          1e9,
			NoiseBoostDB: 10,
		})
	}
}

// AcousticEntities appends live CM as Entity pointers for waterfall / seeker.
func (s *CountermeasureSystem) AcousticEntities(ents []*world.Entity) []*world.Entity {
	if s == nil {
		return ents
	}
	n := 0
	for _, cm := range s.Active {
		if cm != nil && cm.Alive {
			n++
		}
	}
	if n == 0 {
		return ents
	}
	if cap(s.entityPool) < n {
		s.entityPool = make([]world.Entity, n)
		s.entityPtrs = make([]*world.Entity, n)
	} else {
		s.entityPool = s.entityPool[:n]
		s.entityPtrs = s.entityPtrs[:n]
	}
	i := 0
	for _, cm := range s.Active {
		if cm == nil || !cm.Alive {
			continue
		}
		s.entityPtrs[i] = cm.AsEntity(&s.entityPool[i])
		ents = append(ents, s.entityPtrs[i])
		i++
	}
	return ents
}

// AsEntity fills dst with an acoustic entity representing this CM.
func (c *Countermeasure) AsEntity(dst *world.Entity) *world.Entity {
	if c == nil || dst == nil {
		return nil
	}
	sig := "adc"
	name := "ADC"
	switch c.Kind {
	case CMExpendableJitter:
		sig = "jitter"
		name = "JITTER"
	case CMTowedNixie:
		sig = "nixie"
		name = "NIXIE"
	}
	*dst = world.Entity{
		ID:          c.ID,
		Name:        name,
		Kind:        world.KindCountermeasure,
		Side:        c.Side,
		Status:      world.StatusActive,
		SignatureID: sig,
		X:           c.X,
		Y:           c.Y,
		DepthFt:     c.DepthFt,
		HeadingDeg:  c.HeadingDeg,
		SpeedKts:    c.SpeedKts,
		LengthFt:    6,
	}
	return dst
}

// JitterInCone reports whether a jammer is inside the seeker cone (confuses ship locks).
func JitterConfuseFactor(targets []*world.Entity, ox, oy, headingDeg, coneHalf, maxRangeYd float64) float64 {
	factor := 1.0
	for _, tgt := range targets {
		if tgt == nil || !tgt.Alive() || tgt.Kind != world.KindCountermeasure {
			continue
		}
		if tgt.SignatureID != "jitter" {
			continue
		}
		d := math.Hypot(tgt.X-ox, tgt.Y-oy)
		if d < 1 || d > maxRangeYd*1.15 {
			continue
		}
		brg := math.Atan2(tgt.X-ox, tgt.Y-oy) * 180 / math.Pi
		if brg < 0 {
			brg += 360
		}
		diff := math.Mod(brg-headingDeg+540, 360) - 180
		if math.Abs(diff) > coneHalf*1.1 {
			continue
		}
		prox := 1 - d/(maxRangeYd*1.15)
		j := 0.40 + 0.40*prox // 0.40..0.80 lock quality on real hulls
		if j < factor {
			factor = j
		}
	}
	return factor
}

// EnemyJitterConfuseFactor is a stronger jam penalty for gullible hostile seekers.
func EnemyJitterConfuseFactor(targets []*world.Entity, ox, oy, headingDeg, coneHalf, maxRangeYd float64) float64 {
	f := JitterConfuseFactor(targets, ox, oy, headingDeg, coneHalf, maxRangeYd)
	if f >= 0.995 {
		return f
	}
	// Push ship-lock quality further down when a jammer is in the cone.
	return 0.55*f + 0.18
}
