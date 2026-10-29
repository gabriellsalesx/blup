// Package pwa turns a built site into an installable app that works
// offline: icons for every device made from the site's favicon, a
// favicon.ico, and a service worker that keeps the pages in the browser.
//
// The theme writes the manifest (manifest.json) and publishes the favicon
// as blup-icon.<ext>; Write reads both, writes the rest and deletes the
// copy of the favicon.
package pwa

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"  // favicon formats
	_ "image/jpeg" //
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// SourceName is the favicon copy the theme publishes (blup-icon.svg,
// blup-icon.png...).
const SourceName = "blup-icon"

// Write adds the icons and the service worker to the built site in dir.
// It does nothing when the theme wrote no manifest (an older theme).
func Write(dir string) error {
	man, err := readManifest(filepath.Join(dir, "manifest.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	src, err := findSource(dir)
	if err != nil {
		return err
	}
	if src != "" {
		err := writeIcons(dir, src, man.background())
		os.Remove(src) // internal: never published
		if err != nil {
			return fmt.Errorf("app icons: %w", err)
		}
	}
	return writeWorker(dir, man.Scope)
}

type manifest struct {
	Scope      string `json:"scope"`
	Background string `json:"background_color"`
}

func readManifest(path string) (manifest, error) {
	var m manifest
	b, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("%s: %w", path, err)
	}
	if !strings.HasSuffix(m.Scope, "/") {
		m.Scope += "/"
	}
	return m, nil
}

// background is the manifest's background color (#rrggbb), white otherwise.
func (m manifest) background() color.Color {
	h := strings.TrimPrefix(m.Background, "#")
	if v, err := strconv.ParseUint(h, 16, 32); err == nil && len(h) == 6 {
		return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}
	}
	return color.White
}

func findSource(dir string) (string, error) {
	m, err := filepath.Glob(filepath.Join(dir, SourceName+".*"))
	if err != nil || len(m) == 0 {
		return "", err
	}
	return m[0], nil
}

// icon draws the favicon into a rectangle of an image.
type icon func(dst *image.RGBA, r image.Rectangle)

func loadIcon(path string) (icon, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		svg, err := oksvg.ReadIconStream(bytes.NewReader(b), oksvg.IgnoreErrorMode)
		if err != nil {
			return nil, err
		}
		return func(dst *image.RGBA, r image.Rectangle) {
			x, y, w, h := fit(svg.ViewBox.W, svg.ViewBox.H, r)
			svg.SetTarget(x, y, w, h)
			sc := rasterx.NewScannerGV(dst.Bounds().Dx(), dst.Bounds().Dy(), dst, dst.Bounds())
			svg.Draw(rasterx.NewDasher(dst.Bounds().Dx(), dst.Bounds().Dy(), sc), 1)
		}, nil
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return func(dst *image.RGBA, r image.Rectangle) {
		sb := img.Bounds()
		x, y, w, h := fit(float64(sb.Dx()), float64(sb.Dy()), r)
		at := image.Rect(int(x), int(y), int(x+w), int(y+h))
		xdraw.CatmullRom.Scale(dst, at, img, sb, draw.Over, nil)
	}, nil
}

// fit centers a w×h shape in r, as large as fits.
func fit(w, h float64, r image.Rectangle) (x, y, fw, fh float64) {
	if w <= 0 || h <= 0 {
		w, h = 1, 1
	}
	rw, rh := float64(r.Dx()), float64(r.Dy())
	s := min(rw/w, rh/h)
	fw, fh = w*s, h*s
	return float64(r.Min.X) + (rw-fw)/2, float64(r.Min.Y) + (rh-fh)/2, fw, fh
}

// render draws the icon on a size×size image, inside a margin (fraction of
// the side), on bg (nil: transparent).
func render(ic icon, size int, margin float64, bg color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	if bg != nil {
		draw.Draw(img, img.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
	}
	m := int(float64(size) * margin)
	ic(img, image.Rect(m, m, size-m, size-m))
	return img
}

func writeIcons(dir, src string, bg color.Color) error {
	ic, err := loadIcon(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "icons"), 0o755); err != nil {
		return err
	}
	files := []struct {
		name   string
		size   int
		margin float64
		bg     color.Color
	}{
		{"icon-192.png", 192, 0, nil},
		{"icon-512.png", 512, 0, nil},
		// Android crops maskable icons to a circle or squircle: keep the
		// favicon in the middle 80% (the safe zone), on a full background.
		{"maskable-512.png", 512, 0.15, bg},
		// iOS adds rounded corners and shows no transparency.
		{"apple-touch-icon.png", 180, 0.1, bg},
	}
	for _, f := range files {
		if err := writePNG(filepath.Join(dir, "icons", f.name), render(ic, f.size, f.margin, f.bg)); err != nil {
			return err
		}
	}
	var ico []*image.RGBA
	for _, s := range []int{16, 32, 48} {
		ico = append(ico, render(ic, s, 0, nil))
	}
	return writeICO(filepath.Join(dir, "favicon.ico"), ico)
}

func writePNG(path string, img image.Image) error {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return err
	}
	return os.WriteFile(path, b.Bytes(), 0o644)
}

// writeICO writes the images as one .ico (PNG entries, as every browser
// since IE 9 reads them).
func writeICO(path string, imgs []*image.RGBA) error {
	var head, data bytes.Buffer
	binary.Write(&head, binary.LittleEndian, [3]uint16{0, 1, uint16(len(imgs))})
	offset := 6 + 16*len(imgs)
	for _, img := range imgs {
		var p bytes.Buffer
		if err := png.Encode(&p, img); err != nil {
			return err
		}
		w := img.Bounds().Dx()
		head.Write([]byte{byte(w), byte(w), 0, 0})           // width, height, palette, reserved
		binary.Write(&head, binary.LittleEndian, uint16(1))  // planes
		binary.Write(&head, binary.LittleEndian, uint16(32)) // bits per pixel
		binary.Write(&head, binary.LittleEndian, uint32(p.Len()))
		binary.Write(&head, binary.LittleEndian, uint32(offset+data.Len()))
		data.Write(p.Bytes())
	}
	return os.WriteFile(path, append(head.Bytes(), data.Bytes()...), 0o644)
}

// precached: what the app downloads on install so the site opens offline.
// Pages, styles, scripts, fonts and icons; photos and the search index are
// kept as they are seen (a site can have many and large ones).
var precached = []string{".css", ".js", ".woff2", ".svg", ".ico", ".json"}

// writeWorker writes sw.js: the list of files to keep, and a version (a
// hash of them) that makes browsers update the copy after each publish.
func writeWorker(dir, scope string) error {
	var urls []string
	h := sha256.New()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == "pagefind" || rel == "vault" {
				return filepath.SkipDir
			}
			return nil
		}
		var url string
		switch {
		case rel == "index.html":
			url = scope
		case strings.HasSuffix(rel, "/index.html"):
			// Pretty URLs: hosts redirect ".../index.html" to ".../".
			url = scope + strings.TrimSuffix(rel, "index.html")
		case strings.HasPrefix(rel, "icons/") || slices.Contains(precached, filepath.Ext(rel)):
			if rel == "sw.js" {
				return nil
			}
			url = scope + rel
		default:
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(url))
		h.Write(b)
		urls = append(urls, url)
		return nil
	})
	if err != nil {
		return err
	}
	slices.Sort(urls)
	list, _ := json.Marshal(urls)
	sc, _ := json.Marshal(scope)
	js := strings.NewReplacer(
		"__VERSION__", hex.EncodeToString(h.Sum(nil))[:16],
		"__SCOPE__", string(sc),
		"__FILES__", string(list),
	).Replace(worker)
	return os.WriteFile(filepath.Join(dir, "sw.js"), []byte(js), 0o644)
}

// worker: pages come from the network when online (always fresh) and from
// the copy when offline; other files from the copy first. GitHub and the
// login service always go to the network.
const worker = `// Made by blup: keeps the site working offline.
const VERSION = "__VERSION__";
const SCOPE = __SCOPE__;
const FILES = __FILES__;
const CACHE = "blup-" + VERSION;
const SEEN = "blup-seen";

self.addEventListener("install", (e) => {
  // One by one: a file that fails doesn't stop the others.
  e.waitUntil(caches.open(CACHE).then((c) => Promise.allSettled(FILES.map((f) => c.add(f)))).then(() => self.skipWaiting()));
});

self.addEventListener("activate", (e) => {
  e.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(keys.filter((k) => k.startsWith("blup-") && k !== CACHE && k !== SEEN).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (e) => {
  const req = e.request;
  const url = new URL(req.url);
  if (req.method !== "GET" || url.origin !== location.origin || !url.pathname.startsWith(SCOPE)) return;
  if (req.mode === "navigate") {
    e.respondWith(
      fetch(req)
        .then((res) => {
          if (res.ok) { const copy = res.clone(); caches.open(CACHE).then((c) => c.put(req, copy)); }
          return res;
        })
        .catch(() => caches.match(req, { ignoreSearch: true }).then((hit) => hit || caches.match(SCOPE))),
    );
    return;
  }
  e.respondWith(
    caches.match(req).then((hit) => hit || fetch(req).then((res) => {
      if (res.ok && res.type === "basic") { const copy = res.clone(); caches.open(SEEN).then((c) => c.put(req, copy)); }
      return res;
    })),
  );
});
`
