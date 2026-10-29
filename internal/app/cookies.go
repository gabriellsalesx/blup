package app

import (
	"errors"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"charm.land/huh/v2"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/ui"
)

// Cookies: the visitor statistics and ads tools of the site (Google
// Analytics, Google Tag Manager, Meta Pixel, Umami, Plausible) and the consent banner, which
// loads them, and YouTube videos, only after the visitor accepts. The theme
// does the rest (consent.html, consent.js).

// cookieTool is one tool: its key, how to ask for its ID and how to check it.
type cookieTool struct {
	key   string
	valid *regexp.Regexp
	value func(c *config.Cookies) *string
}

var cookieTools = []cookieTool{
	{"google_analytics", regexp.MustCompile(`^G-[A-Z0-9]{4,20}$`), func(c *config.Cookies) *string { return &c.GoogleAnalytics }},
	{"google_tag_manager", regexp.MustCompile(`^GTM-[A-Z0-9]{4,12}$`), func(c *config.Cookies) *string { return &c.TagManager }},
	{"meta_pixel", regexp.MustCompile(`^\d{5,20}$`), func(c *config.Cookies) *string { return &c.MetaPixel }},
	{"umami", regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`), func(c *config.Cookies) *string { return &c.Umami }},
	{"plausible", regexp.MustCompile(`^[a-z0-9.-]+\.[a-z]{2,}$`), func(c *config.Cookies) *string { return &c.Plausible }},
}

// editCookies turns the banner on or off and sets the tools (customize).
func editCookies(site *config.Site) error {
	c := site.Cookies // edited on a copy: Esc leaves the site as it was
	if err := ui.Explain(i18n.T("ck.title"), i18n.T("ck.about")); err != nil {
		return err
	}

	// Which tools the site uses, then the ID of each.
	var picked []string
	opts := make([]huh.Option[string], 0, len(cookieTools))
	for _, t := range cookieTools {
		if *t.value(&c) != "" {
			picked = append(picked, t.key)
		}
		opts = append(opts, ui.Option(i18n.T("ck.tool."+t.key), t.key))
	}
	if err := ui.ChooseMany(i18n.T("ck.tools_q"), i18n.T("ck.tools_help"), &picked, opts...); err != nil {
		return err
	}
	for _, t := range cookieTools {
		v := t.value(&c)
		if !slices.Contains(picked, t.key) {
			*v = ""
			continue
		}
		valid := t.valid
		if err := ui.Ask(i18n.T("ck.id."+t.key), i18n.T("ck.id_help."+t.key), i18n.T("ck.id_example."+t.key), v, func(s string) error {
			if !valid.MatchString(strings.TrimSpace(s)) {
				return errors.New(i18n.T("ck.bad_id"))
			}
			return nil
		}); err != nil {
			return err
		}
		*v = strings.TrimSpace(*v)
	}
	// Self-hosted Umami or Plausible: their own script address.
	for _, s := range []struct {
		on     bool
		key    string
		value  *string
		hosted string
	}{
		{c.Umami != "", "umami", &c.UmamiScript, config.UmamiCloudScript},
		{c.Plausible != "", "plausible", &c.PlausibleScript, config.PlausibleScript},
	} {
		if !s.on {
			*s.value = ""
			continue
		}
		if err := ui.Ask(i18n.T("ck.script."+s.key), i18n.T("ck.script_help", s.hosted), s.hosted, s.value, optional(validURL)); err != nil {
			return err
		}
		if strings.TrimSpace(*s.value) == s.hosted {
			*s.value = ""
		}
	}

	on, err := ui.Confirm(i18n.T("ck.banner_q"), i18n.T("ck.banner_help"), c.Banner || c.Tools())
	if err != nil {
		return err
	}
	c.Banner = on

	// The privacy policy: a page of the site (a template note to edit in
	// Obsidian) or an address of the user's.
	page, contact := false, ""
	if c.Banner || c.Tools() {
		where := "page"
		if c.Privacy != "" {
			where = "url"
		}
		if err := ui.Choose(i18n.T("ck.privacy_q"), i18n.T("ck.privacy_help"), &where,
			ui.Option(i18n.T("ck.privacy_page"), "page"),
			ui.Option(i18n.T("ck.privacy_url"), "url"),
		); err != nil {
			return err
		}
		if where == "url" {
			if err := ui.Ask(i18n.T("ck.privacy_url_q"), "", "https://", &c.Privacy, validURL); err != nil {
				return err
			}
			c.Privacy = strings.TrimSpace(c.Privacy)
		} else {
			c.Privacy = ""
			page = true
			if privacyNote(site) == "" {
				if err := ui.Ask(i18n.T("ck.contact_q"), i18n.T("ck.contact_help"), i18n.T("ck.contact_example"), &contact, nil); err != nil {
					return err
				}
				contact = strings.TrimSpace(contact)
			}
		}
	}

	site.Cookies = c
	if page {
		existed := privacyNote(site) != ""
		path, err := writePrivacyNote(site, contact)
		if err != nil {
			return err
		}
		if existed {
			ui.OK(i18n.T("ck.privacy_kept", filepath.Base(path)))
		} else {
			ui.OK(i18n.T("ck.privacy_made", filepath.Base(path)))
		}
	}
	switch {
	case c.Banner:
		ui.OK(i18n.T("ck.on"))
	case c.Tools():
		ui.Warn(i18n.T("ck.no_banner"))
	default:
		ui.OK(i18n.T("ck.off"))
	}
	ui.Hint(i18n.T("fb.republish"))
	return nil
}
