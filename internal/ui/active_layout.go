package ui

import "github.com/bubnov-mikhail/ssn688/internal/layout"

const (
	activePanelX          = 20
	activePanelY          = 50
	activeSideY           = 50
	activeSideW           = layout.PassiveSidePanelW
	activeSideLabelY      = 56
	activeControlsY       = 90
	activeControlLabelW   = 90
	activePingRowY        = activeControlsY + 88
	activePowerRowY       = activePingRowY + 36
	activeRangeScaleY     = activePowerRowY + 40
	activeListY           = 318
	activeListVisibleRows = 17
	activePlotX           = 40
	activePlotY           = 152
	activePlotH           = 528
	activeFlashCrossMin   = 5.0
	activeFlashCrossMax   = 11.0
	activePlotBgR         = 0
	activePlotBgG         = 2
	activePlotBgB         = 16
)

func activePanelW() int { return layout.PassiveMainPanelW }
func activeSideX() int  { return layout.PassiveSidePanelX }
func activeControlsX() int {
	return layout.PassiveSidePanelX + 20
}
func activeControlBtnX() int {
	return activeControlsX() + activeControlLabelW
}
func activePlotW() int { return layout.PassiveMainPanelW - 40 }
