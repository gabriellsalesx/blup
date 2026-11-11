//go:build windows

package main

import (
	"io/fs"
	"syscall"
	"time"
)

// birthTime is the file's creation time, which NTFS keeps.
func birthTime(_ string, st fs.FileInfo) time.Time {
	if a, ok := st.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, a.CreationTime.Nanoseconds())
	}
	return time.Time{}
}
