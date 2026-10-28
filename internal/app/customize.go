package app

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"charm.land/huh/v2"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/gabriellsalesx/blup/internal/scaffold"
	"github.com/gabriellsalesx/blup/internal/ui"
)

// customize edits what the site shows besides the notes: top menu, header
// button, social links, logo, favicon and the author. They are properties of
// notes in Obsidian, so each change is saved there at once.
func customize(ctx context.Context, _ *config.Global, site *config.Site) error {
	for {
		choice := "menu"
		opts := []huh.Option[string]{ui.Option(i18n.T("custom.menu"), "menu")}
		if site.Model == "bento" {
			opts = append(opts, ui.Option(i18n.T("custom.sections"), "sections"))
		}
		if err := ui.Choose(i18n.T("custom.q"), i18n.T("custom.help"), &choice, append(opts,
			ui.Option(i18n.T("custom.social"), "social"),
			ui.Option(i18n.T("custom.images"), "images"),
			ui.Option(i18n.T("custom.about"), "about"),
			ui.Option(i18n.T("custom.domain"), "domain"),
			ui.Option(i18n.T("custom.password"), "password"),
			ui.Option(i18n.T("custom.back"), "back"),
		)...); err != nil || choice == "back" {
			return err
		}
		var err error
		switch choice {
		case "menu":
			err = editMenu(site)
		case "sections":
			err = editSections(site)
		case "social":
			err = editSocial(site)
		case "images":
			err = editImages(site)
		case "about":
			err = editAbout(site)
		case "domain":
			err = editDomain(ctx, site)
		case "password":
			err = editPassword(site)
		}
		if err != nil {
			if ui.Aborted(err) {
				continue
			}
			return err
		}
		if err := site.Save(); err != nil {
			return err
		}
		if err := scaffold.Write(site); err != nil {
			return err
		}
		ui.OK(i18n.T("custom.saved"))
	}
}

// editMenu lets the user tick which pages appear in the top menu, in the
// order shown, then add external links and the button at the top right.
// Everything is saved as properties of the home note (menu, button_text,
// button_link), which can also be edited in Obsidian.
func editMenu(site *config.Site) error {
	home := homeNotePath(site)
	note, err := obsidian.ReadNote(home)
	if err != nil {
		return err
	}
	type item struct{ name, target string }
	parse := func(line string) item {
		name, target, ok := strings.Cut(line, "|")
		if !ok {
			target = name
		}
		return item{strings.TrimSpace(name), strings.TrimSpace(target)}
	}
	var current []item
	for _, l := range note.List("menu") {
		current = append(current, parse(l))
	}

	type entry struct{ key, name, target string }
	entries := []entry{{"home", i18n.T("custom.menu_home"), "/"}}
	authors := obsidian.Sections(site.ContentDir())["authors"]
	for _, f := range obsidian.Folders(site.ContentDir(), 1) {
		if f.Path != authors && f.Notes > 0 {
			entries = append(entries, entry{"page:" + f.Path, f.Path, f.Path})
		}
	}
	entries = append(entries, entry{"tags", i18n.T("custom.menu_tags"), "/tags/"})
	same := func(e entry, it item) bool {
		t := strings.ToLower(strings.Trim(it.target, "/"))
		switch e.key {
		case "home":
			return it.target == "/" || t == "home" || t == "início" || t == "inicio"
		case "tags":
			return t == "tags"
		}
		return t == strings.ToLower(e.target)
	}
	external := func(it item) bool {
		return strings.HasPrefix(it.target, "http://") || strings.HasPrefix(it.target, "https://") || strings.HasPrefix(it.target, "mailto:")
	}

	selected := []string{}
	for _, e := range entries {
		for _, it := range current {
			if same(e, it) {
				selected = append(selected, e.key)
			}
		}
	}
	if len(current) == 0 {
		for _, e := range entries {
			if e.key != "tags" {
				selected = append(selected, e.key)
			}
		}
	}
	opts := make([]huh.Option[string], 0, len(entries))
	for _, e := range entries {
		opts = append(opts, ui.Option(e.name, e.key))
	}
	if err := ui.ChooseMany(i18n.T("custom.menu_q"), i18n.T("custom.menu_help"), &selected, opts...); err != nil {
		return err
	}

	var menu []item
	for _, e := range entries {
		if !slices.Contains(selected, e.key) {
			continue
		}
		it := item{e.name, e.target}
		for _, c := range current {
			if same(e, c) && c.name != "" {
				it.name = c.name // keep a name the user changed before
			}
		}
		menu = append(menu, it)
	}
	// Keep previous external links unless the user removes them.
	for _, c := range current {
		if !external(c) {
			continue
		}
		keep, err := ui.Confirm(i18n.T("custom.menu_keep", c.name), c.target, true)
		if err != nil {
			return err
		}
		if keep {
			menu = append(menu, c)
		}
	}
	for {
		more, err := ui.Confirm(i18n.T("custom.menu_add"), i18n.T("custom.menu_add_help"), false)
		if err != nil {
			return err
		}
		if !more {
			break
		}
		var name, link string
		if err := ui.Ask(i18n.T("custom.link_name"), "", "", &name, notEmpty); err != nil {
			return err
		}
		if err := ui.Ask(i18n.T("custom.link_url"), "", "https://", &link, validURL); err != nil {
			return err
		}
		menu = append(menu, item{strings.TrimSpace(name), strings.TrimSpace(link)})
	}

	// Let the user rename the entries in one pass.
	lines := make([]string, 0, len(menu))
	for i := range menu {
		if err := ui.Ask(i18n.T("custom.menu_rename", i+1), i18n.T("custom.menu_rename_help"), menu[i].name, &menu[i].name, notEmpty); err != nil {
			return err
		}
		lines = append(lines, strings.TrimSpace(menu[i].name)+" | "+menu[i].target)
	}

	text, link := note.String("button_text"), note.String("button_link")
	if err := ui.Ask(i18n.T("custom.button_text"), i18n.T("custom.button_text_help"), "", &text, nil); err != nil {
		return err
	}
	if strings.TrimSpace(text) != "" {
		if err := ui.Ask(i18n.T("custom.button_link"), "", "https://", &link, validURL); err != nil {
			return err
		}
	} else {
		link = ""
	}
	return obsidian.SetProperties(home, listProp("menu", lines), textProp("button_text", text), textProp("button_link", link))
}

// socialOrder is the order networks are asked; the footer finds each
// network's icon from the address itself.
var socialOrder = []struct{ name, match string }{
	{"Instagram", "instagram.com"}, {"X", "x.com"}, {"LinkedIn", "linkedin.com"},
	{"YouTube", "youtube.com"}, {"GitHub", "github.com"}, {"Facebook", "facebook.com"},
	{"WhatsApp", "wa.me"}, {"E-mail", "mailto:"},
}

func socialNetwork(url string) int {
	u := strings.ToLower(url)
	for i, n := range socialOrder {
		if strings.Contains(u, n.match) || (n.name == "X" && strings.Contains(u, "twitter.com")) ||
			(n.name == "YouTube" && strings.Contains(u, "youtu.be")) ||
			(n.name == "WhatsApp" && strings.Contains(u, "whatsapp.com")) ||
			(n.name == "E-mail" && !strings.Contains(u, "://") && strings.Contains(u, "@")) {
			return i
		}
	}
	return -1
}

// editSocial asks one address per network and saves the "social" property
// of the home note. Addresses of other sites already there are kept.
func editSocial(site *config.Site) error {
	home := homeNotePath(site)
	note, err := obsidian.ReadNote(home)
	if err != nil {
		return err
	}
	current := make([]string, len(socialOrder))
	var others []string
	for _, u := range note.List("social") {
		if i := socialNetwork(u); i >= 0 && current[i] == "" {
			current[i] = strings.TrimPrefix(u, "mailto:")
		} else {
			others = append(others, u)
		}
	}
	var out []string
	for i, n := range socialOrder {
		v := current[i]
		check := validURL
		if n.name == "E-mail" {
			check = func(s string) error {
				if !strings.Contains(s, "@") {
					return errors.New(i18n.T("custom.bad_email"))
				}
				return nil
			}
		}
		if err := ui.Ask(n.name, i18n.T("custom.social_help"), "", &v, optional(check)); err != nil {
			return err
		}
		if v = strings.TrimSpace(v); v != "" {
			if n.name == "E-mail" {
				v = "mailto:" + v
			}
			out = append(out, v)
		}
	}
	return obsidian.SetProperties(home, listProp("social", append(out, others...)))
}

// editImages copies the logo and favicon into the site's attachments
// folder and points the home note's properties at them.
func editImages(site *config.Site) error {
	home := homeNotePath(site)
	for _, key := range []string{"logo", "favicon"} {
		v := ""
		if err := ui.Ask(i18n.T("custom."+key), i18n.T("custom.image_help"), "~/Imagens/"+key+".png", &v, optional(validImage)); err != nil {
			return err
		}
		if v = strings.TrimSpace(v); v == "" {
			continue
		}
		link, err := copyAttachment(site, expandHome(v), key)
		if err != nil {
			return err
		}
		if err := obsidian.SetProperties(home, obsidian.Prop{Key: key, Value: link}); err != nil {
			return err
		}
	}
	return nil
}

// editAbout changes the site's name and description (home note) and the
// author: name (blup.toml, the default author of the posts) and the
// author's note with role, photo and bio.
func editAbout(site *config.Site) error {
	home := homeNotePath(site)
	note, err := obsidian.ReadNote(home)
	if err != nil {
		return err
	}
	name := note.String("site_name")
	if name == "" {
		name = site.Title
	}
	desc := note.String("site_description")
	if desc == "" {
		desc = site.Description
	}
	if err := ui.Ask(i18n.T("new.site_title"), i18n.T("new.site_title_help"), "", &name, notEmpty); err != nil {
		return err
	}
	if err := ui.Ask(i18n.T("new.description"), i18n.T("new.optional"), "", &desc, nil); err != nil {
		return err
	}
	site.Title, site.Description = strings.TrimSpace(name), strings.TrimSpace(desc)
	if err := obsidian.SetProperties(home, textProp("site_name", site.Title), textProp("site_description", site.Description)); err != nil {
		return err
	}

	if err := ui.Ask(i18n.T("new.author"), i18n.T("custom.author_help"), "", &site.Author, nil); err != nil {
		return err
	}
	if site.Author = strings.TrimSpace(site.Author); site.Author == "" {
		return nil
	}
	p, err := authorNotePath(site, site.Author)
	if err != nil {
		return err
	}
	author, err := obsidian.ReadNote(p)
	if err != nil {
		return err
	}
	role, bio, photo := author.String("role"), strings.TrimSpace(author.Body), ""
	if err := ui.Ask(i18n.T("custom.role"), i18n.T("new.optional"), "", &role, nil); err != nil {
		return err
	}
	if err := ui.Ask(i18n.T("custom.bio"), i18n.T("new.optional"), "", &bio, nil); err != nil {
		return err
	}
	if err := ui.Ask(i18n.T("custom.avatar"), i18n.T("custom.image_help"), "~/Imagens/foto.png", &photo, optional(validImage)); err != nil {
		return err
	}
	props := []obsidian.Prop{{Key: "role", Value: strings.TrimSpace(role)}}
	if photo = strings.TrimSpace(photo); photo != "" {
		link, err := copyAttachment(site, expandHome(photo), safeFileName(site.Author))
		if err != nil {
			return err
		}
		props = append(props, obsidian.Prop{Key: "photo", Value: link})
	}
	if err := obsidian.SetProperties(p, props...); err != nil {
		return err
	}
	return obsidian.SetBody(p, strings.TrimSpace(bio)+"\n")
}

func validURL(s string) error {
	s = strings.TrimSpace(s)
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return errors.New(i18n.T("custom.bad_url"))
	}
	return nil
}

func validImage(s string) error {
	p := expandHome(s)
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return errors.New(i18n.T("custom.bad_file"))
	}
	switch strings.ToLower(filepath.Ext(p)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg":
		return nil
	}
	return errors.New(i18n.T("custom.bad_image"))
}

// optional accepts an empty answer, otherwise applies check.
func optional(check func(string) error) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return check(s)
	}
}

// editPassword sets the general password used by notes with "password: sim".
func editPassword(site *config.Site) error {
	if err := ui.Explain(i18n.T("custom.password"), i18n.T("lock.explain")); err != nil {
		return err
	}
	pw := ""
	if err := ui.AskSecret(i18n.T("lock.global_q"), i18n.T("lock.global_help"), &pw); err != nil {
		return err
	}
	site.Password = pw
	return nil
}

// editSections adds the docs-model sections the site does not have yet.
// Removing one is done in Obsidian by deleting its folder.
func editSections(site *config.Site) error {
	have := existingSections(site)
	var missing []huh.Option[string]
	var present []string
	for _, k := range docsKinds {
		if folder, ok := have[k]; ok {
			present = append(present, folder)
			continue
		}
		missing = append(missing, ui.Option(i18n.T("new.section_"+k), k))
	}
	help := i18n.T("custom.sections_help", strings.Join(present, ", "))
	if len(missing) == 0 {
		return ui.Explain(i18n.T("custom.sections"), help+"\n\n"+i18n.T("custom.sections_all"))
	}
	if err := ui.Explain(i18n.T("new.sections_about_title"), i18n.T("new.sections_about")); err != nil {
		return err
	}
	var add []string
	if err := ui.ChooseMany(i18n.T("custom.sections_q"), help, &add, missing...); err != nil {
		return err
	}
	for _, k := range add {
		if err := writeSection(site, k); err != nil {
			return err
		}
		ui.OK(i18n.T("custom.section_added", sectionFolder(site, k)))
	}
	return nil
}
