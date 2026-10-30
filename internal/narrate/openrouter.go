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
	"strings"
	"time"
)

// DefaultBaseURL is the root of OpenRouter's API.
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// KeyPage is where a new OpenRouter key is made.
const KeyPage = "https://openrouter.ai/settings/keys"

// Model is the voice model used: it reads Portuguese and English well and
// follows instructions about the tone.
const Model = "openai/gpt-4o-mini-tts-2025-12-15"

// Voices are the voices of Model, the first being the default.
var Voices = []string{"nova", "alloy", "ash", "ballad", "coral", "echo", "fable", "onyx", "sage", "shimmer", "verse"}

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

// Speak reads text aloud with voice and returns the audio as MP3.
// instructions guide the delivery (language, tone). Busy or failing
// servers are tried again a few times.
func (c *Client) Speak(ctx context.Context, text, voice, instructions string) ([]byte, error) {
	body, err := json.Marshal(map[string]any{
		"model":           Model,
		"input":           text,
		"voice":           voice,
		"instructions":    instructions,
		"response_format": "mp3",
	})
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
	switch res.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrInvalidKey, msg)
	case http.StatusPaymentRequired:
		return fmt.Errorf("%w: %s", ErrNoCredits, msg)
	}
	return fmt.Errorf("openrouter: HTTP %d: %s", res.StatusCode, msg)
}
