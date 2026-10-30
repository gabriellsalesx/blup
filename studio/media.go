package main

import (
	"os"
	"path/filepath"
	"runtime"
)

// The window (and the live preview in it) plays audio through the
// system's web view. On Linux that is WebKitGTK, which plays media with
// GStreamer and needs its "good" plugins for MP3 (mpegaudioparse). The
// Studio's packages install them (build/linux/nfpm); this check only keeps
// a system without them from hanging the window: the preview then shows no
// player.

// canPlayMP3 looks for GStreamer's audio parsers (mpegaudioparse) in the
// usual plugin folders; other systems always can.
func canPlayMP3() bool {
	if runtime.GOOS != "linux" {
		return true
	}
	dirs := filepath.SplitList(os.Getenv("GST_PLUGIN_PATH"))
	dirs = append(dirs, "/usr/lib/gstreamer-1.0", "/usr/lib64/gstreamer-1.0", "/usr/local/lib/gstreamer-1.0")
	if m, _ := filepath.Glob("/usr/lib/*-linux-gnu*/gstreamer-1.0"); m != nil {
		dirs = append(dirs, m...)
	}
	for _, d := range dirs {
		if d == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(d, "libgstaudioparsers.so")); err == nil {
			return true
		}
	}
	return false
}
