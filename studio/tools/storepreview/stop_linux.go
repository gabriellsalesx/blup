//go:build linux && !android

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func stopPreview(binary string) error {
	binary, err := filepath.Abs(binary)
	if err != nil {
		return err
	}
	if filepath.Base(binary) != "blup-studio-store-preview" {
		return errors.New("refusing to stop a non-preview executable")
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return err
	}
	stopped := 0
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		exePath := filepath.Join("/proc", entry.Name(), "exe")
		exe, err := os.Readlink(exePath)
		if err != nil || strings.TrimSuffix(exe, " (deleted)") != binary {
			continue
		}
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			return err
		}
		for i := 0; i < 20; i++ {
			time.Sleep(100 * time.Millisecond)
			current, err := os.Readlink(exePath)
			if err != nil || strings.TrimSuffix(current, " (deleted)") != binary {
				break
			}
		}
		if current, err := os.Readlink(exePath); err == nil && strings.TrimSuffix(current, " (deleted)") == binary {
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
				return err
			}
		}
		stopped++
	}
	fmt.Printf("Stopped %d dedicated Studio preview process(es)\n", stopped)
	return nil
}
