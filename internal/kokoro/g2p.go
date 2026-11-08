package kokoro

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// Text to phonemes, as Kokoro was trained: eSpeak NG's IPA (with "^" tying
// the letters of one phoneme and the stress marks), punctuation kept, then
// rewritten into Kokoro's alphabet. This follows Kokoro's own pipeline,
// misaki (github.com/hexgrad/misaki, Apache-2.0): EspeakG2P for Portuguese
// and EspeakFallback for English.

// marks are the punctuation kept between the phonemes (phonemizer's
// preserve_punctuation): Kokoro reads them as pauses and intonation.
const marks = ";:,.!?¡¿—…\"“”«»()"

// e2mPT is misaki's EspeakG2P table (Kokoro v1.0), applied in key order.
var e2mPT = sortedPairs(map[string]string{
	"a^ɪ": "I", "a^ʊ": "W",
	"d^z": "ʣ", "d^ʒ": "ʤ",
	"e^ɪ": "A",
	"o^ʊ": "O", "ə^ʊ": "Q",
	"s^s": "S",
	"t^s": "ʦ", "t^ʃ": "ʧ",
	"ɔ^ɪ": "Y",
}, false)

// e2mEN is misaki's EspeakFallback table, longest first.
var e2mEN = sortedPairs(map[string]string{
	"ʔˌn̩": "ʔn", "ʔn̩": "ʔn",
	"a^ɪ": "I", "a^ʊ": "W",
	"d^ʒ": "ʤ",
	"e^ɪ": "A", "e": "A",
	"t^ʃ": "ʧ",
	"ɔ^ɪ": "Y",
	"ə^l": "ᵊl",
	"ʲo":  "jo", "ʲə": "jə", "ʲ": "",
	"ɚ": "əɹ",
	"r": "ɹ",
	"x": "k", "ç": "k",
	"ɐ": "ə",
	"ɬ": "l",
	"̃": "",
}, true)

type pair struct{ old, new string }

func sortedPairs(m map[string]string, byLength bool) []pair {
	out := make([]pair, 0, len(m))
	for k, v := range m {
		out = append(out, pair{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if byLength && len([]rune(out[i].old)) != len([]rune(out[j].old)) {
			return len([]rune(out[i].old)) > len([]rune(out[j].old))
		}
		return out[i].old < out[j].old
	})
	return out
}

var (
	reFlags    = regexp.MustCompile(`\([a-z-]+\)`) // eSpeak's language switches, "(en)"
	reSyllabic = regexp.MustCompile(`(\S)\x{0329}`)
	reSpaces   = regexp.MustCompile(`\s+`)
)

// espeakVoice is eSpeak NG's voice for a Kokoro voice ("pf_dora" -> pt-br).
func espeakVoice(voice string) string {
	switch {
	case strings.HasPrefix(voice, "p"):
		return "pt-br"
	case strings.HasPrefix(voice, "b"):
		return "en-gb"
	case strings.HasPrefix(voice, "e"):
		return "es"
	}
	return "en-us"
}

// Phonemize turns text into Kokoro's phonemes for voice, running eSpeak NG
// (the program espeak, with its data in dataDir).
func Phonemize(ctx context.Context, espeak, dataDir, voice, text string) (string, error) {
	lang := espeakVoice(voice)
	type piece struct {
		text string // words to phonemize ("" for a mark)
		mark string // punctuation, kept as it is
		out  string
	}
	var pieces []*piece
	var cur strings.Builder
	flush := func() {
		if strings.TrimSpace(cur.String()) != "" {
			pieces = append(pieces, &piece{text: strings.TrimSpace(cur.String())})
		}
		cur.Reset()
	}
	for _, r := range text {
		if strings.ContainsRune(marks, r) {
			flush()
			pieces = append(pieces, &piece{mark: string(r)})
			continue
		}
		cur.WriteRune(r)
	}
	flush()

	// eSpeak NG reads one piece per run; several run at once.
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	sem := make(chan struct{}, runtime.NumCPU())
	for _, p := range pieces {
		if p.text == "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(p *piece) {
			defer func() { <-sem; wg.Done() }()
			out, err := espeakIPA(ctx, espeak, dataDir, lang, p.text)
			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			p.out = out
		}(p)
	}
	wg.Wait()
	if firstErr != nil {
		return "", firstErr
	}

	var b strings.Builder
	for _, p := range pieces {
		if p.mark != "" {
			// Marks stick to the word before and keep a space after.
			s := strings.TrimRight(b.String(), " ")
			b.Reset()
			b.WriteString(s + p.mark + " ")
			continue
		}
		b.WriteString(p.out + " ")
	}
	ps := strings.TrimSpace(reSpaces.ReplaceAllString(b.String(), " "))
	if lang == "pt-br" {
		ps = toKokoroPT(ps)
	} else {
		ps = toKokoroEN(ps, lang == "en-gb")
	}
	return keepKnown(ps), nil
}

// espeakIPA runs eSpeak NG on one piece of text (no punctuation).
func espeakIPA(ctx context.Context, espeak, dataDir, lang, text string) (string, error) {
	// The data folder goes as "." with the program running in it: on
	// Windows eSpeak NG reads its arguments in the ANSI code page, which
	// would garble a folder with accents in the user's name.
	if abs, err := filepath.Abs(espeak); err == nil {
		espeak = abs // a relative path would be taken from Dir
	}
	cmd := exec.CommandContext(ctx, espeak, "--path=.", "-q", "-b", "1", "--ipa", "--tie=^", "-v", lang)
	cmd.Dir = dataDir
	cmd.Stdin = strings.NewReader(text)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	hideWindow(cmd)
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(errb.String()); msg != "" {
			return "", &toolError{"espeak-ng", msg}
		}
		return "", err
	}
	s := reFlags.ReplaceAllString(out.String(), "")
	return strings.TrimSpace(reSpaces.ReplaceAllString(s, " ")), nil
}

type toolError struct{ tool, msg string }

func (e *toolError) Error() string { return e.tool + ": " + e.msg }

// toKokoroPT is misaki's EspeakG2P rewrite (Kokoro v1.0).
func toKokoroPT(ps string) string {
	for _, p := range e2mPT {
		ps = strings.ReplaceAll(ps, p.old, p.new)
	}
	ps = strings.ReplaceAll(ps, "^", "")
	return strings.ReplaceAll(ps, "-", "")
}

// toKokoroEN is misaki's EspeakFallback rewrite (Kokoro v1.0).
func toKokoroEN(ps string, british bool) string {
	for _, p := range e2mEN {
		ps = strings.ReplaceAll(ps, p.old, p.new)
	}
	ps = reSyllabic.ReplaceAllString(ps, "ᵊ$1")
	ps = strings.ReplaceAll(ps, "̩", "")
	if british {
		ps = strings.ReplaceAll(ps, "e^ə", "ɛː")
		ps = strings.ReplaceAll(ps, "iə", "ɪə")
		ps = strings.ReplaceAll(ps, "ə^ʊ", "Q")
	} else {
		ps = strings.ReplaceAll(ps, "o^ʊ", "O")
		ps = strings.ReplaceAll(ps, "ɜːɹ", "ɜɹ")
		ps = strings.ReplaceAll(ps, "ɜː", "ɜɹ")
		ps = strings.ReplaceAll(ps, "ɪə", "iə")
		ps = strings.ReplaceAll(ps, "ː", "")
	}
	ps = strings.ReplaceAll(ps, "o", "ɔ")
	ps = strings.ReplaceAll(ps, "ɾ", "T")
	ps = strings.ReplaceAll(ps, "ʔ", "t")
	return strings.ReplaceAll(ps, "^", "")
}

// keepKnown drops what Kokoro has no token for.
func keepKnown(ps string) string {
	return strings.Map(func(r rune) rune {
		if _, ok := vocab[r]; ok || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, ps)
}
