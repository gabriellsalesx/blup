package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // the zones, whatever the phone has

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/gabriellsalesx/blup/internal/scaffold"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Note is one Markdown file of the vault. The ID is its path in the vault.
// The list carries no text: it would cross to the screen for every note on
// every refresh. The editor reads it (Read) and the search looks in Go
// (Search).
type Note struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Folder    string   `json:"folder"`
	Preview   string   `json:"preview"`
	Tags      []string `json:"tags"`     // the properties' and the #tags in the text
	PropTags  []string `json:"propTags"` // only the properties'
	Created   string   `json:"created"`  // the properties' dates, as 2006-01-02
	Changed   string   `json:"changed"`
	Links     []string `json:"links"`
	Backlinks []string `json:"backlinks"`
	Words     int      `json:"words"`
	Updated   int64    `json:"updated"`
}

// Device is what the interface needs to know about the phone.
type Device struct {
	Platform string  `json:"platform"`
	Vault    string  `json:"vault"`
	Top      float64 `json:"top"`
	Bottom   float64 `json:"bottom"`
}

// NotesService keeps the notes as plain files in the app's private folder.
type NotesService struct{}

var (
	wikilink  = regexp.MustCompile(`\[\[([^\]\n|]+)(?:\|([^\]\n]+))?\]\]`)
	mdNote    = regexp.MustCompile(`\[[^\]\n]*\]\(([^)\s]+\.md)\)`)
	mdAnyLink = regexp.MustCompile(`\[([^\]\n]*)\]\([^)\n]*\)`)
	hashtag   = regexp.MustCompile(`(?:^|\s)#([\p{L}\p{N}_-]+)`)
)

// dataDir is the app's private folder. The Blup packages keep their files
// under the user's config folder, which Android doesn't set: it goes here.
func dataDir() (string, error) {
	root := application.Mobile.StoragePath()
	if root == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		dir := filepath.Join(base, "blup-notes")
		// The folder of the builds made before the app had its name.
		if old := filepath.Join(base, "blup-mobile-"+"prototipo"); dirExists(old) && !dirExists(dir) {
			_ = os.Rename(old, dir)
		}
		return dir, nil
	}
	os.Setenv("HOME", root)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	return root, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func vaultDir() (string, error) {
	root, err := currentRoot()
	if err != nil {
		return "", err
	}
	return notesAt(root)
}

// notesAt is the notes folder of a vault. Only the phone's first vault starts
// with the welcome note: the others are made empty, so the folder is there.
func notesAt(root string) (string, error) {
	dir := filepath.Join(root, "cofre")
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return dir, seed(dir)
	}
	return dir, nil
}

func notePath(id string) (string, error) {
	if !filepath.IsLocal(id) || filepath.Ext(id) != ".md" {
		return "", fail("badNote", id)
	}
	dir, err := vaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id), nil
}

// mdImage is a picture in the text, which the preview of a note leaves out.
var mdImage = regexp.MustCompile(`!\[\[[^\]\n]*\]\]|!\[[^\]\n]*\]\([^)\n]*\)`)

// calloutMark is what opens a callout, which the preview leaves out.
var calloutMark = regexp.MustCompile(`\[![\w-]+\][-+]?\s*`)

// frontmatter is the note's properties block, in the Studio's format.
var frontmatter = regexp.MustCompile(`(?s)\A---\r?\n.*?\r?\n---\r?\n?`)

// day keeps the date of a property, which may come with a time.
func day(v string) string {
	if len(v) > 10 {
		return v[:10]
	}
	return v
}

// previewRunes is as much of the text as the list shows (two lines, at
// most): a long first paragraph would cross to the screen whole, for
// every note, only to be cut there.
const previewRunes = 240

// clip cuts s to n runes at most.
func clip(s string, n int) string {
	for i := range s {
		if n == 0 {
			return s[:i]
		}
		n--
	}
	return s
}

// replaceIf and findIf run re only on text that has its mark: most lines
// have none, and a regexp costs far more than looking for a few bytes.
func replaceIf(re *regexp.Regexp, s, mark, with string) string {
	if !strings.Contains(s, mark) {
		return s
	}
	return re.ReplaceAllString(s, with)
}

func findIf(re *regexp.Regexp, s, mark string) [][]string {
	if !strings.Contains(s, mark) {
		return nil
	}
	return re.FindAllStringSubmatch(s, -1)
}

func parse(id, body string, own obsidian.Note, updated time.Time) Note {
	text := frontmatter.ReplaceAllString(body, "")
	n := Note{ID: id, Updated: updated.UnixMilli(), Created: day(own.String("created")), Changed: day(own.String("updated")), Tags: []string{}, PropTags: []string{}, Links: []string{}, Backlinks: []string{}}
	if d := filepath.Dir(id); d != "." {
		n.Folder = d
	}
	n.Title = strings.TrimSuffix(filepath.Base(id), ".md")
	var preview []string
	for line := range strings.SplitSeq(text, "\n") {
		if len(preview) == 2 {
			break // the rest of the text has nothing for the preview
		}
		line = strings.TrimSpace(replaceIf(mdImage, line, "![", ""))
		if line == "#" || (strings.HasPrefix(line, "# ") && len(preview) == 0) {
			continue
		}
		if line != "" && replaceIf(hashtag, line, "#", "") != "" {
			line = strings.ReplaceAll(replaceIf(wikilink, line, "[[", "$1"), "*", "")
			line = replaceIf(mdAnyLink, line, "](", "$1")
			line = replaceIf(calloutMark, line, "[!", "")
			preview = append(preview, strings.TrimLeft(line, "#->[] x"))
		}
	}
	n.Preview = clip(strings.Join(preview, " "), previewRunes)
	seen := map[string]bool{}
	for _, t := range own.List("tags") {
		t = strings.TrimPrefix(t, "#")
		if t != "" && !seen["#"+t] {
			seen["#"+t] = true
			n.Tags = append(n.Tags, t)
			n.PropTags = append(n.PropTags, t)
		}
	}
	for _, m := range findIf(hashtag, text, "#") {
		if !seen["#"+m[1]] {
			seen["#"+m[1]] = true
			n.Tags = append(n.Tags, m[1])
		}
	}
	for _, m := range findIf(wikilink, text, "[[") {
		if t := strings.TrimSpace(m[1]); !seen[t] {
			seen[t] = true
			n.Links = append(n.Links, t)
		}
	}
	for _, m := range findIf(mdNote, text, "](") {
		name, err := url.PathUnescape(strings.TrimSuffix(filepath.Base(m[1]), ".md"))
		if err == nil && !seen[name] {
			seen[name] = true
			n.Links = append(n.Links, name)
		}
	}
	n.Words = len(strings.Fields(text))
	return n
}

// listCache keeps each note as List last read it, by its file. The list is
// asked again after every save, sync and change: only the files changed
// since then are read and parsed again. A file is known by its full path,
// so each vault of the phone has its own.
var listCache struct {
	sync.Mutex
	notes map[string]cachedNote
}

type cachedNote struct {
	note Note // without backlinks: they come from the other notes
	mod  time.Time
	size int64
}

// List returns every note, the newest first, with its backlinks filled in.
// agentGuide tells whether path is one of the guides Blup Studio keeps at
// the top of the notes folder for AI agents (AGENTS.md, CLAUDE.md,
// GEMINI.md, INSTRUCTIONS.md): they stay in the folder and sync, but are
// not notes of the person, so the phone never lists or finds them (the
// Studio hides them too).
func agentGuide(root, path string) bool {
	return filepath.Dir(path) == root && slices.Contains(scaffold.AgentGuides, filepath.Base(path))
}

func (s *NotesService) List() ([]Note, error) {
	dir, err := vaultDir()
	if err != nil {
		return nil, err
	}
	listCache.Lock()
	defer listCache.Unlock()
	type item struct {
		path string
		info os.FileInfo
	}
	var items []item
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != dir && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir // the sync's own files
		}
		if d.IsDir() || filepath.Ext(path) != ".md" || agentGuide(dir, path) {
			return nil
		}
		// The date and size before the text: a note saved in between is
		// read again next time, never kept with its old text.
		info, err := d.Info()
		if err != nil {
			return err
		}
		items = append(items, item{path, info})
		return nil
	})
	if err != nil {
		return nil, err
	}
	// What changed since the last list is read and parsed by every core of
	// the phone at once; the rest comes from the cache.
	notes := make([]Note, len(items))
	fresh := make(map[string]cachedNote, len(items))
	errs := make([]error, len(items))
	var wg sync.WaitGroup
	work := make(chan int)
	for range min(runtime.NumCPU(), 8) {
		wg.Go(func() {
			for i := range work {
				it := items[i]
				body, err := os.ReadFile(it.path)
				if err != nil {
					errs[i] = err
					continue
				}
				id, err := filepath.Rel(dir, it.path)
				if err != nil {
					errs[i] = err
					continue
				}
				own, err := obsidian.ReadNote(it.path)
				if err != nil {
					own = obsidian.Note{} // broken properties: the note still shows
				}
				notes[i] = parse(id, string(body), own, it.info.ModTime())
			}
		})
	}
	for i, it := range items {
		if c, ok := listCache.notes[it.path]; ok && c.mod.Equal(it.info.ModTime()) && c.size == it.info.Size() {
			notes[i] = c.note
			continue
		}
		work <- i
	}
	close(work)
	wg.Wait()
	for i, it := range items {
		if errs[i] != nil {
			return nil, errs[i]
		}
		fresh[it.path] = cachedNote{notes[i], it.info.ModTime(), it.info.Size()}
	}
	listCache.notes = fresh // the notes gone from the folder go from here too
	// The notes by their title in lower case: a link finds its notes at
	// once, instead of every note being checked against every link.
	byTitle := make(map[string][]int, len(notes))
	for i := range notes {
		key := strings.ToLower(notes[i].Title)
		byTitle[key] = append(byTitle[key], i)
	}
	for j := range notes {
		for _, link := range notes[j].Links {
			for _, i := range byTitle[strings.ToLower(link)] {
				if i != j {
					notes[i].Backlinks = append(notes[i].Backlinks, notes[j].Title)
				}
			}
		}
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].Updated > notes[j].Updated })
	return notes, nil
}

// Read is the note's text, properties included, as the editor opens it.
func (s *NotesService) Read(id string) (string, error) {
	path, err := notePath(id)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

// Search returns the IDs of the notes whose name or text has term, in any
// case, the newest first as in the list.
func (s *NotesService) Search(term string) ([]string, error) {
	term = strings.ToLower(strings.TrimSpace(term))
	dir, err := vaultDir()
	if err != nil {
		return nil, err
	}
	type hit struct {
		id string
		at int64
	}
	var hits []hit
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != dir && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if d.IsDir() || filepath.Ext(path) != ".md" || term == "" || agentGuide(dir, path) {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		id, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		title := strings.TrimSuffix(filepath.Base(id), ".md")
		if !strings.Contains(strings.ToLower(title+"\n"+string(body)), term) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		hits = append(hits, hit{id, info.ModTime().UnixMilli()})
		return nil
	})
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].at > hits[j].at })
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.id
	}
	return out, err
}

// Save writes the note through a temporary file, so a closed app never
// leaves half a note behind, and dates it as the Studio does: created when it
// has none, updated today. It returns the note as it stands.
func (s *NotesService) Save(id, body string) (string, error) {
	path, err := notePath(id)
	if err != nil {
		return "", err
	}
	today := time.Now().Format("2006-01-02")
	dates := func(tmp string) []obsidian.Prop {
		list := []obsidian.Prop{{Key: "updated", Value: today}}
		if own, err := obsidian.ReadNote(tmp); err == nil && own.String("created") == "" {
			list = append(list, obsidian.Prop{Key: "created", Value: today})
		}
		return list
	}
	// The text and its dates reach the disk together. Properties broken by
	// hand keep the dates as they are: the text is what must be saved.
	if _, err := saveWithProps(path, []byte(body), dates); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

// SetProps changes the note's dates and tags and returns it as it stands.
func (s *NotesService) SetProps(id, created, updated string, tags []string) (string, error) {
	path, err := notePath(id)
	if err != nil {
		return "", err
	}
	list := []obsidian.Prop{{Key: "tags", Value: tags}}
	if created != "" {
		list = append(list, obsidian.Prop{Key: "created", Value: created})
	}
	if updated != "" {
		list = append(list, obsidian.Prop{Key: "updated", Value: updated})
	}
	if err := setPropsSafe(path, list...); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

// Create makes a note in the folder as the Studio does: its name is the
// file's, it starts with its dates and takes the tags of the folder's notes.
func (s *NotesService) Create(folder string) (string, error) {
	st, err := site()
	if err != nil {
		return "", err
	}
	for i := 1; ; i++ {
		name := tr("untitled") + ".md"
		if i > 1 {
			name = fmt.Sprintf("%s %d.md", tr("untitled"), i)
		}
		id := filepath.Join(folder, name)
		path, err := notePath(id)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return "", err
		}
		today := time.Now().Format("2006-01-02")
		if err := os.WriteFile(path, []byte("---\ncreated: "+today+"\nupdated: "+today+"\n---\n\n"), 0o600); err != nil {
			return "", err
		}
		if tags := app.FolderTags(st, folder); len(tags) > 0 {
			return id, obsidian.SetProperties(path, obsidian.Prop{Key: "tags", Value: tags})
		}
		return id, nil
	}
}

// Mentions are the notes with a link to this one, the Studio's "mentioned in".
func (s *NotesService) Mentions(id string) ([]app.Mention, error) {
	st, err := site()
	if err != nil {
		return nil, err
	}
	return app.Mentions(st, filepath.ToSlash(id))
}

// Rename gives the note a new name, fixing the links to it, and returns its
// new ID.
func (s *NotesService) Rename(id, name string) (string, error) {
	st, err := site()
	if err != nil {
		return "", err
	}
	to, err := app.RenameItem(st, id, name)
	if err != nil {
		return "", err
	}
	return to, writeReminders()
}

// Move puts the note in another folder ("" for none) and returns its new ID.
func (s *NotesService) Move(id, folder string) (string, error) {
	st, err := site()
	if err != nil {
		return "", err
	}
	to, err := app.MoveItem(st, id, folder)
	if err != nil {
		return "", err
	}
	return to, writeReminders()
}

// Folders lists every folder of the vault, the empty ones too.
func (s *NotesService) Folders() ([]string, error) {
	dir, err := vaultDir()
	if err != nil {
		return nil, err
	}
	out := []string{}
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == dir {
			return err
		}
		if strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if filepath.Dir(path) == dir && d.Name() == attachments {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out, err
}

// CreateFolder makes a folder inside parent ("" for the top) and returns it.
func (s *NotesService) CreateFolder(parent, name string) (string, error) {
	name = strings.TrimSpace(strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|#^[]`, r) {
			return -1
		}
		return r
	}, name))
	if name == "" || strings.HasPrefix(name, ".") {
		return "", fail("folderName")
	}
	dir, err := vaultDir()
	if err != nil {
		return "", err
	}
	rel := filepath.Join(parent, name)
	if !filepath.IsLocal(rel) {
		return "", fail("badFolder", rel)
	}
	if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
		return "", fail("folderTaken")
	}
	return rel, os.MkdirAll(filepath.Join(dir, rel), 0o700)
}

// Delete removes the note for good: an empty one nobody wrote in. What the
// person deletes goes to the trash, by CalendarService.Delete.
func (s *NotesService) Delete(id string) error {
	path, err := notePath(id)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// attachments is the folder of the images, as the Studio has it in Portuguese.
// It holds no notes, so it stays out of the folders on the screen.
const attachments = "Anexos"

// imageFile tells a picture from anything else the vault has.
var imageFile = regexp.MustCompile(`(?i)\.(png|jpe?g|webp|gif|svg|avif)$`)

// AddImage saves a picture (its bytes in base64) in the attachments folder, as
// the Studio does, and returns the Markdown that shows it in the note.
func (s *NotesService) AddImage(note, name, data string) (string, error) {
	if _, err := notePath(note); err != nil {
		return "", err
	}
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	st, err := site()
	if err != nil {
		return "", err
	}
	md, err := app.PasteImage(st, filepath.ToSlash(note), name, b)
	if err != nil {
		return "", fail("imageFail")
	}
	return md, nil
}

// Image returns a picture a note shows (![](src), relative to the note, or
// ![[name.png]]) as a data: URL, "" when missing. A note that came from
// somewhere else may point outside the vault: only the vault's images open.
func (s *NotesService) Image(note, src string) (string, error) {
	st, err := site()
	if err != nil {
		return "", err
	}
	file := ""
	if strings.HasPrefix(src, "[[") {
		file = app.AttachmentFile(st, src)
	} else {
		if u, err := url.PathUnescape(src); err == nil {
			src = u
		}
		file = filepath.Join(st.ContentDir(), filepath.FromSlash(path.Join(path.Dir("/"+filepath.ToSlash(note)), src)))
		if _, err := os.Stat(file); err != nil {
			file = app.AttachmentFile(st, path.Base(src)) // moved: found by its name
		}
	}
	if file == "" || !imageFile.MatchString(file) || !app.Within(st.ContentDir(), file) {
		return "", nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return "", nil
	}
	kind := mime.TypeByExtension(filepath.Ext(file))
	if kind == "" {
		kind = http.DetectContentType(data)
	}
	return "data:" + kind + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// OpenURL opens an address in the phone's browser, mail or phone app. Only
// web, e-mail and phone addresses: a link in a note never reaches another
// kind of app.
func (s *NotesService) OpenURL(url string) {
	if openable(url) {
		application.Mobile.OpenURL(url)
	}
}

func openable(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "mailto", "tel":
		return true
	}
	return false
}

// Device takes the phone's time zone (Go on Android runs in UTC, which
// would put every event hours off) and returns the platform, the vault
// folder and the system bars in px.
func (s *NotesService) Device(zone string) (Device, error) {
	if loc, err := time.LoadLocation(zone); err == nil {
		time.Local = loc
	}
	dir, err := vaultDir()
	if err != nil {
		return Device{}, err
	}
	d := Device{Platform: runtime.GOOS, Vault: dir}
	if err := writeReminders(); err != nil {
		return d, err
	}
	application.Mobile.SetKeyboardWatch(true)
	if raw := application.Mobile.SafeAreaJSON(); raw != "" {
		if err := json.Unmarshal([]byte(raw), &d); err != nil {
			return d, err
		}
	}
	return d, nil
}

// Haptic gives the phone a short tap.
func (s *NotesService) Haptic() { application.Mobile.Haptic("impact-light") }

// HapticMedium is the tap of a long press.
func (s *NotesService) HapticMedium() { application.Mobile.Haptic("impact-medium") }

// HapticStrong is the firm tap of something done, like a note deleted.
func (s *NotesService) HapticStrong() { application.Mobile.Haptic("impact-heavy") }

// StatusBar picks dark or light icons for the status bar.
func (s *NotesService) StatusBar(dark bool) {
	style := "light"
	if dark {
		style = "dark"
	}
	application.Mobile.SetStatusBar(fmt.Sprintf(`{"style":%q}`, style))
}
