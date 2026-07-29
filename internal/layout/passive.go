package layout

const (
	ScreenW = 1600
	ScreenH = 900

	PassiveMainPanelX = 20
	PassiveMainPanelY = 50
	PassiveMainPanelW = 900
	PassiveMainPanelH = 700

	PassiveSidePanelX = 940
	PassiveSidePanelY = 50
	PassiveSidePanelW = 340
	PassiveSidePanelH = 700

	// Static label anchors (engraved light-gray text; no nameplates).
	PassiveTitleLabelX = 36
	PassiveTitleLabelY = 62

	PassiveHintLabelX = 36
	PassiveHintLabelY = 708

	PassiveArrayLabelX = 952
	PassiveArrayLabelY = 56

	PassiveBandLabelX = 952
	PassiveBandLabelY = 118

	PassiveTowedLabelX = 952
	PassiveTowedLabelY = 178

	PassiveContactLabelX = 952
	PassiveContactLabelY = 300

	// Legacy aliases used by button layout helpers.
	PassiveTitlePlateX = PassiveTitleLabelX
	PassiveTitlePlateY = PassiveTitleLabelY
	PassiveTitlePlateW = 320
	PassiveTitlePlateH = 20

	PassiveHintPlateX = PassiveHintLabelX
	PassiveHintPlateY = PassiveHintLabelY
	PassiveHintPlateW = 620
	PassiveHintPlateH = 20

	PassiveArrayPlateX = PassiveArrayLabelX
	PassiveArrayPlateY = PassiveArrayLabelY
	PassiveArrayPlateW = 220
	PassiveArrayPlateH = 18

	PassiveTowedPlateX = PassiveTowedLabelX
	PassiveTowedPlateY = PassiveTowedLabelY
	PassiveTowedPlateW = 220
	PassiveTowedPlateH = 18

	PassivePlotCX         = 470.0
	PassivePlotCY         = 430.0
	PassivePlotR          = 222.0
	PassivePlotMaxRangeYd = 8000.0

	WaterfallMaxRows     = 460 // matches WaterfallPlotH — no off-screen history
	WaterfallSampleSec   = 0.15
	WaterfallRowH        = 1
	WaterfallFrameInset  = 10
	WaterfallPanelX      = PassiveMainPanelX + WaterfallFrameInset
	WaterfallPanelY      = PassiveTitleLabelY + 24
	WaterfallPanelW      = PassiveMainPanelW - 2*WaterfallFrameInset
	WaterfallPanelH      = PassiveHintLabelY - WaterfallPanelY - 14
	WaterfallLabelW      = 46
	WaterfallHeaderH     = 56
	WaterfallAxisH       = 22
	WaterfallNoiseH      = 34
	WaterfallPlotX       = WaterfallPanelX + WaterfallLabelW
	WaterfallPlotY       = WaterfallPanelY + WaterfallHeaderH
	WaterfallPlotW       = WaterfallPanelW - WaterfallLabelW - 8
	WaterfallPlotH       = WaterfallPanelH - WaterfallHeaderH - WaterfallAxisH - WaterfallNoiseH

	PassiveStatusTextX = WaterfallPanelX + 4
	PassiveStatusTextY = WaterfallPanelY + 22
	WaterfallChipY     = WaterfallPanelY + WaterfallHeaderH - 18

	PassiveSelfNoiseMonitorX = WaterfallPlotX
	PassiveSelfNoiseMonitorY = WaterfallPanelY + WaterfallPanelH - WaterfallAxisH - WaterfallNoiseH
	PassiveSelfNoiseMonitorW = WaterfallPlotW
	PassiveSelfNoiseMonitorH = WaterfallNoiseH
	PassiveSelfNoiseLabelX  = PassiveSelfNoiseMonitorX + 8
	PassiveSelfNoiseStatusX = PassiveSelfNoiseLabelX + 92
	PassiveSelfNoiseBarX    = PassiveSelfNoiseStatusX + 96
	PassiveSelfNoiseBarW    = PassiveSelfNoiseMonitorX + PassiveSelfNoiseMonitorW - 8 - PassiveSelfNoiseBarX

	PassiveTowedStatusX = 952
	PassiveTowedStatusY = PassiveTowedLabelY + 22
	PassiveTowedStatusW = 298
	PassiveTowedStatusH = 24

	PassiveArrayButtonsY = PassiveArrayLabelY + 22
	PassiveBandButtonsY  = PassiveBandLabelY + 22
	PassiveTowedButtonsY = PassiveTowedStatusY + PassiveTowedStatusH + 8

	PassiveListX   = 952
	PassiveListY   = PassiveContactLabelY + 22
	PassiveListW   = 310
	PassiveListRow = 22
	PassiveListH   = 310
)
