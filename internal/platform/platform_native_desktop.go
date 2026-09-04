//go:build !android && !ios

package platform

// NativeMobile is false outside Android/iOS (including -tags=ssnmobile on desktop).
func NativeMobile() bool { return false }
