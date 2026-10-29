package app

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/gabriellsalesx/blup/internal/scaffold"
)

// Settings without questions: what the customize questions change, read and
// saved from values a form already has. Blup Studio uses these; each save
// writes the site's files at once, like customize does.

// Settings is what the site shows besides the notes.
type Settings struct {
	Model        string         `json:"model"`
	Lang         string         `json:"lang"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Author       string         `json:"author"`
	Logo         string         `json:"logo"`    // file of the logo, "" for none
	Favicon      string         `json:"favicon"` // file of the favicon, "" for none
	Menu         []MenuItem     `json:"menu"`
	MenuOptions  []MenuItem     `json:"menuOptions"` // pages that can go in the menu
	ButtonText   string         `json:"buttonText"`
	ButtonLink   string         `json:"buttonLink"`
	Social       []string       `json:"social"`
	Sections     []Section      `json:"sections"` // Bento only
	Cookies      config.Cookies `json:"cookies"`
	PrivacyNote  string         `json:"privacyNote"` // the policy note, relative to the notes folder
	Feedback     bool           `json:"feedback"`
	FeedbackRepo string         `json:"feedbackRepo"`
	HasPassword  bool           `json:"hasPassword"`
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
		Feedback: site.Feedback.On(), HasPassword: site.Password != "",
		Logo: attachmentFile(site, note.String("logo")), Favicon: attachmentFile(site, note.String("favicon")),
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
	if site.Model == "bento" {
		have := existingSections(site)
		for _, k := range docsKinds {
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

// AddSection creates a Bento section with its example notes.
func AddSection(site *config.Site, kind string) error {
	if site.Model != "bento" {
		return errors.New("sections are part of Bento")
	}
	if err := writeSection(site, kind); err != nil {
		return err
	}
	return saved(site)
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

// SetPassword sets the general password of protected posts ("" removes it).
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
