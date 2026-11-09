package main

import (
	"os/exec"
	"syscall"

	"github.com/gabriellsalesx/blup/internal/proc"
)

const (
	// createNewProcessGroup keeps cmd out of this Studio's console group.
	createNewProcessGroup = 0x00000200
	// createBreakawayFromJob takes cmd out of a job the Studio may run in
	// (a launcher's, closed with the Studio), so it outlives the Studio.
	createBreakawayFromJob = 0x01000000
)

// detach starts cmd without a window, apart from this Studio; breakaway
// also leaves the Studio's job, which some jobs refuse.
func detach(cmd *exec.Cmd, breakaway bool) {
	proc.Hide(cmd)
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= createNewProcessGroup
	if breakaway {
		cmd.SysProcAttr.CreationFlags |= createBreakawayFromJob
	}
}
