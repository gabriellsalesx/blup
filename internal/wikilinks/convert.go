package wikilinks

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Change is one wikilink rewritten in a note.
type Change struct {
	File string // vault-relative, slash-separated
	Line int    // 1-based
	Old  string // the original wikilink, e.g. "[[Note|alias]]"
	New  string // its replacement, e.g. "[alias](Note.md)"
}

// Problem is a wikilink that could not be converted, or was converted with
// loss (Warning is true then: e.g. a note embed that became a plain link).
type Problem struct {
	File    string
	Line    int
	Link    string
	Kind    Kind   // what went wrong, for callers that translate messages
	Reason  string // English description with details
	Warning bool   // true: the link was converted anyway; false: left untouched
}

// Kind classifies a Problem.
type Kind string

const (
	KindSectionEmbed  Kind = "section-embed"
	KindSameNoteBlock Kind = "same-note-block"
	KindEmpty         Kind = "empty"
	KindNotFound      Kind = "not-found"
	KindOutside       Kind = "outside"
	KindTransclusion  Kind = "transclusion"
	KindFileEmbed     Kind = "file-embed"
	KindHeading       Kind = "heading"
)

// ConvertText rewrites the wikilinks of one note. notePath is the note's
// path relative to the vault root; it decides how relative destinations and
// same-name ambiguities are resolved. See the package documentation for the
// output conventions.
func ConvertText(idx *Index, notePath, text string) (out string, changes []Change, problems []Problem) {
	return convertText(idx, notePath, text, "")
}

// convertText is ConvertText with a scope: when scope is not "", links to
// files outside that vault folder are converted but reported as warnings,
// since they will not be part of the generated site.
func convertText(idx *Index, notePath, text, scope string) (string, []Change, []Problem) {
	notePath = path.Clean(filepath.ToSlash(notePath))
	c := &converter{idx: idx, note: notePath, srcDir: path.Dir(notePath), self: text, scope: scope}

	lines := strings.SplitAfter(text, "\n")
	start := frontmatterLines(lines)
	var b strings.Builder
	b.Grow(len(text))
	for _, l := range lines[:start] {
		b.WriteString(l)
	}
	var fence fenceState
	for i := start; i < len(lines); i++ {
		if fence.inside(lines[i]) {
			b.WriteString(lines[i])
			continue
		}
		b.WriteString(c.line(lines[i], i+1))
	}
	return b.String(), c.changes, c.problems
}

type converter struct {
	idx      *Index
	note     string
	srcDir   string
	self     string
	scope    string
	selfHead []heading
	parsed   bool
	changes  []Change
	problems []Problem
}

// line converts the wikilinks of one line outside code blocks, skipping
// inline code spans and backslash-escaped brackets.
func (c *converter) line(s string, n int) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		switch {
		case s[i] == '\\' && i+1 < len(s):
			b.WriteString(s[i : i+2])
			i += 2
			continue
		case s[i] == '`':
			run := countRun(s[i:], '`')
			if end := closingTicks(s, i+run, run); end >= 0 {
				b.WriteString(s[i:end])
				i = end
			} else {
				b.WriteString(s[i : i+run])
				i += run
			}
			continue
		case strings.HasPrefix(s[i:], "[[") || strings.HasPrefix(s[i:], "![["):
			embed := s[i] == '!'
			open := i + 2
			if embed {
				open++
			}
			if end := strings.Index(s[open:], "]]"); end >= 0 {
				inner := s[open : open+end]
				if !strings.Contains(inner, "[[") && !strings.ContainsAny(inner, "\r\n") {
					raw := s[i : open+end+2]
					b.WriteString(c.link(raw, inner, embed, n))
					i = open + end + 2
					continue
				}
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// closingTicks finds the end (exclusive) of the backtick run of exactly
// length n that closes an inline code span opened before from, or -1.
func closingTicks(s string, from, n int) int {
	for i := from; i < len(s); {
		if s[i] != '`' {
			i++
			continue
		}
		run := countRun(s[i:], '`')
		if run == n {
			return i + run
		}
		i += run
	}
	return -1
}

var sizeRe = regexp.MustCompile(`^\s*\d+(?:x\d+)?\s*$`)

// link converts a single wikilink; raw is the whole "[[...]]" text and inner
// what is between the brackets.
func (c *converter) link(raw, inner string, embed bool, line int) string {
	target, alias, hasAlias := strings.Cut(inner, "|")
	target = strings.TrimSuffix(target, `\`) // "\|" escapes the pipe inside tables
	alias = strings.TrimSpace(alias)
	file, sub, hasSub := strings.Cut(target, "#")
	file = strings.TrimSpace(file)

	var heading string
	block := false
	if hasSub {
		parts := strings.Split(sub, "#")
		heading = strings.TrimSpace(parts[len(parts)-1])
		if strings.HasPrefix(heading, "^") {
			block, heading = true, ""
		}
	}

	problem := func(kind Kind, reason string, warning bool) {
		c.problems = append(c.problems, Problem{File: c.note, Line: line, Link: raw, Kind: kind, Reason: reason, Warning: warning})
	}
	done := func(out string) string {
		c.changes = append(c.changes, Change{File: c.note, Line: line, Old: raw, New: out})
		return out
	}

	// Same-note heading: [[#Heading]].
	if file == "" {
		switch {
		case heading != "":
			anchor := c.anchor(c.note, heading, problem)
			text := heading
			if hasAlias && alias != "" {
				text = alias
			}
			if embed {
				problem(KindSectionEmbed, "embed of a section of the same note converted to a plain link", true)
			}
			return done(fmt.Sprintf("[%s](#%s)", escapeText(text), anchor))
		case block:
			problem(KindSameNoteBlock, "block reference within the same note cannot be converted", false)
		default:
			problem(KindEmpty, "empty link", false)
		}
		return raw
	}

	dest, ok := c.idx.Resolve(c.note, file)
	if !ok {
		problem(KindNotFound, "target not found in the vault", false)
		return raw
	}
	// Attachments outside the site folder are still published (the site
	// mounts the whole vault's attachments), so only notes are reported.
	if c.scope != "" && !inScope(dest, c.scope) && isNote(dest) {
		problem(KindOutside, fmt.Sprintf("target %q is outside %q and will not be part of the site", dest, c.scope), true)
	}

	url := escapePath(relPath(c.srcDir, dest))
	note := isNote(dest)

	// Image embed: ![[img.png]], ![[img.png|300]], ![[img.png|caption]].
	if embed && !note && isImage(dest) {
		alt := strings.TrimSuffix(path.Base(dest), path.Ext(dest))
		width := ""
		if hasAlias {
			if sizeRe.MatchString(alias) {
				width = strings.TrimSpace(alias)
			} else if alias != "" {
				alt = alias
			}
		}
		alt = escapeText(alt)
		if width != "" {
			alt += "|" + width
		}
		return done(fmt.Sprintf("![%s](%s)", alt, url))
	}

	if heading != "" {
		if note {
			url += "#" + c.anchor(dest, heading, problem)
		} else {
			url += "#" + strings.ReplaceAll(heading, " ", "%20") // e.g. doc.pdf#page=3
		}
	}
	text := file
	if heading != "" {
		text = file + " > " + heading
	}
	if hasAlias && alias != "" && !(embed && sizeRe.MatchString(alias)) {
		text = alias
	}
	if embed {
		if note {
			problem(KindTransclusion, "note embed (transclusion) is not supported by Hugo; converted to a plain link", true)
		} else {
			problem(KindFileEmbed, "embed of a non-image attachment converted to a plain link", true)
		}
	}
	return done(fmt.Sprintf("[%s](%s)", escapeText(text), url))
}

// anchor returns the Hugo heading ID for heading in the note at notePath.
// When the heading cannot be found the plain slug is used and a warning is
// reported.
func (c *converter) anchor(notePath, h string, problem func(Kind, string, bool)) string {
	var hs []heading
	if notePath == c.note {
		if !c.parsed {
			c.selfHead, c.parsed = parseHeadings(c.self), true
		}
		hs = c.selfHead
	} else {
		hs = c.idx.noteHeadings(notePath)
	}
	if id, ok := findAnchor(hs, h); ok {
		return id
	}
	problem(KindHeading, fmt.Sprintf("heading %q not found in %s; anchor guessed", h, notePath), true)
	return Slug(plainText(h))
}

func inScope(p, scope string) bool {
	scope = strings.Trim(path.Clean(filepath.ToSlash(scope)), "/")
	return scope == "" || scope == "." || strings.HasPrefix(p, scope+"/")
}

// relPath returns the slash path of to relative to the directory fromDir.
func relPath(fromDir, to string) string {
	r, err := filepath.Rel(filepath.FromSlash(fromDir), filepath.FromSlash(to))
	if err != nil {
		return to
	}
	return filepath.ToSlash(r)
}

// escapePath percent-escapes each segment of a relative slash path the way
// Obsidian writes Markdown links: spaces and characters that would break the
// link are escaped, other (including non-ASCII) characters are kept.
func escapePath(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		var b strings.Builder
		for j := 0; j < len(s); j++ {
			ch := s[j]
			if ch <= 0x20 || ch == 0x7f || strings.IndexByte("%()<>#?[]\\^|\"`", ch) >= 0 {
				fmt.Fprintf(&b, "%%%02X", ch)
			} else {
				b.WriteByte(ch)
			}
		}
		segs[i] = b.String()
	}
	return strings.Join(segs, "/")
}

// escapeText escapes characters that would end or break Markdown link text.
func escapeText(s string) string {
	return strings.NewReplacer(`\`, `\\`, `[`, `\[`, `]`, `\]`).Replace(s)
}
