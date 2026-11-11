package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

// VersionNotes is what one version changed, in one language.
type VersionNotes struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
}

// Releases lists the computer's published versions, newest first: no
// drafts, no pre-releases, and only tags vX.Y.Z, which leaves out Blup
// Notes' releases (notes-vX.Y.Z) and the side ones (the local voice's
// tools).
func (c *Client) Releases(ctx context.Context) ([]Release, error) {
	var list []struct {
		Tag        string `json:"tag_name"`
		Body       string `json:"body"`
		HTMLURL    string `json:"html_url"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+"/repos/"+Repo+"/releases?per_page=30", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("checking for updates: %s", resp.Status)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxSmall*8)).Decode(&list); err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}
	out := []Release{}
	for _, r := range list {
		// Only the computer's versions: vX.Y.Z (Blup Notes' are notes-vX.Y.Z).
		v, isV := strings.CutPrefix(r.Tag, "v")
		if _, ok := parse(v); !isV || !ok || r.Draft || r.Prerelease {
			continue
		}
		rel := Release{Version: v, Notes: strings.TrimSpace(r.Body), URL: r.HTMLURL, assets: map[string]string{}}
		for _, a := range r.Assets {
			rel.assets[a.Name] = a.URL
		}
		out = append(out, rel)
	}
	slices.SortFunc(out, func(a, b Release) int {
		switch {
		case Newer(a.Version, b.Version):
			return -1
		case Newer(b.Version, a.Version):
			return 1
		}
		return 0
	})
	return out, nil
}

// Since is what changed after version current up to latest (both of the
// history, newest first), each in lang.
func Since(history []VersionNotes, current, latest, lang string) []VersionNotes {
	out := []VersionNotes{}
	for _, h := range history {
		if Newer(h.Version, current) && !Newer(h.Version, latest) {
			out = append(out, VersionNotes{Version: h.Version, Notes: NotesFor(h.Notes, lang)})
		}
	}
	return out
}

// The sections of a release's notes, one per language ("## Português",
// "## English", "## Español"), and the order they stand in for each other.
var sectionLang = map[string]string{"português": "pt", "portugues": "pt", "english": "en", "español": "es", "espanol": "es"}
var fallback = map[string][]string{"pt": {"pt", "en", "es"}, "en": {"en", "pt", "es"}, "es": {"es", "en", "pt"}}

// NotesFor keeps the part of a release's notes in lang (else in English,
// then Portuguese), without the lines about checksums and signatures that
// close every release. Notes with no language sections come as they are,
// without those lines.
func NotesFor(notes, lang string) string {
	sections := map[string][]string{}
	var common []string
	cur := ""
	for _, line := range strings.Split(strings.ReplaceAll(notes, "\r\n", "\n"), "\n") {
		if h, ok := strings.CutPrefix(line, "## "); ok {
			if l, known := sectionLang[strings.ToLower(strings.TrimSpace(h))]; known {
				cur = l
				continue
			}
		}
		if strings.Contains(line, "checksums.txt") {
			continue
		}
		if cur == "" {
			common = append(common, line)
		} else {
			sections[cur] = append(sections[cur], line)
		}
	}
	order, ok := fallback[lang]
	if !ok {
		order = fallback["en"]
	}
	for _, l := range order {
		if body := strings.TrimSpace(strings.Join(sections[l], "\n")); body != "" {
			return body
		}
	}
	return strings.TrimSpace(strings.Join(common, "\n"))
}
