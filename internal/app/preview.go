package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/hugo"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/pagefind"
	"github.com/gabriellsalesx/blup/internal/scaffold"
	"github.com/gabriellsalesx/blup/internal/ui"
)

var serverURL = regexp.MustCompile(`https?://(localhost|127\.0\.0\.1):\d+/\S*`)

// preview runs "hugo server" so every saved note shows up in the browser
// immediately (drafts included). Ctrl+C stops it and returns to the menu.
// It renders to memory only: protected notes are shown in clear text in the
// preview, and that must never land in public/.
func preview(ctx context.Context, _ *config.Global, site *config.Site) error {
	bin, err := ensureHugo(ctx)
	if err != nil {
		return err
	}
	if err := scaffold.Write(site); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	previewSearch(ctx, bin, site)

	cmd := bin.Command(ctx, site.Dir(), "server", "--config", scaffold.PreviewConfig, "--renderToMemory", "--buildDrafts", "--navigateToChanged",
		"--bind", "127.0.0.1", "--baseURL", "http://localhost/", "--logLevel", "warn")
	cmd.Env = append(os.Environ(), "HUGO_BLUP_MODE=preview")
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait(); pw.Close() }()

	ui.Step(i18n.T("preview.starting"))
	opened := false
	sc := bufio.NewScanner(pr)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case !opened && serverURL.MatchString(line):
			url := serverURL.FindString(line)
			opened = true
			ui.OK(i18n.T("preview.ready", ui.Highlight(url)))
			ui.Hint(i18n.T("preview.hint"))
			ui.OpenBrowser(url)
		default:
			for _, p := range hugoProblems(line) {
				ui.Warn(p)
			}
		}
	}
	err = <-done
	if ctx.Err() != nil {
		ui.Hint(i18n.T("preview.stopped"))
		return nil
	}
	return err
}

// previewSearch builds the site once into a temporary folder and indexes it,
// so search works in the preview (with the notes as they were at start).
func previewSearch(ctx context.Context, bin hugo.Binary, site *config.Site) {
	tmp, err := os.MkdirTemp("", "blup-preview-")
	if err != nil {
		return
	}
	defer os.RemoveAll(tmp)
	err = ui.Spin(ctx, i18n.T("preview.indexing"), func(ctx context.Context) error {
		cmd := bin.Command(ctx, site.Dir(), "--buildDrafts", "--baseURL", "http://localhost/", "--destination", tmp, "--quiet")
		cmd.Env = append(os.Environ(), "HUGO_BLUP_MODE=preview")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%w: %s", err, out)
		}
		pf, err := pagefind.Ensure(ctx)
		if err != nil {
			return err
		}
		if out, err := pagefind.Index(ctx, pf, tmp); err != nil {
			return fmt.Errorf("%w: %s", err, out)
		}
		dest := filepath.Join(site.Dir(), filepath.FromSlash(scaffold.PreviewStatic), "pagefind")
		if err := os.RemoveAll(dest); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.Rename(filepath.Join(tmp, "pagefind"), dest)
	})
	if err != nil {
		ui.Warn(i18n.T("build.search_failed", err))
	}
}
