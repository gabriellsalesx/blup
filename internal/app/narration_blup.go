package app

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/narrate"
)

// Narration through Blup Cloud: the plan's AI allowance pays (no
// OpenRouter key of one's own). The allowance is monthly, and no week can
// use more than half of it; the Studio shows what a text will use first.

// blupAIIncluded tells whether the account signed in here has AI through
// Blup Cloud.
func blupAIIncluded(ctx context.Context) bool {
	c, _, err := blupClient()
	if err != nil {
		return false
	}
	me, err := c.Me(ctx)
	return err == nil && me.Plan.Limits.AIUsage > 0
}

// AIQuota is how much of the month's AI allowance was used (Percent, 0 to
// 100; Left is what can still be used now, the week's cap counted) and
// when it starts again.
type AIQuota struct {
	Percent  int    `json:"percent"`
	Left     int    `json:"left"`
	ResetsAt string `json:"resetsAt"` // RFC 3339
}

// BlupAIUsage reads the AI allowance of the account signed in here.
func BlupAIUsage(ctx context.Context) (AIQuota, error) {
	c, _, err := blupClient()
	if err != nil {
		return AIQuota{}, errors.New(describe(err))
	}
	u, err := c.AIUsage(ctx)
	if err != nil {
		return AIQuota{}, errors.New(describe(err))
	}
	q := AIQuota{Percent: u.Percent, ResetsAt: u.ResetsAt.Format(time.RFC3339)}
	if u.MonthMicros > 0 {
		q.Left = int(u.Left() * 100 / u.MonthMicros)
	}
	return q, nil
}

// NarrationCost is what narrating a note through Blup Cloud would use.
type NarrationCost struct {
	Percent int  `json:"percent"` // of the month's allowance
	Left    int  `json:"left"`    // percent of the month still usable now
	Enough  bool `json:"enough"`
}

func costOf(u blupcloud.AIUsage, perChar float64, text string) (micros int64, c NarrationCost) {
	micros = int64(math.Ceil(float64(utf8.RuneCountInString(text)) * perChar))
	if u.MonthMicros > 0 {
		c.Percent = int((micros*100 + u.MonthMicros - 1) / u.MonthMicros)
		c.Left = int(u.Left() * 100 / u.MonthMicros)
	}
	c.Enough = micros <= u.Left()
	return micros, c
}

// perCharMicros is what a character costs with model through Blup Cloud
// (millionths of a dollar): the recommended model's as the cloud says,
// another's from OpenRouter's price list (the cloud charges the same).
func perCharMicros(ctx context.Context, u blupcloud.AIUsage, model string) float64 {
	if model == narrate.DefaultModel || model == "" {
		return float64(u.PerCharMicros)
	}
	list, _ := speechModels(ctx)
	for _, m := range list {
		if m.ID == model && m.PerThousand > 0 {
			return m.PerThousand * 1000
		}
	}
	return float64(u.PerCharMicros)
}

// EstimateNarration tells what narrating the note at rel through Blup
// Cloud, with voice (a "blup:" choice), would use of the month's allowance.
func EstimateNarration(ctx context.Context, site *config.Site, rel, voice string) (NarrationCost, error) {
	text, _, err := narrationText(site, rel)
	if err != nil {
		return NarrationCost{}, err
	}
	c, _, err := blupClient()
	if err != nil {
		return NarrationCost{}, errors.New(describe(err))
	}
	u, err := c.AIUsage(ctx)
	if err != nil {
		return NarrationCost{}, errors.New(describe(err))
	}
	_, lang := NoteOrigin(site, rel)
	model, _ := blupChoice(strings.TrimPrefix(voiceFor(lang, voice), blupPrefix))
	_, cost := costOf(u, perCharMicros(ctx, u, model), text)
	return cost, nil
}

// speakBlup narrates through Blup Cloud, part by part, after checking the
// whole text fits what is left of the allowance.
func speakBlup(ctx context.Context, site *config.Site, text, model, voice string, progress func(done, total int)) ([]byte, float64, error) {
	c, creds, err := blupClient()
	if err != nil {
		return nil, 0, errors.New(describe(err))
	}
	u, err := c.AIUsage(ctx)
	if err != nil {
		return nil, 0, errors.New(describe(err))
	}
	if _, cost := costOf(u, perCharMicros(ctx, u, model), text); !cost.Enough {
		return nil, 0, errors.New(byLang(site,
			fmt.Sprintf("Este texto usaria %d%% do seu uso de IA do mês, e agora sobram %d%%. Use a voz no computador (grátis) ou um plano maior.", cost.Percent, cost.Left),
			fmt.Sprintf("This text would use %d%% of your monthly AI allowance, and %d%% is left now. Use the voice on the computer (free) or a bigger plan.", cost.Percent, cost.Left),
			fmt.Sprintf("Este texto usaría %d%% de tu cuota mensual de IA, y ahora te queda %d%%. Usa la voz de la computadora (gratis) o un plan mayor.", cost.Percent, cost.Left)))
	}
	parts := narrate.Split(text, narrate.MaxPart)
	client := &narrate.Client{Key: creds.BlupToken, BaseURL: blupcloud.AIBaseURL()}
	audio := make([][]byte, 0, len(parts))
	for i, part := range parts {
		if progress != nil {
			progress(i, len(parts))
		}
		a, err := client.Speak(ctx, model, part, voice)
		switch {
		case errors.Is(err, narrate.ErrNoCredits):
			return nil, 0, errors.New(byLang(site, "O uso de IA do mês acabou durante a narração. Ele renova no dia 1.", "The month's AI allowance ran out during the narration. It renews on the 1st.", "La cuota mensual de IA se acabó durante la narración. Se renueva el día 1."))
		case errors.Is(err, narrate.ErrInvalidKey):
			return nil, 0, errors.New(describe(blupcloud.ErrSignedOut))
		case err != nil:
			return nil, 0, err
		}
		audio = append(audio, a)
	}
	if progress != nil {
		progress(len(parts), len(parts))
	}
	return narrate.JoinMP3(audio)
}

// NarratesThroughBlup tells whether the voice chosen for lang is read
// through Blup Cloud.
func NarratesThroughBlup(lang, chosen string) bool {
	return strings.HasPrefix(voiceFor(lang, chosen), blupPrefix)
}
