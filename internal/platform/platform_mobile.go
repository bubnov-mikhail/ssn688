//go:build android || ios || ssnmobile

package platform

// Mobile is true for Android/iOS targets and desktop builds with -tags=ssnmobile.
func Mobile() bool { return true }
