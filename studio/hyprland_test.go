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
