package cfpages

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
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Limits, taken from wrangler (packages/wrangler/src/pages/constants.ts) and
// https://developers.cloudflare.com/pages/platform/limits/.
const (
	MaxFileSize           = 25 << 20 // MAX_ASSET_SIZE: 25 MiB per file
	DefaultMaxFiles       = 20_000   // MAX_ASSET_COUNT_DEFAULT (Free plan); JWT claim may raise it
	maxBucketSize         = 40 << 20 // MAX_BUCKET_SIZE (raw bytes, before base64)
	maxBucketFiles        = 2000     // MAX_BUCKET_FILE_COUNT (non-Windows)
	uploadConcurrency     = 3        // BULK_UPLOAD_CONCURRENCY
	maxUploadAttempts     = 5        // MAX_UPLOAD_ATTEMPTS
	maxGatewayErrors      = 5        // MAX_UPLOAD_GATEWAY_ERRORS
	maxCheckMissingTries  = 5        // MAX_CHECK_MISSING_ATTEMPTS
	maxDeploymentAttempts = 3        // MAX_DEPLOYMENT_ATTEMPTS
)

// Progress stages reported to the callback.
const (
	StageScanning  = "scanning"  // walking and hashing files (Done/Total = files hashed)
	StageChecking  = "checking"  // asking which files Cloudflare already has
	StageUploading = "uploading" // Done/Total = files present remotely
	StageFinishing = "finishing" // upsert hashes
	StageDeploying = "deploying" // creating the deployment
	StageDone      = "done"
)

// Progress is reported during Deploy.
type Progress struct {
	Stage       string
	Done, Total int
}

// Result of a successful Deploy.
type Result struct {
	// URL is the production URL (https://<subdomain>/) when deploying the
	// production branch; otherwise the deployment's own URL.
	URL string
	// DeploymentURL is the unique URL of this deployment
	// (https://<hash>.<project>.pages.dev).
	DeploymentURL string
	DeploymentID  string
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

// ignored follows wrangler's validate.ts IGNORE_LIST. Root-only entries:
// _worker.js, _redirects, _headers, _routes.json, .wrangler (and
// "functions", which we do NOT ignore: we never compile Pages Functions, and
// a Hugo section named "functions" must still be published). Anywhere:
// .DS_Store, node_modules, .git.
func ignored(rel string) bool {
	switch path.Base(rel) {
	case ".DS_Store", "node_modules", ".git":
		return true
	}
	if !strings.Contains(rel, "/") {
		switch rel {
		case "_worker.js", "_redirects", "_headers", "_routes.json", ".wrangler":
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
		info, err := os.Stat(p) // follow symlinks, like wrangler's stat()
		if err != nil {
			return err
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		if info.Size() > MaxFileSize {
			return fmt.Errorf("%s is %d bytes, Pages accepts up to 25 MiB per file: %w", rel, info.Size(), ErrFileTooLarge)
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

// jwtSession holds the upload JWT and refreshes it on demand.
type jwtSession struct {
	c                  *Client
	accountID, project string
	mu                 sync.Mutex
	token              string
}

func (s *jwtSession) get() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}

func (s *jwtSession) refresh(ctx context.Context) error {
	var r struct {
		JWT string `json:"jwt"`
	}
	if _, err := s.c.do(ctx, http.MethodGet, projectPath(s.accountID, s.project)+"/upload-token", "", nil, "", &r); err != nil {
		return fmt.Errorf("getting upload token: %w", err)
	}
	if r.JWT == "" {
		return errors.New("getting upload token: empty jwt")
	}
	s.mu.Lock()
	s.token = r.JWT
	s.mu.Unlock()
	return nil
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

func jwtExpired(tok string) bool {
	exp, ok := jwtClaims(tok)["exp"].(float64)
	return ok && exp <= float64(time.Now().Unix())
}

func maxFilesFromJWT(tok string) int {
	if n, ok := jwtClaims(tok)["max_file_count_allowed"].(float64); ok && n > 0 {
		return int(n)
	}
	return DefaultMaxFiles
}

// needsNewJWT mirrors wrangler: refresh on Pages code 8000013 or expiry.
func needsNewJWT(err error, tok string) bool {
	var ae *APIError
	if errors.As(err, &ae) && (ae.hasCode(codePagesUnauthorized) || ae.Status == http.StatusUnauthorized) {
		return true
	}
	return jwtExpired(tok)
}

func isGateway(err error) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.Status == http.StatusBadGateway || ae.Status == http.StatusServiceUnavailable || ae.Status == http.StatusGatewayTimeout
	}
	return false
}

func backoff(attempt int) time.Duration { return time.Duration(1<<attempt) * time.Second }

// Deploy uploads dir as a new production deployment ("main" branch) of the
// Pages project and returns its URLs. progress may be nil.
func (c *Client) Deploy(ctx context.Context, accountID, project, dir string, progress func(Progress)) (Result, error) {
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
	proj, err := c.getProject(ctx, accountID, project)
	if err != nil {
		return Result{}, fmt.Errorf("looking up Pages project %q: %w", project, err)
	}

	sess := &jwtSession{c: c, accountID: accountID, project: project}
	if err := sess.refresh(ctx); err != nil {
		return Result{}, err
	}

	report(Progress{Stage: StageScanning})
	files, err := scan(dir, maxFilesFromJWT(sess.get()), report)
	if err != nil {
		return Result{}, err
	}
	hashes := make([]string, 0, len(files))
	seen := map[string]bool{}
	for _, f := range files {
		if !seen[f.hash] {
			seen[f.hash] = true
			hashes = append(hashes, f.hash)
		}
	}

	report(Progress{Stage: StageChecking, Total: len(files)})
	missing, err := c.checkMissing(ctx, sess, hashes)
	if err != nil {
		return Result{}, err
	}
	missingSet := map[string]bool{}
	for _, h := range missing {
		missingSet[h] = true
	}
	var toUpload []fileEntry
	queued := map[string]bool{}
	uploadedFiles := 0
	for _, f := range files {
		if missingSet[f.hash] {
			uploadedFiles++
			if !queued[f.hash] {
				queued[f.hash] = true
				toUpload = append(toUpload, f)
			}
		}
	}

	if err := c.uploadAll(ctx, sess, toUpload, len(files), len(files)-uploadedFiles, report); err != nil {
		return Result{}, err
	}

	report(Progress{Stage: StageFinishing, Done: len(files), Total: len(files)})
	// wrangler only warns when this fails: assets are uploaded, the next
	// deploy is merely slower. We ignore the error the same way.
	_ = c.upsertHashes(ctx, sess, hashes)

	report(Progress{Stage: StageDeploying, Done: len(files), Total: len(files)})
	manifest := make(map[string]string, len(files))
	for _, f := range files {
		manifest["/"+f.name] = f.hash
	}
	dep, err := c.createDeployment(ctx, accountID, project, dir, manifest)
	if err != nil {
		return Result{}, err
	}
	res := Result{
		URL:           dep.URL,
		DeploymentURL: dep.URL,
		DeploymentID:  dep.ID,
		Uploaded:      uploadedFiles,
		Skipped:       len(files) - uploadedFiles,
	}
	if proj.Subdomain != "" && (proj.ProductionBranch == "" || proj.ProductionBranch == ProductionBranch) {
		res.URL = "https://" + proj.Subdomain + "/"
	}
	report(Progress{Stage: StageDone, Done: len(files), Total: len(files)})
	return res, nil
}

func (c *Client) checkMissing(ctx context.Context, sess *jwtSession, hashes []string) ([]string, error) {
	var lastErr error
	for attempt := 0; attempt <= maxCheckMissingTries; attempt++ {
		var missing []string
		tok := sess.get()
		_, err := c.do(ctx, http.MethodPost, "/pages/assets/check-missing", tok, map[string]any{"hashes": hashes}, "", &missing)
		if err == nil {
			return missing, nil
		}
		lastErr = err
		if ctx.Err() != nil || attempt == maxCheckMissingTries {
			break
		}
		if werr := c.wait(ctx, backoff(attempt)); werr != nil {
			return nil, werr
		}
		if needsNewJWT(err, tok) {
			if rerr := sess.refresh(ctx); rerr != nil {
				return nil, rerr
			}
		}
	}
	return nil, fmt.Errorf("checking which files need upload: %w", lastErr)
}

type bucket struct {
	files     []fileEntry
	remaining int64
}

// makeBuckets reproduces wrangler's bucketing: biggest files first, spread
// round-robin over at least 3 buckets, each <= 40 MiB and <= 2000 files.
// Thousands of tiny files (e.g. a pagefind index) end up in a few buckets
// of up to 2000 files each.
func makeBuckets(files []fileEntry) []*bucket {
	sorted := append([]fileEntry(nil), files...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].size > sorted[j].size })
	buckets := make([]*bucket, uploadConcurrency)
	for i := range buckets {
		buckets[i] = &bucket{remaining: maxBucketSize}
	}
	offset := 0
	for _, f := range sorted {
		inserted := false
		for i := range buckets {
			b := buckets[(i+offset)%len(buckets)]
			if b.remaining >= f.size && len(b.files) < maxBucketFiles {
				b.files = append(b.files, f)
				b.remaining -= f.size
				inserted = true
				break
			}
		}
		if !inserted {
			buckets = append(buckets, &bucket{files: []fileEntry{f}, remaining: maxBucketSize - f.size})
		}
		offset++
	}
	out := buckets[:0]
	for _, b := range buckets {
		if len(b.files) > 0 {
			out = append(out, b)
		}
	}
	return out
}

type uploadItem struct {
	Key      string            `json:"key"`
	Value    string            `json:"value"`
	Metadata map[string]string `json:"metadata"`
	Base64   bool              `json:"base64"`
}

func (c *Client) uploadAll(ctx context.Context, sess *jwtSession, files []fileEntry, total, already int, report func(Progress)) error {
	report(Progress{Stage: StageUploading, Done: already, Total: total})
	if len(files) == 0 {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		mu       sync.Mutex
		done     = already
		firstErr error
		wg       sync.WaitGroup
		sem      = make(chan struct{}, uploadConcurrency)
		gwMu     sync.Mutex // held while backing off from gateway errors (concurrency -> 1)
	)
	for _, b := range makeBuckets(files) {
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
			err := c.uploadBucket(ctx, sess, b, &gwMu)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
					cancel()
				}
				return
			}
			done += len(b.files)
			report(Progress{Stage: StageUploading, Done: min(done, total), Total: total})
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return fmt.Errorf("uploading files: %w", firstErr)
	}
	if done < total {
		report(Progress{Stage: StageUploading, Done: total, Total: total})
	}
	return nil
}

func (c *Client) uploadBucket(ctx context.Context, sess *jwtSession, b *bucket, gwMu *sync.Mutex) error {
	payload := make([]uploadItem, 0, len(b.files))
	for _, f := range b.files {
		data, err := os.ReadFile(f.path)
		if err != nil {
			return err
		}
		payload = append(payload, uploadItem{
			Key:      f.hash,
			Value:    base64.StdEncoding.EncodeToString(data),
			Metadata: map[string]string{"contentType": f.contentType},
			Base64:   true,
		})
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	payload = nil
	attempts, gateway, refreshes := 0, 0, 0
	for {
		gwMu.Lock()
		gwMu.Unlock() //nolint:staticcheck // barrier: wait while another bucket backs off
		tok := sess.get()
		_, err := c.do(ctx, http.MethodPost, "/pages/assets/upload", tok, bytes.NewReader(body), "application/json", nil)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil || attempts >= maxUploadAttempts {
			return err
		}
		if werr := c.wait(ctx, backoff(attempts)); werr != nil {
			return werr
		}
		switch {
		case isGateway(err):
			gwMu.Lock()
			werr := c.wait(ctx, time.Duration(1<<gateway)*5*time.Second)
			gwMu.Unlock()
			if werr != nil {
				return werr
			}
			gateway++
			if gateway >= maxGatewayErrors {
				attempts++
			}
		case needsNewJWT(err, tok):
			if rerr := sess.refresh(ctx); rerr != nil {
				return rerr
			}
			// Like wrangler, a JWT refresh is not a failed attempt, but cap it
			// so a permanently rejected token cannot loop forever.
			refreshes++
			if refreshes >= maxGatewayErrors {
				attempts++
			}
		default:
			// 429 (rate limit) is retried; hard limits are not.
			if errors.Is(err, ErrFileTooLarge) || errors.Is(err, ErrTooManyFiles) || errors.Is(err, ErrMissingPermission) {
				return err
			}
			attempts++
		}
	}
}

func (c *Client) upsertHashes(ctx context.Context, sess *jwtSession, hashes []string) error {
	body := map[string]any{"hashes": hashes}
	tok := sess.get()
	_, err := c.do(ctx, http.MethodPost, "/pages/assets/upsert-hashes", tok, body, "", nil)
	if err == nil {
		return nil
	}
	if werr := c.wait(ctx, time.Second); werr != nil {
		return werr
	}
	if needsNewJWT(err, tok) {
		if rerr := sess.refresh(ctx); rerr != nil {
			return rerr
		}
	}
	_, err = c.do(ctx, http.MethodPost, "/pages/assets/upsert-hashes", sess.get(), body, "", nil)
	return err
}

type apiDeployment struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func (c *Client) createDeployment(ctx context.Context, accountID, project, dir string, manifest map[string]string) (apiDeployment, error) {
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return apiDeployment{}, err
	}
	// _headers and _redirects at the root go as their own form parts (they
	// are excluded from the asset manifest), exactly like wrangler; empty
	// files are skipped (wrangler's `if (_headers)`).
	extras := map[string][]byte{}
	for _, name := range []string{"_headers", "_redirects"} {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil && len(b) > 0 {
			extras[name] = b
		} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return apiDeployment{}, fmt.Errorf("reading %s: %w", name, err)
		}
	}
	build := func() (*bytes.Buffer, string, error) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		if err := w.WriteField("manifest", string(manifestJSON)); err != nil {
			return nil, "", err
		}
		if err := w.WriteField("branch", ProductionBranch); err != nil {
			return nil, "", err
		}
		for _, name := range []string{"_headers", "_redirects"} {
			b, ok := extras[name]
			if !ok {
				continue
			}
			fw, err := w.CreateFormFile(name, name)
			if err != nil {
				return nil, "", err
			}
			if _, err := fw.Write(b); err != nil {
				return nil, "", err
			}
		}
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return &buf, w.FormDataContentType(), nil
	}

	p := projectPath(accountID, project) + "/deployments"
	var lastErr error
	for attempt := 0; attempt < maxDeploymentAttempts; attempt++ {
		buf, ct, err := build()
		if err != nil {
			return apiDeployment{}, err
		}
		var dep apiDeployment
		_, err = c.do(ctx, http.MethodPost, p, "", buf, ct, &dep)
		if err == nil {
			return dep, nil
		}
		lastErr = err
		var ae *APIError
		if !errors.As(err, &ae) || !ae.hasCode(codePagesUnknown) || attempt == maxDeploymentAttempts-1 {
			break
		}
		if werr := c.wait(ctx, backoff(attempt)); werr != nil {
			return apiDeployment{}, werr
		}
	}
	return apiDeployment{}, fmt.Errorf("creating deployment: %w", lastErr)
}
