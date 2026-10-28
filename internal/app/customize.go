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
// button, social links, logo, photo and bio. Each change is saved at once.
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
// order shown, then add external links.
func editMenu(site *config.Site) error {
	type entry struct{ key, name, page string }
	entries := []entry{{"home", i18n.T("custom.menu_home"), "/"}}
	for _, f := range obsidian.Folders(site.ContentDir(), 1) {
		entries = append(entries, entry{"page:/" + f.Path, f.Path, "/" + f.Path})
	}
	entries = append(entries, entry{"tags", i18n.T("custom.menu_tags"), "/tags"})

	selected := []string{}
	for _, m := range site.Menu {
		if m.Page != "" {
			for _, e := range entries {
				if e.page == m.Page {
					selected = append(selected, e.key)
				}
			}
		}
	}
	if len(site.Menu) == 0 {
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

	var menu []config.MenuItem
	for _, e := range entries {
		if !slices.Contains(selected, e.key) {
			continue
		}
		name := e.name
		for _, m := range site.Menu {
			if m.Page == e.page && m.Name != "" {
				name = m.Name // keep a name the user changed before
			}
		}
		menu = append(menu, config.MenuItem{Name: name, Page: e.page})
	}
	// Keep previous external links unless the user removes them.
	for _, m := range site.Menu {
		if m.URL == "" {
			continue
		}
		keep, err := ui.Confirm(i18n.T("custom.menu_keep", m.Name), m.URL, true)
		if err != nil {
			return err
		}
		if keep {
			menu = append(menu, m)
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
		menu = append(menu, config.MenuItem{Name: strings.TrimSpace(name), URL: strings.TrimSpace(link)})
	}

	// Let the user rename the entries in one pass.
	for i := range menu {
		if err := ui.Ask(i18n.T("custom.menu_rename", i+1), i18n.T("custom.menu_rename_help"), menu[i].Name, &menu[i].Name, notEmpty); err != nil {
			return err
		}
	}
	site.Menu = menu
	return nil
}

func editSocial(site *config.Site) error {
	current := map[string]string{}
	for _, s := range site.Social {
		current[s.Network] = s.URL
	}
	var out []config.Social
	for _, n := range scaffold.SocialOrder {
		v := current[n]
		help := i18n.T("custom.social_help")
		check := validURL
		if n == "mail" {
			check = func(s string) error {
				if s != "" && !strings.Contains(s, "@") {
					return errors.New(i18n.T("custom.bad_email"))
				}
				return nil
			}
		}
		if err := ui.Ask(scaffold.SocialNames[n], help, "", &v, optional(check)); err != nil {
			return err
		}
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, config.Social{Network: n, URL: v})
		}
	}
	site.Social = out
	return nil
}

func editImages(site *config.Site) error {
	for _, it := range []struct {
		key  string
		dest *string
	}{{"custom.logo", &site.Logo}, {"custom.avatar", &site.Avatar}} {
		v := *it.dest
		if err := ui.Ask(i18n.T(it.key), i18n.T("custom.image_help"), "~/Imagens/foto.png", &v, optional(validImage)); err != nil {
			return err
		}
		if v = strings.TrimSpace(v); v != "" {
			v = expandHome(v)
		}
		*it.dest = v
	}
	return nil
}

func editAbout(site *config.Site) error {
	if err := ui.Ask(i18n.T("new.description"), i18n.T("new.optional"), "", &site.Description, nil); err != nil {
		return err
	}
	if err := ui.Ask(i18n.T("new.author"), i18n.T("new.optional"), "", &site.Author, nil); err != nil {
		return err
	}
	return ui.Ask(i18n.T("custom.bio"), i18n.T("new.optional"), "", &site.AuthorBio, nil)
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
	var add []string
	if err := ui.ChooseMany(i18n.T("custom.sections_q"), help, &add, missing...); err != nil {
		return err
	}
	for _, k := range add {
		if err := writeSection(site, k); err != nil {
			return err
		}
		ui.OK(i18n.T("custom.section_added", i18n.T("section_folder."+k)))
	}
	return nil
}
