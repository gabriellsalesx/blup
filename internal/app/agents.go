package app

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/scaffold"
)

// The guides for AI agents (Claude Code, Codex, Gemini, run in Blup
// Studio's terminal or anywhere else): INSTRUCTIONS.md at the top of the
// notes folder explains, for the vault's theme, how the notes become the
// site, so agents write notes the site understands. AGENTS.md, CLAUDE.md
// and GEMINI.md, the files each agent reads first, point to it. They are
// never pages (scaffold.AgentGuides). A guide the person changed is never
// rewritten; one still as Blup wrote it follows Blup's newer text.

//go:embed guides/*.md
var guides embed.FS

// pointer is AGENTS.md, CLAUDE.md and GEMINI.md: Claude Code and Gemini
// also read the file named after an @.
func pointer(name string) string {
	s := "# Instructions for AI agents\n\nRead INSTRUCTIONS.md before doing anything in this vault.\n"
	if name != "AGENTS.md" {
		s += "\n@INSTRUCTIONS.md\n"
	}
	return s
}

var languageNames = map[string]string{"pt": "Portuguese", "en": "English", "es": "Spanish"}

// guideTheme is the guide a site gets: its theme's, or the notes one.
func guideTheme(site *config.Site) string {
	if site.Kind == config.KindNotes {
		return "notes"
	}
	switch site.Model {
	case "nori", "bento":
		return site.Model
	}
	return "mochi"
}

// instructions is INSTRUCTIONS.md for a theme ("mochi", "nori", "bento"
// or "notes") and a site language.
func instructions(theme, lang string) string {
	b, err := guides.ReadFile("guides/" + theme + ".md")
	if err != nil {
		panic(err) // embedded: present at build time
	}
	name := languageNames[lang]
	if name == "" {
		name = "English"
	}
	return strings.ReplaceAll(string(b), "{{LANGUAGE}}", name)
}

// agentGuide is what Blup writes in the guide named name.
func agentGuide(site *config.Site, name string) string {
	if name == "INSTRUCTIONS.md" {
		return instructions(guideTheme(site), site.Lang)
	}
	return pointer(name)
}

// written are the texts Blup ever wrote in a guide (by their SHA-256):
// a file still holding one of them was not changed by the person, so it
// may follow the newer text. Earlier versions: the AGENTS.md of each
// language and the "@AGENTS.md" of CLAUDE.md and GEMINI.md.
var written = func() map[[32]byte]bool {
	m := map[[32]byte]bool{}
	for _, h := range []string{
		"10a2e6c50eb6533fa05b615594a684a4467e7eb5772d52271f7eff740734469d",
		"81a9e5d7242485927012b9a3e73d7605122f86ff120ecd711b8403ba2d3af081",
		"32df9b848850a0d375ea63c5e845a93c1043871c8ed525dfe2d96734f37c0132",
		"336cc4fbf19beaada7ccf9986414fa91851a8d7a07dfb3ccbe800a69eed0ab49",
	} {
		var k [32]byte
		hex.Decode(k[:], []byte(h))
		m[k] = true
	}
	for _, name := range scaffold.AgentGuides {
		m[sha256.Sum256([]byte(pointer(name)))] = true
	}
	for _, theme := range []string{"mochi", "nori", "bento", "notes"} {
		for lang := range languageNames {
			m[sha256.Sum256([]byte(instructions(theme, lang)))] = true
		}
	}
	return m
}()

// isAgentGuide tells whether rel (a path in the notes folder, with /) is
// one of the guides: not a note of the person, so out of the map and the
// search.
func isAgentGuide(rel string) bool { return slices.Contains(scaffold.AgentGuides, rel) }

// WriteAgentGuides creates the guides that are missing and brings up to
// date the ones still as Blup wrote them; the ones the person changed
// stay as they are.
func WriteAgentGuides(site *config.Site) error {
	root := site.ContentDir()
	for _, name := range scaffold.AgentGuides {
		body := agentGuide(site, name)
		p := filepath.Join(root, name)
		// Lstat: a link there (to nothing, in a downloaded folder) is left
		// alone, not followed to write the guide wherever it points.
		if fi, err := os.Lstat(p); err == nil {
			if !fi.Mode().IsRegular() || fi.Size() > 1<<20 {
				continue
			}
			old, err := os.ReadFile(p)
			if err != nil || string(old) == body || !written[sha256.Sum256(old)] {
				continue
			}
			if err := WriteInside(root, p, []byte(body)); err != nil {
				return err
			}
			continue
		}
		if err := writeNew(root, p, []byte(body)); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
	}
	return nil
}
