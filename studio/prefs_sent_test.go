package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadSentDamaged: a damaged reminders-sent.json never leaves a nil
// map (writing to it would panic) nor a time that is no time.
func TestReadSentDamaged(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"null": "null", "array": "[1,2]", "string": `"x"`, "wrong type": `{"a":"b"}`, "not json": "{", "empty": "",
	} {
		f := filepath.Join(dir, "sent.json")
		if err := os.WriteFile(f, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		got := readSent(f)
		if got == nil || len(got) != 0 {
			t.Errorf("%s: got %v", name, got)
		}
		got["k"] = 1 // must not panic
	}
	f := filepath.Join(dir, "ok.json")
	os.WriteFile(f, []byte(`{"a":10,"b":-1,"":5,"c":0}`), 0o600)
	if got := readSent(f); len(got) != 1 || got["a"] != 10 {
		t.Errorf("valid entries: got %v", got)
	}
	if got := readSent(filepath.Join(dir, "missing.json")); got == nil {
		t.Error("missing file gave nil")
	}
}

// TestPrefsDamaged: a damaged prefs file gives the defaults, never half of
// it, and values out of range go back to the default.
func TestPrefsDamaged(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	f, err := prefsFile()
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(f), 0o755)
	s := &Studio{}
	def := defaultPrefs()
	// A late error: theme would be applied before the bad field fails.
	for _, body := range []string{"null", "[]", `{"theme":"dark","editorFont":"big"}`, "{"} {
		os.WriteFile(f, []byte(body), 0o644)
		p := s.Prefs()
		if p.Theme != def.Theme || p.EditorFont != def.EditorFont || !p.Notifications {
			t.Errorf("%s: got %+v", body, p)
		}
	}
	os.WriteFile(f, []byte(`{"theme":"neon","editorWidth":"x","editorFont":-5,"termFont":9999,"spellLang":"zz","sideWidth":-1,"previewWidth":1000000,"tray":true}`), 0o644)
	p := s.Prefs()
	if p.Theme != "system" || p.EditorWidth != "narrow" || p.EditorFont != 16 || p.TermFont != 13 || p.SpellLang != "" || p.SideWidth != 0 || p.PreviewWidth != 0 || !p.Tray {
		t.Errorf("out of range: got %+v", p)
	}
	if err := s.SavePrefs(Prefs{Theme: "dark", EditorFont: 1000, EditorWidth: "wide", TermFont: 14}); err != nil {
		t.Fatal(err)
	}
	if p := s.Prefs(); p.Theme != "dark" || p.EditorFont != 16 || p.EditorWidth != "wide" || p.TermFont != 14 {
		t.Errorf("saved: got %+v", p)
	}
}
