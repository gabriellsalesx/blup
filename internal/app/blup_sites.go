package app

import (
	"context"
	"errors"
	"os"
	"sort"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
)

// The sites kept on Blup Cloud, to bring them to a new computer (Blup
// Studio's "I already have an account").

// BlupSite is a site synced through Blup Cloud.
type BlupSite struct {
	Vault string `json:"vault"`
	Title string `json:"title"`
	// Dir is the site's folder when it is already on this computer.
	Dir string `json:"dir"`
}

// localVaults maps a vault to the folder of the site synced with it here.
func localVaults() map[string]string {
	out := map[string]string{}
	g, err := config.LoadGlobal()
	if err != nil {
		return out
	}
	for _, dir := range g.ExistingSites() {
		if s, err := config.LoadSite(dir); err == nil && s.Sync.Remote == RemoteBlup && s.Sync.Vault != "" {
			out[s.Sync.Vault] = dir
		}
	}
	return out
}

// blupKeys is the client and the notes keys of the account signed in (a
// key left by another account is forgotten first: accountKeys).
func blupKeys(ctx context.Context) (*blupcloud.Client, *cloudcrypt.Keys, error) {
	c, creds, err := blupClient()
	if err != nil {
		return nil, nil, errors.New(describe(err))
	}
	keys, err := accountKeys(ctx, c, &creds)
	if err != nil {
		return nil, nil, errors.New(describe(err))
	}
	return c, keys, nil
}

// BlupSites lists the account's sites on Blup Cloud, by name.
func BlupSites(ctx context.Context) ([]BlupSite, error) {
	c, keys, err := blupKeys(ctx)
	if err != nil {
		return nil, err
	}
	vaults, err := c.Vaults(ctx)
	if err != nil {
		return nil, errors.New(describe(err))
	}
	here := localVaults()
	out := make([]BlupSite, 0, len(vaults))
	for _, v := range vaults {
		if v.Head == 0 {
			continue // never synced: nothing to bring
		}
		title, _, err := keys.OpenName(v.ID, v.Name)
		if err != nil {
			continue
		}
		out = append(out, BlupSite{Vault: v.ID, Title: title, Dir: here[v.ID]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out, nil
}

// BringBlupSite makes a site kept on Blup Cloud on this computer: new
// folders for its notes and its files, the notes downloaded, its settings
// applied (they travel with the notes). The site joins the list.
func BringBlupSite(ctx context.Context, vault string) (*config.Site, error) {
	if dir := localVaults()[vault]; dir != "" {
		return config.LoadSite(dir)
	}
	sites, err := BlupSites(ctx)
	if err != nil {
		return nil, err
	}
	title := ""
	for _, s := range sites {
		if s.Vault == vault {
			title = s.Title
		}
	}
	if title == "" {
		return nil, errors.New(i18n.T("bring.not_a_site", vault))
	}
	notes, dir := freeDir(DefaultNotesDir(title)), freeDir(DefaultSiteDir(title))
	for _, d := range []string{notes, dir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	site := config.NewSite(dir)
	site.Title, site.Model, site.Lang = title, "mochi", string(i18n.Current())
	site.Vault, site.Folder = notes, ""
	site.Sync = config.NoteSync{Remote: RemoteBlup, Vault: vault}
	if err := site.Save(); err != nil {
		return nil, err
	}
	// The first sync only receives: the settings come with the notes.
	if _, err := syncSite(ctx, site, false); err != nil {
		return nil, err
	}
	if _, err := os.Stat(sharedPath(site)); err == nil {
		if err := readShared(site); err != nil {
			return nil, err
		}
	}
	if err := RememberSite(dir); err != nil {
		return nil, err
	}
	// A name sealed before it was bound to the vault's id is sealed again
	// now, once (best effort: it opens as it is anyway).
	if c, keys, err := blupKeys(ctx); err == nil {
		_ = c.UpgradeVaultName(ctx, keys, vault)
	}
	return site, nil
}
