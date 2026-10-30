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
	"github.com/gabriellsalesx/blup/internal/kokoro"
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

// localPrefix marks the voices of the local engine ("kokoro:pf_dora"); the
// others are OpenRouter's.
const localPrefix = "kokoro:"

// LocalVoiceDir is where the local voice is installed (with Hugo's
// download, in the user's cache folder).
func LocalVoiceDir() (string, error) {
	d, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "blup", "kokoro"), nil
}

// LocalVoice tells whether this computer can have the local voice, whether
// it is installed and how much it downloads.
type LocalVoice struct {
	Supported bool  `json:"supported"`
	Installed bool  `json:"installed"`
	Bytes     int64 `json:"bytes"`
}

// LocalVoiceState checks the local voice.
func LocalVoiceState() LocalVoice {
	dir, err := LocalVoiceDir()
	return LocalVoice{Supported: kokoro.Supported(), Installed: err == nil && kokoro.Installed(dir), Bytes: kokoro.DownloadSize()}
}

// InstallLocalVoice downloads and sets up the local voice.
func InstallLocalVoice(ctx context.Context, progress kokoro.Progress) error {
	dir, err := LocalVoiceDir()
	if err != nil {
		return err
	}
	return kokoro.Install(ctx, dir, progress)
}

// RemoveLocalVoice deletes the local voice.
func RemoveLocalVoice() error {
	dir, err := LocalVoiceDir()
	if err != nil {
		return err
	}
	return kokoro.Remove(dir)
}

// Voice is a narration voice: Engine "local" (on the computer) or
// "openrouter"; Ready when it can narrate now (installed, or a key saved).
type Voice struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Engine string `json:"engine"`
	Female bool   `json:"female"`
	Ready  bool   `json:"ready"`
}

// NarrationVoices lists the voices for a site in lang ("pt" or "en"): the
// local ones, then OpenRouter's (the default of each first).
func NarrationVoices(ctx context.Context, lang string) ([]Voice, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	installed := LocalVoiceState().Installed
	var out []Voice
	for _, v := range kokoro.Voices {
		if v.Lang == lang || (lang != "pt" && v.Lang == "en") {
			out = append(out, Voice{ID: localPrefix + v.ID, Name: v.Name, Engine: "local", Female: v.Female, Ready: installed})
		}
	}
	ids, err := (&narrate.Client{Key: creds.OpenRouterToken, BaseURL: openRouterURL}).Voices(ctx, lang)
	for _, id := range ids {
		out = append(out, Voice{ID: id, Name: narrate.VoiceName(id), Engine: "openrouter", Ready: creds.OpenRouterToken != ""})
	}
	return out, err
}

// voiceFor is the voice that narrates a site in lang: the chosen one when
// it speaks lang, else the local default when installed, else OpenRouter's.
func voiceFor(lang, chosen string) string {
	if id, ok := strings.CutPrefix(chosen, localPrefix); ok {
		for _, v := range kokoro.Voices {
			if v.ID == id && (v.Lang == lang || (lang != "pt" && v.Lang == "en")) {
				return chosen
			}
		}
	} else if chosen != "" && narrate.VoiceFor(lang, chosen) == chosen {
		return chosen
	}
	if LocalVoiceState().Installed {
		for _, v := range kokoro.Voices {
			if v.Lang == lang || (lang != "pt" && v.Lang == "en") {
				return localPrefix + v.ID
			}
		}
	}
	return narrate.DefaultVoice(lang)
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
	return narrate.Speakable(title, n.String("description"), n.Body), n, nil
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
	voice = voiceFor(site.Lang, voice)
	var mp3 []byte
	var seconds float64
	if id, ok := strings.CutPrefix(voice, localPrefix); ok {
		mp3, seconds, err = speakLocal(ctx, text, id, progress)
	} else {
		mp3, seconds, err = speakOpenRouter(ctx, site, text, voice, progress)
	}
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

// speakLocal narrates on the computer (Kokoro).
func speakLocal(ctx context.Context, text, voice string, progress func(done, total int)) ([]byte, float64, error) {
	dir, err := LocalVoiceDir()
	if err != nil {
		return nil, 0, err
	}
	return kokoro.Engine{Dir: dir}.Speak(ctx, text, voice, progress)
}

// speakOpenRouter narrates with OpenRouter, part by part, and joins the
// MP3s.
func speakOpenRouter(ctx context.Context, site *config.Site, text, voice string, progress func(done, total int)) ([]byte, float64, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, 0, err
	}
	if creds.OpenRouterToken == "" {
		return nil, 0, ErrNoAIKey
	}
	parts := narrate.Split(text, narrate.MaxPart)
	client := &narrate.Client{Key: creds.OpenRouterToken, BaseURL: openRouterURL}
	audio := make([][]byte, 0, len(parts))
	for i, part := range parts {
		if progress != nil {
			progress(i, len(parts))
		}
		a, err := client.Speak(ctx, part, voice)
		if errors.Is(err, narrate.ErrModelGone) {
			return nil, 0, errors.New(byLang(site, "O modelo de voz saiu do OpenRouter. Atualize o Blup para usar o novo.", "The voice model left OpenRouter. Update Blup to use the new one."))
		}
		if err != nil {
			return nil, 0, err
		}
		audio = append(audio, a)
	}
	if progress != nil {
		progress(len(parts), len(parts))
	}
	return narrate.JoinMP3(audio)
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
