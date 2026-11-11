//go:build darwin && !ios

package blupcloud

import (
	"context"
	"os/exec"
	"regexp"
	"time"

	"github.com/gabriellsalesx/blup/internal/proc"
)

var platformUUID = regexp.MustCompile(`"IOPlatformUUID" = "([0-9A-Fa-f-]{36})"`)

// machineID is the Mac's IOPlatformUUID, the hardware UUID System
// Information shows.
func machineID() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := proc.Hide(exec.CommandContext(ctx, "/usr/sbin/ioreg", "-rd1", "-c", "IOPlatformExpertDevice")).Output()
	if err != nil {
		return ""
	}
	if m := platformUUID.FindSubmatch(out); m != nil {
		return string(m[1])
	}
	return ""
}
