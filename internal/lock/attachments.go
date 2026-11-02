package lock

import (
	"encoding/base64"
	"html"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// The vault's attachments are published under /vault/ (the theme mounts the
// vault as Hugo assets), with their original names and as processed images
// ("name_hu_<hash>.webp"). Hugo publishes every one a page uses, also when
// the page is protected or the use is inside an Obsidian comment (%% %%),
// which the theme removes from the HTML afterwards. So:
//
//   - Process puts the attachments of a protected post inside its
//     ciphertext, as data: URLs (only someone with the password sees them);
//   - Prune then deletes every attachment no published page refers to.
//
// An attachment also used by a public page stays published.

// attachmentTypes are the media types of the files Prune may delete: the
// attachments the site mounts (internal/scaffold.Attachments) and the
// images Hugo makes from them.
var attachmentTypes = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".webp": "image/webp",
	".avif": "image/avif", ".gif": "image/gif", ".svg": "image/svg+xml", ".bmp": "image/bmp",
	".tif": "image/tiff", ".tiff": "image/tiff", ".pdf": "application/pdf",
	".mp3": "audio/mpeg", ".m4a": "audio/mp4", ".ogg": "audio/ogg", ".wav": "audio/wav",
	".flac": "audio/flac", ".mp4": "video/mp4", ".webm": "video/webm", ".mov": "video/quicktime",
	".zip": "application/zip",
}

// MaxInline is the largest attachment put inside a protected post. A larger
// one is left out of the site (unless a public page uses it): the post
// shows it broken after it is opened.
const MaxInline = 10 << 20

// vaultFile is the published attachment a URL in dir points to ("" when it
// is not one). The URL may be relative to the site, carry the site's own
// path (GitHub Pages: /repo/vault/...) or be absolute.
func vaultFile(dir, ref string) string {
	ref = html.UnescapeString(strings.TrimSpace(ref))
	u, err := url.Parse(ref)
	if err != nil || u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	p := u.Path
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	for i := strings.Index(p, "/vault/"); i >= 0; {
		rel := path.Clean(p[i+1:])
		if strings.HasPrefix(rel, "vault/") && filepath.IsLocal(filepath.FromSlash(rel)) {
			f := filepath.Join(dir, filepath.FromSlash(rel))
			if _, ok := attachmentTypes[strings.ToLower(path.Ext(rel))]; ok {
				if info, err := os.Lstat(f); err == nil && info.Mode().IsRegular() {
					return f
				}
			}
		}
		next := strings.Index(p[i+1:], "/vault/")
		if next < 0 {
			break
		}
		i += 1 + next
	}
	return ""
}

// Attribute values come quoted or, after Hugo's --minify, bare.
const attrValue = `(?:"([^"]*)"|'([^']*)'|([^\s"'>]+))`

var (
	urlAttr    = regexp.MustCompile(`(?i)(\s)(src|href|poster)=` + attrValue)
	srcsetAttr = regexp.MustCompile(`(?i)\s(?:srcset|sizes)=` + attrValue)
	imgTag     = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	srcsetOne  = regexp.MustCompile(`(?i)\ssrcset=` + attrValue)
	srcOne     = regexp.MustCompile(`(?i)(\s)src=` + attrValue)
)

// value is the attribute value matched by attrValue at groups i..i+2.
func value(m []string, i int) string { return m[i] + m[i+1] + m[i+2] }

// inlineAttachments rewrites the HTML of a protected post so its
// attachments travel inside it, as data: URLs.
func inlineAttachments(dir, plain string) string {
	// An image with several widths: keep one (the largest up to 1200 px)
	// as its only source.
	plain = imgTag.ReplaceAllStringFunc(plain, func(tag string) string {
		m := srcsetOne.FindStringSubmatch(tag)
		if m == nil {
			return tag
		}
		if best := pickWidth(html.UnescapeString(value(m, 1))); best != "" && vaultFile(dir, best) != "" {
			tag = srcOne.ReplaceAllStringFunc(tag, func(a string) string {
				return srcOne.FindStringSubmatch(a)[1] + `src="` + html.EscapeString(best) + `"`
			})
		}
		return srcsetAttr.ReplaceAllString(tag, "")
	})
	return urlAttr.ReplaceAllStringFunc(plain, func(attr string) string {
		m := urlAttr.FindStringSubmatch(attr)
		f := vaultFile(dir, value(m, 3))
		if f == "" {
			return attr
		}
		data, ok := dataURL(f)
		if !ok {
			return attr
		}
		out := m[1] + m[2] + `="` + data + `"`
		if strings.EqualFold(m[2], "href") {
			// Browsers do not open data: links in the page; they save them.
			out += ` download="` + html.EscapeString(filepath.Base(f)) + `"`
		}
		return out
	})
}

// pickWidth chooses from a srcset the widest image up to 1200 px wide, or
// the narrowest when all are wider.
func pickWidth(srcset string) string {
	best, bestW, small, smallW := "", 0, "", 0
	for _, part := range strings.Split(srcset, ",") {
		f := strings.Fields(part)
		if len(f) == 0 {
			continue
		}
		w := 0
		if len(f) > 1 {
			w, _ = strconv.Atoi(strings.TrimSuffix(f[1], "w"))
		}
		if w <= 1200 && w >= bestW {
			best, bestW = f[0], w
		}
		if small == "" || w < smallW {
			small, smallW = f[0], w
		}
	}
	if best == "" {
		return small
	}
	return best
}

func dataURL(file string) (string, bool) {
	info, err := os.Stat(file)
	if err != nil || info.Size() > MaxInline {
		return "", false
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return "", false
	}
	return "data:" + attachmentTypes[strings.ToLower(filepath.Ext(file))] + ";base64," + base64.StdEncoding.EncodeToString(b), true
}

// textFiles are the published files that can refer to an attachment.
var textFiles = []string{".html", ".htm", ".xml", ".json", ".js", ".mjs", ".css", ".txt", ".webmanifest", ".svg"}

// vaultRef finds what follows "vault/" in a URL written in a file.
var vaultRef = regexp.MustCompile(`vault/[^"'<>\s\\]+`)

// Prune deletes the attachments under dir/vault that no published file
// refers to: the ones used only by protected posts (now inside their
// ciphertext) or only inside comments. It returns how many it deleted.
// Run it after Process.
func Prune(dir string) (int, error) {
	vault := filepath.Join(dir, "vault")
	if info, err := os.Lstat(vault); err != nil || !info.IsDir() {
		return 0, nil
	}
	used := map[string]bool{} // lowercase, slash-separated, from dir
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !d.Type().IsRegular() || !slices.Contains(textFiles, strings.ToLower(filepath.Ext(p))) {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		text := strings.ReplaceAll(string(b), `\/`, "/")
		for _, ref := range vaultRef.FindAllString(text, -1) {
			ref = html.UnescapeString(ref)
			if i := strings.IndexAny(ref, "?#"); i >= 0 {
				ref = ref[:i]
			}
			if u, err := url.PathUnescape(ref); err == nil {
				ref = u
			}
			used[strings.ToLower(path.Clean(ref))] = true
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	removed := 0
	var dirs []string
	err = filepath.WalkDir(vault, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, p)
			return nil
		}
		if _, ok := attachmentTypes[strings.ToLower(filepath.Ext(p))]; !ok {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		if used[strings.ToLower(filepath.ToSlash(rel))] {
			return nil
		}
		if err := os.Remove(p); err != nil {
			return err
		}
		removed++
		return nil
	})
	// Folders left empty, deepest first.
	for i := len(dirs) - 1; i >= 0; i-- {
		os.Remove(dirs[i]) // fails, as it should, when not empty
	}
	return removed, err
}
