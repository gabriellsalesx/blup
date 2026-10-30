package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/narrate"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// Narration of posts, for Blup Studio: a voice model on OpenRouter reads
// the note aloud and the MP3 goes to the attachments, linked from the
// note's "audio" property; the themes show a player on the post. The key
// is global (credentials.toml), so every site uses it.

// openRouterURL is OpenRouter's API (narrate.DefaultBaseURL when empty); tests
// point it to a fake server.
var openRouterURL = ""

// ErrNoAIKey means no OpenRouter key was saved.
var ErrNoAIKey = errors.New("no OpenRouter key")

// maxAudio is the largest narration published (Cloudflare Pages takes
// files up to 25 MiB).
const maxAudio = 25 << 20

// AIConnected tells whether an OpenRouter key is saved.
func AIConnected() (bool, error) {
	creds, err := config.LoadCredentials()
	return creds.OpenRouterToken != "", err
}

// ConnectOpenRouter checks an OpenRouter key and saves it.
func ConnectOpenRouter(ctx context.Context, key string) error {
	key = strings.TrimSpace(key)
	if err := (&narrate.Client{Key: key}).CheckKey(ctx); err != nil {
		return err
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	creds.OpenRouterToken = key
	return creds.Save()
}

// Narration is a narration just made: the property value that links it
// ("[[name.mp3]]"), the fingerprint of the text it reads (audio_hash), its
// length and size.
type Narration struct {
	Link    string  `json:"link"`
	Hash    string  `json:"hash"`
	Seconds float64 `json:"seconds"`
	Bytes   int     `json:"bytes"`
}

// narrationText is what gets read aloud from the note at rel, with the
// note's properties.
func narrationText(site *config.Site, rel string) (string, obsidian.Note, error) {
	p, err := notesPath(site, rel)
	if err != nil {
		return "", obsidian.Note{}, err
	}
	n, err := obsidian.ReadNote(p)
	if err != nil {
		return "", n, err
	}
	title := n.String("title")
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	}
	return narrate.Speakable(title, n.Body), n, nil
}

func textHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:8])
}

// NarrationHash is the fingerprint of what the note at rel reads now; when
// it differs from the note's audio_hash, the narration is out of date.
func NarrationHash(site *config.Site, rel string) (string, error) {
	text, _, err := narrationText(site, rel)
	return textHash(text), err
}

// Narrate reads the note at rel aloud with voice, in the site's language,
// and saves the MP3 in the attachments (over the note's previous
// narration, so its link stays). progress gets the parts done of total.
// Posts with a password are not narrated: the audio would give them away.
func Narrate(ctx context.Context, site *config.Site, rel, voice string, progress func(done, total int)) (Narration, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return Narration{}, err
	}
	if creds.OpenRouterToken == "" {
		return Narration{}, ErrNoAIKey
	}
	text, note, err := narrationText(site, rel)
	if err != nil {
		return Narration{}, err
	}
	if note.String("password") != "" && note.String("password") != "false" {
		return Narration{}, errors.New(byLang(site, "Posts com senha não ganham narração: o áudio entregaria o texto.", "Password protected posts are not narrated: the audio would give the text away."))
	}
	if strings.Count(text, " ") < 3 {
		return Narration{}, errors.New(byLang(site, "A nota ainda não tem texto para narrar.", "The note has no text to narrate yet."))
	}
	if !contains(narrate.Voices, voice) {
		voice = narrate.Voices[0]
	}
	parts := narrate.Split(text, narrate.MaxPart)
	client := &narrate.Client{Key: creds.OpenRouterToken, BaseURL: openRouterURL}
	instructions := byLang(site,
		"Narre em português do Brasil, com voz calma, natural e clara, como um podcast. Respeite a pontuação e faça pausas curtas entre os parágrafos.",
		"Narrate in English with a calm, natural and clear voice, like a podcast. Follow the punctuation and pause briefly between paragraphs.")
	audio := make([][]byte, 0, len(parts))
	for i, part := range parts {
		if progress != nil {
			progress(i, len(parts))
		}
		a, err := client.Speak(ctx, part, voice, instructions)
		if err != nil {
			return Narration{}, err
		}
		audio = append(audio, a)
	}
	if progress != nil {
		progress(len(parts), len(parts))
	}
	mp3, seconds, err := narrate.JoinMP3(audio)
	if err != nil {
		return Narration{}, err
	}
	if len(mp3) > maxAudio {
		return Narration{}, errors.New(byLang(site, "A narração passou de 25 MB, o máximo que o site publica. Divida o post em partes.", "The narration is over 25 MB, the most the site publishes. Split the post into parts."))
	}
	file := narrationFile(site, rel, note)
	dest := filepath.Join(site.ContentDir(), attachmentsFolder(site), file)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return Narration{}, err
	}
	if err := os.WriteFile(dest, mp3, 0o644); err != nil {
		return Narration{}, err
	}
	return Narration{Link: "[[" + file + "]]", Hash: textHash(text), Seconds: seconds, Bytes: len(mp3)}, nil
}

// narrationFile is the attachment's name: the one the note already links
// to (an MP3 in the attachments), else the note's name.
func narrationFile(site *config.Site, rel string, note obsidian.Note) string {
	cur := strings.TrimSuffix(strings.TrimPrefix(note.String("audio"), "[["), "]]")
	if strings.EqualFold(filepath.Ext(cur), ".mp3") && filepath.Base(cur) == cur {
		if _, err := os.Stat(filepath.Join(site.ContentDir(), attachmentsFolder(site), cur)); err == nil {
			return cur
		}
	}
	name := safeFileName(strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel)))
	if name == "" {
		name = "narration"
	}
	// A taken name (another note's narration, a file of the user) gets a number.
	file := name + ".mp3"
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(site.ContentDir(), attachmentsFolder(site), file)); errors.Is(err, os.ErrNotExist) {
			return file
		}
		file = fmt.Sprintf("%s %d.mp3", name, i)
	}
}

// byLang picks the text in the site's language.
func byLang(site *config.Site, pt, en string) string {
	if site.Lang == "pt" {
		return pt
	}
	return en
}
