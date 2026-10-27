package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/gabriellsalesx/blup/internal/scaffold"
	"github.com/gabriellsalesx/blup/internal/ui"
)

const otherChoice = "\x00other"

// newSite is the wizard: vault → folder → model → names → where to save →
// adjustments in Obsidian → files. Each question has a sensible default so
// pressing Enter all the way through works.
func newSite(ctx context.Context, g *config.Global) error {
	ui.Step(i18n.T("new.title"))

	vault, err := askVault()
	if err != nil {
		return err
	}
	folder, err := askFolder(vault)
	if err != nil {
		return err
	}

	model := "blog"
	if err := ui.Choose(i18n.T("new.model"), "", &model,
		ui.Option(i18n.T("new.model_blog"), "blog"),
		ui.Option(i18n.T("new.model_docs"), "docs"),
	); err != nil {
		return err
	}

	title := filepath.Base(filepath.Join(vault, folder))
	if err := ui.Ask(i18n.T("new.site_title"), i18n.T("new.site_title_help"), title, &title, notEmpty); err != nil {
		return err
	}
	desc := ""
	if err := ui.Ask(i18n.T("new.description"), i18n.T("new.optional"), i18n.T("new.description_ph"), &desc, nil); err != nil {
		return err
	}
	author := ""
	if err := ui.Ask(i18n.T("new.author"), i18n.T("new.optional"), "", &author, nil); err != nil {
		return err
	}
	siteLang := string(i18n.Current())
	if err := ui.Choose(i18n.T("new.site_lang"), i18n.T("new.site_lang_help"), &siteLang,
		ui.Option("Português", "pt"), ui.Option("English", "en"),
	); err != nil {
		return err
	}

	dir, err := askSiteDir(title)
	if err != nil {
		return err
	}

	content := filepath.Join(vault, folder)
	where := i18n.T("new.whole_vault")
	if folder != "" {
		where = folder
	}
	summary := i18n.T("new.summary", title, i18n.T("model."+model), vault, where, dir)
	if ok, err := ui.Confirm(i18n.T("new.confirm"), summary, true); err != nil || !ok {
		if err == nil {
			ui.Hint(i18n.T("cancelled"))
		}
		return err
	}

	// From here on, nothing else is asked except the Obsidian adjustments.
	site := config.NewSite(dir)
	site.Title, site.Description, site.Author = title, desc, author
	site.Lang, site.Model, site.Vault, site.Folder = siteLang, model, vault, folder

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := adjustObsidian(site); err != nil {
		return err
	}
	if obsidian.CountNotes(content) == 0 {
		if err := writeSamples(site); err != nil {
			return err
		}
		ui.OK(i18n.T("new.samples"))
	} else if err := ensureHome(site); err != nil {
		return err
	}
	if err := writeTemplates(site); err != nil {
		ui.Warn(i18n.T("new.templates_failed", err))
	}
	if err := scaffold.Write(site); err != nil {
		return err
	}
	if err := site.Save(); err != nil {
		return err
	}
	g.AddSite(dir)
	if err := g.Save(); err != nil {
		return err
	}
	ui.OK(i18n.T("new.created", dir))

	if err := convertLinksIfAny(ctx, site); err != nil && !ui.Aborted(err) {
		ui.Warn(describe(err))
	}
	if _, err := ensureHugo(ctx); err != nil {
		return err
	}

	ui.Explain(i18n.T("new.done_title"), i18n.T("new.done_body", content))
	if ok, _ := ui.Confirm(i18n.T("new.preview_now"), "", true); ok {
		return preview(ctx, g, site)
	}
	return nil
}

func askVault() (string, error) {
	vaults := obsidian.Vaults()
	choice := otherChoice
	if len(vaults) > 0 {
		choice = vaults[0].Path
		var opts []huh.Option[string]
		for _, v := range vaults {
			opts = append(opts, ui.Option(fmt.Sprintf("%s  %s", v.Name, ui.Highlight(v.Path)), v.Path))
		}
		opts = append(opts, ui.Option(i18n.T("new.vault_other"), otherChoice))
		if err := ui.Choose(i18n.T("new.vault"), i18n.T("new.vault_help"), &choice, opts...); err != nil {
			return "", err
		}
	} else {
		ui.Hint(i18n.T("new.vault_none"))
	}
	if choice != otherChoice {
		return choice, nil
	}
	path := ""
	err := ui.Ask(i18n.T("new.vault_path"), i18n.T("new.vault_path_help"), "~/Documents/My Vault", &path, func(s string) error {
		p := expandHome(s)
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			return errors.New(i18n.T("new.vault_missing"))
		}
		if !obsidian.IsVault(p) {
			return errors.New(i18n.T("new.vault_not_vault"))
		}
		return nil
	})
	return expandHome(path), err
}

func askFolder(vault string) (string, error) {
	mode := "folder"
	if err := ui.Choose(i18n.T("new.scope"), i18n.T("new.scope_help"), &mode,
		ui.Option(i18n.T("new.scope_folder"), "folder"),
		ui.Option(i18n.T("new.scope_vault", obsidian.CountNotes(vault)), "vault"),
	); err != nil || mode == "vault" {
		return "", err
	}

	folders := obsidian.Folders(vault, 2)
	choice := otherChoice
	var opts []huh.Option[string]
	opts = append(opts, ui.Option(i18n.T("new.folder_create"), otherChoice))
	for _, f := range folders {
		opts = append(opts, ui.Option(i18n.T("new.folder_item", f.Path, f.Notes), f.Path))
	}
	if err := ui.Choose(i18n.T("new.folder"), i18n.T("new.folder_help"), &choice, opts...); err != nil {
		return "", err
	}
	if choice != otherChoice {
		return choice, nil
	}
	name := i18n.T("new.folder_default")
	err := ui.Ask(i18n.T("new.folder_name"), i18n.T("new.folder_name_help"), name, &name, func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, ".") || strings.Contains(s, "..") {
			return errors.New(i18n.T("new.folder_invalid"))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	name = strings.Trim(filepath.ToSlash(strings.TrimSpace(name)), "/")
	return name, os.MkdirAll(filepath.Join(vault, filepath.FromSlash(name)), 0o755)
}

func askSiteDir(title string) (string, error) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, i18n.T("new.sites_folder"), Slug(title))
	err := ui.Ask(i18n.T("new.site_dir"), i18n.T("new.site_dir_help"), dir, &dir, func(s string) error {
		p := expandHome(s)
		if fileExists(filepath.Join(p, config.SiteFile)) {
			return errors.New(i18n.T("new.site_dir_exists"))
		}
		if entries, err := os.ReadDir(p); err == nil && len(entries) > 0 {
			return errors.New(i18n.T("new.site_dir_not_empty"))
		}
		return nil
	})
	return expandHome(dir), err
}

// adjustObsidian switches the vault to standard relative Markdown links,
// after explaining why. Declining is allowed: wikilinks can be converted
// later from the menu.
func adjustObsidian(site *config.Site) error {
	s, err := obsidian.ReadLinkSettings(site.Vault)
	if err != nil {
		return err
	}
	if s.UsesHugoFriendlyLinks() {
		return nil
	}
	ok, err := ui.Confirm(i18n.T("obs.links_q"), i18n.T("obs.links_help"), true)
	if err != nil || !ok {
		return err
	}
	if err := obsidian.EnableHugoFriendlyLinks(site.Vault); err != nil {
		return err
	}
	ui.OK(i18n.T("obs.links_done"))
	return nil
}

func notEmpty(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New(i18n.T("required"))
	}
	return nil
}

func expandHome(p string) string {
	p = strings.TrimSpace(p)
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		p = filepath.Join(home, strings.TrimPrefix(p, "~"))
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}
