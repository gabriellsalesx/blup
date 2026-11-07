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

	"github.com/gabriellsalesx/blup/internal/blupcloud"
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

// blupChoice is a Blup Cloud voice's model and voice ("blup:" taken off):
// "<model>|<voice>", or a recommended model's voice alone (older choices).
func blupChoice(rest string) (model, voice string) {
	if m, v, ok := strings.Cut(rest, "|"); ok {
		return m, v
	}
	return narrate.DefaultModel, rest
}

// ModelOption is an OpenRouter voice model for a site's language.
type ModelOption struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	PerPost     float64 `json:"perPost"` // US$ for a post of about 7,000 characters (0: priced otherwise)
	Recommended bool    `json:"recommended"`
	Voices      int     `json:"voices"`
}

// VoiceModels are the voice models offered for a language, in the order
// the staff put them (Recommended only on the one the owner marked): Own
// with the person's OpenRouter key, Plan through Blup Cloud on the plan.
type VoiceModels struct {
	Own  []ModelOption `json:"own"`
	Plan []ModelOption `json:"plan"`
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

// NarrationModels lists the voice models the staff offers (Blup Cloud's
// /staff/ai) that can read a site in lang: with one's own key and through
// the plan (free ones aren't offered through the plan).
func NarrationModels(ctx context.Context, lang string) (VoiceModels, error) {
	list, err := speechModels(ctx)
	ai := studioAI(ctx)
	pick := func(l blupcloud.VoiceList, plan bool) []ModelOption {
		out := []ModelOption{}
		for _, id := range l.Models {
			i := slices.IndexFunc(list, func(m narrate.ModelInfo) bool { return m.ID == id })
			if i < 0 || (plan && list[i].PerThousand <= 0) {
				continue
			}
			m := list[i]
			if n := len(narrate.VoicesFor(m, lang)); n > 0 {
				out = append(out, ModelOption{ID: m.ID, Name: m.Name, PerPost: m.PerThousand * 7, Recommended: m.ID == l.Recommended, Voices: n})
			}
		}
		return out
	}
	return VoiceModels{Own: pick(ai.Voice.Own, false), Plan: pick(ai.Voice.Plan, true)}, err
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

// NarrationVoices lists the voices for a site in lang ("pt", "en" or "es"): the
// local ones, then those of the OpenRouter model (the recommended one when
// model is "").
func NarrationVoices(ctx context.Context, lang, model, blupModel string) ([]Voice, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	installed := LocalVoiceState().Installed
	var out []Voice
	for _, v := range kokoro.Voices {
		if localSpeaks(v, lang) {
			out = append(out, Voice{ID: localPrefix + v.ID, Name: v.Name, Engine: "local", Female: v.Female, Ready: installed, Sample: true})
		}
	}
	// Only the models the staff offers; another (or none) is the list's first.
	ai := studioAI(ctx)
	model, blupModel = pickModel(model, ai.Voice.Own), pickModel(blupModel, ai.Voice.Plan)
	list, err := speechModels(ctx)
	// Through Blup Cloud, on plans with AI: the voices of the model chosen
	// there (any of OpenRouter's; a dearer one uses the allowance faster).
	included := blupAIIncluded(ctx)
	for _, m := range list {
		if m.ID != blupModel || m.PerThousand <= 0 { // free ones aren't offered through the plan
			continue
		}
		for _, v := range narrate.VoicesFor(m, lang) {
			name := narrate.VoiceName(v)
			if v == "" {
				name = byLangCode(lang, "Voz padrão do modelo", "The model's default voice", "Voz predeterminada del modelo")
			}
			out = append(out, Voice{ID: blupPrefix + joinChoice(m.ID, v), Name: name, Engine: "blup", Ready: included, Sample: m.ID == narrate.DefaultModel && narrate.HasSample(v)})
		}
	}
	for _, m := range list {
		if m.ID != model {
			continue
		}
		for _, v := range narrate.VoicesFor(m, lang) {
			name := narrate.VoiceName(v)
			if v == "" {
				name = byLangCode(lang, "Voz padrão do modelo", "The model's default voice", "Voz predeterminada del modelo")
			}
			out = append(out, Voice{ID: joinChoice(m.ID, v), Name: name, Engine: "openrouter", Ready: creds.OpenRouterToken != "", Sample: m.ID == narrate.DefaultModel && narrate.HasSample(v)})
		}
	}
	return out, err
}

// localSpeaks tells whether a local voice reads a site in lang: its own
// language, English for the languages without local voices.
func localSpeaks(v kokoro.Voice, lang string) bool {
	switch lang {
	case "pt", "en", "es":
		return v.Lang == lang
	}
	return v.Lang == "en"
}

func byLangCode(lang, pt, en, es string) string {
	switch lang {
	case "pt":
		return pt
	case "es":
		return es
	}
	return en
}

// voiceFor is the voice that narrates a site in lang: the chosen one (a
// local voice that speaks lang, or an OpenRouter one, checked when it
// narrates), else the same engine's voice for lang. Without a choice it is
// the local voice: the Studio never sends a note to an AI in the cloud the
// person did not pick.
func voiceFor(lang, chosen string) string {
	if rest, ok := strings.CutPrefix(chosen, blupPrefix); ok {
		if model, v := blupChoice(rest); model != narrate.DefaultModel {
			return blupPrefix + joinChoice(model, v)
		} else {
			return blupPrefix + joinChoice(model, narrate.VoiceFor(lang, v))
		}
	} else if model, voice := splitChoice(chosen); model != "" && !strings.HasPrefix(chosen, localPrefix) {
		if model == narrate.DefaultModel {
			voice = narrate.VoiceFor(lang, voice)
		}
		return joinChoice(model, voice)
	}
	id := strings.TrimPrefix(chosen, localPrefix)
	for _, v := range kokoro.Voices {
		if v.ID == id && localSpeaks(v, lang) {
			return localPrefix + id
		}
	}
	for _, v := range kokoro.Voices {
		if localSpeaks(v, lang) {
			return localPrefix + v.ID
		}
	}
	return ""
}

// cloudOf is the AI in the cloud a voice sends the text to ("" for the
// local voice).
func cloudOf(voice string) string {
	switch {
	case voice == "" || strings.HasPrefix(voice, localPrefix):
		return ""
	case strings.HasPrefix(voice, blupPrefix):
		return AIBlup
	}
	return AIOpenRouter
}

// errNoLocalVoice: no voice was chosen and the local one is not installed.
func errNoLocalVoice(lang string) error {
	return errors.New(byLangCode(lang,
		"Escolha uma voz nas configurações de narração, ou baixe a voz no computador (grátis).",
		"Pick a voice in the narration settings, or download the voice on the computer (free).",
		"Elige una voz en la configuración de narración, o descarga la voz en la computadora (gratis)."))
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
	return "", "", "", errors.New(byLangCode(lang, "O OpenRouter não tem nenhuma voz para o idioma do site agora. Use a voz local.", "OpenRouter has no voice for the site's language right now. Use the local voice.", "OpenRouter no tiene ninguna voz para el idioma del sitio en este momento. Usa la voz local."))
}

func voiceName(v, lang string) string {
	if v == "" {
		return byLangCode(lang, "voz padrão", "default voice", "voz predeterminada")
	}
	return narrate.VoiceName(v)
}

// PreviewVoice is a short sample of an OpenRouter voice without a recorded
// one: made once (it costs a fraction of a cent) and kept in the cache.
func PreviewVoice(ctx context.Context, lang, choice string) ([]byte, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	// A Blup Cloud voice is heard through the plan; the others through the
	// person's OpenRouter key.
	key, base := creds.OpenRouterToken, openRouterURL
	model, voice := splitChoice(choice)
	if rest, ok := strings.CutPrefix(choice, blupPrefix); ok {
		model, voice = blupChoice(rest)
		key, base = creds.BlupToken, blupcloud.AIBaseURL()
	}
	if key == "" {
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
		"Hi! This is the voice that will read your posts. Anyone who would rather listen just presses play and hears it all, like a podcast.",
		"¡Hola! Esta es la voz que va a leer tus posts. Quien prefiera escuchar solo presiona play y lo oye todo, como un podcast.")
	b, err := (&narrate.Client{Key: key, BaseURL: base}).Speak(ctx, model, text, voice)
	if err != nil {
		return nil, err
	}
	// Only a cache: without it the sample is made again next time.
	if os.MkdirAll(filepath.Dir(path), 0o755) == nil {
		_ = os.WriteFile(path, b, 0o644)
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

// Narrate reads the note at rel aloud with voice, in the note's language
// (the site's, or its translation's: NoteOrigin),
// and saves the MP3 in the attachments (over the note's previous
// narration, so its link stays). progress gets the parts done of total.
// Posts with a password are not narrated: the audio would give them away.
func Narrate(ctx context.Context, site *config.Site, rel, voice string, progress func(done, total int)) (Narration, error) {
	text, note, err := narrationText(site, rel)
	if err != nil {
		return Narration{}, err
	}
	if noteLocked(note) {
		return Narration{}, errors.New(byLang(site, "Posts com senha não ganham narração: o áudio entregaria o texto.", "Password protected posts are not narrated: the audio would give the text away.", "Los posts con contraseña no tienen narración: el audio revelaría el texto."))
	}
	if strings.Count(text, " ") < 3 {
		return Narration{}, errors.New(byLang(site, "A nota ainda não tem texto para narrar.", "The note has no text to narrate yet.", "La nota todavía no tiene texto para narrar."))
	}
	_, lang := NoteOrigin(site, rel)
	voice = voiceFor(lang, voice)
	switch cloud := cloudOf(voice); {
	case cloud == "" && !LocalVoiceState().Installed:
		return Narration{}, errNoLocalVoice(site.Lang)
	case cloud == AIOpenRouter:
		// The key first: without it there is nothing to accept yet.
		if creds, _ := config.LoadCredentials(); creds.OpenRouterToken == "" {
			return Narration{}, ErrNoAIKey
		}
		fallthrough
	case cloud != "":
		if err := needConsent(cloud); err != nil {
			return Narration{}, err
		}
	}
	var mp3 []byte
	var seconds float64
	var used string
	if id, ok := strings.CutPrefix(voice, localPrefix); ok {
		mp3, seconds, err = speakLocal(ctx, text, id, progress)
	} else if rest, ok := strings.CutPrefix(voice, blupPrefix); ok {
		model, v := blupChoice(rest)
		mp3, seconds, err = speakBlup(ctx, site, text, model, v, progress)
	} else {
		mp3, seconds, used, err = speakOpenRouter(ctx, lang, text, voice, progress)
	}
	if err != nil {
		return Narration{}, err
	}
	if len(mp3) > maxAudio {
		return Narration{}, errors.New(byLang(site, "A narração passou de 25 MB, o máximo que o site publica. Divida o post em partes.", "The narration is over 25 MB, the most the site publishes. Split the post into parts.", "La narración pasa de 25 MB, el máximo que publica el sitio. Divide el post en partes."))
	}
	file := narrationFile(site, rel, lang, note)
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
func speakOpenRouter(ctx context.Context, lang, text, choice string, progress func(done, total int)) ([]byte, float64, string, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, 0, "", err
	}
	if creds.OpenRouterToken == "" {
		return nil, 0, "", ErrNoAIKey
	}
	model, voice, used, err := resolveOpenRouter(ctx, lang, choice)
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
			return nil, 0, "", errors.New(byLangCode(lang, "O modelo de voz saiu do OpenRouter agora há pouco. Escolha outro nas configurações de IA.", "The voice model just left OpenRouter. Pick another in the AI settings.", "El modelo de voz acaba de salir de OpenRouter. Elige otro en la configuración de IA."))
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
func narrationFile(site *config.Site, rel, lang string, note obsidian.Note) string {
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
	// A translation's narration: the same name, with its language.
	if lang != site.Lang {
		name += " (" + lang + ")"
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
func byLang(site *config.Site, pt, en, es string) string {
	return byLangCode(site.Lang, pt, en, es)
}
