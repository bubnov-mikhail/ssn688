package sim

// Clock tracks simulation time and playback speed.
type Clock struct {
	GameTime  float64
	TimeScale float64
	Paused    bool
}

func NewClock() Clock {
	return Clock{TimeScale: 1.0}
}

func (c *Clock) Advance(dt float64) {
	if c.Paused || c.TimeScale <= 0 {
		return
	}
	// dt is already a simulation step; TimeScale is applied via Engine.Accum.
	c.GameTime += dt
}

func (c *Clock) TogglePause() {
	c.Paused = !c.Paused
}

func (c *Clock) CycleSpeed() {
	switch {
	case c.TimeScale < 0.5:
		c.TimeScale = 0.5
	case c.TimeScale < 1:
		c.TimeScale = 1
	case c.TimeScale < 2:
		c.TimeScale = 2
	case c.TimeScale < 4:
		c.TimeScale = 4
	case c.TimeScale < 8:
		c.TimeScale = 8
	default:
		c.TimeScale = 0.5
	}
}

func (c *Clock) SpeedLabel() string {
	if c.Paused {
		return "PAUSED"
	}
	switch c.TimeScale {
	case 0.5:
		return "0.5x"
	case 1:
		return "1x"
	case 2:
		return "2x"
	case 4:
		return "4x"
	case 8:
		return "8x"
	default:
		return "1x"
	}
}
