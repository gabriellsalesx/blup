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

// Limits (wrangler: workers-shared/utils/constants.ts and deploy-helpers
// assets.ts; https://developers.cloudflare.com/workers/platform/limits/#static-assets).
const (
	MaxFileSize = 25 << 20 // MAX_ASSET_SIZE: 25 MiB per file
	// MaxFiles is the global ceiling (MAX_ASSET_COUNT, Workers Paid). Workers
	// Free allows 20,000; that limit is enforced by the API at session start
	// and surfaces as ErrTooManyFiles / ErrQuota.
	MaxFiles = 100_000

	bulkUploadConcurrency   = 3  // BULK_UPLOAD_CONCURRENCY
	edgeKVUploadConcurrency = 50 // EDGE_KV_UPLOAD_CONCURRENCY (single-asset mode default)
	maxUploadAttempts       = 5  // MAX_UPLOAD_ATTEMPTS
	maxGatewayErrors        = 5  // MAX_UPLOAD_GATEWAY_ERRORS
	maxSessions             = 2  // restart the upload session once if its JWT expires
	maxScriptAttempts       = 3  // retries for the script PUT on 5xx
)

// DefaultCompatibilityDate is sent with the assets-only Worker.
const DefaultCompatibilityDate = "2026-09-01"

// Asset routing used for every deploy (Hugo output conventions).
const (
	HTMLHandling     = "auto-trailing-slash"
	NotFoundHandling = "404-page"
)

// Progress stages reported to the callback.
const (
	StageScanning  = "scanning"  // walking and hashing files
	StageChecking  = "checking"  // assets upload session (which files are new)
	StageUploading = "uploading" // Done/Total = files present remotely
	StageDeploying = "deploying" // uploading the Worker version
	StageRouting   = "routing"   // enabling the workers.dev route
	StageDone      = "done"
)

// Progress is reported during Deploy.
type Progress struct {
	Stage       string
	Done, Total int
}

// Result of a successful Deploy.
type Result struct {
	// URL is https://<script>.<account-subdomain>.workers.dev/.
	URL string
	// DeploymentID identifies the uploaded Worker version/deployment.
	DeploymentID string
	// Uploaded files were new; Skipped were already stored on Cloudflare.
	Uploaded, Skipped int
}

type fileEntry struct {
	name        string // "/a/b.html"
	path        string // on disk
	size        int64
	hash        string
	contentType string
}

// ignored: wrangler ignores /.assetsignore, /_redirects and /_headers (sent
// as config instead). We also skip .DS_Store, .git and node_modules anywhere
// (Pages did this by default; Workers expects an .assetsignore for it) and a
// root _worker.js (wrangler refuses to publish it as an asset). Custom
// .assetsignore patterns are not supported: our CLI controls the folder.
func ignored(rel string) bool {
	switch path.Base(rel) {
	case ".DS_Store", "node_modules", ".git":
		return true
	}
	switch rel {
	case ".assetsignore", "_redirects", "_headers", "_worker.js":
		return true
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
		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		if info.Size() > MaxFileSize {
			return fmt.Errorf("%s is %d bytes, Workers accepts up to 25 MiB per file: %w", rel, info.Size(), ErrFileTooLarge)
		}
		files = append(files, fileEntry{name: "/" + rel, path: p, size: info.Size(), contentType: contentTypeFor(rel)})
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
		files[i].hash = HashFile(files[i].path, b)
		if progress != nil && (i%50 == 0 || i == len(files)-1) {
			progress(Progress{Stage: StageScanning, Done: i + 1, Total: len(files)})
		}
	}
	return files, nil
}

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

func jwtExpired(tok string) bool {
	exp, ok := jwtClaims(tok)["exp"].(float64)
	return ok && exp <= float64(time.Now().Unix())
}

// singleAssetMode / uploadConcurrency follow wrangler's JWT claims
// wrangler_single_asset_uploads and edge_kv_upload_concurrency.
func singleAssetMode(tok string) bool {
	v, _ := jwtClaims(tok)["wrangler_single_asset_uploads"].(bool)
	return v
}

func uploadConcurrency(tok string) int {
	if !singleAssetMode(tok) {
		return bulkUploadConcurrency
	}
	if n, ok := jwtClaims(tok)["edge_kv_upload_concurrency"].(float64); ok && n >= 1 {
		return int(n)
	}
	return edgeKVUploadConcurrency
}

func isGateway(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && (ae.Status == http.StatusBadGateway || ae.Status == http.StatusServiceUnavailable || ae.Status == http.StatusGatewayTimeout)
}

func backoff(attempt int) time.Duration { return time.Duration(1<<attempt) * time.Second }

var errSessionExpired = errors.New("upload session expired")

// Deploy uploads dir as the assets of an assets-only Worker named script,
// deploys it, and enables its workers.dev route. progress may be nil.
func (c *Client) Deploy(ctx context.Context, accountID, script, dir string, progress func(Progress)) (Result, error) {
	report := func(p Progress) {
		if progress != nil {
			progress(p)
		}
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return Result{}, fmt.Errorf("site folder %s: %w", dir, err)
	}
	site, err := c.EnsureSite(ctx, accountID, script)
	if err != nil {
		return Result{}, err
	}

	report(Progress{Stage: StageScanning})
	files, err := scan(dir, MaxFiles, report)
	if err != nil {
		return Result{}, err
	}
	cfg := map[string]any{"html_handling": HTMLHandling, "not_found_handling": NotFoundHandling}
	for _, name := range []string{"_headers", "_redirects"} {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			cfg[name] = string(b)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return Result{}, fmt.Errorf("reading %s: %w", name, err)
		}
	}

	var completion string
	uploaded := 0
	for session := 0; ; session++ {
		report(Progress{Stage: StageChecking, Total: len(files)})
		completion, uploaded, err = c.syncAssets(ctx, accountID, script, files, report)
		if errors.Is(err, errSessionExpired) && session+1 < maxSessions {
			continue // already-uploaded files are kept; a new session resumes
		}
		if errors.Is(err, errSessionExpired) {
			return Result{}, fmt.Errorf("uploading files: %w", ErrUploadExpired)
		}
		if err != nil {
			return Result{}, err
		}
		break
	}

	report(Progress{Stage: StageDeploying, Done: len(files), Total: len(files)})
	depID, err := c.putScript(ctx, accountID, script, completion, cfg)
	if err != nil {
		return Result{}, err
	}

	report(Progress{Stage: StageRouting, Done: len(files), Total: len(files)})
	if err := c.enableWorkersDev(ctx, accountID, script); err != nil {
		return Result{}, err
	}
	report(Progress{Stage: StageDone, Done: len(files), Total: len(files)})
	return Result{URL: site.URL, DeploymentID: depID, Uploaded: uploaded, Skipped: len(files) - uploaded}, nil
}

type manifestEntry struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// syncAssets runs one upload session and returns the completion token and
// how many files had to be uploaded.
func (c *Client) syncAssets(ctx context.Context, accountID, script string, files []fileEntry, report func(Progress)) (string, int, error) {
	manifest := make(map[string]manifestEntry, len(files))
	byHash := map[string]fileEntry{}
	for _, f := range files {
		manifest[f.name] = manifestEntry{Hash: f.hash, Size: f.size}
		if _, ok := byHash[f.hash]; !ok {
			byHash[f.hash] = f
		}
	}
	var sess struct {
		JWT     string     `json:"jwt"`
		Buckets [][]string `json:"buckets"`
	}
	_, err := c.do(ctx, request{method: http.MethodPost, path: scriptPath(accountID, script) + "/assets-upload-session",
		body: map[string]any{"manifest": manifest}}, &sess)
	if err != nil {
		return "", 0, fmt.Errorf("starting asset upload: %w", err)
	}
	if sess.JWT == "" {
		return "", 0, errors.New("starting asset upload: unexpected empty response from Cloudflare, please try again")
	}

	var buckets [][]fileEntry
	toUpload := 0
	for _, hb := range sess.Buckets {
		var b []fileEntry
		for _, h := range hb {
			f, ok := byHash[h]
			if !ok {
				return "", 0, fmt.Errorf("starting asset upload: Cloudflare requested unknown file hash %s", h)
			}
			b = append(b, f)
		}
		if len(b) > 0 {
			buckets = append(buckets, b)
			toUpload += len(b)
		}
	}
	total := len(files)
	if toUpload == 0 {
		report(Progress{Stage: StageUploading, Done: total, Total: total})
		return sess.JWT, 0, nil
	}
	single := singleAssetMode(sess.JWT)
	if single {
		var flat [][]fileEntry
		for _, b := range buckets {
			for _, f := range b {
				flat = append(flat, []fileEntry{f})
			}
		}
		buckets = flat
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		mu         sync.Mutex
		done       = total - toUpload
		completion string
		firstErr   error
		wg         sync.WaitGroup
		sem        = make(chan struct{}, uploadConcurrency(sess.JWT))
		gate       sync.Mutex // held during gateway / Retry-After pauses
	)
	report(Progress{Stage: StageUploading, Done: done, Total: total})
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
			tok, err := c.uploadBucket(ctx, accountID, sess.JWT, b, single, &gate)
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
		if errors.Is(firstErr, errSessionExpired) {
			return "", 0, firstErr
		}
		return "", 0, fmt.Errorf("uploading files: %w", firstErr)
	}
	if completion == "" {
		return "", 0, errors.New("uploading files: Cloudflare did not return a completion token, please try again")
	}
	return completion, toUpload, nil
}

func buildBulkBody(b []fileEntry) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range b {
		data, err := os.ReadFile(f.path)
		if err != nil {
			return nil, "", err
		}
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, f.hash, f.hash))
		h.Set("Content-Type", f.contentType)
		pw, err := w.CreatePart(h)
		if err != nil {
			return nil, "", err
		}
		if _, err := pw.Write([]byte(base64.StdEncoding.EncodeToString(data))); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

func (c *Client) uploadBucket(ctx context.Context, accountID, jwt string, b []fileEntry, single bool, gate *sync.Mutex) (string, error) {
	attempts, gateway := 0, 0
	for {
		gate.Lock()
		gate.Unlock() //nolint:staticcheck // barrier while another upload pauses

		var req request
		if single {
			data, err := os.ReadFile(b[0].path)
			if err != nil {
				return "", err
			}
			req = request{method: http.MethodPost, bearer: jwt, contentType: b[0].contentType, body: bytes.NewReader(data),
				path: "/accounts/" + url.PathEscape(accountID) + "/workers/assets/upload/" + b[0].hash}
		} else {
			body, ct, err := buildBulkBody(b)
			if err != nil {
				return "", err
			}
			req = request{method: http.MethodPost, bearer: jwt, contentType: ct, body: body,
				path: "/accounts/" + url.PathEscape(accountID) + "/workers/assets/upload?base64=true"}
		}
		var res struct {
			JWT string `json:"jwt"`
		}
		_, err := c.do(ctx, req, &res)
		if err == nil {
			return res.JWT, nil
		}
		var ae *APIError
		if errors.As(err, &ae) && ae.Status == http.StatusUnauthorized || jwtExpired(jwt) {
			return "", fmt.Errorf("%w: %w", errSessionExpired, err)
		}
		if ctx.Err() != nil || attempts >= maxUploadAttempts ||
			errors.Is(err, ErrFileTooLarge) || errors.Is(err, ErrTooManyFiles) || errors.Is(err, ErrMissingPermission) {
			return "", err
		}
		delay := backoff(attempts)
		if isGateway(err) {
			delay += time.Duration(1<<gateway) * 5 * time.Second
			gateway++
			if gateway >= maxGatewayErrors {
				attempts++
			}
		} else {
			attempts++
		}
		if ae != nil && ae.RetryAfter > 0 {
			delay = ae.RetryAfter // the API asked for a pause: honor it exactly
		}
		gate.Lock()
		werr := c.wait(ctx, delay)
		gate.Unlock()
		if werr != nil {
			return "", werr
		}
	}
}

// putScript uploads an assets-only Worker version (no code) exactly like
// wrangler's createWorkerUploadForm short-circuit for assets-only Workers.
func (c *Client) putScript(ctx context.Context, accountID, script, completion string, cfg map[string]any) (string, error) {
	compat := c.CompatibilityDate
	if compat == "" {
		compat = DefaultCompatibilityDate
	}
	meta, err := json.Marshal(map[string]any{
		"assets":             map[string]any{"jwt": completion, "config": cfg},
		"compatibility_date": compat,
	})
	if err != nil {
		return "", err
	}
	var lastErr error
	for attempt := 0; attempt < maxScriptAttempts; attempt++ {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		if err := w.WriteField("metadata", string(meta)); err != nil {
			return "", err
		}
		if err := w.Close(); err != nil {
			return "", err
		}
		var res struct {
			ID           string `json:"id"`
			DeploymentID string `json:"deployment_id"`
			Etag         string `json:"etag"`
		}
		_, err := c.do(ctx, request{method: http.MethodPut, path: scriptPath(accountID, script) + "?excludeScript=true",
			body: &buf, contentType: w.FormDataContentType()}, &res)
		if err == nil {
			switch {
			case res.DeploymentID != "":
				return res.DeploymentID, nil
			case res.ID != "":
				return res.ID, nil
			}
			return res.Etag, nil
		}
		lastErr = err
		var ae *APIError
		if !errors.As(err, &ae) || ae.Status < 500 || attempt == maxScriptAttempts-1 {
			break
		}
		if werr := c.wait(ctx, backoff(attempt)); werr != nil {
			return "", werr
		}
	}
	return "", fmt.Errorf("deploying worker %q: %w", script, lastErr)
}

// enableWorkersDev turns on https://<script>.<subdomain>.workers.dev/
// (same call and API-date header as wrangler's triggers/deploy.ts).
func (c *Client) enableWorkersDev(ctx context.Context, accountID, script string) error {
	_, err := c.do(ctx, request{
		method:  http.MethodPost,
		path:    scriptPath(accountID, script) + "/subdomain",
		body:    map[string]bool{"enabled": true, "previews_enabled": false},
		headers: map[string]string{"Cloudflare-Workers-Script-Api-Date": "2025-08-01"},
	}, nil)
	if err != nil {
		return fmt.Errorf("enabling the workers.dev address: %w", err)
	}
	return nil
}
