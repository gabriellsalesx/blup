package app

import "testing"

// The stores and the system's packages update the Studio: it doesn't look
// for new versions there.
func TestUpdatesBy(t *testing.T) {
	env := func(kv map[string]string) func(string) string { return func(k string) string { return kv[k] } }
	none := env(nil)
	for _, c := range []struct {
		goos, exe string
		env       func(string) string
		want      string
	}{
		{"windows", `C:\Program Files\WindowsApps\Blup.Studio_1.2.0.0_x64__abc\blup-studio.exe`, none, "msstore"},
		{"windows", `C:\Users\ana\AppData\Local\Programs\Blup Studio\blup-studio.exe`, none, ""},
		{"linux", "/app/bin/blup-studio", env(map[string]string{"FLATPAK_ID": "io.github.gabriellsalesx.BlupStudio"}), "flatpak"},
		{"linux", "/usr/bin/blup-studio", none, "package"},
		{"linux", "/usr/local/bin/blup-studio", none, ""},
		{"linux", "/tmp/.mount_BlupSt/usr/bin/blup-studio", env(map[string]string{"APPIMAGE": "/home/ana/Blup_Studio.AppImage"}), ""},
		{"linux", "/home/ana/.local/bin/blup-studio", none, ""},
		{"darwin", "/Applications/Blup Studio.app/Contents/MacOS/blup-studio", none, ""},
	} {
		if got := updatesBy(c.goos, c.exe, c.env); got != c.want {
			t.Errorf("%s %s = %q, want %q", c.goos, c.exe, got, c.want)
		}
	}
	// A store build says so itself.
	Channel = "msstore"
	defer func() { Channel = "" }()
	if got := updatesBy("windows", `D:\anywhere\blup-studio.exe`, none); got != "msstore" {
		t.Errorf("store build = %q", got)
	}
}

// A store build never asks GitHub for new versions.
func TestStoreBuildDoesntLookUp(t *testing.T) {
	Channel, Version = "msstore", "1.2.0"
	defer func() { Channel, Version = "", "dev" }()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1") // any request would fail
	info, err := CheckUpdate(t.Context(), true)
	if err != nil || info.UpdatesBy != "msstore" || info.Newer || info.Latest != "" {
		t.Fatalf("store build = %+v, %v", info, err)
	}
}
