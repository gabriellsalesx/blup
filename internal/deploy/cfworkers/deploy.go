package cfworkers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Limits, from wrangler (@cloudflare/workers-shared utils/constants.ts and
// deploy-helpers assets.ts) and https://developers.cloudflare.com/workers/platform/limits/.
const (
	MaxFileSize        = 25 << 20 // MAX_ASSET_SIZE: 25 MiB per file
	MaxFiles           = 100_000  // MAX_ASSET_COUNT (Paid plan; Free is 20,000, which Cloudflare enforces)
	uploadConcurrency  = 3        // BULK_UPLOAD_CONCURRENCY
	singleConcurrency  = 50       // EDGE_KV_UPLOAD_CONCURRENCY (single-file mode)
	maxUploadAttempts  = 5        // MAX_UPLOAD_ATTEMPTS
	maxGatewayErrors   = 5        // MAX_UPLOAD_GATEWAY_ERRORS
	maxPublishAttempts = 3
)

// CompatibilityDate of the Workers Blup publishes. A site of static files
// runs no code, so it only sets the date the asset serving follows.
const CompatibilityDate = "2026-11-01"

// scriptAPIDate is the version of the Worker subdomain API wrangler uses
// (Cloudflare-Workers-Script-Api-Date).
const scriptAPIDate = "2025-08-01"

// Stage is a step of Deploy, reported to the callback.
type Stage string

// Progress stages reported to the callback.
const (
	StageScanning  Stage = "scanning"  // walking and hashing files (Done/Total = files hashed)
	StageChecking  Stage = "checking"  // sending the manifest: which files Cloudflare already has
	StageUploading Stage = "uploading" // Done/Total = files present remotely
	StageDeploying Stage = "deploying" // publishing the version and turning on workers.dev
	StageDone      Stage = "done"
)

// Progress is reported during Deploy.
type Progress struct {
	Stage       Stage
	Done, Total int
}

// Result of a successful Deploy.
type Result struct {
	// VersionID is the Worker version Cloudflare created.
	VersionID string
	// Uploaded files were new; Skipped were already stored on Cloudflare.
	Uploaded, Skipped int
}

type fileEntry struct {
	name        string // slash-separated, relative to dir
	path        string // on disk
	size        int64
	hash        string
	contentType string
}

// ignored follows wrangler: _headers, _redirects and .assetsignore at the
// root are configuration, not files to serve (createAssetsIgnoreFunction),
// and _worker.js is never published as a file (it may hold server code).
// Anywhere: .DS_Store, node_modules, .git, as Pages did.
func ignored(rel string) bool {
	switch path.Base(rel) {
	case ".DS_Store", "node_modules", ".git":
		return true
	}
	if !strings.Contains(rel, "/") {
		switch rel {
		case "_worker.js", "_redirects", "_headers", ".assetsignore":
			return true
		}
	}
	return false
}

func scan(dir string, maxFiles int, progress func(Progress)) ([]fileEntry, error) {
	var files []fileEntry
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == dir {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if ignored(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		// Only regular files: a symlink could point outside the site and
		// publish whatever it reaches (wrangler skips them too).
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > MaxFileSize {
			return fmt.Errorf("%s is %d bytes, Workers accepts up to 25 MiB per file: %w", rel, info.Size(), ErrFileTooLarge)
		}
		files = append(files, fileEntry{name: rel, path: p, size: info.Size(), contentType: contentTypeFor(rel)})
		if len(files) > maxFiles {
			return fmt.Errorf("%s has more than %d files: %w", dir, maxFiles, ErrTooManyFiles)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i := range files {
		b, err := os.ReadFile(files[i].path)
		if err != nil {
			return nil, err
		}
		files[i].hash = HashFile(files[i].name, b)
		if progress != nil && (i%50 == 0 || i == len(files)-1) {
			progress(Progress{Stage: StageScanning, Done: i + 1, Total: len(files)})
		}
	}
	return files, nil
}

// jwtClaims decodes the JWT payload without verifying it (as wrangler does).
func jwtClaims(tok string) map[string]any {
	parts := strings.Split(tok, ".")
	if len(parts) < 2 {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}

// singleUploads: Cloudflare may ask for one request per file (a claim of
// the upload JWT), as wrangler's isSingleAssetUploadMode reads it.
func singleUploads(tok string) bool {
	b, _ := jwtClaims(tok)["wrangler_single_asset_uploads"].(bool)
	return b
}

func singleConcurrencyFrom(tok string) int {
	if n, ok := jwtClaims(tok)["edge_kv_upload_concurrency"].(float64); ok && n >= 1 {
		return int(n)
	}
	return singleConcurrency
}

func isGateway(err error) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.Status == http.StatusBadGateway || ae.Status == http.StatusServiceUnavailable || ae.Status == http.StatusGatewayTimeout
	}
	return false
}

func backoff(attempt int) time.Duration { return time.Duration(1<<attempt) * time.Second }

// Deploy uploads dir as the static files of the Worker name (creating it
// on the first call), publishes it and turns on its workers.dev address.
// The caller checks first that name is not someone else's Worker (Exists):
// publishing replaces whatever Worker has the name. progress may be nil.
func (c *Client) Deploy(ctx context.Context, accountID, name, dir string, progress func(Progress)) (Result, error) {
	report := func(p Progress) {
		if progress != nil {
			progress(p)
		}
	}
	if err := ValidateName(name); err != nil {
		return Result{}, err
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return Result{}, fmt.Errorf("site folder %s: %w", dir, err)
	}

	report(Progress{Stage: StageScanning})
	files, err := scan(dir, MaxFiles, report)
	if err != nil {
		return Result{}, err
	}

	report(Progress{Stage: StageChecking, Total: len(files)})
	manifest := make(map[string]manifestEntry, len(files))
	byHash := map[string]fileEntry{}
	for _, f := range files {
		manifest["/"+f.name] = manifestEntry{Hash: f.hash, Size: f.size}
		byHash[f.hash] = f
	}
	var sess struct {
		JWT     string     `json:"jwt"`
		Buckets [][]string `json:"buckets"`
	}
	if err := c.retry(ctx, func() error {
		_, err := c.do(ctx, http.MethodPost, scriptPath(accountID, name)+"/assets-upload-session", "", map[string]any{"manifest": manifest}, "", &sess)
		return err
	}); err != nil {
		return Result{}, fmt.Errorf("starting the upload: %w", err)
	}
	if sess.JWT == "" {
		return Result{}, errors.New("starting the upload: Cloudflare sent no upload token")
	}

	// The files Cloudflare asked for, in its buckets. Every file with a
	// requested hash counts as uploaded (two files may share one hash).
	var buckets [][]fileEntry
	wanted := map[string]bool{}
	for _, b := range sess.Buckets {
		var bucket []fileEntry
		for _, h := range b {
			f, ok := byHash[h]
			if !ok {
				return Result{}, fmt.Errorf("starting the upload: Cloudflare asked for a file that is not in the site (%s)", h)
			}
			wanted[h] = true
			bucket = append(bucket, f)
		}
		if len(bucket) > 0 {
			buckets = append(buckets, bucket)
		}
	}
	uploaded := 0
	for _, f := range files {
		if wanted[f.hash] {
			uploaded++
		}
	}

	completion := sess.JWT // with nothing to upload, the session token completes it
	if len(buckets) > 0 {
		if singleUploads(sess.JWT) {
			var single [][]fileEntry
			for _, b := range buckets {
				for _, f := range b {
					single = append(single, []fileEntry{f})
				}
			}
			buckets = single
		}
		tok, err := c.uploadAll(ctx, accountID, sess.JWT, buckets, len(files), len(files)-uploaded, report)
		if err != nil {
			return Result{}, err
		}
		completion = tok
	}

	report(Progress{Stage: StageDeploying, Done: len(files), Total: len(files)})
	version, err := c.publish(ctx, accountID, name, dir, completion)
	if err != nil {
		return Result{}, err
	}
	if err := c.EnableWorkersDev(ctx, accountID, name); err != nil {
		return Result{}, err
	}
	report(Progress{Stage: StageDone, Done: len(files), Total: len(files)})
	return Result{VersionID: version, Uploaded: uploaded, Skipped: len(files) - uploaded}, nil
}

type manifestEntry struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// retry runs fn up to maxPublishAttempts times on Cloudflare's own
// hiccups (5xx, rate limit), like wrangler's retryOnAPIFailure.
func (c *Client) retry(ctx context.Context, fn func() error) error {
	var err error
	for attempt := 0; attempt < maxPublishAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		var ae *APIError
		if !errors.As(err, &ae) || (ae.Status < 500 && ae.Status != http.StatusTooManyRequests) || attempt == maxPublishAttempts-1 {
			return err
		}
		if werr := c.wait(ctx, backoff(attempt)); werr != nil {
			return werr
		}
	}
	return err
}

// uploadAll sends the buckets (3 at a time, or many at a time in
// single-file mode) and returns the completion JWT, which Cloudflare sends
// with the last bucket.
func (c *Client) uploadAll(ctx context.Context, accountID, jwt string, buckets [][]fileEntry, total, already int, report func(Progress)) (string, error) {
	report(Progress{Stage: StageUploading, Done: already, Total: total})
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	single := singleUploads(jwt)
	limit := uploadConcurrency
	if single {
		limit = singleConcurrencyFrom(jwt)
	}
	var (
		mu         sync.Mutex
		done       = already
		completion string
		firstErr   error
		wg         sync.WaitGroup
		sem        = make(chan struct{}, limit)
		gwMu       sync.Mutex // held while backing off from gateway errors
	)
	for _, b := range buckets {
		b := b
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			tok, err := c.uploadBucket(ctx, accountID, jwt, b, single, &gwMu)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
					cancel()
				}
				return
			}
			if tok != "" {
				completion = tok
			}
			done += len(b)
			report(Progress{Stage: StageUploading, Done: min(done, total), Total: total})
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return "", fmt.Errorf("uploading files: %w", firstErr)
	}
	if completion == "" {
		return "", errors.New("uploading files: Cloudflare did not confirm the upload; try again")
	}
	report(Progress{Stage: StageUploading, Done: total, Total: total})
	return completion, nil
}

// uploadBucket sends one bucket: a multipart body whose fields are the
// files' hashes and whose values are their contents in base64, each part
// typed with the type the file is served with (wrangler's FormData).
func (c *Client) uploadBucket(ctx context.Context, accountID, jwt string, files []fileEntry, single bool, gwMu *sync.Mutex) (string, error) {
	build := func() (*bytes.Buffer, string, string, error) {
		if single {
			data, err := os.ReadFile(files[0].path)
			if err != nil {
				return nil, "", "", err
			}
			p := "/accounts/" + url.PathEscape(accountID) + "/workers/assets/upload/" + url.PathEscape(files[0].hash)
			return bytes.NewBuffer(data), files[0].contentType, p, nil
		}
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		for _, f := range files {
			data, err := os.ReadFile(f.path)
			if err != nil {
				return nil, "", "", err
			}
			h := textproto.MIMEHeader{}
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, f.hash, f.hash))
			h.Set("Content-Type", f.contentType)
			pw, err := w.CreatePart(h)
			if err != nil {
				return nil, "", "", err
			}
			if _, err := pw.Write([]byte(base64.StdEncoding.EncodeToString(data))); err != nil {
				return nil, "", "", err
			}
		}
		if err := w.Close(); err != nil {
			return nil, "", "", err
		}
		return &buf, w.FormDataContentType(), "/accounts/" + url.PathEscape(accountID) + "/workers/assets/upload?base64=true", nil
	}

	attempts, gateway := 0, 0
	for {
		// Barrier: wait while another bucket backs off.
		gwMu.Lock()
		//lint:ignore SA2001 intentional barrier
		gwMu.Unlock()
		body, ct, p, err := build()
		if err != nil {
			return "", err
		}
		var r struct {
			JWT string `json:"jwt"`
		}
		_, err = c.do(ctx, http.MethodPost, p, jwt, body, ct, &r)
		if err == nil {
			return r.JWT, nil
		}
		if ctx.Err() != nil || attempts >= maxUploadAttempts {
			return "", err
		}
		// Hard limits and a refused token are not retried.
		if errors.Is(err, ErrFileTooLarge) || errors.Is(err, ErrTooManyFiles) || errors.Is(err, ErrMissingPermission) || errors.Is(err, ErrInvalidToken) {
			return "", err
		}
		if werr := c.wait(ctx, backoff(attempts)); werr != nil {
			return "", werr
		}
		if isGateway(err) {
			gwMu.Lock()
			werr := c.wait(ctx, time.Duration(1<<gateway)*5*time.Second)
			gwMu.Unlock()
			if werr != nil {
				return "", werr
			}
			gateway++
			if gateway >= maxGatewayErrors {
				attempts++
			}
			continue
		}
		attempts++
	}
}

// assetConfig is how Cloudflare serves the files: Hugo's pretty URLs
// (/post/ for /post/index.html) and its 404.html for missing pages, plus
// the site's own _headers and _redirects.
type assetConfig struct {
	HTMLHandling     string `json:"html_handling"`
	NotFoundHandling string `json:"not_found_handling"`
	Headers          string `json:"_headers,omitempty"`
	Redirects        string `json:"_redirects,omitempty"`
}

// publish uploads the Worker version that serves the files: metadata only,
// no code (wrangler's assets-only upload), and returns its version id.
func (c *Client) publish(ctx context.Context, accountID, name, dir, completion string) (string, error) {
	cfg := assetConfig{HTMLHandling: "auto-trailing-slash", NotFoundHandling: "404-page"}
	for _, f := range []struct {
		name string
		to   *string
	}{{"_headers", &cfg.Headers}, {"_redirects", &cfg.Redirects}} {
		b, err := os.ReadFile(filepath.Join(dir, f.name))
		if err == nil {
			*f.to = string(b)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("reading %s: %w", f.name, err)
		}
	}
	meta, err := json.Marshal(map[string]any{
		"assets":             map[string]any{"jwt": completion, "config": cfg},
		"compatibility_date": CompatibilityDate,
	})
	if err != nil {
		return "", err
	}
	var out struct {
		ID           string `json:"id"`
		DeploymentID string `json:"deployment_id"`
	}
	err = c.retry(ctx, func() error {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="metadata"`)
		h.Set("Content-Type", "application/json")
		pw, err := w.CreatePart(h)
		if err != nil {
			return err
		}
		if _, err := pw.Write(meta); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		_, err = c.do(ctx, http.MethodPut, scriptPath(accountID, name), "", &buf, w.FormDataContentType(), &out)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("publishing the Worker %q: %w", name, err)
	}
	return out.DeploymentID, nil
}

// EnableWorkersDev turns on <name>.<account>.workers.dev for the Worker,
// with its preview URLs off (Blup publishes one version at a time).
func (c *Client) EnableWorkersDev(ctx context.Context, accountID, name string) error {
	err := c.retry(ctx, func() error {
		_, err := c.do(ctx, http.MethodPost, scriptPath(accountID, name)+"/subdomain", "",
			map[string]bool{"enabled": true, "previews_enabled": false}, "", nil,
			[2]string{"Cloudflare-Workers-Script-Api-Date", scriptAPIDate})
		return err
	})
	if err != nil {
		return fmt.Errorf("turning on the workers.dev address: %w", err)
	}
	return nil
}
