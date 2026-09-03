package campaign

import (
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/weapons"
)

// TubeLoadout is the ordnance type loaded in each of four tubes at mission start.
type TubeLoadout [4]string

// Mix returns Harpoon tube fraction in [0,1] (0 = all Mk48, 1 = all Harpoon).
func (t TubeLoadout) Mix() float64 {
	harp := 0
	for i := range t {
		if weapons.NormalizeOrdnance(t[i]) == weapons.OrdnanceHarpoon {
			harp++
		}
	}
	return float64(harp) / 4.0
}

// DefaultTubeLoadout is the standard tube ordnance at mission prep (independent of magazine slider).
func DefaultTubeLoadout() TubeLoadout {
	fc := weapons.NewFireControl()
	var t TubeLoadout
	for i := range t {
		t[i] = fc.Tubes[i].TorpedoType
	}
	return t
}

// LoadoutFromMix derives tube types from the Mk48/Harpoon slider position.
func LoadoutFromMix(mix float64) TubeLoadout {
	fc := weapons.NewFireControl()
	ApplyPlayerLoadout(&fc, mix)
	var t TubeLoadout
	for i := range t {
		t[i] = fc.Tubes[i].TorpedoType
	}
	return t
}

// PlayerWeaponSlots is interchangeable magazine space (Mk48 or Harpoon, one slot each).
func PlayerWeaponSlots() int {
	return weapons.PlayerMagazineCapacity + weapons.PlayerHarpoonMagazine
}

func magazineLeftFromMix(mix float64, mk48Loaded, harpLoaded int) (mk48Mag, harpMag int) {
	if mix < 0 {
		mix = 0
	}
	if mix > 1 {
		mix = 1
	}
	total := PlayerWeaponSlots()
	totalHarp := int(math.Round(float64(total) * mix))
	totalMk48 := total - totalHarp
	if totalMk48 < mk48Loaded {
		totalMk48 = mk48Loaded
		totalHarp = total - totalMk48
	}
	if totalHarp < harpLoaded {
		totalHarp = harpLoaded
		totalMk48 = total - totalHarp
	}
	return totalMk48 - mk48Loaded, totalHarp - harpLoaded
}

// ApplyTubeLoadout configures tubes from player choice; magazine counts scale with mix.
func ApplyTubeLoadout(fc *weapons.FireControl, t TubeLoadout, mix float64) {
	if fc == nil {
		return
	}
	mk48Loaded := 0
	harpLoaded := 0
	for i := range t {
		ord := weapons.NormalizeOrdnance(t[i])
		if ord == "" {
			ord = weapons.OrdnanceMk48
		}
		fc.Tubes[i] = weapons.Tube{
			Number:      i + 1,
			State:       weapons.TubeLoaded,
			TorpedoType: ord,
		}
		if ord == weapons.OrdnanceMk48 {
			mk48Loaded++
		} else {
			harpLoaded++
		}
	}
	fc.MagazineLeft, fc.HarpoonMagLeft = magazineLeftFromMix(mix, mk48Loaded, harpLoaded)
}

// PreviewFireControl builds a fire-control snapshot for loadout UI preview.
func PreviewFireControl(t TubeLoadout, mix float64) weapons.FireControl {
	fc := weapons.NewFireControl()
	ApplyTubeLoadout(&fc, t, mix)
	return fc
}

// ApplyPlayerLoadout configures tubes and magazines from mix in [0,1]:
// 0 = Mk48-heavy, 1 = Harpoon-heavy.
func ApplyPlayerLoadout(fc *weapons.FireControl, mix float64) {
	if fc == nil {
		return
	}
	if mix < 0 {
		mix = 0
	}
	if mix > 1 {
		mix = 1
	}
	mk48Share := 1 - mix
	mk48Tubes := int(math.Round(mk48Share * 4))
	if mk48Tubes < 0 {
		mk48Tubes = 0
	}
	if mk48Tubes > 4 {
		mk48Tubes = 4
	}
	harpTubes := 4 - mk48Tubes
	fc.MagazineLeft, fc.HarpoonMagLeft = magazineLeftFromMix(mix, mk48Tubes, harpTubes)
	for i := range fc.Tubes {
		ord := weapons.OrdnanceMk48
		if i >= mk48Tubes {
			ord = weapons.OrdnanceHarpoon
		}
		fc.Tubes[i] = weapons.Tube{
			Number:      i + 1,
			State:       weapons.TubeLoaded,
			TorpedoType: ord,
		}
	}
}
