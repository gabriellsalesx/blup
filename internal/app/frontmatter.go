package app

import "strings"

// splitFront splits a note into its front matter (the YAML between the
// first two --- lines, without them) and its text. A BOM and Windows line
// ends are allowed; an empty front matter (---, ---) is fine. A note
// without one is all text.
func splitFront(text string) (fm, body string) {
	norm := strings.ReplaceAll(strings.TrimPrefix(text, "\ufeff"), "\r\n", "\n")
	if !strings.HasPrefix(norm, "---\n") {
		return "", norm
	}
	rest := norm[4:]
	for at := 0; at <= len(rest); {
		line, after, found := strings.Cut(rest[at:], "\n")
		if strings.TrimRight(line, " \t") == "---" {
			if !found {
				return rest[:at], ""
			}
			return rest[:at], after
		}
		if !found {
			break
		}
		at += len(line) + 1
	}
	return "", norm // never closed: not a front matter
}
