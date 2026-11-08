package kokoro

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestChunk(t *testing.T) {
	ps := strings.Repeat("ola mundo, ", 80) + "fim."
	parts := chunk(ps, 100)
	if len(parts) < 8 {
		t.Fatalf("%d parts", len(parts))
	}
	for _, p := range parts {
		if len(p) > 100 {
			t.Fatalf("part of %d symbols", len(p))
		}
	}
	if s := string(parts[0]); !strings.HasSuffix(s, ",") {
		t.Errorf("should cut after a comma: %q", s)
	}
	if got := chunk("  ", 10); len(got) != 0 {
		t.Errorf("empty: %v", got)
	}
}

func TestPauses(t *testing.T) {
	if pauseFor(0, 3) != 0.9 || pauseFor(1, 3) != 0.6 || pauseFor(2, 3) != 0 || pauseFor(0, 1) != 0 {
		t.Fatal("pauses: title 0.9 s, paragraphs 0.6 s, none at the end")
	}
}

func TestToKokoroPT(t *testing.T) {
	// eSpeak NG's IPA (tied with ^) to Kokoro's letters, as misaki does.
	in := "ˌest^ʃy ɛ ʊ blˈup lˈe^ɪŋdw ʊ se^ʊ pˈost^ʃ vˈɔz ˈa^ʊtæ stud^ʒˈi^ʊ"
	want := "ˌesʧy ɛ ʊ blˈup lˈAŋdw ʊ seʊ pˈosʧ vˈɔz ˈWtæ stuʤˈiʊ"
	if got := toKokoroPT(in); got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if got := keepKnown("abc☃ ʧ"); got != "abc ʧ" {
		t.Errorf("keepKnown: %q", got)
	}
	if espeakVoice("pf_dora") != "pt-br" || espeakVoice("bf_emma") != "en-gb" || espeakVoice("am_michael") != "en-us" {
		t.Error("espeakVoice")
	}
}

func TestLoadVoice(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.bin")
	os.WriteFile(p, make([]byte, 100), 0o644)
	if _, err := loadVoice(p); err == nil {
		t.Fatal("a short file is not a voice")
	}
	os.WriteFile(p, make([]byte, voiceSize), 0o644)
	if v, err := loadVoice(p); err != nil || len(v) != maxTokens*256 {
		t.Fatalf("%d %v", len(v), err)
	}
}

func targz(t *testing.T, files map[string]string) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
		tw.Write([]byte(body))
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func TestInstall(t *testing.T) {
	key := runtime.GOOS + "/" + runtime.GOARCH
	if _, ok := current(); !ok {
		t.Skip("no local voice on", key)
	}
	ort := targz(t, map[string]string{"ort/lib/libonnxruntime.so": "lib", "ort/LICENSE": "MIT"})
	tools := targz(t, map[string]string{exe("espeak-ng"): "e", exe("lame"): "l", "espeak-ng-data/pt_dict": "d"})
	modelBytes := []byte("model")
	voiceBytes := make([]byte, voiceSize)
	files := map[string][]byte{"/ort.tgz": ort, "/tools.tar.gz": tools, "/onnx/model_quantized.onnx": modelBytes}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/voices/") {
			w.Write(voiceBytes)
			return
		}
		if b, ok := files[r.URL.Path]; ok {
			w.Write(b)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	oldP, oldModel, oldHF, oldVoices := platforms[key], model, hf, voiceFiles
	defer func() { platforms[key], model, hf, voiceFiles = oldP, oldModel, oldHF, oldVoices }()
	platforms[key] = platform{
		ort:        download{srv.URL + "/ort.tgz", sum(ort), int64(len(ort))},
		ortLib:     "ort/lib/libonnxruntime.so",
		ortLicense: "ort/LICENSE",
		tools:      download{srv.URL + "/tools.tar.gz", sum(tools), int64(len(tools))},
	}
	hf = srv.URL + "/"
	model = download{hf + "onnx/model_quantized.onnx", sum(modelBytes), int64(len(modelBytes))}
	voiceFiles = map[string]string{"pf_dora": sum(voiceBytes)}

	dir := filepath.Join(t.TempDir(), "kokoro")
	var last int64
	if err := Install(context.Background(), dir, func(done, total int64, step string) { last = done }); err != nil {
		t.Fatal(err)
	}
	if !Installed(dir) || last != DownloadSize() {
		t.Fatalf("installed %v, progress %d of %d", Installed(dir), last, DownloadSize())
	}
	for _, f := range []string{"ort/libonnxruntime.so", "ort/LICENSE", "tools/espeak-ng-data/pt_dict", "voices/pf_dora.bin", "model.onnx"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f))); err != nil {
			t.Error(err)
		}
	}
	if info, _ := os.Stat(filepath.Join(dir, "tools", exe("lame"))); info == nil || info.Mode().Perm()&0o100 == 0 {
		t.Error("the tools must stay executable")
	}

	// A file that doesn't match its checksum is refused and the old
	// installation stays.
	modelBytes[0] = 'M'
	if err := Install(context.Background(), dir, nil); err == nil || !strings.Contains(err.Error(), "integrity") {
		t.Fatalf("tampered model: %v", err)
	}
	if !Installed(dir) {
		t.Fatal("a failed update must keep the installation")
	}
	if Remove(dir) != nil || Installed(dir) {
		t.Fatal("remove")
	}
}

// TestSpeak reads aloud with a real installation (BLUP_KOKORO_DIR).
func TestSpeak(t *testing.T) {
	dir := os.Getenv("BLUP_KOKORO_DIR")
	if dir == "" || !Installed(dir) {
		t.Skip("set BLUP_KOKORO_DIR to an installed local voice")
	}
	mp3, secs, err := Engine{dir}.Speak(context.Background(), "Olá! Um teste rápido da voz local, em português.", "pf_dora", nil)
	if err != nil {
		t.Fatal(err)
	}
	if secs < 1 || len(mp3) < 4000 || mp3[0] != 0xFF {
		t.Fatalf("%.1fs, %d bytes, % x", secs, len(mp3), mp3[:4])
	}
}

// TestRealInstall installs the local voice for real (ONNX Runtime, model
// and voices from their pinned addresses) with a tools package just built
// (BLUP_VOICE_TOOLS_FILE, as .github/workflows/voice-tools.yml does on each
// system) and reads aloud in each language with it.
func TestRealInstall(t *testing.T) {
	file := os.Getenv("BLUP_VOICE_TOOLS_FILE")
	if file == "" {
		t.Skip("set BLUP_VOICE_TOOLS_FILE to a tools package built for this system")
	}
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	key := runtime.GOOS + "/" + runtime.GOARCH
	p, ok := platforms[key]
	if !ok {
		t.Fatalf("no platform %s", key)
	}
	defer func(old platform) { platforms[key] = old }(p)
	p.tools = download{file, sum(b), int64(len(b))}
	platforms[key] = p
	if !Supported() {
		t.Fatalf("the local voice is not supported on %s (cgo needed on Linux and macOS)", key)
	}
	// Not t.TempDir: Windows keeps a loaded DLL from being deleted.
	tmp, err := os.MkdirTemp("", "blup-kokoro")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	dir := filepath.Join(tmp, "kokoro")
	if err := Install(context.Background(), dir, nil); err != nil {
		t.Fatal(err)
	}
	for voice, text := range map[string]string{
		"pf_dora":  "Olá! Um teste rápido da voz local, em português.",
		"af_heart": "Hello! A quick test of the local voice, in English.",
		"ef_dora":  "¡Hola! Una prueba rápida de la voz local, en español.",
	} {
		mp3, secs, err := Engine{dir}.Speak(context.Background(), text, voice, nil)
		if err != nil {
			t.Fatalf("%s: %v", voice, err)
		}
		if secs < 1 || len(mp3) < 4000 || mp3[0] != 0xFF {
			t.Fatalf("%s: %.1fs, %d bytes", voice, secs, len(mp3))
		}
		t.Logf("%s: %.1fs, %d bytes of MP3", voice, secs, len(mp3))
	}
}
