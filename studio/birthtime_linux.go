//go:build linux

package main

import (
	"io/fs"
	"time"

	"golang.org/x/sys/unix"
)

// birthTime is the file's creation time (statx's btime), zero when the
// file system doesn't keep one.
func birthTime(path string, _ fs.FileInfo) time.Time {
	var sx unix.Statx_t
	if unix.Statx(unix.AT_FDCWD, path, unix.AT_STATX_SYNC_AS_STAT, unix.STATX_BTIME, &sx) != nil || sx.Mask&unix.STATX_BTIME == 0 {
		return time.Time{}
	}
	return time.Unix(sx.Btime.Sec, int64(sx.Btime.Nsec))
}
