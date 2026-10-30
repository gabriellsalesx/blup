package main

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// The window (and the live preview in it) plays audio through the
// system's web view. On Linux that is WebKitGTK, which plays media with
// GStreamer and needs its "good" plugins for MP3 (mpegaudioparse); many
// systems (Arch, Omarchy) don't install them with WebKitGTK. Without them,
// trying to play the narration can hang the window, so the Studio checks
// first and tells how to install them.

// MediaSupport tells whether the window can play the narration (MP3) and,
// when it can't, the command that installs what is missing.
type MediaSupport struct {
	MP3     bool   `json:"mp3"`
	Command string `json:"command"`
}

// Media checks the window's MP3 support.
func (s *Studio) Media() MediaSupport {
	if canPlayMP3() {
		return MediaSupport{MP3: true}
	}
	return MediaSupport{Command: installCommand()}
}

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

// installCommand is the command that installs GStreamer's good plugins on
// this distribution (from /etc/os-release).
func installCommand() string {
	ids := ""
	if f, err := os.Open("/etc/os-release"); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if k, v, ok := strings.Cut(sc.Text(), "="); ok && (k == "ID" || k == "ID_LIKE") {
				ids += " " + strings.Trim(v, `"`)
			}
		}
		f.Close()
	}
	has := func(names ...string) bool {
		for _, n := range names {
			if strings.Contains(" "+ids+" ", " "+n+" ") {
				return true
			}
		}
		return false
	}
	switch {
	case has("arch", "omarchy", "manjaro", "endeavouros", "cachyos"):
		return "sudo pacman -S gst-plugins-good"
	case has("fedora", "rhel", "centos"):
		return "sudo dnf install gstreamer1-plugins-good"
	case has("opensuse", "suse"):
		return "sudo zypper install gstreamer-plugins-good"
	default:
		return "sudo apt install gstreamer1.0-plugins-good"
	}
}
