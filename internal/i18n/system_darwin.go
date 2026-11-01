//go:build darwin

package i18n

import (
	"os/exec"
	"strings"
)

// macOS apps opened from the Dock get no LANG: the preferred languages are
// in the user's defaults.
func systemLocales() []string {
	out := envLocales()
	if b, err := exec.Command("defaults", "read", "-g", "AppleLanguages").Output(); err == nil {
		// ( "pt-BR", "en-US" )
		for _, f := range strings.FieldsFunc(string(b), func(r rune) bool { return r == '(' || r == ')' || r == ',' || r == '"' || r == ' ' || r == '\n' }) {
			out = append(out, f)
		}
	}
	return out
}
