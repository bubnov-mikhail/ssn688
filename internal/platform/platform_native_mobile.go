//go:build android || ios

package platform

// NativeMobile is true on Android/iOS device builds (not -tags=ssnmobile desktop).
func NativeMobile() bool { return true }
