package app

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/hugo"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/pagefind"
	"github.com/gabriellsalesx/blup/internal/scaffold"
	"github.com/gabriellsalesx/blup/internal/ui"
)

// ensureHugo returns a usable Hugo, downloading it on first use.
func ensureHugo(ctx context.Context) (hugo.Binary, error) {
	if b, ok := hugo.Find(ctx); ok {
		return b, nil
	}
	var bin hugo.Binary
	err := ui.Spin(ctx, i18n.T("hugo.downloading", hugo.Pinned), func(ctx context.Context) error {
		var err error
		bin, err = hugo.Download(ctx, nil)
		return err
	})
	if errors.Is(err, hugo.ErrUnsupportedPlatform) {
		return bin, errors.New(i18n.T("hugo.install_manually"))
	}
	if err != nil {
		return bin, err
	}
	ui.OK(i18n.T("hugo.ready", bin.Version))
	return bin, nil
}

// build refreshes the generated files and renders the site into public/.
func build(ctx context.Context, site *config.Site, baseURL string) (string, error) {
	bin, err := ensureHugo(ctx)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(site.ContentDir()); err != nil {
		return "", errors.New(i18n.T("build.no_folder", site.ContentDir()))
	}
	if err := scaffold.Write(site); err != nil {
		return "", err
	}
	out := filepath.Join(site.Dir(), "public")
	var log bytes.Buffer
	err = ui.Spin(ctx, i18n.T("build.running"), func(ctx context.Context) error {
		cmd := bin.Command(ctx, site.Dir(), "--gc", "--minify", "--cleanDestinationDir",
			"--environment", "production", "--baseURL", baseURL, "--destination", out, "--logLevel", "warn")
		cmd.Stdout, cmd.Stderr = &log, &log
		return cmd.Run()
	})
	warnings := hugoProblems(log.String())
	if err != nil {
		for _, w := range warnings {
			ui.Error(w)
		}
		return "", errors.New(i18n.T("build.failed"))
	}
	for _, w := range warnings {
		ui.Warn(w)
	}
	ui.OK(i18n.T("build.done"))
	searchIndex(ctx, out)
	return out, nil
}

// searchIndex builds the Pagefind index inside dir. Search is a nicety: a
// failure is reported but never blocks publishing.
func searchIndex(ctx context.Context, dir string) {
	err := ui.Spin(ctx, i18n.T("build.search"), func(ctx context.Context) error {
		bin, err := pagefind.Ensure(ctx)
		if err != nil {
			return err
		}
		if out, err := pagefind.Index(ctx, bin, dir); err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	})
	if err != nil {
		ui.Warn(i18n.T("build.search_failed", err))
	}
}

// hugoProblems keeps only Hugo's WARN/ERROR lines, without the prefix.
func hugoProblems(log string) []string {
	var out []string
	sc := bufio.NewScanner(strings.NewReader(log))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		for _, p := range []string{"WARN", "ERROR"} {
			if rest, ok := strings.CutPrefix(line, p); ok {
				out = append(out, strings.TrimSpace(rest))
			}
		}
	}
	return out
}

func siteURL(site *config.Site, fallback string) string {
	if site.Domain != "" {
		return fmt.Sprintf("https://%s/", site.Domain)
	}
	return fallback
}
