// Package translate turns a note into another language with an AI model:
// through the person's OpenRouter key (chat completions), or through Blup
// Cloud on the plan's AI allowance (its /translate, which builds the same
// request on the server). A note goes in parts, split at blank lines
// outside code, so a long note never exceeds what a request carries.
package translate

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

// Model is the OpenRouter model that translates: cheap, fast and good at
// Portuguese, English and Spanish.
const Model = "anthropic/claude-haiku-5.5"

// MaxPart is the most a part carries, in bytes.
const MaxPart = 12000

// Names are the languages' names in English, for the instructions.
var Names = map[string]string{"pt": "Brazilian Portuguese", "en": "English", "es": "Spanish (neutral Latin American)"}

var (
	ErrInvalidKey = errors.New("translate: invalid key")
	ErrNoCredits  = errors.New("translate: no credits")
	ErrLimit      = errors.New("translate: AI allowance used up")
)

// Client translates through OpenRouter (Blup false) or Blup Cloud (Blup
// true, Key being the Blup session).
// Models, on OpenRouter, is the order to try (the first answers, the
// others take over when it fails, OpenRouter's fallback); empty, Model.
type Client struct {
	Key     string
	BaseURL string
	Blup    bool
	Models  []string
	HTTP    *http.Client
}

// Prompt is the instruction for a part of a note, the same on OpenRouter
// and on Blup Cloud.
func Prompt(from, to string) string {
	return fmt.Sprintf(`You translate Markdown notes of a website from %s to %s.
Rules:
- Output only the translation, nothing before or after it.
- Keep the Markdown exactly: headings, lists, tables, quotes, callouts ("> [!tip] Title": translate the title, never the type in brackets), bold, italics, line breaks and blank lines.
- Never change code blocks, inline code, HTML tags, URLs, the targets of links (the part in parentheses) or of [[wikilinks]] (translate only the text after |), image paths, file names, {.attributes} or math between $.
- Keep names of people, products and brands.
- Write naturally, as a native speaker would, keeping the tone.`, Names[from], Names[to])
}

// Text translates text (a whole note's body, or a short field) from one
// language to another.
func (c *Client) Text(ctx context.Context, text, from, to string, progress func(done, total int)) (string, error) {
	if strings.TrimSpace(text) == "" {
		return text, nil
	}
	parts := Split(text, MaxPart)
	var out strings.Builder
	for i, p := range parts {
		if progress != nil {
			progress(i, len(parts))
		}
		lead, body, trail := cutSpace(p)
		if body == "" {
			out.WriteString(p)
			continue
		}
		t, err := c.part(ctx, body, from, to)
		if err != nil {
			return "", err
		}
		out.WriteString(lead + strings.TrimSpace(t) + trail)
	}
	if progress != nil {
		progress(len(parts), len(parts))
	}
	return out.String(), nil
}

// cutSpace splits the blank space around s, kept as it is.
func cutSpace(s string) (lead, body, trail string) {
	body = strings.TrimLeft(s, " \t\r\n")
	lead = s[:len(s)-len(body)]
	t := strings.TrimRight(body, " \t\r\n")
	return lead, t, body[len(t):]
}

// Split cuts text into parts of at most max bytes at blank lines outside
// code blocks: paragraphs are packed into a part while they fit, so a part
// only goes over max when one paragraph (or code block) alone does.
func Split(text string, max int) []string {
	var parts []string
	var cur, block strings.Builder
	fence := false
	flush := func() {
		if cur.Len() > 0 && cur.Len()+block.Len() > max {
			parts = append(parts, cur.String())
			cur.Reset()
		}
		cur.WriteString(block.String())
		block.Reset()
	}
	for line := range strings.Lines(text) {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			fence = !fence
		}
		block.WriteString(line)
		if !fence && trim == "" {
			flush()
		}
	}
	flush()
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

func (c *Client) part(ctx context.Context, text, from, to string) (string, error) {
	var body []byte
	var url string
	if c.Blup {
		url = c.BaseURL + "/translate"
		body, _ = json.Marshal(map[string]string{"text": text, "from": from, "to": to})
	} else {
		url = c.BaseURL + "/chat/completions"
		req := map[string]any{
			"temperature": 0.2,
			"messages": []map[string]string{
				{"role": "system", "content": Prompt(from, to)},
				{"role": "user", "content": text},
			},
		}
		if len(c.Models) > 0 {
			req["models"] = c.Models
		} else {
			req["model"] = Model
		}
		body, _ = json.Marshal(req)
	}
	var last error
	for try := range 3 {
		if try > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(try*try) * 2 * time.Second):
			}
		}
		out, retry, err := c.post(ctx, url, body)
		if err == nil {
			return out, nil
		}
		last = err
		if !retry {
			break
		}
	}
	return "", last
}

func (c *Client) post(ctx context.Context, url string, body []byte) (text string, retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/gabriellsalesx/blup")
	req.Header.Set("X-OpenRouter-Title", "Blup Studio")
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 3 * time.Minute}
	}
	res, err := hc.Do(req)
	if err != nil {
		return "", ctx.Err() == nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return "", true, err
	}
	if res.StatusCode != http.StatusOK {
		return "", res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500, apiError(res.StatusCode, b)
	}
	if c.Blup {
		var r struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(b, &r); err != nil || r.Text == "" {
			return "", true, errors.New("translate: empty answer")
		}
		return r.Text, false, nil
	}
	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(b, &r); err != nil || len(r.Choices) == 0 || r.Choices[0].Message.Content == "" {
		return "", true, errors.New("translate: empty answer")
	}
	if r.Choices[0].FinishReason == "length" {
		return "", false, errors.New("translate: the answer was cut")
	}
	return r.Choices[0].Message.Content, false, nil
}

// apiError reads OpenRouter's (and Blup Cloud's) error body.
func apiError(status int, b []byte) error {
	var e struct {
		Error struct {
			Message string `json:"message"`
			Code    any    `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(b, &e)
	msg := e.Error.Message
	if msg == "" {
		msg = http.StatusText(status)
	}
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrInvalidKey, msg)
	case status == http.StatusPaymentRequired && fmt.Sprint(e.Error.Code) != "402" && fmt.Sprint(e.Error.Code) != "<nil>":
		return fmt.Errorf("%w: %s", ErrLimit, msg)
	case status == http.StatusPaymentRequired:
		return fmt.Errorf("%w: %s", ErrNoCredits, msg)
	}
	return fmt.Errorf("translate: HTTP %d: %s", status, msg)
}
