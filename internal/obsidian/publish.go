package obsidian

import (
	"fmt"
	"strings"
)

// Unpublished reports whether a note's properties keep it off the site, as
// Obsidian Publish does: "publish: false" or "private: true". Obsidian
// reads text values too, so "false"/"no" and "true"/"yes" (and the
// Portuguese and Spanish words) count the same.
func Unpublished(props map[string]any) bool {
	if v, ok := props["publish"]; ok && isNo(v) {
		return true
	}
	if v, ok := props["private"]; ok && Yes(v) {
		return true
	}
	return false
}

func word(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.ToLower(strings.TrimSpace(x))
	default:
		return strings.ToLower(strings.TrimSpace(fmt.Sprint(x)))
	}
}

// Yes reports whether a property value means yes: true, "true", "yes",
// "sim", "sí"... (Obsidian reads text values too).
func Yes(v any) bool {
	switch word(v) {
	case "true", "yes", "y", "on", "1", "sim", "s", "sí", "si":
		return true
	}
	return false
}

func isNo(v any) bool {
	switch word(v) {
	case "false", "no", "n", "off", "0", "não", "nao":
		return true
	}
	return false
}
