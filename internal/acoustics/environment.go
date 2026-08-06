package acoustics

import "math"

// WaterLayer is a vertical slice of the ocean with distinct acoustic properties.
type WaterLayer struct {
	Name                 string
	TopDepthFt           float64
	BottomDepthFt        float64 // <=0 means no lower bound
	AmbientNoiseDB       float64
	CrossBoundaryAttenDB float64 // extra loss when a ray crosses this layer's top boundary
}

// Environment models ocean acoustic conditions for a scenario.
// The vertical profile is currently uniform across the map (no geographic variation).
type Environment struct {
	SeaState      int
	Layers        []WaterLayer
	BottomDepthFt float64 // seafloor depth (feet) — clearance under keel

	// Layer survey (SSXBT / bathythermograph cast). Boundaries are hidden in UI until known.
	LayerSurveyKnown   bool
	LayerSurveyStartAt float64
	LayerSurveyEndAt   float64 // GameTime when cast completes; 0 = idle
}

// LayerSurveyDurationSec is compressed SSXBT cast time for gameplay pacing.
const LayerSurveyDurationSec = 15.0

// DefaultEnvironment returns a coastal thermocline profile (depths in feet).
// BottomDepthFt is overwritten in-mission from the active bathymetry chart.
func DefaultEnvironment() Environment {
	return Environment{
		SeaState:      2,
		BottomDepthFt: 280,
		Layers: []WaterLayer{
			{Name: "mixed", TopDepthFt: 0, BottomDepthFt: 240, AmbientNoiseDB: 68, CrossBoundaryAttenDB: 0},
			{Name: "thermocline", TopDepthFt: 240, BottomDepthFt: 800, AmbientNoiseDB: 60, CrossBoundaryAttenDB: 16},
			{Name: "deep", TopDepthFt: 800, BottomDepthFt: 0, AmbientNoiseDB: 54, CrossBoundaryAttenDB: 0},
		},
	}
}

func (e Environment) layerIndex(depthFt float64) int {
	for i, l := range e.Layers {
		if depthFt >= l.TopDepthFt && (l.BottomDepthFt <= 0 || depthFt < l.BottomDepthFt) {
			return i
		}
	}
	if len(e.Layers) > 0 {
		return len(e.Layers) - 1
	}
	return 0
}

func (e Environment) AmbientSpectrum(depthFt float64) Spectrum {
	layer := e.Layers[e.layerIndex(depthFt)]
	sea := float64(e.SeaState) * 1.8
	s := NewSpectrumFlat(layer.AmbientNoiseDB + sea)
	if depthFt < 120 {
		for i := NumBands / 2; i < NumBands; i++ {
			s[i] += float64(i-NumBands/2) * 0.15
		}
	}
	return s
}

// LayerCrossingLoss returns extra attenuation when sound crosses layer boundaries.
func (e Environment) LayerCrossingLoss(srcDepthFt, dstDepthFt float64) float64 {
	src := e.layerIndex(srcDepthFt)
	dst := e.layerIndex(dstDepthFt)
	if src == dst {
		return 0
	}
	loss := 0.0
	lo, hi := src, dst
	if lo > hi {
		lo, hi = hi, lo
	}
	for i := lo + 1; i <= hi; i++ {
		if i < len(e.Layers) {
			loss += e.Layers[i].CrossBoundaryAttenDB
		}
	}
	if src > dst && e.Layers[dst].Name == "mixed" {
		loss += 6
	}
	return loss
}

// ColumnAttenuationDB is continuous vertical transmission loss through the water
// column (volume scattering / multipath), separate from discrete LayerCrossingLoss.
// Deeper immersion and larger vertical separation both weaken the received signal.
func (e Environment) ColumnAttenuationDB(srcDepthFt, dstDepthFt, freqHz float64) float64 {
	dz := math.Abs(srcDepthFt - dstDepthFt)
	if dz < 1 {
		return 0
	}
	// Higher frequencies scatter more along the vertical path.
	freqFac := 1.0 + 0.35*math.Min(2.0, freqHz/700.0)
	const rateNear = 0.012 // dB/ft for the first ~600 ft of Δdepth
	const rateFar = 0.005  // diminishing returns beyond that
	loss := 0.0
	if dz <= 600 {
		loss = dz * rateNear * freqFac
	} else {
		loss = (600*rateNear + (dz-600)*rateFar) * freqFac
	}
	// Extra isolation when one platform is shallow and the other deep below the
	// mixed layer — models increasing duct/shadow loss with submergence.
	shallow := math.Min(srcDepthFt, dstDepthFt)
	deep := math.Max(srcDepthFt, dstDepthFt)
	thermo := 240.0
	if len(e.Layers) > 1 {
		thermo = e.Layers[1].TopDepthFt
	}
	if shallow < thermo && deep > thermo {
		below := deep - thermo
		loss += math.Min(7.0, below*0.007)
	}
	return loss
}

func (e Environment) LayerName(depthFt float64) string {
	return e.Layers[e.layerIndex(depthFt)].Name
}

// LayerNameKnown returns the layer name only after a successful BT survey.
func (e Environment) LayerNameKnown(depthFt float64) string {
	if !e.LayerSurveyKnown {
		return "unknown"
	}
	return e.LayerName(depthFt)
}

// KeelClearanceFt is water under the keel (seafloor − own depth).
func (e Environment) KeelClearanceFt(depthFt float64) float64 {
	bottom := e.BottomDepthFt
	if bottom <= 0 {
		bottom = 2000
	}
	c := bottom - depthFt
	if c < 0 {
		return 0
	}
	return c
}

// KnownBoundaryDepthsFt returns layer tops (except surface) once surveyed.
func (e Environment) KnownBoundaryDepthsFt() []float64 {
	if !e.LayerSurveyKnown {
		return nil
	}
	var out []float64
	for _, l := range e.Layers {
		if l.TopDepthFt > 0 && l.TopDepthFt < 800 {
			out = append(out, l.TopDepthFt)
		}
	}
	return out
}

// StartLayerSurvey begins an SSXBT cast. Returns false if already running.
// A completed profile can be re-surveyed (depths/layers may change).
func (e *Environment) StartLayerSurvey(gameTime float64) bool {
	if e.LayerSurveyEndAt > gameTime {
		return false
	}
	e.LayerSurveyStartAt = gameTime
	e.LayerSurveyEndAt = gameTime + LayerSurveyDurationSec
	return true
}

// UpdateLayerSurvey completes the cast when GameTime reaches LayerSurveyEndAt.
func (e *Environment) UpdateLayerSurvey(gameTime float64) {
	if e.LayerSurveyEndAt <= 0 {
		return
	}
	if gameTime >= e.LayerSurveyEndAt {
		e.LayerSurveyKnown = true
		e.LayerSurveyEndAt = 0
	}
}

// LayerSurveyActive reports whether a cast is in progress.
func (e Environment) LayerSurveyActive(gameTime float64) bool {
	return e.LayerSurveyEndAt > 0 && gameTime < e.LayerSurveyEndAt
}

// LayerSurveyProgress returns 0..1 for an active survey (0 when idle).
func (e Environment) LayerSurveyProgress(gameTime float64) float64 {
	if e.LayerSurveyEndAt <= 0 || e.LayerSurveyStartAt <= 0 {
		return 0
	}
	dur := e.LayerSurveyEndAt - e.LayerSurveyStartAt
	if dur <= 0 {
		return 0
	}
	p := (gameTime - e.LayerSurveyStartAt) / dur
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

// LayerSurveyRemainingSec remaining cast time, or 0.
func (e Environment) LayerSurveyRemainingSec(gameTime float64) float64 {
	if !e.LayerSurveyActive(gameTime) {
		return 0
	}
	return e.LayerSurveyEndAt - gameTime
}

// CavitationDepthFt estimates the shallowest depth (feet) before heavy cavitation at given speed.
func CavitationDepthFt(speedKts float64) float64 {
	return 40 + speedKts*8
}

// CavitationSeverity returns 0..1 how much the platform is cavitating.
func CavitationSeverity(depthFt, speedKts float64) float64 {
	speedKts = math.Abs(speedKts)
	// Surface / near-surface platforms: depth≈0 would otherwise always read as
	// "fully cavitating" under the submarine margin curve. Use speed instead.
	if depthFt < 25 {
		switch {
		case speedKts < 8:
			return 0
		case speedKts < 14:
			return (speedKts - 8) / 12 // 0..0.5 cruise freighters
		case speedKts < 22:
			return 0.5 + (speedKts-14)/16 // 0.5..1 warship sprint
		default:
			return 1
		}
	}
	margin := depthFt - CavitationDepthFt(speedKts)
	if margin >= 30 {
		return 0
	}
	if margin <= -40 {
		return 1
	}
	return 1 - (margin+40)/70
}

func absorptionDBPerKy(freqHz, rangeKy float64) float64 {
	f2 := freqHz * freqHz
	alpha := 0.0004 + f2*1.5e-9 + 0.02*f2/(f2+250000)
	return alpha * rangeKy
}

func spreadingLossDB(rangeYd float64) float64 {
	if rangeYd < 100 {
		rangeYd = 100
	}
	return 20 * math.Log10(rangeYd/100)
}

// passiveSpreadingLossDB models one-way passive transmission loss (spherical + extra
// range dependence so contacts fade noticeably with distance on the waterfall).
func passiveSpreadingLossDB(rangeYd float64) float64 {
	if rangeYd < 100 {
		rangeYd = 100
	}
	// 15·log(r) + 10·log(r) ≈ practical deep-water passive TL curve.
	return 25 * math.Log10(rangeYd/100)
}
