package main

import "os"

// On Wayland (Hyprland, GNOME, KDE) WebKitGTK hands each painted frame to
// the compositor as a GPU buffer (DMA-BUF). With the Intel and NVIDIA
// cards of a laptop that hand-off can show old and new frames together:
// text redrawn while typing looks doubled, switches flicker. WebKit's own
// copy to shared memory doesn't. Wails already does this when NVIDIA
// renders; here it is on for every Wayland session, unless the variable
// was set by hand (WEBKIT_DISABLE_DMABUF_RENDERER=0 brings the GPU path
// back).
func init() {
	if os.Getenv("WAYLAND_DISPLAY") != "" && os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") == "" {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}
}
