package pwa

import (
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" fill="#3b82f6"/></svg>`

func site(t *testing.T, scope string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	man, _ := json.Marshal(map[string]string{"scope": scope, "background_color": "#fafafa"})
	files["manifest.json"] = string(man)
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func readPNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestWriteFromSVG(t *testing.T) {
	dir := site(t, "/docs/", map[string]string{
		"blup-icon.svg":        svg,
		"index.html":           "<h1>home</h1>",
		"guide/index.html":     "<h1>guide</h1>",
		"404.html":             "nope",
		"css/site.css":         "body{}",
		"js/app.js":            "1",
		"pagefind/index.js":    "search",
		"photos/big.jpg":       "jpg",
		"vault/notes/file.svg": svg,
	})
	if err := Write(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "blup-icon.svg")); !os.IsNotExist(err) {
		t.Fatal("the favicon copy must be deleted")
	}
	for name, size := range map[string]int{"icon-192.png": 192, "icon-512.png": 512, "maskable-512.png": 512, "apple-touch-icon.png": 180} {
		img := readPNG(t, filepath.Join(dir, "icons", name))
		if img.Bounds().Dx() != size || img.Bounds().Dy() != size {
			t.Errorf("%s: %v", name, img.Bounds())
		}
		// The middle is the favicon's blue.
		if r, g, b, _ := img.At(size/2, size/2).RGBA(); r>>8 > 80 || g>>8 > 160 || b>>8 < 200 {
			t.Errorf("%s: middle is %d,%d,%d", name, r>>8, g>>8, b>>8)
		}
	}
	// Maskable: favicon inside the safe zone, background in the corner.
	if c := color.RGBAModel.Convert(readPNG(t, filepath.Join(dir, "icons", "maskable-512.png")).At(2, 2)).(color.RGBA); c != (color.RGBA{0xfa, 0xfa, 0xfa, 0xff}) {
		t.Errorf("maskable corner = %v", c)
	}

	ico, err := os.ReadFile(filepath.Join(dir, "favicon.ico"))
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint16(ico[2:]) != 1 || binary.LittleEndian.Uint16(ico[4:]) != 3 {
		t.Fatalf("ico header % x", ico[:6])
	}
	for i, want := range []byte{16, 32, 48} {
		entry := ico[6+16*i:]
		if entry[0] != want {
			t.Errorf("ico entry %d: %d", i, entry[0])
		}
		off := binary.LittleEndian.Uint32(entry[12:])
		if string(ico[off+1:off+4]) != "PNG" {
			t.Errorf("ico entry %d is not a PNG", i)
		}
	}

	sw, err := os.ReadFile(filepath.Join(dir, "sw.js"))
	if err != nil {
		t.Fatal(err)
	}
	js := string(sw)
	for _, want := range []string{`"/docs/"`, `"/docs/guide/"`, `"/docs/css/site.css"`, `"/docs/js/app.js"`, `"/docs/icons/icon-192.png"`, `"/docs/favicon.ico"`, `"/docs/manifest.json"`} {
		if !strings.Contains(js, want) {
			t.Errorf("sw.js misses %s", want)
		}
	}
	for _, not := range []string{"pagefind", "big.jpg", "vault", "404", "index.html", `"/docs/sw.js"`} {
		if strings.Contains(js[strings.Index(js, "FILES"):strings.Index(js, "const CACHE")], not) {
			t.Errorf("sw.js must not precache %s", not)
		}
	}
}

func TestWriteFromPNGAndVersion(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 64, 32)) // not square: centered
	for x := range 64 {
		for y := range 32 {
			src.Set(x, y, color.RGBA{0xff, 0, 0, 0xff})
		}
	}
	dir := site(t, "/", map[string]string{"index.html": "a"})
	out, _ := os.Create(filepath.Join(dir, "blup-icon.png"))
	png.Encode(out, src)
	out.Close()
	if err := Write(dir); err != nil {
		t.Fatal(err)
	}
	img := readPNG(t, filepath.Join(dir, "icons", "icon-192.png"))
	if _, _, _, a := img.At(96, 10).RGBA(); a != 0 {
		t.Error("a wide favicon leaves the top transparent")
	}
	if r, _, _, _ := img.At(96, 96).RGBA(); r>>8 < 200 {
		t.Error("the middle is the favicon")
	}
	first, _ := os.ReadFile(filepath.Join(dir, "sw.js"))

	// Same files: same version. A changed page: new version.
	if err := Write(dir); err != nil {
		t.Fatal(err)
	}
	same, _ := os.ReadFile(filepath.Join(dir, "sw.js"))
	if string(first) != string(same) {
		t.Error("the version must not change without changes")
	}
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("b"), 0o644)
	Write(dir)
	changed, _ := os.ReadFile(filepath.Join(dir, "sw.js"))
	if string(first) == string(changed) {
		t.Error("a changed page must change the version")
	}
}

func TestWriteWithoutManifest(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sw.js")); !os.IsNotExist(err) {
		t.Fatal("no manifest: no app")
	}
}
