package campaign

import (
	"testing"

	"github.com/bubnov-mikhail/ssn688/internal/weapons"
)

func TestApplyPlayerLoadoutExtremes(t *testing.T) {
	fc := weapons.NewFireControl()
	ApplyPlayerLoadout(&fc, 0)
	if fc.Tubes[0].TorpedoType != weapons.OrdnanceMk48 || fc.Tubes[3].TorpedoType != weapons.OrdnanceMk48 {
		t.Fatal("mix 0 should load all Mk48 tubes")
	}
	ApplyPlayerLoadout(&fc, 1)
	if fc.Tubes[0].TorpedoType != weapons.OrdnanceHarpoon {
		t.Fatal("mix 1 should load all Harpoon tubes")
	}
}

func TestApplyTubeLoadoutMagazines(t *testing.T) {
	tubes := TubeLoadout{
		weapons.OrdnanceMk48,
		weapons.OrdnanceMk48,
		weapons.OrdnanceMk48,
		weapons.OrdnanceHarpoon,
	}
	fc := PreviewFireControl(tubes, 0)
	if fc.MagazineLeft != 26 {
		t.Fatalf("Mk48 mag at mix 0 want 26 got %d", fc.MagazineLeft)
	}
	if fc.HarpoonMagLeft != 0 {
		t.Fatalf("Harpoon mag at mix 0 want 0 got %d", fc.HarpoonMagLeft)
	}

	fc = PreviewFireControl(tubes, 1)
	if fc.HarpoonMagLeft != 26 {
		t.Fatalf("Harpoon mag at mix 1 want 26 got %d", fc.HarpoonMagLeft)
	}
	if fc.MagazineLeft != 0 {
		t.Fatalf("Mk48 mag at mix 1 want 0 got %d", fc.MagazineLeft)
	}
}

func weaponCount(fc weapons.FireControl, tubes TubeLoadout) int {
	n := fc.MagazineLeft + fc.HarpoonMagLeft
	for i := range tubes {
		if weapons.NormalizeOrdnance(tubes[i]) == weapons.OrdnanceHarpoon {
			n++
		} else {
			n++
		}
	}
	return n
}

func TestMagazineMixConstantTotal(t *testing.T) {
	tubes := DefaultTubeLoadout()
	want := PlayerWeaponSlots()
	for _, mix := range []float64{0, 0.25, 0.5, 0.75, 1} {
		fc := PreviewFireControl(tubes, mix)
		if got := weaponCount(fc, tubes); got != want {
			t.Fatalf("mix %.2f total weapons want %d got %d (mk48 mag %d harp mag %d)",
				mix, want, got, fc.MagazineLeft, fc.HarpoonMagLeft)
		}
	}
}

func TestMagazineMixScalesWithSlider(t *testing.T) {
	tubes := DefaultTubeLoadout()
	fc0 := PreviewFireControl(tubes, 0)
	fc1 := PreviewFireControl(tubes, 1)
	if fc1.HarpoonMagLeft <= fc0.HarpoonMagLeft {
		t.Fatalf("harpoon mag should increase with mix: mix0=%d mix1=%d", fc0.HarpoonMagLeft, fc1.HarpoonMagLeft)
	}
	if fc1.MagazineLeft >= fc0.MagazineLeft {
		t.Fatalf("mk48 mag should decrease with mix: mix0=%d mix1=%d", fc0.MagazineLeft, fc1.MagazineLeft)
	}
}

func TestLoadoutFromMixRoundTrip(t *testing.T) {
	for _, mix := range []float64{0, 0.25, 0.5, 0.75, 1} {
		tubes := LoadoutFromMix(mix)
		got := tubes.Mix()
		if got < mix-0.01 || got > mix+0.01 {
			t.Fatalf("mix %.2f -> tubes mix %.2f", mix, got)
		}
	}
}

func TestDemoScenarioRegistry(t *testing.T) {
	sc := ScenarioByID(DemoScenarioID)
	if sc == nil || len(sc.Missions) != 2 {
		t.Fatal("demo scenario should have two missions")
	}
	if len(sc.CoverData) == 0 && sc.CoverFile == "" {
		t.Fatal("demo scenario needs cover art")
	}
}
