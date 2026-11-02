package main

import (
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// A voice sample played by the system itself, on Linux: WebKitGTK's Web
// Audio opened its stream but sent only silence to Bluetooth headphones
// (measured on a PipeWire computer), while the system's players reach
// them. The window still decodes the sample (for its length) and draws
// the ring; this only makes the sound. macOS and Windows keep Web Audio.

var sound struct {
	mu  sync.Mutex
	cmd *exec.Cmd
}

// soundPlayer is a command playing file with the system's own player:
// GStreamer's (which WebKitGTK needs anyway), else PipeWire's or
// PulseAudio's; nil when none is there.
func soundPlayer(file string) *exec.Cmd {
	if p, err := exec.LookPath("gst-launch-1.0"); err == nil {
		return exec.Command(p, "-q", "playbin", "uri="+(&url.URL{Scheme: "file", Path: filepath.ToSlash(file)}).String())
	}
	for _, name := range []string{"pw-play", "paplay"} {
		if p, err := exec.LookPath(name); err == nil {
			return exec.Command(p, file)
		}
	}
	return nil
}

// SystemSound tells whether samples play through the system (Linux with a
// player); otherwise the window plays them.
func (s *Studio) SystemSound() bool {
	return runtime.GOOS == "linux" && soundPlayer("x") != nil
}

// PlaySound plays an MP3 through the system, stopping the one playing; the
// "soundEnded" event says when it ends.
func (s *Studio) PlaySound(mp3 []byte) error {
	s.StopSound()
	dir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	file := filepath.Join(dir, "blup", "sample-playing.mp3")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(file, mp3, 0o600); err != nil {
		return err
	}
	cmd := soundPlayer(file)
	if cmd == nil {
		return errors.New("no sound player")
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	sound.mu.Lock()
	sound.cmd = cmd
	sound.mu.Unlock()
	go func() {
		cmd.Wait()
		sound.mu.Lock()
		mine := sound.cmd == cmd
		if mine {
			sound.cmd = nil
		}
		sound.mu.Unlock()
		if mine {
			application.Get().Event.Emit("soundEnded", nil)
		}
	}()
	return nil
}

// StopSound stops the sample playing, if any.
func (s *Studio) StopSound() {
	sound.mu.Lock()
	cmd := sound.cmd
	sound.cmd = nil
	sound.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
	}
}
