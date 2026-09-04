package layout

// Base logical resolution (desktop design). Height stays fixed on mobile;
// width grows with device aspect so main panels widen while the right column
// keeps its desktop height and sits on the right edge.
const (
	BaseScreenW = 1600
	BaseScreenH = 900

	// Right column (ownship / CM / minimap): left edge is ScreenW - RightColInset.
	RightColInset = 300
	PanelGap      = 20 // gap between main↔side and side↔right column

	PassiveMainPanelX = 20
	PassiveMainPanelY = 50
	PassiveMainPanelH = 700

	PassiveSidePanelY = 50
	PassiveSidePanelW = 340
	PassiveSidePanelH = 700

	// Static label anchors (engraved light-gray text; no nameplates).
	PassiveTitleLabelX = 36
	PassiveTitleLabelY = 62

	PassiveHintLabelX = 36
	PassiveHintLabelY = 708

	// Legacy aliases used by button layout helpers.
	PassiveTitlePlateX = PassiveTitleLabelX
	PassiveTitlePlateY = PassiveTitleLabelY
	PassiveTitlePlateW = 320
	PassiveTitlePlateH = 20

	PassiveHintPlateX = PassiveHintLabelX
	PassiveHintPlateY = PassiveHintLabelY
	PassiveHintPlateW = 620
	PassiveHintPlateH = 20

	PassiveArrayPlateW = 220
	PassiveArrayPlateH = 18

	PassiveTowedPlateW = 220
	PassiveTowedPlateH = 18

	PassivePlotCX         = 470.0
	PassivePlotCY         = 430.0
	PassivePlotR          = 222.0
	PassivePlotMaxRangeYd = 8000.0

	WaterfallMaxRows    = 460 // matches WaterfallPlotH — no off-screen history
	WaterfallSampleSec  = 0.15
	WaterfallRowH       = 1
	WaterfallFrameInset = 10
	WaterfallLabelW     = 46
	WaterfallHeaderH    = 56
	WaterfallAxisH      = 22
	WaterfallNoiseH     = 34

	PassiveListRow = 22
	PassiveListH   = 410
	PassiveTowedStatusH = 24
)

// ScreenW / ScreenH are the logical canvas size (updated on mobile via SetScreenSize).
var (
	ScreenW = BaseScreenW
	ScreenH = BaseScreenH
)

// Geometry derived from ScreenW (kept in sync by SetScreenSize / syncGeometry).
var (
	// Dual-column screens (PASSIVE / ACTIVE): main | gap | side | gap | right col.
	PassiveMainPanelW int
	PassiveSidePanelX int

	PassiveArrayLabelX   int
	PassiveArrayLabelY   = 56
	PassiveBandLabelX    int
	PassiveBandLabelY    = 118
	PassiveTowedLabelX   int
	PassiveTowedLabelY   = 178
	PassiveContactLabelX int
	PassiveContactLabelY = 300

	PassiveArrayPlateX int
	PassiveArrayPlateY = PassiveArrayLabelY
	PassiveTowedPlateX int
	PassiveTowedPlateY = PassiveTowedLabelY

	WaterfallPanelX int
	WaterfallPanelY int
	WaterfallPanelW int
	WaterfallPanelH int
	WaterfallPlotX  int
	WaterfallPlotY  int
	WaterfallPlotW  int
	WaterfallPlotH  int

	PassiveStatusTextX int
	PassiveStatusTextY int
	WaterfallChipY     int

	PassiveSelfNoiseMonitorX int
	PassiveSelfNoiseMonitorY int
	PassiveSelfNoiseMonitorW int
	PassiveSelfNoiseMonitorH int
	PassiveSelfNoiseLabelX   int
	PassiveSelfNoiseStatusX  int
	PassiveSelfNoiseBarX     int
	PassiveSelfNoiseBarW     int

	PassiveTowedStatusX int
	PassiveTowedStatusY int
	PassiveTowedStatusW int

	PassiveArrayButtonsY int
	PassiveBandButtonsY  int
	PassiveTowedButtonsY int

	PassiveListX int
	PassiveListY int
	PassiveListW int

	// Full-width main console (PLOT / HELM / LIBRARY / WEPS / MAST / SPECTRUM / DC).
	FullMainPanelW int
)

func init() {
	syncGeometry()
}

// SetScreenSize updates the logical canvas and recomputes panel geometry.
// Desktop builds keep BaseScreenW×BaseScreenH; mobile Layout widens ScreenW.
func SetScreenSize(w, h int) {
	if w < BaseScreenW {
		w = BaseScreenW
	}
	if h < 1 {
		h = BaseScreenH
	}
	if w == ScreenW && h == ScreenH {
		return
	}
	ScreenW, ScreenH = w, h
	syncGeometry()
}

func syncGeometry() {
	// Side panel sits PanelGap left of the right column.
	PassiveSidePanelX = ScreenW - RightColInset - PanelGap - PassiveSidePanelW
	PassiveMainPanelW = PassiveSidePanelX - PanelGap - PassiveMainPanelX
	FullMainPanelW = ScreenW - RightColInset - PanelGap - PassiveMainPanelX

	sideLabelX := PassiveSidePanelX + 12
	PassiveArrayLabelX = sideLabelX
	PassiveBandLabelX = sideLabelX
	PassiveTowedLabelX = sideLabelX
	PassiveContactLabelX = sideLabelX
	PassiveArrayPlateX = PassiveArrayLabelX
	PassiveTowedPlateX = PassiveTowedLabelX

	WaterfallPanelX = PassiveMainPanelX + WaterfallFrameInset
	WaterfallPanelY = PassiveTitleLabelY + 24
	WaterfallPanelW = PassiveMainPanelW - 2*WaterfallFrameInset
	WaterfallPanelH = PassiveHintLabelY - WaterfallPanelY - 14
	WaterfallPlotX = WaterfallPanelX + WaterfallLabelW
	WaterfallPlotY = WaterfallPanelY + WaterfallHeaderH
	WaterfallPlotW = WaterfallPanelW - WaterfallLabelW - 8
	WaterfallPlotH = WaterfallPanelH - WaterfallHeaderH - WaterfallAxisH - WaterfallNoiseH

	PassiveStatusTextX = WaterfallPanelX + 4
	PassiveStatusTextY = WaterfallPanelY + 22
	WaterfallChipY = WaterfallPanelY + WaterfallHeaderH - 18

	PassiveSelfNoiseMonitorX = WaterfallPlotX
	PassiveSelfNoiseMonitorY = WaterfallPanelY + WaterfallPanelH - WaterfallAxisH - WaterfallNoiseH
	PassiveSelfNoiseMonitorW = WaterfallPlotW
	PassiveSelfNoiseMonitorH = WaterfallNoiseH
	PassiveSelfNoiseLabelX = PassiveSelfNoiseMonitorX + 8
	PassiveSelfNoiseStatusX = PassiveSelfNoiseLabelX + 92
	PassiveSelfNoiseBarX = PassiveSelfNoiseStatusX + 96
	PassiveSelfNoiseBarW = PassiveSelfNoiseMonitorX + PassiveSelfNoiseMonitorW - 8 - PassiveSelfNoiseBarX

	PassiveTowedStatusX = sideLabelX
	PassiveTowedStatusY = PassiveTowedLabelY + 22
	PassiveTowedStatusW = PassiveSidePanelW - 42

	PassiveArrayButtonsY = PassiveArrayLabelY + 22
	PassiveBandButtonsY = PassiveBandLabelY + 22
	PassiveTowedButtonsY = PassiveTowedStatusY + PassiveTowedStatusH + 8

	PassiveListX = sideLabelX
	PassiveListY = PassiveContactLabelY + 22
	PassiveListW = PassiveSidePanelW - 30
}
