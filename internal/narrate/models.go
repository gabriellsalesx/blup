package narrate

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// OpenRouter's voice models change often (a model left the list once), so
// the user picks the model too: Models reads the list, and VoicesFor tells
// which voices of a model speak a site's language.

// ModelInfo is a voice model offered by OpenRouter.
type ModelInfo struct {
	ID           string
	Name         string
	PerThousand  float64 // US$ per 1,000 characters (0: priced per second of audio, or free)
	Voices       []string
	Multilingual bool // its voices speak any language (when they don't say one)
}

// pcmOnly are the models that answer only raw audio (OpenRouter's
// documentation): the narration needs MP3.
var pcmOnly = []string{"google/gemini", "heygen/"}

// Models lists OpenRouter's voice models that answer MP3, cheapest first
// (DefaultModel, the recommended one, first of all).
func (c *Client) Models(ctx context.Context) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+"/models?output_modalities=speech", nil)
	if err != nil {
		return nil, err
	}
	c.headers(req)
	res, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, apiError(res)
	}
	var list struct {
		Data []struct {
			ID          string   `json:"id"`
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Voices      []string `json:"supported_voices"`
			Pricing     struct {
				Prompt string `json:"prompt"`
			} `json:"pricing"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		return nil, err
	}
	var out []ModelInfo
	for _, m := range list.Data {
		skip := false
		for _, p := range pcmOnly {
			skip = skip || strings.HasPrefix(m.ID, p)
		}
		if skip {
			continue
		}
		price, _ := strconv.ParseFloat(m.Pricing.Prompt, 64)
		d := strings.ToLower(m.Description)
		out = append(out, ModelInfo{
			ID: m.ID, Name: m.Name, PerThousand: price * 1000, Voices: m.Voices,
			Multilingual: strings.Contains(d, "multilingual") || strings.Contains(d, "languages") || strings.Contains(d, "portuguese"),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].ID == DefaultModel) != (out[j].ID == DefaultModel) {
			return out[i].ID == DefaultModel
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

var (
	reLocale   = regexp.MustCompile(`^([a-z]{2})[-_][A-Z]{2}[-_]`)           // pt-BR-Luana (Microsoft)
	reSuffix   = regexp.MustCompile(`-([a-z]{2})$`)                          // aura-2-thalia-en (Deepgram)
	reKokoro   = regexp.MustCompile(`^([abefhijpz])[fm]_`)                   // pf_dora (Kokoro)
	rePrefix   = regexp.MustCompile(`^(en|gb|us|pt|br|es|fr|de|it|nl|ja)_`)  // en_paul (Mistral)
	reLangWord = regexp.MustCompile(`^(English|Portuguese|Spanish|French)_`) // English_x (MiniMax)
)

var kokoroLangs = map[string]string{"a": "en", "b": "en", "p": "pt", "e": "es", "f": "fr", "h": "hi", "i": "it", "j": "ja", "z": "zh"}

// voiceLang is the language a voice's name says ("" when it says none).
func voiceLang(v string) string {
	if m := reLocale.FindStringSubmatch(v); m != nil {
		return m[1]
	}
	if m := reKokoro.FindStringSubmatch(v); m != nil {
		return kokoroLangs[m[1]]
	}
	if m := rePrefix.FindStringSubmatch(v); m != nil {
		switch m[1] {
		case "gb", "us":
			return "en"
		case "br":
			return "pt"
		}
		return m[1]
	}
	if m := reLangWord.FindStringSubmatch(v); m != nil {
		return map[string]string{"English": "en", "Portuguese": "pt", "Spanish": "es", "French": "fr"}[m[1]]
	}
	if m := reSuffix.FindStringSubmatch(v); m != nil {
		return m[1]
	}
	return ""
}

// VoicesFor are the voices of m that read a site in lang ("pt" or "en"):
// those whose name says that language, and those that say none when the
// model speaks many languages (else only for English). The model's
// default voice ("") stands for a model without named voices.
func VoicesFor(m ModelInfo, lang string) []string {
	lang = langOf(lang)
	if len(m.Voices) == 0 {
		if m.Multilingual || lang == "en" {
			return []string{""}
		}
		return nil
	}
	var out []string
	for _, v := range m.Voices {
		switch vl := voiceLang(v); {
		case vl == lang:
			out = append(out, v)
		case vl == "" && (m.Multilingual || lang == "en"):
			out = append(out, v)
		}
	}
	if m.ID == DefaultModel {
		// The default voice of the recommended model first.
		def := DefaultVoice(lang)
		for i, v := range out {
			if v == def {
				out[0], out[i] = out[i], out[0]
			}
		}
	}
	return out
}
