package kokoro

import (
	"os/exec"
	"syscall"
)

// hideWindow keeps the tools from flashing a console window.
func hideWindow(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }
