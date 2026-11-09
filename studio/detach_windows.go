package main

import (
	"os/exec"
	"syscall"

	"github.com/gabriellsalesx/blup/internal/proc"
)

// createNewProcessGroup keeps cmd out of this Studio's console group, so
// it goes on after the Studio closes.
const createNewProcessGroup = 0x00000200

// detach starts cmd without a window, apart from this Studio.
func detach(cmd *exec.Cmd) {
	proc.Hide(cmd)
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= createNewProcessGroup
}
