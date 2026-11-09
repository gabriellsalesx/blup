package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// Taking the narrations off a blog (the Podcast screen): only the ones the
// Studio made (a note with audio and audio_hash), translations included,
// never an audio of the person's own (audio without audio_hash). Their
// MP3s go to the Blup trash together (one item), the notes lose the
// narration's properties, and RestoreNarrations puts both back.

// narrationKeys are the properties a narration made by the Studio writes.
var narrationKeys = []string{"audio", "audio_hash", "audio_seconds"}

// generatedNarration is a note with a narration the Studio made.
type generatedNarration struct {
	Rel     string `json:"rel"`
	Lang    string `json:"lang"`
	Audio   string `json:"audio"`
	Hash    string `json:"hash"`
	Seconds any    `json:"seconds,omitempty"`
	File    string `json:"file"` // the MP3 moved to the trash ("" when kept: another note links it)
}

// audioName is the file an "audio" property links ("[[a.mp3]]" → "a.mp3").
func audioName(link string) string {
	name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(link), "[["), "]]"))
	if i := strings.Index(name, "|"); i >= 0 {
		name = name[:i]
	}
	return strings.ToLower(path.Base(filepath.ToSlash(name)))
}

// generatedNarrations walks the site's notes (translations included, not
// the hidden folders nor the templates) for the narrations the Studio made,
// and counts how many notes link each audio file (any note, own audio too).
func generatedNarrations(site *config.Site) ([]generatedNarration, map[string]int, error) {
	root := site.ContentDir()
	templates := ""
	if t := obsidian.TemplatesFolder(site.Vault); t != "" {
		if rel, err := filepath.Rel(root, filepath.Join(site.Vault, t)); err == nil {
			templates = filepath.ToSlash(rel)
		}
	}
	var out []generatedNarration
	links := map[string]int{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if p != root && (strings.HasPrefix(d.Name(), ".") || rel == templates) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(path.Ext(rel), ".md") {
			return nil
		}
		n, err := obsidian.ReadNote(p)
		if err != nil || n.String("audio") == "" {
			return nil
		}
		links[audioName(n.String("audio"))]++
		if n.String("audio_hash") == "" {
			return nil // the person's own audio
		}
		_, lang := NoteOrigin(site, rel)
		out = append(out, generatedNarration{Rel: rel, Lang: lang, Audio: n.String("audio"), Hash: n.String("audio_hash"), Seconds: n.Props["audio_seconds"]})
		return nil
	})
	return out, links, err
}

// NarrationsMade tells how many notes have a narration the Studio made,
// and in which languages (the site's first).
func NarrationsMade(site *config.Site) (int, []string, error) {
	list, _, err := generatedNarrations(site)
	return len(list), narrationLangs(site, list), err
}

func narrationLangs(site *config.Site, list []generatedNarration) []string {
	langs := []string{}
	for _, l := range append([]string{site.Lang}, site.OtherLangs()...) {
		for _, g := range list {
			if g.Lang == l {
				langs = append(langs, l)
				break
			}
		}
	}
	return langs
}

// RemovedNarrations is what RemoveNarrations did: how many notes, in which
// languages, and the trash folder that undoes it.
type RemovedNarrations struct {
	Notes int      `json:"notes"`
	Langs []string `json:"langs"`
	Trash string   `json:"trash"`
}

// manifestName keeps, in the trash folder, what each note had (to undo).
const manifestName = ".narrations.json"

// RemoveNarrations takes the narrations the Studio made off the site's
// notes: their MP3s go to one folder of the Blup trash (an MP3 another
// note still links stays where it is) and the notes lose audio,
// audio_hash and audio_seconds (a note deleted meanwhile is left alone).
func RemoveNarrations(site *config.Site) (RemovedNarrations, error) {
	list, links, err := generatedNarrations(site)
	if err != nil || len(list) == 0 {
		return RemovedNarrations{Langs: []string{}}, err
	}
	// The files every linking note of which goes now.
	removing := map[string]int{}
	for _, g := range list {
		removing[audioName(g.Audio)]++
	}
	stamp := time.Now().Format("2006-01-02-150405")
	dir := filepath.Join(trashDir(site), stamp)
	for i := 2; ; i++ {
		if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
			break
		}
		dir = filepath.Join(trashDir(site), fmt.Sprintf("%s.%d", stamp, i))
	}
	folder := filepath.Join(dir, byLang(site, "Narrações", "Narrations", "Narraciones"))
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return RemovedNarrations{}, err
	}
	moved := map[string]string{}
	done := []generatedNarration{}
	for _, g := range list {
		name := audioName(g.Audio)
		if dst, ok := moved[name]; ok {
			g.File = dst
		} else if links[name] == removing[name] {
			if src := attachmentFile(site, g.Audio); src != "" && !inTrash(site, src) {
				dst := filepath.Base(src)
				if err := moveDir(src, filepath.Join(folder, dst)); err != nil {
					return RemovedNarrations{}, err
				}
				moved[name] = dst
				g.File = dst
			}
		}
		p, err := notesPath(site, g.Rel)
		if err != nil {
			return RemovedNarrations{}, err
		}
		props := make([]obsidian.Prop, 0, len(narrationKeys))
		for _, k := range narrationKeys {
			props = append(props, obsidian.Prop{Key: k, Value: nil})
		}
		if err := obsidian.UpdateProperties(p, props...); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return RemovedNarrations{}, err
		}
		done = append(done, g)
	}
	b, err := json.Marshal(done)
	if err != nil {
		return RemovedNarrations{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, manifestName), b, 0o644); err != nil {
		return RemovedNarrations{}, err
	}
	// Where the MP3s were, for the trash screen.
	origin := path.Join(attachmentsFolder(site), filepath.Base(folder))
	if err := os.WriteFile(filepath.Join(dir, ".origin"), []byte(origin), 0o644); err != nil {
		return RemovedNarrations{}, err
	}
	if len(moved) == 0 {
		os.Remove(folder) // nothing in it: no empty item in the trash
	}
	return RemovedNarrations{Notes: len(done), Langs: narrationLangs(site, done), Trash: folder}, nil
}

// RestoreNarrations undoes RemoveNarrations: the MP3s go back to the
// attachments (a name taken meanwhile keeps the file in the trash) and the
// notes get their narration's properties again.
func RestoreNarrations(site *config.Site, trash string) error {
	dir := filepath.Dir(trash)
	if filepath.Dir(dir) != trashDir(site) {
		return errors.New("not in the Blup trash")
	}
	b, err := os.ReadFile(filepath.Join(dir, manifestName))
	if err != nil {
		return err
	}
	var list []generatedNarration
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}
	var first error
	keep := func(err error) {
		if first == nil {
			first = err
		}
	}
	att := filepath.Join(site.ContentDir(), attachmentsFolder(site))
	back := map[string]bool{}
	for _, g := range list {
		if g.File != "" && !back[g.File] {
			back[g.File] = true
			dst := filepath.Join(att, g.File)
			if _, err := os.Stat(dst); err == nil {
				keep(fmt.Errorf("%s: a file with that name is in the attachments", g.File))
			} else if err := moveDir(filepath.Join(trash, g.File), dst); err != nil {
				keep(err)
			}
		}
		p, err := notesPath(site, g.Rel)
		if err != nil {
			keep(err)
			continue
		}
		if err := obsidian.UpdateProperties(p, obsidian.Prop{Key: "audio", Value: g.Audio}, obsidian.Prop{Key: "audio_hash", Value: g.Hash}, obsidian.Prop{Key: "audio_seconds", Value: g.Seconds}); err != nil && !errors.Is(err, os.ErrNotExist) {
			keep(err)
		}
	}
	if left, _ := os.ReadDir(trash); len(left) == 0 {
		os.RemoveAll(dir)
	}
	return first
}
