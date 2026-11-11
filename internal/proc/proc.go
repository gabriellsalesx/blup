// Package proc prepares the programs the Studio and the CLI start (Hugo,
// Pagefind, the local voice's tools, PowerShell). On Windows a console
// program started from a window app opens its own console window, a big
// black window in the middle of the screen: Hide stops that. From an
// AppImage, it keeps the AppImage's own libraries away from them.
package proc

import (
	"os"
	"os/exec"
)

// Hide keeps cmd from opening a console window. Call it before Start, Run,
// Output or CombinedOutput. It also gives cmd the system's environment
// when the Studio runs from an AppImage (see Environ); an Env set after
// Hide should start from Environ(), not os.Environ().
func Hide(cmd *exec.Cmd) *exec.Cmd {
	hide(cmd)
	if cmd.Env == nil && os.Getenv("APPDIR") != "" {
		cmd.Env = Environ()
	}
	return cmd
}
