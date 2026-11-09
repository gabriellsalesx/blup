package app

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/kokoro"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// Narrating every post of a blog at once (the Podcast screen), with the
// same rules as one note's narration (Narrate): what the site publishes as
// a post, never a password protected one (nor its translations: the audio
// would give the text away), only text that changed since its narration.
// Translations are read in their own language, with the voice chosen for
// it.

// NarrationItem is a note the batch would narrate.
type NarrationItem struct {
	Rel  string `json:"rel"`
	Lang string `json:"lang"`
	// Outdated: it has a narration, of a text that changed since.
	Outdated bool `json:"outdated"`
	Chars    int  `json:"chars"`
}

// NarrationVoice is the voice that reads one language of the batch.
type NarrationVoice struct {
	Lang   string `json:"lang"`
	Name   string `json:"name"`
	Engine string `json:"engine"` // "local", "blup" or "openrouter"
	Notes  int    `json:"notes"`
	Chars  int    `json:"chars"`
}

// NarrationPlan is what narrating the blog's posts would do: posts of the
// site's language without a narration (Missing) or with an out of date one
// (Outdated), translations per language, password protected posts left
// out (Locked), the voice of each language and, when some go through Blup
// Cloud, what they would use of the month's AI allowance.
type NarrationPlan struct {
	Missing      int              `json:"missing"`
	Outdated     int              `json:"outdated"`
	Translations map[string]int   `json:"translations"`
	Locked       int              `json:"locked"`
	Total        int              `json:"total"`
	Chars        int              `json:"chars"`
	Voices       []NarrationVoice `json:"voices"`
	Items        []NarrationItem  `json:"items"`
	// Cost is set when some notes go through Blup Cloud; CostError says
	// why it could not be worked out (the batch then can't start on it).
	Cost      *NarrationCost `json:"cost"`
	CostError string         `json:"costError"`
}

// minNarrationSpaces is the least text a note needs to be narrated (the
// same as Narrate's).
const minNarrationSpaces = 3

// narrationState is what a note's narration is now: "" when it has none,
// "done" when up to date, "outdated" when the text changed since (or its
// file is gone).
func narrationState(site *config.Site, n obsidian.Note, text string) string {
	link := n.String("audio")
	if link == "" {
		return ""
	}
	if n.String("audio_hash") != textHash(text) || attachmentFile(site, link) == "" {
		return "outdated"
	}
	return "done"
}

// isPost tells whether the note at rel is a post the blog publishes and
// lists (so a podcast episode once narrated): not a section's or the
// home's note, not an author's profile, not in the podcast's folder, not
// one of the site's own pages (privacy and the like, marked "blup:"), and
// not hidden, a draft or kept off the site.
func isPost(site *config.Site, rel string, n obsidian.Note) bool {
	if strings.HasPrefix(path.Base(rel), "_") || isAuthor(site, rel) || n.String("blup") != "" {
		return false
	}
	sections := existingSections(site)
	top, _, _ := strings.Cut(rel, "/")
	for _, kind := range []string{"authors", "podcast"} {
		if f, ok := sections[kind]; ok && strings.EqualFold(top, f) {
			return false
		}
	}
	return !obsidian.Yes(n.Props["hidden"]) && !obsidian.Yes(n.Props["draft"]) && !obsidian.Unpublished(n.Props)
}

// candidate is what a note is to the batch.
type candidate int

const (
	notPost  candidate = iota // not a post, or nothing to narrate in it
	upToDate                  // a post whose narration is up to date (or too short)
	locked                    // a password protected post
	needed                    // a post to narrate
)

// narrationCandidate reads the note at rel and tells what it is to the
// batch, with the item when it needs a narration.
func narrationCandidate(site *config.Site, rel, lang string) (NarrationItem, candidate, obsidian.Note) {
	var item NarrationItem
	text, n, err := narrationText(site, rel)
	if err != nil || !isPost(site, rel, n) {
		return item, notPost, n
	}
	if noteLocked(n) {
		return item, locked, n
	}
	if strings.Count(text, " ") < minNarrationSpaces {
		return item, upToDate, n
	}
	switch narrationState(site, n, text) {
	case "done":
		return item, upToDate, n
	case "outdated":
		item.Outdated = true
	}
	item.Rel, item.Lang, item.Chars = rel, lang, utf8.RuneCountInString(text)
	return item, needed, n
}

// NarrationTodo is what narrating the blog's posts would do, with the
// voice chosen for each language (voices, by language code; "" for none).
// It reads the notes only: the cost is NarrationPlanCost's.
func NarrationTodo(site *config.Site, voices map[string]string) (NarrationPlan, error) {
	plan := NarrationPlan{Translations: map[string]int{}, Items: []NarrationItem{}, Voices: []NarrationVoice{}}
	originals, err := siteOriginals(site)
	if err != nil {
		return plan, err
	}
	for _, rel := range originals {
		item, kind, n := narrationCandidate(site, rel, site.Lang)
		switch kind {
		case notPost:
			continue
		case locked:
			plan.Locked++
			continue // its translations neither: the audio would give the text away
		case needed:
			plan.Items = append(plan.Items, item)
			if item.Outdated {
				plan.Outdated++
			} else {
				plan.Missing++
			}
		}
		for _, lang := range site.OtherLangs() {
			tr := translationPath(site, rel, n, lang)
			if tr == "" {
				continue
			}
			if titem, kind, _ := narrationCandidate(site, tr, lang); kind == needed {
				plan.Items = append(plan.Items, titem)
				plan.Translations[lang]++
			}
		}
	}
	byLang := map[string]*NarrationVoice{}
	order := append([]string{site.Lang}, site.OtherLangs()...)
	for _, it := range plan.Items {
		plan.Total++
		plan.Chars += it.Chars
		v := byLang[it.Lang]
		if v == nil {
			voice := voiceFor(it.Lang, voices[it.Lang])
			v = &NarrationVoice{Lang: it.Lang, Name: voiceLabel(it.Lang, voice), Engine: engineOf(voice)}
			byLang[it.Lang] = v
		}
		v.Notes++
		v.Chars += it.Chars
	}
	for _, l := range order {
		if v := byLang[l]; v != nil {
			plan.Voices = append(plan.Voices, *v)
		}
	}
	return plan, nil
}

// engineOf names who reads with voice: "local", "blup" or "openrouter".
func engineOf(voice string) string {
	switch cloudOf(voice) {
	case AIBlup:
		return "blup"
	case AIOpenRouter:
		return "openrouter"
	}
	return "local"
}

// voiceLabel is a voice's name for the screen ("" when there is no voice).
func voiceLabel(lang, voice string) string {
	if id, ok := strings.CutPrefix(voice, localPrefix); ok {
		for _, v := range kokoro.Voices {
			if v.ID == id {
				return v.Name
			}
		}
		return id
	}
	if voice == "" {
		return ""
	}
	rest := strings.TrimPrefix(voice, blupPrefix)
	_, v := splitChoice(rest)
	return voiceName(v, lang)
}

// NarrationPlanCost is what the plan's notes read through Blup Cloud would
// use of the month's AI allowance (nil when none go through it).
func NarrationPlanCost(ctx context.Context, site *config.Site, plan NarrationPlan, voices map[string]string) (*NarrationCost, error) {
	chars := map[string]int{}
	for _, it := range plan.Items {
		if NarratesThroughBlup(it.Lang, voices[it.Lang]) {
			chars[it.Lang] += it.Chars
		}
	}
	if len(chars) == 0 {
		return nil, nil
	}
	c, _, err := blupClient()
	if err != nil {
		return nil, errors.New(describe(err))
	}
	u, err := c.AIUsage(ctx)
	if err != nil {
		return nil, errors.New(describe(err))
	}
	var micros int64
	for lang, n := range chars {
		model, _ := blupChoice(strings.TrimPrefix(voiceFor(lang, voices[lang]), blupPrefix))
		per, ok := perCharMicros(ctx, u, model)
		if !ok {
			return nil, errUnknownPrice(lang)
		}
		micros += int64(math.Ceil(float64(n) * per))
	}
	cost := costOfMicros(u, micros)
	return &cost, nil
}

// costOfMicros is what micros of AI use are of the allowance (as costOf).
func costOfMicros(u blupcloud.AIUsage, micros int64) NarrationCost {
	var c NarrationCost
	if u.MonthMicros > 0 {
		c.Percent = int((micros*100 + u.MonthMicros - 1) / u.MonthMicros)
		c.Left = int(u.Left() * 100 / u.MonthMicros)
	}
	c.Enough = micros <= u.Left()
	return c
}

// NarrateReady says whether every language of the plan can be narrated
// now (its voice: the local one installed, an OpenRouter key, the consent
// to send the text to the AI in the cloud), before a batch starts on its
// own.
func NarrateReady(site *config.Site, plan NarrationPlan, voices map[string]string) error {
	for _, v := range plan.Voices {
		voice := voiceFor(v.Lang, voices[v.Lang])
		switch cloud := cloudOf(voice); {
		case cloud == "" && !LocalVoiceState().Installed:
			return errNoLocalVoice(site.Lang)
		case cloud == AIOpenRouter:
			if creds, _ := config.LoadCredentials(); creds.OpenRouterToken == "" {
				return ErrNoAIKey
			}
			if err := needConsent(cloud); err != nil {
				return err
			}
		case cloud == AIBlup:
			if err := needConsent(cloud); err != nil {
				return err
			}
		}
	}
	return nil
}

// SiteNarration is what narrating the blog's posts did.
type SiteNarration struct {
	Done int `json:"done"`
	// Failed are the notes that could not be narrated this time (the
	// rest went on); narrating again tries only them.
	Failed []string `json:"failed"`
}

// narrateNote narrates one note (Narrate, a seam for the tests).
var narrateNote = Narrate

// sameErrorStop is how many notes in a row failing with the same error
// stop the batch: that is something every note would hit (the allowance
// ran out, the connection fell), not a note of its own.
const sameErrorStop = 3

// NarrateSite narrates the plan's notes one by one, each with the voice
// chosen for its language, and links each narration in its note (audio,
// audio_hash, audio_seconds), as narrating it alone does. progress gets
// the notes done of total and the note being read. A note that fails alone
// is listed in Failed and the rest go on; the same failure on several
// notes in a row (or a cancel) stops the batch, with the notes done kept.
func NarrateSite(ctx context.Context, site *config.Site, plan NarrationPlan, voices map[string]string, progress func(done, total int, note string)) (SiteNarration, error) {
	res := SiteNarration{Failed: []string{}}
	total := len(plan.Items)
	lastErr, streak := "", 0
	for i, it := range plan.Items {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		if progress != nil {
			progress(i, total, it.Rel)
		}
		r, err := narrateNote(ctx, site, it.Rel, voices[it.Lang], nil)
		if err == nil {
			err = linkNarration(site, it.Rel, r)
		}
		if err != nil {
			if ctx.Err() != nil {
				return res, ctx.Err()
			}
			var consent *ConsentError
			if errors.As(err, &consent) || errors.Is(err, ErrNoAIKey) {
				return res, err
			}
			res.Failed = append(res.Failed, it.Rel)
			if err.Error() == lastErr {
				streak++
			} else {
				lastErr, streak = err.Error(), 1
			}
			if streak >= sameErrorStop {
				return res, err
			}
			continue
		}
		lastErr, streak = "", 0
		res.Done++
	}
	if progress != nil {
		progress(total, total, "")
	}
	return res, nil
}

// linkNarration writes the narration's properties in its note.
func linkNarration(site *config.Site, rel string, r Narration) error {
	p, err := notesPath(site, rel)
	if err != nil {
		return err
	}
	return obsidian.SetProperties(p,
		obsidian.Prop{Key: "audio", Value: r.Link},
		obsidian.Prop{Key: "audio_hash", Value: r.Hash},
		obsidian.Prop{Key: "audio_seconds", Value: int64(math.Round(r.Seconds))},
	)
}

// ErrBatchAllowance: the batch would use more of the month's AI allowance
// than is left.
func ErrBatchAllowance(site *config.Site, c NarrationCost) error {
	return errors.New(byLang(site,
		fmt.Sprintf("Narrar todos usaria %d%% do seu uso de IA do mês, e agora sobram %d%%. Narre alguns posts por vez, use a voz no computador (grátis) ou um plano maior.", c.Percent, c.Left),
		fmt.Sprintf("Narrating them all would use %d%% of your monthly AI allowance, and %d%% is left now. Narrate a few posts at a time, use the voice on the computer (free) or a bigger plan.", c.Percent, c.Left),
		fmt.Sprintf("Narrarlos todos usaría %d%% de tu cuota mensual de IA, y ahora te queda %d%%. Narra algunos posts a la vez, usa la voz de la computadora (gratis) o un plan mayor.", c.Percent, c.Left)))
}

// ErrBatchTranslating: the site's translation runs; narrating meanwhile
// would read translations still being written.
func ErrBatchTranslating(site *config.Site) error {
	return errors.New(byLang(site,
		"O site está sendo traduzido agora. Espere a tradução terminar para narrar os posts.",
		"The site is being translated now. Wait for the translation to finish to narrate the posts.",
		"El sitio se está traduciendo ahora. Espera a que termine la traducción para narrar los posts."))
}

// ErrBatchNarrating: the site's posts are being narrated; translating
// them meanwhile would change notes the narration reads.
func ErrBatchNarrating(site *config.Site) error {
	return errors.New(byLang(site,
		"Os posts estão sendo narrados agora. Espere a narração terminar para traduzir.",
		"The posts are being narrated now. Wait for the narration to finish to translate.",
		"Los posts se están narrando ahora. Espera a que termine la narración para traducir."))
}
