package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

// localPrefix marks the voices of the local engine ("kokoro:pf_dora"),
// blupPrefix those read through Blup Cloud on the plan's allowance
// ("blup:pt-BR-Luana:MAI-Voice-2.1"); the others are OpenRouter's, with
// the person's own key.
const (
	localPrefix = "kokoro:"
	blupPrefix  = "blup:"
)

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

// An OpenRouter voice is saved as "<model>|<voice>" (the voice may be ""
// for a model's default); older choices are a MAI-Voice name alone.
func splitChoice(c string) (model, voice string) {
	if m, v, ok := strings.Cut(c, "|"); ok {
		return m, v
	}
	if c != "" {
		return narrate.DefaultModel, c
	}
	return "", ""
}

func joinChoice(model, voice string) string { return model + "|" + voice }

// ModelOption is an OpenRouter voice model for a site's language.
type ModelOption struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	PerPost     float64 `json:"perPost"` // US$ for a post of about 7,000 characters (0: priced otherwise)
	Recommended bool    `json:"recommended"`
	Voices      int     `json:"voices"`
}

// speechModels reads OpenRouter's voice models (the recommended one alone
// when the list can't be read).
func speechModels(ctx context.Context) ([]narrate.ModelInfo, error) {
	creds, _ := config.LoadCredentials()
	list, err := (&narrate.Client{Key: creds.OpenRouterToken, BaseURL: openRouterURL}).Models(ctx)
	if err != nil || len(list) == 0 {
		return []narrate.ModelInfo{narrate.Known()}, err
	}
	// Kokoro runs on the computer for free (the local voice): not offered
	// paid through OpenRouter. Seed's voices aren't used either.
	return slices.DeleteFunc(list, func(m narrate.ModelInfo) bool {
		id := strings.ToLower(m.ID + " " + m.Name)
		return strings.Contains(id, "kokoro") || strings.Contains(id, "seed")
	}), nil
}

// NarrationModels lists OpenRouter's voice models that can read a site in
// lang, the recommended one first.
func NarrationModels(ctx context.Context, lang string) ([]ModelOption, error) {
	list, err := speechModels(ctx)
	var out []ModelOption
	for _, m := range list {
		if n := len(narrate.VoicesFor(m, lang)); n > 0 {
			out = append(out, ModelOption{ID: m.ID, Name: m.Name, PerPost: m.PerThousand * 7, Recommended: m.ID == narrate.DefaultModel, Voices: n})
		}
	}
	return out, err
}

// Voice is a narration voice: Engine "local" (on the computer) or
// "openrouter"; Ready when it can narrate now (installed, or a key saved);
// Sample when the Studio has a recorded sample (else one is made on demand).
type Voice struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Engine string `json:"engine"`
	Female bool   `json:"female"`
	Ready  bool   `json:"ready"`
	Sample bool   `json:"sample"`
}

// NarrationVoices lists the voices for a site in lang ("pt" or "en"): the
// local ones, then those of the OpenRouter model (the recommended one when
// model is "").
func NarrationVoices(ctx context.Context, lang, model string) ([]Voice, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	installed := LocalVoiceState().Installed
	var out []Voice
	for _, v := range kokoro.Voices {
		if v.Lang == lang || (lang != "pt" && v.Lang == "en") {
			out = append(out, Voice{ID: localPrefix + v.ID, Name: v.Name, Engine: "local", Female: v.Female, Ready: installed, Sample: true})
		}
	}
	// Through Blup Cloud: the recommended model's voices, on paid plans.
	included := blupAIIncluded(ctx)
	for _, v := range narrate.VoicesFor(narrate.Known(), lang) {
		out = append(out, Voice{ID: blupPrefix + v, Name: narrate.VoiceName(v), Engine: "blup", Ready: included, Sample: true})
	}
	if model == "" {
		model = narrate.DefaultModel
	}
	list, err := speechModels(ctx)
	for _, m := range list {
		if m.ID != model {
			continue
		}
		for _, v := range narrate.VoicesFor(m, lang) {
			name := narrate.VoiceName(v)
			if v == "" {
				name = byLangCode(lang, "Voz padrão do modelo", "The model's default voice")
			}
			out = append(out, Voice{ID: joinChoice(m.ID, v), Name: name, Engine: "openrouter", Ready: creds.OpenRouterToken != "", Sample: m.ID == narrate.DefaultModel && narrate.HasSample(v)})
		}
	}
	return out, err
}

func byLangCode(lang, pt, en string) string {
	if lang == "pt" {
		return pt
	}
	return en
}

// voiceFor is the voice that narrates a site in lang: the chosen one (a
// local voice that speaks lang, or an OpenRouter one, checked when it
// narrates), else the local default when installed, else OpenRouter's
// recommended one.
func voiceFor(lang, chosen string) string {
	if id, ok := strings.CutPrefix(chosen, localPrefix); ok {
		for _, v := range kokoro.Voices {
			if v.ID == id && (v.Lang == lang || (lang != "pt" && v.Lang == "en")) {
				return chosen
			}
		}
	} else if v, ok := strings.CutPrefix(chosen, blupPrefix); ok {
		return blupPrefix + narrate.VoiceFor(lang, v)
	} else if model, voice := splitChoice(chosen); model != "" && (model != narrate.DefaultModel || narrate.VoiceFor(lang, voice) == voice) {
		return joinChoice(model, voice)
	}
	if LocalVoiceState().Installed {
		for _, v := range kokoro.Voices {
			if v.Lang == lang || (lang != "pt" && v.Lang == "en") {
				return localPrefix + v.ID
			}
		}
	}
	return joinChoice(narrate.DefaultModel, narrate.DefaultVoice(lang))
}

// resolveOpenRouter checks a choice against OpenRouter's current list, so
// a model or voice that left doesn't stop the narration: the voice falls
// back to the model's first for lang, the model to the recommended one,
// then to any that reads lang. note tells what was used instead ("" when
// the choice stands).
func resolveOpenRouter(ctx context.Context, lang, choice string) (model, voice, note string, err error) {
	model, voice = splitChoice(choice)
	list, lerr := (&narrate.Client{BaseURL: openRouterURL}).Models(ctx)
	if lerr != nil || len(list) == 0 {
		return model, voice, "", nil // can't check: try the choice as it is
	}
	find := func(id string) (narrate.ModelInfo, []string) {
		for _, m := range list {
			if m.ID == id {
				return m, narrate.VoicesFor(m, lang)
			}
		}
		return narrate.ModelInfo{}, nil
	}
	if _, vs := find(model); len(vs) > 0 {
		if slices.Contains(vs, voice) {
			return model, voice, "", nil
		}
		return model, vs[0], voiceName(vs[0], lang), nil
	}
	if m, vs := find(narrate.DefaultModel); len(vs) > 0 {
		return m.ID, vs[0], m.Name + ", " + voiceName(vs[0], lang), nil
	}
	for _, m := range list {
		if vs := narrate.VoicesFor(m, lang); len(vs) > 0 {
			return m.ID, vs[0], m.Name + ", " + voiceName(vs[0], lang), nil
		}
	}
	return "", "", "", errors.New(byLangCode(lang, "O OpenRouter não tem nenhuma voz para o idioma do site agora. Use a voz local.", "OpenRouter has no voice for the site's language right now. Use the local voice."))
}

func voiceName(v, lang string) string {
	if v == "" {
		return byLangCode(lang, "voz padrão", "default voice")
	}
	return narrate.VoiceName(v)
}

// PreviewVoice is a short sample of an OpenRouter voice without a recorded
// one: made once (it costs a fraction of a cent) and kept in the cache.
func PreviewVoice(ctx context.Context, lang, choice string) ([]byte, error) {
	model, voice := splitChoice(choice)
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	if creds.OpenRouterToken == "" {
		return nil, ErrNoAIKey
	}
	h := sha256.Sum256([]byte(lang + "\x00" + choice))
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "blup", "voice-samples", hex.EncodeToString(h[:8])+".mp3")
	if b, err := os.ReadFile(path); err == nil {
		return b, nil
	}
	text := byLangCode(lang,
		"Olá! Esta é a voz que vai ler os seus posts. Quem prefere escutar aperta o play e ouve tudo, como um podcast.",
		"Hi! This is the voice that will read your posts. Anyone who would rather listen just presses play and hears it all, like a podcast.")
	b, err := (&narrate.Client{Key: creds.OpenRouterToken, BaseURL: openRouterURL}).Speak(ctx, model, text, voice)
	if err != nil {
		return nil, err
	}
	if os.MkdirAll(filepath.Dir(path), 0o755) == nil {
		os.WriteFile(path, b, 0o644)
	}
	return b, nil
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
	Used    string  `json:"used"` // the voice used instead of the chosen one, when it left OpenRouter
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
	var used string
	if id, ok := strings.CutPrefix(voice, localPrefix); ok {
		mp3, seconds, err = speakLocal(ctx, text, id, progress)
	} else if v, ok := strings.CutPrefix(voice, blupPrefix); ok {
		mp3, seconds, err = speakBlup(ctx, site, text, v, progress)
	} else {
		mp3, seconds, used, err = speakOpenRouter(ctx, site, text, voice, progress)
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
	return Narration{Link: "[[" + file + "]]", Hash: textHash(text), Seconds: seconds, Bytes: len(mp3), Used: used}, nil
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
// MP3s. used names the voice used instead of the choice, when it left.
func speakOpenRouter(ctx context.Context, site *config.Site, text, choice string, progress func(done, total int)) ([]byte, float64, string, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, 0, "", err
	}
	if creds.OpenRouterToken == "" {
		return nil, 0, "", ErrNoAIKey
	}
	model, voice, used, err := resolveOpenRouter(ctx, site.Lang, choice)
	if err != nil {
		return nil, 0, "", err
	}
	parts := narrate.Split(text, narrate.MaxPart)
	client := &narrate.Client{Key: creds.OpenRouterToken, BaseURL: openRouterURL}
	audio := make([][]byte, 0, len(parts))
	for i, part := range parts {
		if progress != nil {
			progress(i, len(parts))
		}
		a, err := client.Speak(ctx, model, part, voice)
		if errors.Is(err, narrate.ErrModelGone) {
			return nil, 0, "", errors.New(byLang(site, "O modelo de voz saiu do OpenRouter agora há pouco. Escolha outro nas configurações de IA.", "The voice model just left OpenRouter. Pick another in the AI settings."))
		}
		if err != nil {
			return nil, 0, "", err
		}
		audio = append(audio, a)
	}
	if progress != nil {
		progress(len(parts), len(parts))
	}
	mp3, secs, err := narrate.JoinMP3(audio)
	return mp3, secs, used, err
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
