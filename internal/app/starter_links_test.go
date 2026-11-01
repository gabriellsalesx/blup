package app

import (
	"io/fs"
	"net/url"
	"path"
	"regexp"
	"strings"
	"testing"
)

var (
	fenceRe    = regexp.MustCompile("(?ms)^```.*?^```")
	codeSpanRe = regexp.MustCompile("`[^`\n]*`")
	mdLinkRe   = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	wikiLinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	menuItemRe = regexp.MustCompile(`(?m)^  - (.+)$`)
	fileExtRe  = regexp.MustCompile(`(?i)\.(jpg|jpeg|png|webp|svg|gif|pdf|mp3|mp4|zip)$`)
	// calloutTitleRe is the title line of a callout, where [[...]] is text.
	calloutTitleRe = regexp.MustCompile(`(?m)^>\s*\[![^\]]+\].*$`)
)

// TestStarterLinks checks, for every language and model of the starter, that
// the links between notes point to notes that exist in the same tree, that
// images linked into the attachments folder use the folder of that language
// and exist in starter/covers, and that the menu of the home page names
// folders or notes of the tree.
func TestStarterLinks(t *testing.T) {
	for lang, attach := range attachmentsFolders {
		for _, model := range []string{"mochi", "bento", "nori"} {
			root := path.Join("starter", lang, model)
			notes := map[string]bool{} // note names, without .md
			files := map[string]bool{} // paths of files and folders
			fs.WalkDir(starter, root, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					t.Fatal(err)
				}
				files[p] = true
				if !d.IsDir() {
					notes[strings.TrimSuffix(d.Name(), ".md")] = true
				}
				return nil
			})
			if len(files) < 10 {
				t.Fatalf("%s: starter tree missing", root)
			}
			for p := range files {
				if !strings.HasSuffix(p, ".md") {
					continue
				}
				b, _ := starter.ReadFile(p)
				body := codeSpanRe.ReplaceAllString(fenceRe.ReplaceAllString(string(b), ""), "")
				body = calloutTitleRe.ReplaceAllString(body, "")
				for _, m := range mdLinkRe.FindAllStringSubmatch(body, -1) {
					target := m[1]
					if strings.Contains(target, "://") || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") {
						continue
					}
					target, _, _ = strings.Cut(target, "#")
					dec, err := url.PathUnescape(target)
					if err != nil {
						t.Errorf("%s: bad link %q", p, target)
						continue
					}
					if fileExtRe.MatchString(dec) {
						dir := path.Base(path.Dir(dec))
						if dir != attach {
							t.Errorf("%s: %q should be in the %s folder", p, dec, attach)
						}
						if _, err := fs.Stat(starter, path.Join("starter", "covers", path.Base(dec))); err != nil {
							t.Errorf("%s: image %q not in starter/covers", p, dec)
						}
						continue
					}
					if !files[path.Join(path.Dir(p), dec)] {
						t.Errorf("%s: broken link %q", p, m[1])
					}
				}
				for _, m := range wikiLinkRe.FindAllStringSubmatch(body, -1) {
					target, _, _ := strings.Cut(m[1], "|")
					target, _, _ = strings.Cut(target, "#")
					if fileExtRe.MatchString(target) {
						continue // images, some of them meant to be added by the user
					}
					if !notes[path.Base(target)] {
						t.Errorf("%s: broken wikilink [[%s]]", p, m[1])
					}
				}
			}
			home, _ := starter.ReadFile(path.Join(root, "_index.md"))
			front, _, _ := strings.Cut(strings.TrimPrefix(string(home), "---\n"), "\n---")
			_, block, ok := strings.Cut(front, "\nmenu:\n")
			if !ok {
				t.Fatalf("%s: home page has no menu", root)
			}
			for _, line := range strings.Split(block, "\n") {
				m := menuItemRe.FindStringSubmatch(line)
				if m == nil {
					break // end of the menu list
				}
				if strings.Contains(m[1], "|") {
					continue // external link
				}
				item := strings.Trim(m[1], `"'`)
				if !files[path.Join(root, item)] && !files[path.Join(root, item+".md")] {
					t.Errorf("%s: menu item %q is not a folder or note", root, item)
				}
			}
		}
	}
}
