package i18n

import (
	"os"
	"strings"
)

// System is the computer's language as Blup speaks it: Portuguese when the
// system is in Portuguese (Brazil or elsewhere), Spanish when it is in
// Spanish, English otherwise.
func System() Lang {
	for _, l := range systemLocales() {
		l = strings.ToLower(l)
		if l == "" || l == "c" || l == "posix" || strings.HasPrefix(l, "c.") {
			continue
		}
		if strings.HasPrefix(l, "pt") {
			return "pt"
		}
		if strings.HasPrefix(l, "es") {
			return "es"
		}
		return "en"
	}
	return "en"
}

// envLocales are the POSIX locale variables, most specific first.
func envLocales() []string {
	var out []string
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" {
			out = append(out, v)
		}
	}
	// LANGUAGE is a list ("pt_BR:en").
	if v := os.Getenv("LANGUAGE"); v != "" {
		out = append(out, strings.Split(v, ":")...)
	}
	return out
}
