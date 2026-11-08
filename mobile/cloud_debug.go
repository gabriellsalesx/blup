//go:build debug && !production

package main

// The debug build (build-emu.sh, wails3 task android:build) talks to a Blup
// Cloud running on the developer's computer: "adb reverse tcp:18820
// tcp:18820" takes the phone's localhost there. Release builds never have it.
func init() {
	if cloudURL == "" {
		cloudURL = "http://localhost:18820"
	}
}
