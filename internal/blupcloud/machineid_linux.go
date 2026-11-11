//go:build linux && !android

package blupcloud

import (
	"os"
	"strings"
)

// machineID is systemd's machine-id (also inside a Flatpak, which shares
// it), or D-Bus's on systems without systemd.
func machineID() string {
	for _, f := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if b, err := os.ReadFile(f); err == nil {
			if id := strings.TrimSpace(string(b)); id != "" {
				return id
			}
		}
	}
	return ""
}
