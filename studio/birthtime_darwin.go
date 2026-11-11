//go:build darwin

package main

import (
	"io/fs"
	"syscall"
	"time"
)

// birthTime is the file's creation time, which APFS and HFS+ keep.
func birthTime(_ string, st fs.FileInfo) time.Time {
	if s, ok := st.Sys().(*syscall.Stat_t); ok {
		return time.Unix(s.Birthtimespec.Sec, s.Birthtimespec.Nsec)
	}
	return time.Time{}
}
