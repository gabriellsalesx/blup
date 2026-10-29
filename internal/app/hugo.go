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
	"github.com/gabriellsalesx/blup/internal/lock"
	"github.com/gabriellsalesx/blup/internal/pagefind"
	"github.com/gabriellsalesx/blup/internal/pwa"
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
	runHugo := func() error {
		var log bytes.Buffer
		err := ui.Spin(ctx, i18n.T("build.running"), func(ctx context.Context) error {
			cmd := bin.Command(ctx, site.Dir(), "--gc", "--minify", "--cleanDestinationDir",
				"--environment", "production", "--baseURL", baseURL, "--destination", out, "--logLevel", "warn")
			cmd.Env = append(os.Environ(), "HUGO_BLUP_MODE=publish")
			cmd.Stdout, cmd.Stderr = &log, &log
			return cmd.Run()
		})
		warnings := hugoProblems(log.String())
		if err != nil {
			for _, w := range warnings {
				ui.Error(w)
			}
			return errors.New(i18n.T("build.failed"))
		}
		for _, w := range warnings {
			ui.Warn(w)
		}
		return nil
	}
	if err := runHugo(); err != nil {
		return "", err
	}
	// Feedback (Bento): open issues for new pages; when a page got one, build
	// again so the page shows it.
	newIssues, ferr := syncFeedback(ctx, site, out)
	if ferr != nil {
		ui.Warn(i18n.T("fb.failed", describe(ferr)))
	}
	if newIssues {
		if err := runHugo(); err != nil {
			return "", err
		}
	}
	os.Remove(filepath.Join(out, feedbackManifest))
	if err := encryptLocked(ctx, site, out); err != nil {
		// Never leave a build with protected posts in clear text around.
		os.RemoveAll(out)
		return "", err
	}
	ui.OK(i18n.T("build.done"))
	searchIndex(ctx, out)
	// Last: the offline copy lists every file of the finished site.
	if err := pwa.Write(out); err != nil {
		ui.Warn(i18n.T("pwa.failed", describe(err)))
	}
	return out, nil
}

// encryptLocked encrypts the password-protected notes of the build in dir,
// asking for the general password when a note uses it and none is set.
func encryptLocked(ctx context.Context, site *config.Site, dir string) error {
	posts, needsGlobal, err := lock.Pending(dir)
	if err != nil {
		return err
	}
	if posts == 0 {
		// The (empty) manifest is internal: never publish it.
		return lock.Discard(dir)
	}
	if needsGlobal && site.Password == "" {
		if !ui.Interactive() {
			return errors.New(i18n.T("lock.no_global"))
		}
		if err := ui.AskSecret(i18n.T("lock.global_q"), i18n.T("lock.global_help"), &site.Password); err != nil {
			return err
		}
		if err := site.Save(); err != nil {
			return err
		}
	}
	var n int
	err = ui.Spin(ctx, i18n.T("lock.encrypting"), func(context.Context) error {
		n, err = lock.Process(dir, site.Password)
		return err
	})
	if err != nil {
		return err
	}
	ui.OK(i18n.T("lock.done", n))
	return nil
}

// searchIndex builds the Pagefind index inside dir. Search is a nicety: a
// failure is reported but never blocks publishing.
func searchIndex(ctx context.Context, dir string) {
	if os.Getenv("BLUP_NO_SEARCH") != "" {
		return // e.g. in CI, or to skip downloading Pagefind
	}
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
