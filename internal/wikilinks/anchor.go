package wikilinks

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// heading is one heading of a note: its plain text and the ID Hugo gives it.
type heading struct {
	text string
	id   string
}

// Slug returns the anchor ID Hugo (goldmark, autoHeadingIDType = "github")
// generates for a heading with the given plain text, without the "-1", "-2"
// suffixes added for repeated headings: lowercase, each space or "-" becomes
// "-", letters of any script, digits and "_" are kept, everything else is
// dropped. An empty result becomes "heading", as in Hugo.
func Slug(text string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(text) {
		switch {
		case r == ' ' || r == '-':
			b.WriteByte('-')
		case r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		}
	}
	if b.Len() == 0 {
		return "heading"
	}
	return b.String()
}

var (
	atxRe       = regexp.MustCompile(`^ {0,3}(#{1,6})(?:[ \t]+(.*?))?[ \t]*$`)
	atxCloseRe  = regexp.MustCompile(`(?:^|[ \t]+)#+$`)
	customIDRe  = regexp.MustCompile(`[ \t]*\{[^{}]*#([^\s{}]+)[^{}]*\}$`)
	setextH1Re  = regexp.MustCompile(`^ {0,3}=+[ \t]*$`)
	setextH2Re  = regexp.MustCompile(`^ {0,3}-+[ \t]*$`)
	listStartRe = regexp.MustCompile(`^ {0,3}(?:[-+*]|\d+[.)])(?:[ \t]|$)`)
	wikiInRe    = regexp.MustCompile(`!?\[\[([^\[\]|]*)(?:\|([^\[\]]*))?\]\]`)
	mdLinkRe    = regexp.MustCompile(`!?\[([^\[\]]*)\]\([^()]*\)`)
)

// parseHeadings lists the headings of a note (ATX and setext, outside
// frontmatter and fenced code) with the IDs Hugo would assign them.
func parseHeadings(text string) []heading {
	lines := strings.Split(text, "\n")
	start := frontmatterLines(lines)
	var (
		out   []heading
		seen  = map[string]bool{}
		fence fenceState
		prev  string // previous paragraph line, for setext headings
	)
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		var id string
		if m := customIDRe.FindStringSubmatchIndex(raw); m != nil {
			id = raw[m[2]:m[3]]
			raw = strings.TrimSpace(raw[:m[0]])
		}
		plain := plainText(raw)
		if id == "" {
			id = Slug(plain)
			if seen[id] {
				for i := 1; ; i++ {
					if c := id + "-" + strconv.Itoa(i); !seen[c] {
						id = c
						break
					}
				}
			}
		}
		seen[id] = true
		out = append(out, heading{text: plain, id: id})
	}
	for _, line := range lines[start:] {
		line = strings.TrimSuffix(line, "\r")
		if fence.inside(line) {
			prev = ""
			continue
		}
		if m := atxRe.FindStringSubmatch(line); m != nil {
			add(atxCloseRe.ReplaceAllString(m[2], ""))
			prev = ""
			continue
		}
		if prev != "" && (setextH1Re.MatchString(line) || setextH2Re.MatchString(line)) {
			add(prev)
			prev = ""
			continue
		}
		t := strings.TrimSpace(line)
		if t == "" || listStartRe.MatchString(line) || strings.HasPrefix(t, ">") || strings.HasPrefix(line, "    ") {
			prev = ""
		} else {
			prev = t
		}
	}
	return out
}

// plainText approximates the text Hugo renders for a heading's inline
// Markdown: links keep their text, emphasis/code/highlight markers go away.
func plainText(s string) string {
	s = wikiInRe.ReplaceAllStringFunc(s, func(m string) string {
		sm := wikiInRe.FindStringSubmatch(m)
		if sm[2] != "" {
			return sm[2]
		}
		return sm[1]
	})
	s = mdLinkRe.ReplaceAllString(s, "$1")
	s = strings.NewReplacer("*", "", "`", "", "~~", "", "==", "").Replace(s)
	// A run of "_" is emphasis only at a word boundary (snake_case keeps it).
	r := []rune(s)
	var b strings.Builder
	for i := 0; i < len(r); {
		if r[i] != '_' {
			b.WriteRune(r[i])
			i++
			continue
		}
		j := i
		for j < len(r) && r[j] == '_' {
			j++
		}
		if i > 0 && isAlnum(r[i-1]) && j < len(r) && isAlnum(r[j]) {
			b.WriteString(string(r[i:j]))
		}
		i = j
	}
	return b.String()
}

func isAlnum(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// normHeading reduces heading text to lowercase words so that a wikilink
// heading (where Obsidian replaces characters like ":" or "#") matches the
// real heading.
func normHeading(s string) string {
	f := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return strings.Join(f, " ")
}

// findAnchor returns the Hugo ID of the heading in hs that the wikilink
// heading h refers to.
func findAnchor(hs []heading, h string) (string, bool) {
	want := normHeading(plainText(h))
	for _, x := range hs {
		if x.text == h {
			return x.id, true
		}
	}
	for _, x := range hs {
		if normHeading(x.text) == want {
			return x.id, true
		}
	}
	return "", false
}

// frontmatterLines returns how many leading lines form a YAML frontmatter
// block (0 when there is none).
func frontmatterLines(lines []string) int {
	if len(lines) == 0 || trimEOL(strings.TrimPrefix(lines[0], "\uFEFF")) != "---" {
		return 0
	}
	for i := 1; i < len(lines); i++ {
		if t := strings.TrimRight(trimEOL(lines[i]), " \t"); t == "---" || t == "..." {
			return i + 1
		}
	}
	return 0
}

func trimEOL(s string) string { return strings.TrimRight(s, "\r\n") }

// fenceState tracks whether we are inside a fenced code block.
type fenceState struct {
	char byte // '`' or '~' while open, 0 when closed
	n    int  // length of the opening fence
}

// inside updates the state with line and reports whether the line belongs to
// a fenced code block (opening and closing fence lines included).
func (f *fenceState) inside(line string) bool {
	s := stripContainers(trimEOL(line))
	if f.char != 0 {
		n := countRun(s, f.char)
		if n >= f.n && strings.TrimSpace(s[n:]) == "" {
			f.char, f.n = 0, 0
		}
		return true
	}
	for _, c := range []byte{'`', '~'} {
		n := countRun(s, c)
		if n >= 3 {
			if c == '`' && strings.Contains(s[n:], "`") {
				return false // not a fence: info string can't hold a backtick
			}
			f.char, f.n = c, n
			return true
		}
	}
	return false
}

// stripContainers removes leading indentation and blockquote markers so that
// fences inside lists and blockquotes are recognised.
func stripContainers(s string) string {
	return strings.TrimLeft(s, " \t>")
}

func countRun(s string, c byte) int {
	n := 0
	for n < len(s) && s[n] == c {
		n++
	}
	return n
}
