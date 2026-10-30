package narrate

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxPart is the longest part sent to the voice model at once (the model
// takes about 2,000 tokens; 2,500 characters stay well under it).
const MaxPart = 2500

var (
	reComment    = regexp.MustCompile(`(?s)%%.*?%%`)
	reHTMLCmt    = regexp.MustCompile(`(?s)<!--.*?-->`)
	reFence      = regexp.MustCompile("(?ms)^\\s*(```|~~~).*?^\\s*(```|~~~)\\s*$")
	reMathBlock  = regexp.MustCompile(`(?s)\$\$.*?\$\$`)
	reMathInline = regexp.MustCompile(`\$[^$\n]+\$`)
	reEmbed      = regexp.MustCompile(`!\[\[[^\]]*\]\]|!\[[^\]]*\]\([^)]*\)`)
	reWikiAlias  = regexp.MustCompile(`\[\[[^\]|]*\|([^\]]*)\]\]`)
	reWiki       = regexp.MustCompile(`\[\[([^\]#|]*)(#[^\]]*)?\]\]`)
	reLink       = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	reFootRef    = regexp.MustCompile(`\[\^[^\]]+\]`)
	reTag        = regexp.MustCompile(`<[^>]+>`)
	reAttr       = regexp.MustCompile(`\{[.#][^}]*\}`)
	reCode       = regexp.MustCompile("`([^`]*)`")
	reMarks      = regexp.MustCompile(`\*\*|__|~~|==|\*|(^|\s)_|_(\s|$)`)
	reCallout    = regexp.MustCompile(`^\[![\w-]+\][+-]?\s*`)
	reHeading    = regexp.MustCompile(`^#{1,6}\s+`)
	reList       = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])\s+(\[[ xX]\]\s+)?`)
	reRule       = regexp.MustCompile(`^\s*([-*_]\s*){3,}$`)
	reFootDef    = regexp.MustCompile(`^\[\^[^\]]+\]:`)
	reSpaces     = regexp.MustCompile(`[ \t]+`)
)

// Speakable turns a note into the text to read aloud: the title, then the
// text without Markdown. Code, tables, math, images and comments are left
// out (they make no sense spoken); headings and list items end in a pause.
func Speakable(title, body string) string {
	body = reComment.ReplaceAllString(body, "")
	body = reHTMLCmt.ReplaceAllString(body, "")
	body = reFence.ReplaceAllString(body, "\n")
	body = reMathBlock.ReplaceAllString(body, "")

	var paras []string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			paras = append(paras, strings.Join(cur, " "))
			cur = nil
		}
	}
	if t := clean(title); t != "" {
		paras = append(paras, sentence(t))
	}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, " \t\r")
		trim := strings.TrimSpace(line)
		switch {
		case trim == "":
			flush()
			continue
		case strings.HasPrefix(trim, "|"), reRule.MatchString(trim), reFootDef.MatchString(trim):
			flush() // tables, rules and footnotes are not read
			continue
		}
		// Quotes and callouts: the text without ">"; a callout's title is
		// read as a heading.
		quote := false
		for strings.HasPrefix(trim, ">") {
			trim = strings.TrimSpace(strings.TrimPrefix(trim, ">"))
			quote = true
		}
		if quote && reCallout.MatchString(trim) {
			flush()
			if t := clean(reCallout.ReplaceAllString(trim, "")); t != "" {
				paras = append(paras, sentence(t))
			}
			continue
		}
		if reHeading.MatchString(trim) {
			flush()
			if t := clean(reHeading.ReplaceAllString(trim, "")); t != "" {
				paras = append(paras, sentence(t))
			}
			continue
		}
		if reList.MatchString(trim) {
			flush()
			if t := clean(reList.ReplaceAllString(trim, "")); t != "" {
				paras = append(paras, sentence(t))
			}
			continue
		}
		if t := clean(trim); t != "" {
			cur = append(cur, t)
		}
	}
	flush()
	return strings.Join(paras, "\n\n")
}

// clean removes the inline Markdown of one line, keeping the words.
func clean(s string) string {
	s = reEmbed.ReplaceAllString(s, "")
	s = reWikiAlias.ReplaceAllString(s, "$1")
	s = reWiki.ReplaceAllString(s, "$1")
	s = reLink.ReplaceAllString(s, "$1")
	s = reFootRef.ReplaceAllString(s, "")
	s = reMathInline.ReplaceAllString(s, "")
	s = reTag.ReplaceAllString(s, "")
	s = reAttr.ReplaceAllString(s, "")
	s = reCode.ReplaceAllString(s, "$1")
	s = reMarks.ReplaceAllString(s, "$1$2")
	return strings.TrimSpace(reSpaces.ReplaceAllString(s, " "))
}

// sentence makes a heading or list item end in a pause.
func sentence(s string) string {
	r, _ := utf8.DecodeLastRuneInString(s)
	if unicode.IsPunct(r) {
		return s
	}
	return s + "."
}

// Split cuts text into parts of at most max characters, at paragraph
// breaks when it can, else at the end of a sentence, else between words.
func Split(text string, max int) []string {
	var parts []string
	var cur strings.Builder
	add := func(piece, sep string) {
		if cur.Len() > 0 && cur.Len()+len(sep)+len(piece) > max {
			parts = append(parts, cur.String())
			cur.Reset()
		}
		if cur.Len() > 0 {
			cur.WriteString(sep)
		}
		cur.WriteString(piece)
	}
	for _, para := range strings.Split(text, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if len(para) <= max {
			add(para, "\n\n")
			continue
		}
		for _, s := range sentences(para) {
			if len(s) <= max {
				add(s, " ")
				continue
			}
			for _, w := range strings.Fields(s) {
				add(w, " ")
			}
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

// sentences splits a paragraph after ".", "!" or "?" followed by a space.
func sentences(p string) []string {
	var out []string
	start := 0
	for i := 0; i < len(p)-1; i++ {
		if (p[i] == '.' || p[i] == '!' || p[i] == '?') && p[i+1] == ' ' {
			out = append(out, p[start:i+1])
			start = i + 2
		}
	}
	if start < len(p) {
		out = append(out, p[start:])
	}
	return out
}
