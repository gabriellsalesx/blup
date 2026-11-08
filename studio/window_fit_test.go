package main

import "testing"

func TestFitSize(t *testing.T) {
	for _, c := range []struct {
		name               string
		w, h, areaW, areaH int
		wantW, wantH       int
		wantChanged        bool
	}{
		{"fits on a large screen", 1360, 860, 1920, 1040, 1360, 860, false},
		{"shorter screen: only the height shrinks", 1360, 860, 1920, 760, 1360, 712, true},
		{"small screen: both shrink", 1360, 860, 1280, 720, 1232, 672, true},
		{"never below the minimum", 1360, 860, 960, 600, 912, 560, true},
		{"smaller than the minimum: the whole area", 1360, 860, 800, 500, 800, 500, true},
		{"unknown screen: unchanged", 1360, 860, 0, 0, 1360, 860, false},
	} {
		w, h, changed := fitSize(c.w, c.h, 900, 560, c.areaW, c.areaH)
		if w != c.wantW || h != c.wantH || changed != c.wantChanged {
			t.Errorf("%s: got %dx%d %v, want %dx%d %v", c.name, w, h, changed, c.wantW, c.wantH, c.wantChanged)
		}
	}
}
