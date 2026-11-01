// Package narrate turns a note into spoken audio for Blup Studio: the text
// is prepared for reading (Speakable), split into parts a voice model
// accepts (Split), spoken by a model on OpenRouter (Client.Speak) and the
// MP3 parts are joined into one file (JoinMP3).
package narrate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

// DefaultBaseURL is the root of OpenRouter's API.
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// KeyPage is where a new OpenRouter key is made.
const KeyPage = "https://openrouter.ai/settings/keys"

// DefaultModel is the recommended voice model: Microsoft's MAI-Voice-2.1,
// with native voices for each language (pt-BR-Luana, en-US-Harper...).
const DefaultModel = "microsoft/mai-voice-2.1"

// locales maps a site language to the voices' locale.
var locales = map[string]string{"pt": "pt-BR", "en": "en-US", "es": "es-MX"}

// defaultVoices are the voices used until the user picks another.
var defaultVoices = map[string]string{"pt": "pt-BR-Luana:MAI-Voice-2.1", "en": "en-US-Harper:MAI-Voice-2.1", "es": "es-MX-Valeria:MAI-Voice-2.1"}

// knownVoices are DefaultModel's voices for each language, used when
// OpenRouter's list can't be read.
// Known is DefaultModel with its known voices, for when OpenRouter's list
// can't be read.
func Known() ModelInfo {
	return ModelInfo{ID: DefaultModel, Name: "Microsoft AI: MAI-Voice-2.1", PerThousand: 0.022, Voices: append(append([]string{}, knownVoices["pt"]...), knownVoices["en"]...)}
}

var knownVoices = map[string][]string{
	"pt": {"pt-BR-Luana:MAI-Voice-2.1", "pt-BR-Caio:MAI-Voice-2.1", "pt-BR-Rafael:MAI-Voice-2.1", "pt-BR-Pedro:MAI-Voice-2.1", "pt-BR-Harper:MAI-Voice-2.1", "pt-BR-Grant:MAI-Voice-2.1"},
	"en": {"en-US-Harper:MAI-Voice-2.1", "en-US-Olivia:MAI-Voice-2.1", "en-US-Iris:MAI-Voice-2.1", "en-US-Ethan:MAI-Voice-2.1", "en-US-Grant:MAI-Voice-2.1", "en-US-Jasper:MAI-Voice-2.1", "en-US-Sage:MAI-Voice-2.1"},
}

// HasSample tells whether the Studio carries a recorded sample of voice
// (the recommended model's voices from Brazil and the US). OpenRouter
// lists others too (Portugal, the UK, Australia, India): their sample is
// made on demand.
func HasSample(voice string) bool {
	for _, l := range knownVoices {
		if slices.Contains(l, voice) {
			return true
		}
	}
	return false
}

func langOf(lang string) string {
	switch lang {
	case "pt", "es":
		return lang
	}
	return "en"
}

// DefaultVoice is the voice for a site in lang ("pt", "en" or "es").
func DefaultVoice(lang string) string { return defaultVoices[langOf(lang)] }

// VoiceFor returns voice when it speaks lang, else the language's default.
func VoiceFor(lang, voice string) string {
	if strings.HasPrefix(voice, locales[langOf(lang)]+"-") {
		return voice
	}
	return DefaultVoice(lang)
}

// VoiceName is a voice's name ("Luana" for "pt-BR-Luana:MAI-Voice-2.1").
func VoiceName(voice string) string {
	name, _, _ := strings.Cut(voice, ":")
	if i := strings.LastIndex(name, "-"); i >= 0 {
		name = name[i+1:]
	}
	return name
}

// ErrModelGone means OpenRouter no longer offers the model asked for.
var ErrModelGone = errors.New("openrouter: the voice model is no longer offered")

var (
	// ErrInvalidKey: the key is wrong, revoked or disabled.
	ErrInvalidKey = errors.New("openrouter: invalid key")
	// ErrNoCredits: the account has no credits left for this request.
	ErrNoCredits = errors.New("openrouter: not enough credits")
)

// Client calls OpenRouter with a key.
type Client struct {
	Key     string
	BaseURL string       // DefaultBaseURL when empty
	HTTP    *http.Client // a client with a generous timeout when nil
}

func (c *Client) base() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return DefaultBaseURL
}

func (c *Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 3 * time.Minute}
}

// CheckKey tells whether the key works.
func (c *Client) CheckKey(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+"/key", nil)
	if err != nil {
		return err
	}
	c.headers(req)
	res, err := c.client().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return apiError(res)
	}
	return nil
}

// Speak reads text aloud with a model and voice ("" for the model's
// default) and returns the audio as MP3. Busy or failing servers are tried
// again a few times.
func (c *Client) Speak(ctx context.Context, model, text, voice string) ([]byte, error) {
	req := map[string]any{"model": model, "input": text, "response_format": "mp3"}
	if voice != "" {
		req["voice"] = voice
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var last error
	for try := 0; try < 3; try++ {
		if try > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(try*try) * 2 * time.Second):
			}
		}
		audio, retry, err := c.speak(ctx, body)
		if err == nil {
			return audio, nil
		}
		last = err
		if !retry {
			break
		}
	}
	return nil, last
}

func (c *Client) speak(ctx context.Context, body []byte) (audio []byte, retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base()+"/audio/speech", bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}
	c.headers(req)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client().Do(req)
	if err != nil {
		return nil, ctx.Err() == nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		retry = res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500
		return nil, retry, apiError(res)
	}
	audio, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, true, err
	}
	if len(audio) == 0 {
		return nil, true, errors.New("openrouter: empty audio")
	}
	return audio, false, nil
}

func (c *Client) headers(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.Key)
	// OpenRouter's app attribution.
	req.Header.Set("HTTP-Referer", "https://github.com/gabriellsalesx/blup")
	req.Header.Set("X-Title", "Blup Studio")
}

// apiError reads OpenRouter's error body ({"error":{"message":...}}).
func apiError(res *http.Response) error {
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	b, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	_ = json.Unmarshal(b, &e)
	msg := e.Error.Message
	if msg == "" {
		msg = http.StatusText(res.StatusCode)
	}
	if res.StatusCode == http.StatusBadRequest && strings.Contains(msg, "does not exist") {
		return fmt.Errorf("%w: %s", ErrModelGone, msg)
	}
	switch res.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrInvalidKey, msg)
	case http.StatusPaymentRequired:
		return fmt.Errorf("%w: %s", ErrNoCredits, msg)
	}
	return fmt.Errorf("openrouter: HTTP %d: %s", res.StatusCode, msg)
}
