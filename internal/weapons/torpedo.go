package weapons

import (
	"fmt"
	"math"

	"github.com/ssn688/sim/internal/world"
)

type TubeState int

const (
	TubeEmpty TubeState = iota
	TubeLoaded
	TubeDoorOpen
	TubeFired
)

type Tube struct {
	Number      int
	State       TubeState
	TorpedoType string
	WireIntact  bool
}

type TorpedoMode int

const (
	ModeWire TorpedoMode = iota
	ModeSearch
)

// Torpedo is a weapon in the water.
type Torpedo struct {
	ID           string
	ParentSubID  string
	TargetID     string
	Side         world.Side
	X, Y         float64
	DepthFt      float64
	HeadingDeg   float64
	SpeedKts     float64
	RunDepthFt   float64
	SeekerOn     bool
	WireCut      bool
	Armed        bool
	Mode         TorpedoMode
	Alive        bool
	Age          float64
}

// FireControl manages 688-style torpedo firing.
type FireControl struct {
	Tubes           [4]Tube
	SelectedTube    int
	GyroAngleDeg    float64
	RunDepthFt      float64
	SpeedSetting    string // "LOW", "HIGH"
	SeekerEnabled   bool
	ActiveTorpedoes []*Torpedo
	torpedoSeq      int
}

func NewFireControl() FireControl {
	fc := FireControl{
		SelectedTube:  1,
		GyroAngleDeg:  0,
		RunDepthFt:    400,
		SpeedSetting:  "HIGH",
		SeekerEnabled: false,
	}
	for i := range fc.Tubes {
		fc.Tubes[i] = Tube{
			Number:      i + 1,
			State:       TubeLoaded,
			TorpedoType: "Mk48",
		}
	}
	return fc
}

func (fc *FireControl) SelectTube(n int) {
	if n >= 1 && n <= 4 {
		fc.SelectedTube = n
	}
}

func (fc *FireControl) OpenOuterDoor() bool {
	t := &fc.Tubes[fc.SelectedTube-1]
	if t.State == TubeLoaded {
		t.State = TubeDoorOpen
		return true
	}
	return false
}

func (fc *FireControl) CloseOuterDoor() bool {
	t := &fc.Tubes[fc.SelectedTube-1]
	if t.State == TubeDoorOpen {
		t.State = TubeLoaded
		return true
	}
	return false
}

func (fc *FireControl) Shoot(sub *world.Entity) *Torpedo {
	t := &fc.Tubes[fc.SelectedTube-1]
	if t.State != TubeDoorOpen && t.State != TubeLoaded {
		return nil
	}
	fc.torpedoSeq++
	t.State = TubeFired
	t.WireIntact = true

	launchAngle := sub.HeadingDeg + fc.GyroAngleDeg
	rad := launchAngle * math.Pi / 180
	offset := 50.0
	torp := &Torpedo{
		ID:          fmt.Sprintf("MK48-%d", fc.torpedoSeq),
		ParentSubID: sub.ID,
		Side:        sub.Side,
		X:           sub.X + math.Sin(rad)*offset,
		Y:           sub.Y + math.Cos(rad)*offset,
		DepthFt:     sub.DepthFt,
		HeadingDeg:  launchAngle,
		SpeedKts:    speedKts(fc.SpeedSetting),
		RunDepthFt:  fc.RunDepthFt,
		SeekerOn:    fc.SeekerEnabled,
		Armed:       true,
		Mode:        ModeWire,
		Alive:       true,
	}
	if fc.SeekerEnabled {
		torp.Mode = ModeSearch
	}
	fc.ActiveTorpedoes = append(fc.ActiveTorpedoes, torp)
	return torp
}

func (fc *FireControl) CutWire(torp *Torpedo) {
	torp.WireCut = true
	torp.Mode = ModeSearch
	for i := range fc.Tubes {
		if fc.Tubes[i].State == TubeFired {
			fc.Tubes[i].WireIntact = false
		}
	}
}

func (fc *FireControl) EnableSeeker(torp *Torpedo) {
	torp.SeekerOn = true
	torp.Mode = ModeSearch
}

func speedKts(s string) float64 {
	if s == "LOW" {
		return 28
	}
	return 55
}

func (t *Torpedo) Advance(dt float64, targets []*world.Entity) *world.Entity {
	if !t.Alive {
		return nil
	}
	t.Age += dt
	t.DepthFt += (t.RunDepthFt - t.DepthFt) * dt * 0.5

	if t.Mode == ModeWire && !t.WireCut && t.Age > 90 {
		t.WireCut = true
		t.Mode = ModeSearch
	}

	// Homing
	if t.SeekerOn || t.Mode == ModeSearch {
		var best *world.Entity
		bestDist := 1e9
		for _, tgt := range targets {
			if !tgt.Alive() || tgt.Side == t.Side {
				continue
			}
			d := math.Hypot(tgt.X-t.X, tgt.Y-t.Y)
			if d < bestDist && d < 6000 {
				bestDist = d
				best = tgt
			}
		}
		if best != nil {
			desired := bearing(t.X, t.Y, best.X, best.Y)
			diff := normalizeAngle(desired - t.HeadingDeg)
			t.HeadingDeg += clamp(diff*dt*0.8, -dt*8, dt*8)
			t.TargetID = best.ID
		}
	}

	rad := t.HeadingDeg * math.Pi / 180
	yps := t.SpeedKts * world.KnotsToYPS
	t.X += math.Sin(rad) * yps * dt
	t.Y += math.Cos(rad) * yps * dt

	// Hit detection
	for _, tgt := range targets {
		if !tgt.Alive() || tgt.Side == t.Side {
			continue
		}
		d := math.Hypot(tgt.X-t.X, tgt.Y-t.Y)
		depthDiff := math.Abs(tgt.DepthFt - t.DepthFt)
		if d < 30 && depthDiff < 50 && t.Armed && t.Age > 2 {
			t.Alive = false
			if tgt.Kind == world.KindSubmarine {
				tgt.Status = world.StatusSunk
			} else {
				tgt.Status = world.StatusDestroyed
			}
			return tgt
		}
	}
	if t.Age > 600 {
		t.Alive = false
	}
	return nil
}

func bearing(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	deg := math.Atan2(dx, dy) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}

func normalizeAngle(a float64) float64 {
	for a >= 360 {
		a -= 360
	}
	for a < 0 {
		a += 360
	}
	return a
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
