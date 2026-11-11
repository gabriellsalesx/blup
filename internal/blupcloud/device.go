package blupcloud

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
)

// The device's mark: the same Blup Studio or Blup Notes signing in again
// on the same computer or phone (installed again, or after signing out)
// takes the place of its old session on Blup Cloud instead of counting as
// one more device against the plan's limit (cloud/worker/devices.ts).
//
// The mark comes from what the system keeps for the device across installs
// (machine-id on Linux, the platform UUID on macOS, MachineGuid on Windows,
// ANDROID_ID on Android, which Android already scopes to the app's signing
// key). It never leaves as it is: an HMAC under a key of Blup's own turns
// it into an id only Blup's requests carry, which says nothing about the
// device to anyone else (as systemd's machine-id(5) asks of applications).

// DeviceHeader carries the mark when a device asks for a sign-in code.
const DeviceHeader = "X-Blup-Device"

const deviceKey = "blup-device-v1"

var thisDevice struct {
	sync.Mutex
	set        bool
	mark, name string
}

// SetDevice gives the device's system id and its name, for an app that
// reads them from its platform (Blup Notes, from Android). An empty id
// leaves the device unmarked.
func SetDevice(systemID, name string) {
	thisDevice.Lock()
	defer thisDevice.Unlock()
	thisDevice.set = true
	thisDevice.mark = markOf(systemID)
	thisDevice.name = cleanName(name)
}

// DeviceMark is the device's mark ("" when the system gives none).
func DeviceMark() string {
	thisDevice.Lock()
	defer thisDevice.Unlock()
	if !thisDevice.set {
		thisDevice.set = true
		thisDevice.mark = markOf(machineID())
	}
	return thisDevice.mark
}

// deviceName is the name an app gave with SetDevice ("" for the Studio,
// which names itself by the computer's host name).
func deviceName() string {
	thisDevice.Lock()
	defer thisDevice.Unlock()
	return thisDevice.name
}

func markOf(systemID string) string {
	id := strings.TrimSpace(systemID)
	if id == "" {
		return ""
	}
	m := hmac.New(sha256.New, []byte(deviceKey))
	m.Write([]byte(id))
	return hex.EncodeToString(m.Sum(nil))
}

// cleanName keeps a device name fit for the User-Agent and the panel: no
// control characters, nor the signs that delimit it there, at most 64
// characters.
func cleanName(name string) string {
	name = strings.TrimSpace(strings.Map(func(r rune) rune {
		if r == '(' || r == ')' || r == ';' || r < ' ' || r == 0x7f {
			return -1
		}
		return r
	}, name))
	if r := []rune(name); len(r) > 64 {
		name = strings.TrimSpace(string(r[:64]))
	}
	return name
}
