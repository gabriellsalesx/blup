package kokoro

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// The local voice's downloads, pinned: ONNX Runtime from Microsoft's
// releases, the tools (eSpeak NG and LAME, built from their official
// sources by studio/build/voice/build-tools.sh) from Blup's releases, the
// model and voices from Hugging Face at a fixed revision. Every file is
// SHA-256 checked before it is used.

// Version is the layout of an installation; a newer Blup that changes the
// downloads bumps it and the user downloads again.
const Version = "2" // 2: Spanish (its voices and eSpeak NG dictionary)

type download struct {
	url    string
	sha256 string
	size   int64
}

// platform is what a system needs besides the model and voices.
type platform struct {
	ort        download
	ortLib     string // the library inside ONNX Runtime's archive
	ortLicense string
	tools      download // zero when not built for this system yet
}

const ortRelease = "https://github.com/microsoft/onnxruntime/releases/download/v1.31.1/"
const toolsRelease = "https://github.com/gabriellsalesx/blup/releases/download/voice-tools-1/"

var platforms = map[string]platform{
	"linux/amd64": {
		ort:        download{ortRelease + "onnxruntime-linux-x64-1.31.1.tgz", "c849e36a1ef3e22d4ff217b9d1ce4a3cf5b9467b975610913ae959ad5b16bdc9", 11413655},
		ortLib:     "onnxruntime-linux-x64-1.31.1/lib/libonnxruntime.so.1.31.1",
		ortLicense: "onnxruntime-linux-x64-1.31.1/LICENSE",
		tools:      download{toolsRelease + "blup-voice-tools-linux-amd64.tar.gz", "43e5f9490eb284585f28253b8a299f75b0c4e979eeba8d3aac1b50d54c84a591", 1777126},
	},
	// Built by the release workflow (macOS and Windows runners), then
	// added here with their checksums.
	"linux/arm64": {
		ort:        download{ortRelease + "onnxruntime-linux-aarch64-1.31.1.tgz", "5317505e2da99be65f8e2775ba3c2afd6d2961fd78aba1bd8f56159f48820c78", 10332381},
		ortLib:     "onnxruntime-linux-aarch64-1.31.1/lib/libonnxruntime.so.1.31.1",
		ortLicense: "onnxruntime-linux-aarch64-1.31.1/LICENSE",
	},
	"darwin/arm64": {
		ort:        download{ortRelease + "onnxruntime-osx-arm64-1.31.1.tgz", "2e1344a1926ee50eb25b38153cb6992c3567515aef0e677a1a520a1798896d6f", 36936435},
		ortLib:     "onnxruntime-osx-arm64-1.31.1/lib/libonnxruntime.1.31.1.dylib",
		ortLicense: "onnxruntime-osx-arm64-1.31.1/LICENSE",
	},
}

var hf = "https://huggingface.co/onnx-community/Kokoro-82M-v1.0-ONNX/resolve/1939ad2a8e416c0acfeecc08a694d14ef25f2231/"

// model is Kokoro 82M quantized to 8 bits: the same voice as the full
// model at a quarter of the size (the 16-bit one breaks on CPUs).
var model = download{hf + "onnx/model_quantized.onnx", "fbae9257e1e05ffc727e951ef9b9c98418e6d79f1c9b6b13bd59f5c9028a1478", 92361116}

var voiceFiles = map[string]string{
	"pf_dora":    "3da7b5b2d91847ebf5646f57631af6ececae3c29a89cd300f06edf9aa6cfe9ee",
	"pm_alex":    "0175c753f59c54e7fd5a995bedef0c5ff2fb67e0043dd3dcb2ae74ec2acbeb2a",
	"pm_santa":   "8b012db3185778afe2e45a62cbad69db73021774fe68dda634bcc748a982eede",
	"af_heart":   "d583ccff3cdca2f7fae535cb998ac07e9fcb90f09737b9a41fa2734ec44a8f0b",
	"af_bella":   "f69d836209b78eb8c66e75e3cda491e26ea838a3674257e9d4e5703cbaf55c8b",
	"am_michael": "1d1f21dd8da39c30705cd4c75d039d265e9bc4a2a93ed09bc9e1b1225eb95ba1",
	"am_fenrir":  "c27989f741f7ee34d273a39d8a595cc0837d35f5ced9a29b7cc162614616df43",
	"bf_emma":    "669fe0647f9dd04fcab92f1439a40eeb4c8b4ab1f82e4996fe3d918ce4a63b73",
	"bm_george":  "c4b235a4c1f2cd3b939fed08b899ce9385638b763f7b73a59616c4fc9bd6c9bc",
	"ef_dora":    "f66ec66bd295acb18372e37008533a9a3228483ccd294e7538d5d9294ac9a532",
	"em_alex":    "27809e9eafdcbcfff90a3016c697568676531de2a2c39cee29c96c7bd6b83e95",
	"em_santa":   "ad43b774e1ca24d05c6161297d8aeb770ac3d29bb95daf516727af5f7d543683",
}

const voiceSize = maxTokens * 256 * 4

// ToolsOverride, when set (BLUP_VOICE_TOOLS: a path or URL), replaces the
// tools package's address, to try a build before it is published. Its
// checksum is still checked.
var ToolsOverride = os.Getenv("BLUP_VOICE_TOOLS")

// current is this system's platform; ok is false when it has none (no
// package of tools yet, or the session can't load ONNX Runtime).
func current() (platform, bool) {
	p, ok := platforms[runtime.GOOS+"/"+runtime.GOARCH]
	if !ok || p.tools.url == "" {
		return p, false
	}
	if _, err := openSession("", ""); errors.Is(err, ErrUnsupported) {
		return p, false
	}
	return p, true
}

// Supported tells whether the local voice can be installed here.
func Supported() bool {
	_, ok := current()
	return ok
}

// DownloadSize is the size of everything Install downloads, in bytes.
func DownloadSize() int64 {
	p, ok := current()
	if !ok {
		return 0
	}
	return p.ort.size + p.tools.size + model.size + int64(len(voiceFiles))*voiceSize
}

type stamp struct {
	Version string `json:"version"`
	Library string `json:"library"`
}

// Installed tells whether dir holds a complete installation.
func Installed(dir string) bool {
	_, err := libraryPath(dir)
	return err == nil
}

func libraryPath(dir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(dir, "ready.json"))
	if err != nil {
		return "", ErrNotInstalled
	}
	var s stamp
	if json.Unmarshal(b, &s) != nil || s.Version != Version {
		return "", ErrNotInstalled
	}
	lib := filepath.Join(dir, "ort", s.Library)
	for _, f := range []string{lib, filepath.Join(dir, "model.onnx"), filepath.Join(dir, "tools", exe("espeak-ng")), filepath.Join(dir, "tools", exe("lame"))} {
		if _, err := os.Stat(f); err != nil {
			return "", ErrNotInstalled
		}
	}
	return lib, nil
}

// Progress reports Install: bytes done of total, and what is downloading.
type Progress func(done, total int64, step string)

// Install downloads everything into dir. It works in dir+".part" and
// swaps it in at the end, so a failure leaves the old state as it was.
func Install(ctx context.Context, dir string, progress Progress) error {
	p, ok := current()
	if !ok {
		return ErrUnsupported
	}
	if ToolsOverride != "" {
		p.tools.url = ToolsOverride
	}
	tmp := dir + ".part"
	os.RemoveAll(tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "voices"), 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	total := DownloadSize()
	var done int64
	get := func(d download, step string) ([]byte, error) {
		b, err := fetch(ctx, d, func(n int64) {
			if progress != nil {
				progress(done+n, total, step)
			}
		})
		done += d.size
		return b, err
	}

	ort, err := get(p.ort, "runtime")
	if err != nil {
		return err
	}
	lib := path.Base(p.ortLib)
	if err := extract(ort, p.ort.url, map[string]string{p.ortLib: filepath.Join(tmp, "ort", lib), p.ortLicense: filepath.Join(tmp, "ort", "LICENSE")}); err != nil {
		return err
	}
	tools, err := get(p.tools, "tools")
	if err != nil {
		return err
	}
	if err := extract(tools, p.tools.url, map[string]string{"*": filepath.Join(tmp, "tools")}); err != nil {
		return err
	}
	m, err := get(model, "model")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, "model.onnx"), m, 0o644); err != nil {
		return err
	}
	for id, sum := range voiceFiles {
		v, err := get(download{hf + "voices/" + id + ".bin", sum, voiceSize}, "voices")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(tmp, "voices", id+".bin"), v, 0o644); err != nil {
			return err
		}
	}
	st, _ := json.Marshal(stamp{Version, lib})
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

// fetch downloads d (or reads it, for a local path) and checks its SHA-256.
func fetch(ctx context.Context, d download, progress func(int64)) ([]byte, error) {
	var r io.Reader
	if strings.HasPrefix(d.url, "http://") || strings.HasPrefix(d.url, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.url, nil)
		if err != nil {
			return nil, err
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("download %s: HTTP %d", path.Base(d.url), res.StatusCode)
		}
		r = res.Body
	} else {
		f, err := os.Open(strings.TrimPrefix(d.url, "file://"))
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}
	h := sha256.New()
	var buf bytes.Buffer
	chunk := make([]byte, 256<<10)
	var n int64
	for {
		k, err := r.Read(chunk)
		if k > 0 {
			buf.Write(chunk[:k])
			h.Write(chunk[:k])
			n += int64(k)
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
	if got := hex.EncodeToString(h.Sum(nil)); got != d.sha256 {
		return nil, fmt.Errorf("%s failed the integrity check (SHA-256 %s, expected %s)", path.Base(d.url), got, d.sha256)
	}
	return buf.Bytes(), nil
}

// extract copies files out of a .tgz or .zip: want maps an archive path to
// its destination, or "*" to a folder that receives everything.
func extract(data []byte, name string, want map[string]string) error {
	put := func(p string, mode os.FileMode, r io.Reader) error {
		p = strings.TrimPrefix(path.Clean("/"+p), "/")
		dest, ok := want[p]
		if all, star := want["*"]; star {
			dest, ok = filepath.Join(all, filepath.FromSlash(p)), true
			if !strings.HasPrefix(dest, filepath.Clean(all)+string(filepath.Separator)) {
				return fmt.Errorf("unsafe path in %s: %s", path.Base(name), p)
			}
		}
		if !ok {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode|0o600)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, r); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	}
	if strings.HasSuffix(name, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return err
		}
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}
			err = put(f.Name, f.Mode().Perm(), rc)
			rc.Close()
			if err != nil {
				return err
			}
		}
		return checkExtracted(want)
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		if err := put(h.Name, os.FileMode(h.Mode).Perm(), tr); err != nil {
			return err
		}
	}
	return checkExtracted(want)
}

func checkExtracted(want map[string]string) error {
	for k, dest := range want {
		if k == "*" {
			continue
		}
		if _, err := os.Stat(dest); err != nil {
			return fmt.Errorf("%s missing from the download", k)
		}
	}
	return nil
}
