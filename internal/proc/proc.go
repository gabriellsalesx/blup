// Package proc prepares the programs the Studio and the CLI start (Hugo,
// Pagefind, the local voice's tools, PowerShell). On Windows a console
// program started from a window app opens its own console window, a big
// black window in the middle of the screen: Hide stops that. Elsewhere it
// does nothing.
package proc

import "os/exec"

// Hide keeps cmd from opening a console window. Call it before Start, Run,
// Output or CombinedOutput.
func Hide(cmd *exec.Cmd) *exec.Cmd {
	hide(cmd)
	return cmd
}
