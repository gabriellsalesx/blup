// Package localtr installs the translation that runs on the computer, for
// free and without internet: Firefox's own translation engine (Bergamot,
// Marian compiled to WebAssembly) and its models, from Mozilla, the same
// files Firefox downloads (Remote Settings, pinned here with their SHA-256).
// The Studio's window runs them in a worker (public/bergamot); this package
// downloads, checks and unpacks them, and serves them to the window.
//
// The models translate between English and Portuguese and between English
// and Spanish; Portuguese to Spanish goes through English (the engine
// chains two models). Engine and models are MPL-2.0 (Mozilla).
package localtr

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// Version is the layout of an installation; a Blup that changes the files
// bumps it and the person downloads again.
const Version = "1"

const cdn = "https://firefox-settings-attachments.cdn.mozilla.net/"

// File is one download: zstd compressed (Size, SHA256) and as kept on disk
// (Raw, RawSHA256).
type File struct {
	Pair      string // "enpt": English to Portuguese; "" for the engine
	Type      string // model, lex, vocab, wasm
	Name      string
	URL       string
	Size      int64
	SHA256    string
	Raw       int64
	RawSHA256 string
}

// Engine is Bergamot 4.0 (wasm); its JavaScript glue ships with the Studio.
var Engine = File{Type: "wasm", Name: "bergamot-translator.wasm", URL: cdn + "main-workspace/translations-wasm-v2/a76f9efe-b185-4a7f-9c3b-5f5d216d12e6.zst", Size: 1211955, SHA256: "327fcfc7b7e9d95f6fa0844ebf899cfc05469872ef02d3aef5783182839a6255", Raw: 4960506, RawSHA256: "f38ef807636a7c994afedaab7ff8ffe0d590d21897f05139f747b80fd7bbe926"}

// Models are version 3.0 ("base-memory") of each direction.
var Models = []File{
	{Pair: "enes", Type: "lex", Name: "lex.50.50.enes.s2t.bin", URL: cdn + "main-workspace/translations-models-v2/51318160-1249-451f-80fb-12e61f8c1def.zst", Size: 1687731, SHA256: "0dd2945dc8a5bbccad6fb1e1dfce137d363de839f4492cab717f60f1f38c09e6", Raw: 4198436, RawSHA256: "7d51237c0a07027dcd61643cfbbb0f8c48597d19907ef53d2cae9d6bec2cf25c"},
	{Pair: "enes", Type: "model", Name: "model.enes.intgemm.alphas.bin", URL: cdn + "main-workspace/translations-models-v2/1d705201-9be0-40c4-a0b4-18d1e3973777.zst", Size: 22085725, SHA256: "ce7ba731d3352d7e595d96ed4fac513744568eaf9a271daf90b5fa8fa33b195d", Raw: 31561787, RawSHA256: "3b1c399511c01c84c36fae5c0524df44096288efdc8236e182b5c97d7ad2244c"},
	{Pair: "enes", Type: "vocab", Name: "vocab.enes.spm", URL: cdn + "main-workspace/translations-models-v2/b2b5907b-8759-4cc8-a721-89c283e6e45b.zst", Size: 348996, SHA256: "76b9ef220b22cec379f097852ca6312cabf065d5ddfb05149d5af31aa050ec22", Raw: 816054, RawSHA256: "5ae254fa9b15aa182e70fd2a6186b1333c63a29a48043a9224c6aa4fcac058ad"},
	{Pair: "enpt", Type: "lex", Name: "lex.50.50.enpt.s2t.bin", URL: cdn + "main-workspace/translations-models-v2/94532a3d-b009-4b56-9ffb-213a052c22fe.zst", Size: 1588521, SHA256: "c51d8fa1524a4fd70ac32dc6ef8df2a7683ec8da1b7888ef0b82dc1fcbafd14b", Raw: 3970340, RawSHA256: "ccb4c31c9e1899d77a200e71a86958e6ef6c8649627d0bbf3f873b57d9f236bd"},
	{Pair: "enpt", Type: "model", Name: "model.enpt.intgemm.alphas.bin", URL: cdn + "main-workspace/translations-models-v2/90cb3b8f-d3bd-4584-9b49-f3072a952582.zst", Size: 22901341, SHA256: "b6ae4dfe27a5fd5bfaaeb4d84c4297e69c83a329e7eb26ae6ceefafe5ab541c9", Raw: 31561787, RawSHA256: "07892fd2544ee79dcb643615d8f2debb9793fae16842e87c328e27a3dd26a770"},
	{Pair: "enpt", Type: "vocab", Name: "vocab.enpt.spm", URL: cdn + "main-workspace/translations-models-v2/1ae15c1a-90b6-4c32-8dfa-50dbb710904c.zst", Size: 348792, SHA256: "5b2ef32ade5c06aa3503186dc8c73e0ff21374e097a071f04d73f8bb1ab0e052", Raw: 816726, RawSHA256: "d9f46182823d5bbc84201252b2dfcac28f63e561f0ec827ed858f241864c9def"},
	{Pair: "esen", Type: "lex", Name: "lex.50.50.esen.s2t.bin", URL: cdn + "main-workspace/translations-models-v2/64f16997-3cf3-4cc8-a069-b143760b1173.zst", Size: 1880457, SHA256: "5c3aae52d8fddeb2e09098171cdecf059ea453474633edcebc77375b04cbcaee", Raw: 4636248, RawSHA256: "e2610211d3b9577d012638fe7e7e74ed7b4b708ce96b9e792e67c282a6492daa"},
	{Pair: "esen", Type: "model", Name: "model.esen.intgemm.alphas.bin", URL: cdn + "main-workspace/translations-models-v2/fccccf6a-6d29-407d-82a5-0cf73b23d892.zst", Size: 22849210, SHA256: "7d1a512f3d3c420a0d87e52dccf6c83bf7eba7f4c54608824d3f86b757993a3e", Raw: 31561787, RawSHA256: "4aed7734152ae0045d1a69ae49c86cfda18f53c61f90e95e1d1de1c7c7c3b033"},
	{Pair: "esen", Type: "vocab", Name: "vocab.esen.spm", URL: cdn + "main-workspace/translations-models-v2/2239f64b-dfc4-41c1-9e06-d532dce7b246.zst", Size: 348996, SHA256: "76b9ef220b22cec379f097852ca6312cabf065d5ddfb05149d5af31aa050ec22", Raw: 816054, RawSHA256: "5ae254fa9b15aa182e70fd2a6186b1333c63a29a48043a9224c6aa4fcac058ad"},
	{Pair: "pten", Type: "lex", Name: "lex.50.50.pten.s2t.bin", URL: cdn + "main-workspace/translations-models-v2/35d7b065-8aca-4f7e-82dd-9b2cc287a7f6.zst", Size: 1850036, SHA256: "30d224e25a2e1bdc3aa31af7398f4e317d96d71d463efe3284d4aef230f951c7", Raw: 4634492, RawSHA256: "2685d8b6530be92a4db4cc61f15a097eb114552be15cb6c1699e9d2d99d24470"},
	{Pair: "pten", Type: "model", Name: "model.pten.intgemm.alphas.bin", URL: cdn + "main-workspace/translations-models-v2/193c17c2-1c17-4fd3-b810-7de7d376a4ea.zst", Size: 22030075, SHA256: "f2543035eb4e0cf7814d373e7ae69a10d6b48bdcb47121b372ae7b796bbdf9e6", Raw: 31561787, RawSHA256: "7b854f1ec5a485dd33efd7c1bc01dd7d5a57f566957c5e47722af333f0ce9157"},
	{Pair: "pten", Type: "vocab", Name: "vocab.pten.spm", URL: cdn + "main-workspace/translations-models-v2/ae51a6b3-800f-4198-8b27-c805216f1979.zst", Size: 348792, SHA256: "5b2ef32ade5c06aa3503186dc8c73e0ff21374e097a071f04d73f8bb1ab0e052", Raw: 816726, RawSHA256: "d9f46182823d5bbc84201252b2dfcac28f63e561f0ec827ed858f241864c9def"},
}

// ErrNotInstalled means the local translation was not downloaded.
var ErrNotInstalled = errors.New("local translation not installed")

// DownloadSize is what Install downloads, in bytes.
func DownloadSize() int64 {
	n := Engine.Size
	for _, f := range Models {
		n += f.Size
	}
	return n
}

// Installed tells whether dir holds a complete installation.
func Installed(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "ready.json"))
	if err != nil {
		return false
	}
	var s struct{ Version string }
	if json.Unmarshal(b, &s) != nil || s.Version != Version {
		return false
	}
	for _, f := range append([]File{Engine}, Models...) {
		if st, err := os.Stat(filepath.Join(dir, f.Name)); err != nil || st.Size() != f.Raw {
			return false
		}
	}
	return true
}

// Progress reports Install: bytes done of total.
type Progress func(done, total int64)

// Install downloads, checks and unpacks everything into dir. It works in
// dir+".part" and swaps it in at the end, so a failure changes nothing.
func Install(ctx context.Context, dir string, progress Progress) error {
	tmp := dir + ".part"
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return err
	}
	defer dec.Close()
	total, done := DownloadSize(), int64(0)
	for _, f := range append([]File{Engine}, Models...) {
		packed, err := fetch(ctx, f, func(n int64) {
			if progress != nil {
				progress(done+n, total)
			}
		})
		if err != nil {
			return err
		}
		done += f.Size
		raw, err := dec.DecodeAll(packed, make([]byte, 0, f.Raw))
		if err != nil {
			return fmt.Errorf("%s: %w", f.Name, err)
		}
		if sum := sha256.Sum256(raw); hex.EncodeToString(sum[:]) != f.RawSHA256 || int64(len(raw)) != f.Raw {
			return fmt.Errorf("%s failed the integrity check after unpacking", f.Name)
		}
		if err := os.WriteFile(filepath.Join(tmp, f.Name), raw, 0o644); err != nil {
			return err
		}
	}
	st, _ := json.Marshal(map[string]string{"version": Version})
	if err := os.WriteFile(filepath.Join(tmp, "ready.json"), st, 0o644); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.Rename(tmp, dir)
}

// Remove deletes the installation.
func Remove(dir string) error { return os.RemoveAll(dir) }

// fetch downloads f and checks the SHA-256 of what came.
func fetch(ctx context.Context, f File, progress func(int64)) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.URL, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: HTTP %d", f.Name, res.StatusCode)
	}
	h := sha256.New()
	var buf bytes.Buffer
	buf.Grow(int(f.Size))
	chunk := make([]byte, 256<<10)
	var n int64
	for {
		k, err := res.Body.Read(chunk)
		if k > 0 {
			buf.Write(chunk[:k])
			h.Write(chunk[:k])
			n += int64(k)
			if n > f.Size {
				return nil, fmt.Errorf("download %s: larger than expected", f.Name)
			}
			if progress != nil {
				progress(n)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != f.SHA256 {
		return nil, fmt.Errorf("%s failed the integrity check (SHA-256 %s, expected %s)", f.Name, got, f.SHA256)
	}
	return buf.Bytes(), nil
}

// Serve answers the window's requests for the files at /local-translate/
// (the engine and the models, read from dir), and passes on the others.
func Serve(dir func() string, next http.Handler) http.Handler {
	known := map[string]bool{Engine.Name: true}
	for _, f := range Models {
		known[f.Name] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, ok := strings.CutPrefix(r.URL.Path, "/local-translate/")
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		// Only the files of the installation, never another path.
		if !known[name] {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, filepath.Join(dir(), name))
	})
}
