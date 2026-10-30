//go:build !windows

package kokoro

import "os/exec"

func hideWindow(*exec.Cmd) {}
