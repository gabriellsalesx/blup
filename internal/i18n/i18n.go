// Package i18n holds every user-facing string of the CLI in English and
// Portuguese. Keys are short English identifiers; T falls back to English
// and then to the key itself, so a missing translation never breaks output.
package i18n

import (
	"fmt"
)

// Lang is a supported interface language.
type Lang string

const (
	EN Lang = "en"
	PT Lang = "pt"
)

var current = EN

// Set changes the active language.
func Set(l Lang) {
	if l == PT {
		current = PT
		return
	}
	current = EN
}

// Current returns the active language.
func Current() Lang { return current }

// T returns the translation of key, formatted with args when given.
func T(key string, args ...any) string {
	s, ok := messages[current][key]
	if !ok {
		if s, ok = messages[EN][key]; !ok {
			s = key
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}
