package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/ui"
)

type buildOptions struct {
	url string // where the site will live; "" = the site's own address
	out string // folder to copy the result to; "" = <site>/public only
}

// buildMarker marks a folder filled by --out, so a later build may replace
// its contents (and never wipes a folder Blup did not write).
const buildMarker = ".blup-build"

// parseBuildFlags reads --url/--endereco and --out/--saida (with a value
// after a space or "="), returning the other arguments.
func parseBuildFlags(args []string) (buildOptions, []string, error) {
	var o buildOptions
	var rest []string
	for i := 0; i < len(args); i++ {
		name, value, hasValue := strings.Cut(args[i], "=")
		var dst *string
		switch name {
		case "--url", "--endereco":
			dst = &o.url
		case "--out", "--saida":
			dst = &o.out
		default:
			rest = append(rest, args[i])
			continue
		}
		if !hasValue {
			if i+1 >= len(args) {
				return o, nil, fmt.Errorf(i18n.T("build.flag_value"), name)
			}
			i++
			value = args[i]
		}
		*dst = value
	}
	if o.url != "" {
		u, err := url.Parse(o.url)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return o, nil, errors.New(i18n.T("build.bad_url", o.url))
		}
		if !strings.HasSuffix(u.Path, "/") {
			u.Path += "/"
		}
		o.url = u.String()
	}
	if o.out != "" {
		o.out = expandHome(o.out)
	}
	return o, rest, nil
}

func buildCommand(ctx context.Context, site *config.Site, o buildOptions) error {
	base := o.url
	if base == "" {
		base = siteURL(site, "/")
	}
	dir, err := build(ctx, site, base)
	if err != nil || o.out == "" {
		return err
	}
	out, err := filepath.Abs(o.out)
	if err != nil {
		return err
	}
	if err := clearOut(out); err != nil {
		return err
	}
	if err := os.CopyFS(out, os.DirFS(dir)); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, buildMarker), []byte("Built by blup. This folder is replaced on every build.\n"), 0o644); err != nil {
		return err
	}
	ui.OK(i18n.T("build.copied", out))
	return nil
}

// clearOut empties out for a new build: only when it is missing, empty, or
// was filled by Blup before (it has the marker).
func clearOut(out string) error {
	entries, err := os.ReadDir(out)
	if errors.Is(err, fs.ErrNotExist) {
		return os.MkdirAll(out, 0o755)
	}
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	if !fileExists(filepath.Join(out, buildMarker)) {
		return errors.New(i18n.T("build.out_not_ours", out))
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(out, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
