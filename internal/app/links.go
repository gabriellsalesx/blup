package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/ui"
	"github.com/gabriellsalesx/blup/internal/wikilinks"
)

// convertLinks is the menu entry: it always reports, even when there is
// nothing to convert.
func convertLinks(ctx context.Context, _ *config.Global, site *config.Site) error {
	n, err := convertLinksFlow(ctx, site)
	if err == nil && n == 0 {
		ui.OK(i18n.T("links.none"))
	}
	return err
}

// convertLinksIfAny runs during site creation and stays silent when the
// notes have no wikilinks.
func convertLinksIfAny(ctx context.Context, site *config.Site) error {
	_, err := convertLinksFlow(ctx, site)
	return err
}

// convertLinksFlow does a dry run, shows what would change, backs up the
// affected notes into the site folder and rewrites them.
func convertLinksFlow(ctx context.Context, site *config.Site) (int, error) {
	var idx *wikilinks.Index
	var rep wikilinks.Report
	err := ui.Spin(ctx, i18n.T("links.scanning"), func(context.Context) error {
		var err error
		if idx, err = wikilinks.NewIndex(site.Vault); err != nil {
			return err
		}
		rep, err = wikilinks.ConvertTree(idx, site.Vault, site.Folder, true)
		return err
	})
	if err != nil || rep.Files == 0 {
		return 0, err
	}

	examples := ""
	for i, c := range rep.Changes {
		if i == 3 {
			break
		}
		examples += fmt.Sprintf("\n  %s\n  → %s\n", c.Old, c.New)
	}
	help := i18n.T("links.help", len(rep.Changes), rep.Files) + "\n" + examples
	ok, err := ui.Confirm(i18n.T("links.q"), help, true)
	if err != nil || !ok {
		return 0, err
	}

	backups := filepath.Join(site.Dir(), "backups")
	if err := os.MkdirAll(backups, 0o755); err != nil {
		return 0, err
	}
	zip := filepath.Join(backups, "notes-"+time.Now().Format("2006-01-02-150405")+".zip")
	if err := wikilinks.BackupFiles(site.Vault, rep.Changed, zip); err != nil {
		return 0, err
	}
	if rep, err = wikilinks.ConvertTree(idx, site.Vault, site.Folder, false); err != nil {
		return 0, err
	}
	ui.OK(i18n.T("links.done", len(rep.Changes), rep.Files))
	ui.Hint(i18n.T("links.backup", zip))
	warned := 0
	for _, p := range rep.Problems {
		if warned == 5 {
			ui.Hint(i18n.T("links.more_problems", len(rep.Problems)-warned))
			break
		}
		ui.Warn(fmt.Sprintf("%s:%d  %s  %s", p.File, p.Line, p.Link, i18n.T("links.kind."+string(p.Kind))))
		warned++
	}
	return rep.Files, nil
}
