//go:build android

package webrtc

import "github.com/wlynxg/anet"

// Built without cgo, anet cannot read the Android API level and falls back to the standard
// interface listing, which Android 11 and later refuse to apps (netlinkrib: permission denied).
// Its Android path (RTM_GETADDR only) works on every version the player supports, so use it always.
func init() {
	anet.SetAndroidVersion(11)
}
