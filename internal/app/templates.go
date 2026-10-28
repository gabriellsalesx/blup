package app

import (
	"path/filepath"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/gabriellsalesx/blup/internal/ui"
)

// note is a file written into the vault: path relative to its folder.
type note struct{ path, body string }

// writeTemplates adds note templates for the core Templates plugin, in the
// vault's templates folder (created and configured if there is none). The
// folder is excluded from the site.
func writeTemplates(site *config.Site) error {
	folder := obsidian.TemplatesFolder(site.Vault)
	if folder == "" {
		folder = i18n.T("templates_folder")
		if err := obsidian.SetTemplatesFolder(site.Vault, folder); err != nil {
			return err
		}
	}
	list := templates[site.Lang+"/"+site.Model]
	if site.Model == "bento" {
		list = docsTemplates[site.Lang]
	}
	for _, n := range list {
		if _, err := obsidian.WriteNoteIfMissing(filepath.Join(site.Vault, filepath.FromSlash(folder), n.path), n.body); err != nil {
			return err
		}
	}
	ui.OK(i18n.T("new.templates", folder))
	return nil
}

var templates = map[string][]note{
	"pt/mochi": {{"Post do blog.md", "---\ntitle: \"{{title}}\"\ndate: {{date:YYYY-MM-DD}}\ndescription: \ntags: \ndraft: false\n---\n\n"}},
	"en/mochi": {{"Blog post.md", "---\ntitle: \"{{title}}\"\ndate: {{date:YYYY-MM-DD}}\ndescription: \ntags: \ndraft: false\n---\n\n"}},
}

var docsTemplates = map[string][]note{
	"pt": {
		{"Página da documentação.md", "---\ntitle: \"{{title}}\"\ndescription: \nweight: 10\n---\n\n"},
		{"Item do roadmap.md", "---\ntitle: \"{{title}}\"\ndescription: \nstatus: planejado\ncategory: \n---\n\n"},
		{"Versão do changelog.md", "---\ntitle: \"{{title}}\"\ndate: {{date:YYYY-MM-DD}}\ntags: \n---\n\n"},
	},
	"en": {
		{"Docs page.md", "---\ntitle: \"{{title}}\"\ndescription: \nweight: 10\n---\n\n"},
		{"Roadmap item.md", "---\ntitle: \"{{title}}\"\ndescription: \nstatus: planned\ncategory: \n---\n\n"},
		{"Changelog version.md", "---\ntitle: \"{{title}}\"\ndate: {{date:YYYY-MM-DD}}\ntags: \n---\n\n"},
	},
}
