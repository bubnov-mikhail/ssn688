//go:build !android && !ios && !ssnmobile

package platform

// Mobile is false on desktop builds (macOS / Windows / Linux without -tags=ssnmobile).
func Mobile() bool { return false }
