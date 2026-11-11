package blupcloud

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// Sites published through Blup Cloud (paid plans), at <slug>.getblup.site.
// The site is built by Hugo already minified; here its text files are
// gzipped (the cloud keeps and sends them that way) and only the files the
// site doesn't have yet go up, by the hash of what is stored. A deploy then
// names every path's file and goes live at once (cloud/worker/sites.ts).

var (
	// ErrPlan: the plan has no sites (Free).
	ErrPlan = errors.New("blupcloud: the plan does not include sites")
	// ErrSitesLimit: the plan's sites are all in use.
	ErrSitesLimit = errors.New("blupcloud: the plan's sites are all in use")
	// ErrSlugTaken: another site has that address.
	ErrSlugTaken = errors.New("blupcloud: that address is taken")
	// ErrFileTooLarge: a file is over the plan's size limit.
	ErrFileTooLarge = errors.New("blupcloud: a file is over the plan's limit")
	// ErrSiteMissing (404 NOT_FOUND): the account has no site with that id
	// (taken down in the panel, or another account).
	ErrSiteMissing = errors.New("blupcloud: site not found")
)

// SiteSlug is a site's address on Blup Cloud: <slug>.getblup.site.
var SiteSlug = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])$`)

// Site is a site on Blup Cloud.
type Site struct {
	ID    string `json:"id"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Live  bool   `json:"live"`
	Bytes int64  `json:"bytes"`
	Files int    `json:"files"`
}

// Sites lists the account's sites and how many the plan allows (0: none,
// -1: no limit).
func (c *Client) Sites(ctx context.Context) ([]Site, int, error) {
	var r struct {
		Limit *int    `json:"limit"`
		Sites *[]Site `json:"sites"`
	}
	if err := c.do(ctx, http.MethodGet, "/sites", nil, "", &r); err != nil {
		return nil, 0, siteErr(err)
	}
	// Without the list, "no sites" would read as every site gone.
	if r.Limit == nil || *r.Limit < -1 || r.Sites == nil {
		return nil, 0, fmt.Errorf("%w: sites", ErrBadAnswer)
	}
	for _, s := range *r.Sites {
		if err := s.check(); err != nil {
			return nil, 0, err
		}
	}
	return *r.Sites, *r.Limit, nil
}

// CreateSite reserves slug.getblup.site for a new site.
func (c *Client) CreateSite(ctx context.Context, slug, title string) (Site, error) {
	var s Site
	if err := c.do(ctx, http.MethodPost, "/sites", map[string]string{"slug": slug, "title": title}, "", &s); err != nil {
		return Site{}, siteErr(err)
	}
	if err := s.check(); err != nil {
		return Site{}, err
	}
	if s.Slug != slug {
		return Site{}, fmt.Errorf("%w: site created at another address", ErrBadAnswer)
	}
	return s, nil
}

// check tells whether a site from the cloud has what the Studio uses: its
// id (it goes in addresses), its address and a web address to open.
func (s Site) check() error {
	if !validOpaqueID(s.ID) || !SiteSlug.MatchString(s.Slug) || !webURL(s.URL) || s.Bytes < 0 || s.Files < 0 {
		return fmt.Errorf("%w: site", ErrBadAnswer)
	}
	return nil
}

// validOpaqueID is an id as the cloud makes them (opaqueId, a site's or an
// account's): 1 to 128 letters, digits, "_" and "-".
func validOpaqueID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// webURL is an http(s) address with a host and no user in it.
func webURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && u.User == nil
}

// SlugCheck says whether a name can be a new site's address: "free",
// "yours" (a site of this account has it), "taken" (with free names close
// to it), "reserved" (kept by Blup) or "invalid".
type SlugCheck struct {
	Status      string   `json:"status"`
	Suggestions []string `json:"suggestions"`
}

// CheckSlug asks whether slug.getblup.site is free.
func (c *Client) CheckSlug(ctx context.Context, slug string) (SlugCheck, error) {
	var r SlugCheck
	if err := c.do(ctx, http.MethodGet, "/sites/check?slug="+url.QueryEscape(slug), nil, "", &r); err != nil {
		return SlugCheck{}, siteErr(err)
	}
	if !slugStatuses[r.Status] || len(r.Suggestions) > 10 {
		return SlugCheck{}, fmt.Errorf("%w: address check", ErrBadAnswer)
	}
	// Only names that can be an address are offered.
	r.Suggestions = slices.DeleteFunc(r.Suggestions, func(s string) bool { return !SiteSlug.MatchString(s) })
	if r.Suggestions == nil {
		r.Suggestions = []string{}
	}
	return r, nil
}

var slugStatuses = map[string]bool{"free": true, "yours": true, "taken": true, "reserved": true, "invalid": true}

// DeleteSite takes a site down and deletes its files.
func (c *Client) DeleteSite(ctx context.Context, id string) error {
	return siteErr(c.do(ctx, http.MethodDelete, "/sites/"+url.PathEscape(id), nil, "", nil))
}

func siteErr(err error) error {
	var e *apiError
	if errors.As(err, &e) {
		switch e.Code {
		case "PLAN":
			return ErrPlan
		case "SITES_LIMIT":
			return ErrSitesLimit
		case "SLUG_TAKEN":
			return ErrSlugTaken
		case "TOO_LARGE":
			return ErrFileTooLarge
		case "NOT_FOUND":
			return ErrSiteMissing
		}
	}
	return err
}

// SiteFile is one file of a built site, as it goes up.
type SiteFile struct {
	Path string `json:"path"` // "/posts/oi/index.html"
	Hash string `json:"hash"` // SHA-256 of what is stored
	Type string `json:"type"`
	GZ   bool   `json:"gz"`
	src  string // the file on disk
	data []byte // what is stored, when gzipped (text files are small)
}

// Files that compress well: gzipped before going up.
var textTypes = []string{"text/", "application/javascript", "application/json", "application/xml", "application/rss+xml", "application/atom+xml", "application/manifest+json", "image/svg+xml", "application/wasm", "font/ttf", "font/otf", "image/x-icon"}

func mediaType(name string) string {
	ext := strings.ToLower(path.Ext(name))
	switch ext {
	case ".webmanifest":
		return "application/manifest+json"
	case ".xml":
		return "application/xml; charset=utf-8"
	case ".mp3":
		return "audio/mpeg"
	case ".woff2":
		return "font/woff2"
	case ".avif":
		return "image/avif"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".wasm":
		return "application/wasm"
	case ".pf_meta", ".pf_index", ".pf_fragment", ".pagefind":
		return "application/octet-stream" // Pagefind's index: already compressed
	}
	t := mime.TypeByExtension(ext)
	if t == "" {
		return "application/octet-stream"
	}
	// Go adds "; charset=utf-8" to text types; the cloud takes only that one.
	if base, params, ok := strings.Cut(t, ";"); ok {
		if strings.Contains(strings.ToLower(params), "utf-8") {
			return base + "; charset=utf-8"
		}
		return base
	}
	return t
}

func compressible(t string) bool {
	return slices.ContainsFunc(textTypes, func(p string) bool { return strings.HasPrefix(t, p) })
}

// skipped are Cloudflare's own config files (_headers, _redirects): they mean nothing here.
var skipped = map[string]bool{"/_headers": true, "/_redirects": true, "/_routes.json": true, "/_worker.js": true}

// PackSite reads a built site (dir): every file with its stored hash, text
// files gzipped at the best level when that makes them smaller.
func PackSite(dir string) ([]SiteFile, error) {
	var out []SiteFile
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && p != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		f := SiteFile{Path: "/" + filepath.ToSlash(rel), Type: mediaType(d.Name()), src: p}
		if skipped[f.Path] {
			return nil
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		stored := raw
		if compressible(f.Type) && len(raw) > 256 {
			var buf bytes.Buffer
			w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
			w.Write(raw)
			if err := w.Close(); err != nil {
				return err
			}
			if buf.Len() < len(raw) {
				stored, f.GZ, f.data = buf.Bytes(), true, buf.Bytes()
			}
		}
		sum := sha256.Sum256(stored)
		f.Hash = hex.EncodeToString(sum[:])
		out = append(out, f)
		return nil
	})
	return out, err
}

func (f SiteFile) stored() ([]byte, error) {
	if f.data != nil {
		return f.data, nil
	}
	return os.ReadFile(f.src)
}

// DeployResult says what went up.
type DeployResult struct {
	URL      string
	Uploaded int // files sent
	Skipped  int // files the site already had
}

// siteUploads is how many files go at once: each waits mostly on the
// network and on Blup Cloud, so more side by side finish a site of a
// thousand files in a fraction of the time (well under the server's 500
// requests per 10 s an account).
const siteUploads = 16

// uploadProgressKey carries, in a context, who hears the files sent.
type uploadProgressKey struct{}

// WithUploadProgress is ctx with fn hearing, while a site goes up, how
// many of its files were sent of how many it lacked.
func WithUploadProgress(ctx context.Context, fn func(done, total int)) context.Context {
	return context.WithValue(ctx, uploadProgressKey{}, fn)
}

// DeploySite sends what the site lacks and makes files its live version.
func (c *Client) DeploySite(ctx context.Context, siteID string, files []SiteFile) (DeployResult, error) {
	base := "/sites/" + url.PathEscape(siteID)
	byHash := map[string]SiteFile{}
	for _, f := range files {
		byHash[f.Hash] = f
	}
	hashes := make([]string, 0, len(byHash))
	for h := range byHash {
		hashes = append(hashes, h)
	}
	var res DeployResult
	progress, _ := ctx.Value(uploadProgressKey{}).(func(done, total int))
	send := func(missing []string) error {
		var mu sync.Mutex
		done := 0
		if progress != nil {
			progress(0, len(missing))
		}
		jobs := make(chan string)
		errs := make(chan error, siteUploads)
		var wg sync.WaitGroup
		for range siteUploads {
			wg.Go(func() {
				for h := range jobs {
					data, err := byHash[h].stored()
					if err == nil {
						err = siteErr(c.do(ctx, http.MethodPut, base+"/files/"+h, data, "application/octet-stream", nil))
					}
					if err == nil && progress != nil {
						mu.Lock()
						done++
						progress(done, len(missing))
						mu.Unlock()
					}
					if err != nil {
						select {
						case errs <- err:
						default:
						}
					}
				}
			})
		}
	feed:
		for _, h := range missing {
			select {
			case jobs <- h:
			case err := <-errs:
				close(jobs)
				wg.Wait()
				return err
			case <-ctx.Done():
				break feed
			}
		}
		close(jobs)
		wg.Wait()
		select {
		case err := <-errs:
			return err
		default:
		}
		res.Uploaded += len(missing)
		return ctx.Err()
	}
	missingOf := func() ([]string, error) {
		var missing []string
		for i := 0; i < len(hashes); i += 1000 {
			part := hashes[i:min(i+1000, len(hashes))]
			r, err := missingAnswer(func(r any) error {
				return c.do(ctx, http.MethodPost, base+"/missing", map[string]any{"hashes": part}, "", r)
			}, part)
			if err != nil {
				return nil, siteErr(err)
			}
			missing = append(missing, r...)
		}
		return missing, nil
	}
	missing, err := missingOf()
	if err != nil {
		return res, err
	}
	res.Skipped = len(hashes) - len(missing)
	if err := send(missing); err != nil {
		return res, err
	}
	// A deploy finishing elsewhere may have cleared files meanwhile: those
	// go again, once.
	for attempt := 0; ; attempt++ {
		var r struct {
			URL string `json:"url"`
		}
		err := c.do(ctx, http.MethodPost, base+"/deploys", map[string]any{"files": files}, "", &r)
		var e *apiError
		if errors.As(err, &e) && e.Code == "MISSING_FILES" && attempt == 0 {
			again, err := missingOf()
			if err != nil {
				return res, err
			}
			if err := send(again); err != nil {
				return res, err
			}
			continue
		}
		if err != nil {
			return res, siteErr(err)
		}
		if !webURL(r.URL) {
			return res, fmt.Errorf("%w: deploy", ErrBadAnswer)
		}
		res.URL = r.URL
		return res, nil
	}
}
