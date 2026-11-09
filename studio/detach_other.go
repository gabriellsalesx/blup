//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// detach starts cmd in a session of its own: it outlives this Studio and
// a terminal closing never stops it.
func detach(cmd *exec.Cmd, _ bool) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
