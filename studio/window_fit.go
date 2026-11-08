package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The window opens at its default size, centered. On a screen shorter than
// that (a small laptop, a virtual machine) a centered window starts above
// the top of the screen, and its title bar, with the buttons to move,
// minimise and close it, is out of reach. fitWindow shrinks it to the
// screen's work area first, then centers it again.

// fitMargin is the room kept around the window on a small screen.
const fitMargin = 48

// fitSize is the window size that fits in a work area of areaW x areaH:
// the same size when it already fits, else at most the area minus the
// margin, never below the minimum unless the area itself is smaller.
func fitSize(w, h, minW, minH, areaW, areaH int) (int, int, bool) {
	if areaW <= 0 || areaH <= 0 || (w+fitMargin <= areaW && h+fitMargin <= areaH) {
		return w, h, false
	}
	fit := func(size, minimum, area int) int {
		if size+fitMargin <= area {
			return size
		}
		size = max(area-fitMargin, minimum)
		return min(size, area)
	}
	return fit(w, minW, areaW), fit(h, minH, areaH), true
}

func fitWindow(win *application.WebviewWindow, minW, minH int) {
	screen, err := win.GetScreen()
	if err != nil || screen == nil {
		return
	}
	w, h := win.Size()
	if nw, nh, changed := fitSize(w, h, minW, minH, screen.WorkArea.Width, screen.WorkArea.Height); changed {
		win.SetSize(nw, nh)
		win.Center()
	}
}
