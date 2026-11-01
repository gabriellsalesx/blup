package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every message exists in every language with the same placeholders.
func TestCatalogsMatch(t *testing.T) {
	verbs := regexp.MustCompile(`%[dsvq]`)
	for _, l := range []Lang{PT, ES} {
		for key, en := range messages[EN] {
			tr, ok := messages[l][key]
			if !ok {
				t.Errorf("missing %s translation for %q", l, key)
				continue
			}
			if a, b := verbs.FindAllString(en, -1), verbs.FindAllString(tr, -1); strings.Join(a, "") != strings.Join(b, "") {
				t.Errorf("%s %q: placeholders differ: %v vs %v", l, key, a, b)
			}
		}
		for key := range messages[l] {
			if _, ok := messages[EN][key]; !ok {
				t.Errorf("missing EN translation for %s key %q", l, key)
			}
		}
	}
}

// Every key used in the code exists in the catalog.
func TestKeysUsedExist(t *testing.T) {
	re := regexp.MustCompile(`i18n\.T\("([a-z_.]+)"`)
	knownRe := regexp.MustCompile(`knownError\{[^,]+, "([a-z_.]+)"\}`)
	err := filepath.WalkDir("../..", func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		b, _ := os.ReadFile(p)
		for _, r := range []*regexp.Regexp{re, knownRe} {
			for _, m := range r.FindAllStringSubmatch(string(b), -1) {
				if strings.HasSuffix(m[1], ".") || strings.HasSuffix(m[1], "_") {
					continue // dynamic key such as "model."+name
				}
				if _, ok := messages[EN][m[1]]; !ok {
					t.Errorf("%s: unknown key %q", p, m[1])
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
