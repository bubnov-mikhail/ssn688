package layout

import "testing"

func TestSyncGeometryDesktopBaseline(t *testing.T) {
	SetScreenSize(BaseScreenW, BaseScreenH)
	if PassiveMainPanelW != 900 {
		t.Fatalf("PassiveMainPanelW=%d want 900", PassiveMainPanelW)
	}
	if PassiveSidePanelX != 940 {
		t.Fatalf("PassiveSidePanelX=%d want 940", PassiveSidePanelX)
	}
	if FullMainPanelW != 1260 {
		t.Fatalf("FullMainPanelW=%d want 1260", FullMainPanelW)
	}
	if WaterfallPanelW != 880 {
		t.Fatalf("WaterfallPanelW=%d want 880", WaterfallPanelW)
	}
	// Gaps: main|side and side|right column.
	if PassiveSidePanelX-(PassiveMainPanelX+PassiveMainPanelW) != PanelGap {
		t.Fatalf("main↔side gap")
	}
	rightX := ScreenW - RightColInset
	if rightX-(PassiveSidePanelX+PassiveSidePanelW) != PanelGap {
		t.Fatalf("side↔right gap")
	}
}

func TestSyncGeometryWiderMobile(t *testing.T) {
	t.Cleanup(func() { SetScreenSize(BaseScreenW, BaseScreenH) })
	SetScreenSize(2000, BaseScreenH)
	if FullMainPanelW != 1660 {
		t.Fatalf("FullMainPanelW=%d want 1660", FullMainPanelW)
	}
	if PassiveMainPanelW != 1300 {
		t.Fatalf("PassiveMainPanelW=%d want 1300", PassiveMainPanelW)
	}
	rightX := ScreenW - RightColInset
	if PassiveSidePanelX+PassiveSidePanelW+PanelGap != rightX {
		t.Fatalf("side panel not flush to right column gap")
	}
}
