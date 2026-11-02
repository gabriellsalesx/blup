package app

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TestStarterFrontMatter checks that the properties of every example note
// are valid YAML once the placeholders are filled in: one bad line (a colon
// in the middle of an unquoted text) and Hugo builds no page at all.
func TestStarterFrontMatter(t *testing.T) {
	placeholder := regexp.MustCompile(`\{\{[^}]*\}\}`)
	fs.WalkDir(starter, "starter", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Fatal(err)
		}
		if d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		b, _ := starter.ReadFile(p)
		s := string(b)
		if !strings.HasPrefix(s, "---\n") {
			return nil
		}
		end := strings.Index(s[4:], "\n---")
		if end < 0 {
			t.Errorf("%s: properties not closed", p)
			return nil
		}
		var props map[string]any
		if err := yaml.Unmarshal([]byte(placeholder.ReplaceAllString(s[4:4+end], "x")), &props); err != nil {
			t.Errorf("%s: %v", p, err)
		}
		return nil
	})
}
