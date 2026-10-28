// Package app holds the CLI flows: the main menu, the new-site wizard,
// preview and publish. Every flow is interactive and translated.
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"charm.land/huh/v2"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/ui"
)

// Version is set at build time with -ldflags "-X ...app.Version=v1.2.3".
var Version = "dev"

// Run dispatches the command line. With no arguments it opens the menu.
func Run(ctx context.Context, args []string) error {
	g, err := config.LoadGlobal()
	if err != nil {
		return err
	}
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "version", "--version", "-v":
		fmt.Println("blup", Version)
		return nil
	case "help", "--help", "-h":
		chooseLanguageIfNeeded(&g)
		fmt.Println(i18n.T("help"))
		return nil
	}
	if cmd == "build" || cmd == "gerar" || !ui.Interactive() {
		// No questions when scripted: the saved language, else English.
		i18n.Set(i18n.EN)
		if g.Lang != "" {
			i18n.Set(i18n.Lang(g.Lang))
		}
	} else if err := chooseLanguageIfNeeded(&g); err != nil {
		return err
	}
	switch cmd {
	case "":
		return menu(ctx, &g)
	case "new", "novo":
		return newSite(ctx, &g)
	case "preview", "ver":
		return withSite(ctx, &g, args[1:], preview)
	case "publish", "publicar":
		return withSite(ctx, &g, args[1:], publish)
	case "build", "gerar":
		// Non-interactive build into <site>/public, for scripts and CI.
		return withSite(ctx, &g, args[1:], func(ctx context.Context, _ *config.Global, s *config.Site) error {
			_, err := build(ctx, s, siteURL(s, "/"))
			return err
		})
	case "convert", "converter":
		return withSite(ctx, &g, args[1:], convertLinks)
	case "customize", "personalizar":
		return withSite(ctx, &g, args[1:], customize)
	case "sync", "sincronizar":
		return withSite(ctx, &g, args[1:], syncNotes)
	case "language", "idioma":
		return chooseLanguage(&g)
	case "update", "atualizar":
		return updateCommand(ctx, args[1:])
	default:
		fmt.Println(i18n.T("help"))
		return fmt.Errorf("%s: %s", i18n.T("unknown_command"), cmd)
	}
}

func chooseLanguageIfNeeded(g *config.Global) error {
	if g.Lang != "" {
		i18n.Set(i18n.Lang(g.Lang))
		return nil
	}
	i18n.Set(i18n.EN) // English is the default
	return chooseLanguage(g)
}

func chooseLanguage(g *config.Global) error {
	lang := i18n.Current()
	if err := ui.Choose("Language / Idioma", "", &lang,
		ui.Option("English", i18n.EN),
		ui.Option("Português", i18n.PT),
	); err != nil {
		return err
	}
	i18n.Set(lang)
	g.Lang = string(lang)
	return g.Save()
}

func menu(ctx context.Context, g *config.Global) error {
	ui.Banner("Blup · " + i18n.T("tagline"))
	if offerUpdate(ctx) {
		return nil // the new version runs on the next start
	}
	sites := g.ExistingSites()
	if len(sites) == 0 {
		ui.Hint(i18n.T("menu.first_time"))
		return newSite(ctx, g)
	}
	for {
		site, err := loadSite(sites[0])
		if err != nil {
			return err
		}
		action := "preview"
		opts := []huh.Option[string]{
			ui.Option(i18n.T("menu.preview", site.Title), "preview"),
			ui.Option(i18n.T("menu.publish", site.Title), "publish"),
			ui.Option(i18n.T("menu.customize"), "customize"),
			ui.Option(i18n.T("menu.sync"), "sync"),
			ui.Option(i18n.T("menu.convert"), "convert"),
		}
		if len(sites) > 1 {
			opts = append(opts, ui.Option(i18n.T("menu.switch"), "switch"))
		}
		opts = append(opts,
			ui.Option(i18n.T("menu.new"), "new"),
			ui.Option(i18n.T("menu.language"), "language"),
			ui.Option(i18n.T("menu.exit"), "exit"),
		)
		if err := ui.Choose(i18n.T("menu.question"), i18n.T("menu.current", site.Title, i18n.T("model."+site.Model), site.Dir()), &action, opts...); err != nil {
			return err
		}
		switch action {
		case "preview", "ver":
			err = preview(ctx, g, site)
		case "publish", "publicar":
			err = publish(ctx, g, site)
		case "customize", "personalizar":
			err = customize(ctx, g, site)
		case "sync":
			err = syncNotes(ctx, g, site)
		case "convert", "converter":
			err = convertLinks(ctx, g, site)
		case "switch":
			var dir string
			if dir, err = pickSite(sites); err == nil {
				g.AddSite(dir)
				err = g.Save()
				sites = g.ExistingSites()
			}
		case "new", "novo":
			err = newSite(ctx, g)
			sites = g.ExistingSites()
		case "language", "idioma":
			err = chooseLanguage(g)
		case "exit":
			return nil
		}
		if err != nil && !ui.Aborted(err) {
			ui.Error(describe(err))
		}
	}
}

// withSite resolves the site for a subcommand: an explicit folder, the
// current folder, the only known site, or a choice among known sites.
func withSite(ctx context.Context, g *config.Global, args []string, fn func(context.Context, *config.Global, *config.Site) error) error {
	dir := ""
	switch {
	case len(args) > 0:
		dir, _ = filepath.Abs(args[0])
	case fileExists(config.SiteFile):
		dir, _ = os.Getwd()
	default:
		sites := g.ExistingSites()
		switch len(sites) {
		case 0:
			ui.Warn(i18n.T("no_site"))
			return newSite(ctx, g)
		case 1:
			dir = sites[0]
		default:
			var err error
			if dir, err = pickSite(sites); err != nil {
				return err
			}
		}
	}
	site, err := loadSite(dir)
	if err != nil {
		return err
	}
	g.AddSite(dir)
	_ = g.Save()
	return fn(ctx, g, site)
}

func pickSite(sites []string) (string, error) {
	dir := sites[0]
	var opts []huh.Option[string]
	for _, s := range sites {
		label := s
		if site, err := config.LoadSite(s); err == nil {
			label = fmt.Sprintf("%s  (%s)", site.Title, s)
		}
		opts = append(opts, ui.Option(label, s))
	}
	return dir, ui.Choose(i18n.T("pick_site"), "", &dir, opts...)
}

func loadSite(dir string) (*config.Site, error) {
	s, err := config.LoadSite(dir)
	if errors.Is(err, config.ErrNewerSite) {
		return nil, errors.New(i18n.T("site.newer"))
	}
	var unknown config.UnknownModelError
	if errors.As(err, &unknown) {
		return nil, errors.New(i18n.T("site.unknown_model", unknown.Model))
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("site_unreadable", dir), err)
	}
	return s, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// Slug turns a title into a lowercase a-z0-9- name for folders, repos and
// projects ("Cozinha da Vó" → "cozinha-da-vo").
func Slug(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ = transform.String(t, strings.ToLower(s))
	var b strings.Builder
	dash := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 50 {
		out = strings.Trim(out[:50], "-")
	}
	if out == "" {
		out = "site"
	}
	return out
}

// describe turns an error into a friendly, translated sentence when we
// know it; otherwise it returns the raw message.
func describe(err error) string {
	for _, d := range knownErrors {
		if errors.Is(err, d.err) {
			return i18n.T(d.key)
		}
	}
	return err.Error()
}

type knownError struct {
	err error
	key string
}

var knownErrors []knownError

func registerErrors(list ...knownError) { knownErrors = append(knownErrors, list...) }
