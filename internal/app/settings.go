package app

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/gabriellsalesx/blup/internal/scaffold"
)

// Settings without questions: what the customize questions change, read and
// saved from values a form already has. Blup Studio uses these; each save
// writes the site's files at once, like customize does.

// Settings is what the site shows besides the notes.
type Settings struct {
	Model           string         `json:"model"`
	Lang            string         `json:"lang"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Author          string         `json:"author"`
	Logo            string         `json:"logo"`     // file of the logo, "" for none
	ShowName        bool           `json:"showName"` // the site's name next to the logo in the header
	Favicon         string         `json:"favicon"`  // file of the favicon, "" for none
	Menu            []MenuItem     `json:"menu"`
	MenuOptions     []MenuItem     `json:"menuOptions"` // pages that can go in the menu
	ButtonText      string         `json:"buttonText"`
	ButtonLink      string         `json:"buttonLink"`
	Social          []string       `json:"social"`
	Sections        []Section      `json:"sections"` // Bento only
	Cookies         config.Cookies `json:"cookies"`
	PrivacyNote     string         `json:"privacyNote"` // the policy note, relative to the notes folder
	Feedback        bool           `json:"feedback"`
	FeedbackRepo    string         `json:"feedbackRepo"`
	HasPassword     bool           `json:"hasPassword"`
	GeneralPassword bool           `json:"generalPassword"` // the password for every site is set
}

// MenuItem is an entry of the top menu: a name and where it leads (a
// folder of the site, "/", "/tags/" or an address).
type MenuItem struct {
	Name   string `json:"name"`
	Target string `json:"target"`
}

// Section is one of Bento's parts.
type Section struct {
	Kind    string `json:"kind"`   // docs, roadmap, changelog, help, blog
	Folder  string `json:"folder"` // its folder, "" when the site doesn't have it
	Present bool   `json:"present"`
}

// LoadSettings reads the site's settings (blup.toml and the home note).
func LoadSettings(site *config.Site) (Settings, error) {
	note, err := obsidian.ReadNote(homeNotePath(site))
	if err != nil {
		return Settings{}, err
	}
	s := Settings{
		Model: site.Model, Lang: site.Lang,
		Name: firstOf(note.String("site_name"), site.Title), Description: firstOf(note.String("site_description"), site.Description),
		Author: site.Author, ButtonText: note.String("button_text"), ButtonLink: note.String("button_link"),
		Social: note.List("social"), Cookies: site.Cookies,
		Feedback: site.Feedback.On(), HasPassword: site.Password != "", GeneralPassword: HasGeneralPassword(),
		Logo: attachmentFile(site, note.String("logo")), Favicon: attachmentFile(site, note.String("favicon")),
		ShowName: note.Props["header_name"] != false,
	}
	if s.Social == nil {
		s.Social = []string{}
	}
	if site.Feedback.Repo != "" {
		s.FeedbackRepo = site.Feedback.Owner + "/" + site.Feedback.Repo
	}
	if p := privacyNote(site); p != "" {
		s.PrivacyNote, _ = filepath.Rel(site.ContentDir(), p)
		s.PrivacyNote = filepath.ToSlash(s.PrivacyNote)
	}
	for _, l := range note.List("menu") {
		name, target, ok := strings.Cut(l, "|")
		if !ok {
			target = name
		}
		s.Menu = append(s.Menu, MenuItem{strings.TrimSpace(name), strings.TrimSpace(target)})
	}
	if s.Menu == nil {
		s.Menu = []MenuItem{}
	}
	s.MenuOptions = []MenuItem{{"Home", "/"}}
	authors := obsidian.Sections(site.ContentDir())["authors"]
	for _, f := range obsidian.Folders(site.ContentDir(), 1) {
		if f.Path != authors && f.Notes > 0 {
			s.MenuOptions = append(s.MenuOptions, MenuItem{f.Path, f.Path})
		}
	}
	s.MenuOptions = append(s.MenuOptions, MenuItem{"Tags", "/tags/"})
	{
		have := existingSections(site)
		// Bento's parts, and a blog's podcast. The feedback board and the
		// podcast are the Studio's only (the CLI doesn't offer them).
		for _, k := range sectionKinds(site.Model) {
			folder, ok := have[k]
			if !ok {
				folder = ""
			}
			s.Sections = append(s.Sections, Section{Kind: k, Folder: folder, Present: ok})
		}
	}
	return s, nil
}

func firstOf(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// attachmentFile finds the file of an image property ("[[logo.png]]").
func attachmentFile(site *config.Site, prop string) string {
	name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(prop), "[["), "]]"))
	if name == "" {
		return ""
	}
	if i := strings.Index(name, "|"); i >= 0 {
		name = name[:i]
	}
	direct := filepath.Join(site.ContentDir(), attachmentsFolder(site), filepath.Base(name))
	if _, err := os.Stat(direct); err == nil {
		return direct
	}
	found := ""
	filepath.WalkDir(site.ContentDir(), func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == filepath.Base(name) {
			found = p
			return fs.SkipAll
		}
		return nil
	})
	return found
}

// saved writes blup.toml and the generated files, as customize does after
// each change.
func saved(site *config.Site) error {
	if err := site.Save(); err != nil {
		return err
	}
	return scaffold.Write(site)
}

// SaveAbout changes the site's name, description and author.
func SaveAbout(site *config.Site, name, description, author string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("the site needs a name")
	}
	site.Title, site.Description, site.Author = strings.TrimSpace(name), strings.TrimSpace(description), strings.TrimSpace(author)
	if err := obsidian.SetProperties(homeNotePath(site), textProp("site_name", site.Title), textProp("site_description", site.Description)); err != nil {
		return err
	}
	return saved(site)
}

// SaveMenu sets the top menu and the button at the top right (an empty
// text removes it).
func SaveMenu(site *config.Site, menu []MenuItem, buttonText, buttonLink string) error {
	lines := make([]string, 0, len(menu))
	for _, m := range menu {
		name, target := strings.TrimSpace(m.Name), strings.TrimSpace(m.Target)
		if name == "" || target == "" {
			continue
		}
		lines = append(lines, name+" | "+target)
	}
	if strings.TrimSpace(buttonText) == "" {
		buttonLink = ""
	} else if err := validURL(buttonLink); err != nil {
		return err
	}
	if err := obsidian.SetProperties(homeNotePath(site), listProp("menu", lines), textProp("button_text", buttonText), textProp("button_link", buttonLink)); err != nil {
		return err
	}
	return saved(site)
}

// SaveSocial sets the social links of the footer (addresses, or e-mails).
func SaveSocial(site *config.Site, links []string) error {
	var out []string
	for _, l := range links {
		l = strings.TrimSpace(l)
		switch {
		case l == "":
			continue
		case !strings.Contains(l, "://") && !strings.HasPrefix(l, "mailto:") && strings.Contains(l, "@"):
			l = "mailto:" + l
		case !strings.HasPrefix(l, "mailto:"):
			if err := validURL(l); err != nil {
				return errors.New(l + ": " + err.Error())
			}
		}
		out = append(out, l)
	}
	if err := obsidian.SetProperties(homeNotePath(site), listProp("social", out)); err != nil {
		return err
	}
	return saved(site)
}

// SetImage copies an image into the site as its logo or favicon; src ""
// removes it. It returns the image's file in the site.
func SetImage(site *config.Site, key, src string) (string, error) {
	if key != "logo" && key != "favicon" {
		return "", errors.New("unknown image " + key)
	}
	if src == "" {
		if err := obsidian.SetProperties(homeNotePath(site), obsidian.Prop{Key: key}); err != nil {
			return "", err
		}
		return "", saved(site)
	}
	if err := validImage(src); err != nil {
		return "", err
	}
	link, err := copyAttachment(site, src, key)
	if err != nil {
		return "", err
	}
	if err := obsidian.SetProperties(homeNotePath(site), obsidian.Prop{Key: key, Value: link}); err != nil {
		return "", err
	}
	return attachmentFile(site, link), saved(site)
}

// AddSection creates a section with its example notes: a part of Bento, or
// a blog's podcast. A site with its own menu gets the section in it.
func AddSection(site *config.Site, kind string) error {
	if !slices.Contains(sectionKinds(site.Model), kind) {
		return errors.New("this site can't have a " + kind + " section")
	}
	if err := writeSection(site, kind); err != nil {
		return err
	}
	if site.Model != "bento" {
		if err := menuAdd(site, sectionFolder(site, kind)); err != nil {
			return err
		}
	}
	return saved(site)
}

// menuAdd puts folder in the home note's menu, when the site chose its own
// menu (without one, the theme lists every folder already).
func menuAdd(site *config.Site, folder string) error {
	if folder == "" {
		return nil
	}
	p := homeNotePath(site)
	note, err := obsidian.ReadNote(p)
	if err != nil {
		return nil
	}
	menu := note.List("menu")
	if len(menu) == 0 {
		return nil
	}
	for _, l := range menu {
		name, target, _ := strings.Cut(l, "|")
		if strings.EqualFold(strings.TrimSpace(target), folder) || strings.EqualFold(strings.TrimSpace(name), folder) {
			return nil
		}
	}
	return obsidian.SetProperties(p, obsidian.Prop{Key: "menu", Value: append(menu, folder)})
}

// SetShowName shows or hides the site's name next to the logo in the
// header (header_name on the home note; shown when the property is absent).
func SetShowName(site *config.Site, show bool) error {
	var v any // shown: no property, as on a site that never chose
	if !show {
		v = false
	}
	return obsidian.SetProperties(homeNotePath(site), obsidian.Prop{Key: "header_name", Value: v})
}

// CheckCookieTool tells whether value is a valid ID for a cookie tool
// (google_analytics, google_tag_manager, meta_pixel, umami, plausible).
func CheckCookieTool(key, value string) error {
	for _, t := range cookieTools {
		if t.key == key {
			if !t.valid.MatchString(strings.TrimSpace(value)) {
				return errors.New("invalid ID")
			}
			return nil
		}
	}
	return errors.New("unknown tool " + key)
}

// SaveCookies sets the tools and the banner. With privacyPage, the site's
// privacy policy note is created (from the tools) unless it exists; it
// returns that note, relative to the notes folder.
func SaveCookies(site *config.Site, c config.Cookies, privacyPage bool, contact string) (string, error) {
	for _, t := range cookieTools {
		v := t.value(&c)
		*v = strings.TrimSpace(*v)
		if *v != "" {
			if err := CheckCookieTool(t.key, *v); err != nil {
				return "", errors.New(t.key + ": " + err.Error())
			}
		}
	}
	for _, s := range []*string{&c.UmamiScript, &c.PlausibleScript, &c.Privacy} {
		if *s = strings.TrimSpace(*s); *s != "" {
			if err := validURL(*s); err != nil {
				return "", err
			}
		}
	}
	if privacyPage {
		c.Privacy = ""
	}
	site.Cookies = c
	note := ""
	if privacyPage && (c.Banner || c.Tools()) {
		p, err := writePrivacyNote(site, strings.TrimSpace(contact))
		if err != nil {
			return "", err
		}
		note, _ = filepath.Rel(site.ContentDir(), p)
		note = filepath.ToSlash(note)
	}
	return note, saved(site)
}

// PostPassword is the general password of a site's protected posts: its
// own, or else the one for every site.
func PostPassword(site *config.Site) string {
	if site.Password != "" {
		return site.Password
	}
	c, _ := config.LoadCredentials()
	return c.PostPassword
}

// HasGeneralPassword tells whether the password for every site is set.
func HasGeneralPassword() bool {
	c, _ := config.LoadCredentials()
	return c.PostPassword != ""
}

// SetGeneralPassword sets the password for every site ("" removes it).
func SetGeneralPassword(password string) error {
	c, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	c.PostPassword = password
	return c.Save()
}

// SetPassword sets the site's own general password of protected posts (""
// removes it: the one for every site counts).
func SetPassword(site *config.Site, password string) error {
	site.Password = password
	return saved(site)
}

// SetFeedback turns comments and reactions off, or back on for a site that
// already has its repository (turning them on the first time needs GitHub).
func SetFeedback(site *config.Site, on bool) error {
	if on && site.Feedback.Repo == "" {
		return errors.New("comments need to be set up with GitHub first")
	}
	site.Feedback.Enabled = on
	return saved(site)
}

// AttachmentFile is the file of an image property ("[[cover.jpg]]").
func AttachmentFile(site *config.Site, prop string) string { return attachmentFile(site, prop) }

// AddImage copies an image into the site's attachments folder, keeping its
// name, and returns the link to use in a property ("[[name.jpg]]").
func AddImage(site *config.Site, src string) (string, error) {
	if err := validImage(src); err != nil {
		return "", err
	}
	name := safeFileName(strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)))
	if name == "" {
		name = "image"
	}
	return copyAttachment(site, src, name)
}

// RemoveSection takes a Bento section off the site: its folder goes to the
// Blup trash of the site (.blup/trash, outside the notes) and the menu
// stops pointing to it. It returns the folder in the trash, for
// RestoreSection.
func RemoveSection(site *config.Site, kind string) (string, error) {
	folder, ok := existingSections(site)[kind]
	if !ok {
		return "", errors.New("the site has no " + kind + " section")
	}
	dst, err := toTrash(site, folder)
	if err != nil {
		return "", err
	}
	// The menu entry of the section would lead nowhere.
	note, err := obsidian.ReadNote(homeNotePath(site))
	if err != nil {
		return dst, err
	}
	var menu []string
	for _, l := range note.List("menu") {
		_, target, ok := strings.Cut(l, "|")
		if !ok {
			target = l
		}
		if !strings.EqualFold(strings.Trim(strings.TrimSpace(target), "/"), folder) {
			menu = append(menu, l)
		}
	}
	if len(menu) != len(note.List("menu")) {
		if err := obsidian.SetProperties(homeNotePath(site), listProp("menu", menu)); err != nil {
			return dst, err
		}
	}
	return dst, saved(site)
}

// RestoreSection brings back a section taken off with RemoveSection.
func RestoreSection(site *config.Site, trashed string) error {
	if !inTrash(site, trashed) {
		return errors.New("not in the Blup trash")
	}
	back := trashOrigin(trashed)
	if back == "" {
		back = filepath.Base(trashed)
	}
	dst := filepath.Join(site.ContentDir(), filepath.FromSlash(back))
	if _, err := os.Stat(dst); err == nil {
		return errors.New(i18n.T("trash.exists", back))
	}
	if err := moveDir(trashed, dst); err != nil {
		return err
	}
	os.RemoveAll(filepath.Dir(trashed)) // the dated folder and its .origin
	return saved(site)
}

// rename is os.Rename; tests make it fail as across disks.
var rename = os.Rename

// moveDir moves a folder, copying it when it goes to another disk.
func moveDir(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := rename(src, dst); err == nil {
		return nil
	}
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	})
	if err != nil {
		return err
	}
	return os.RemoveAll(src)
}

// The Blup trash of a site: folders taken off the site, kept outside the
// notes (one dated folder per removal) until restored or deleted.

func trashDir(site *config.Site) string { return filepath.Join(site.Dir(), ".blup", "trash") }

func inTrash(site *config.Site, p string) bool {
	rel, err := filepath.Rel(trashDir(site), p)
	return err == nil && rel != "." && !strings.HasPrefix(rel, "..") && strings.Count(filepath.ToSlash(rel), "/") == 1
}

// TrashItem is a folder in the Blup trash.
type TrashItem struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Removed string `json:"removed"` // when, "2006-01-02T15:04:05"
	Notes   int    `json:"notes"`
	Origin  string `json:"origin"` // where it was in the notes
	Dir     bool   `json:"dir"`
}

// ListTrash lists the trash, newest first.
func ListTrash(site *config.Site) ([]TrashItem, error) {
	dates, err := os.ReadDir(trashDir(site))
	if errors.Is(err, fs.ErrNotExist) {
		return []TrashItem{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []TrashItem{}
	for i := len(dates) - 1; i >= 0; i-- {
		d := dates[i]
		stamp, _, _ := strings.Cut(d.Name(), ".")
		when, err := time.ParseInLocation("2006-01-02-150405", stamp, time.Local)
		if !d.IsDir() || err != nil {
			continue
		}
		items, _ := os.ReadDir(filepath.Join(trashDir(site), d.Name()))
		for _, it := range items {
			if strings.HasPrefix(it.Name(), ".") {
				continue
			}
			p := filepath.Join(trashDir(site), d.Name(), it.Name())
			notes := 1
			if it.IsDir() {
				notes = obsidian.CountNotes(p)
			}
			out = append(out, TrashItem{Path: p, Name: strings.TrimSuffix(it.Name(), ".md"), Removed: when.Format("2006-01-02T15:04:05"),
				Notes: notes, Origin: firstOf(trashOrigin(p), it.Name()), Dir: it.IsDir()})
		}
	}
	return out, nil
}

// DeleteFromTrash deletes a folder of the trash for good.
func DeleteFromTrash(site *config.Site, p string) error {
	if !inTrash(site, p) {
		return errors.New("not in the Blup trash")
	}
	return os.RemoveAll(filepath.Dir(p)) // with its .origin
}

// EmptyTrash deletes everything in the trash.
func EmptyTrash(site *config.Site) error { return os.RemoveAll(trashDir(site)) }
