//go:build !(linux || darwin || windows)

package main

import (
	"io/fs"
	"time"
)

// birthTime is unknown here: the list shows only the change time.
func birthTime(string, fs.FileInfo) time.Time { return time.Time{} }
