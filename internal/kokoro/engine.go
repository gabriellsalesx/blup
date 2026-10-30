// Package kokoro is Blup Studio's local voice: Kokoro 82M
// (github.com/hexgrad/kokoro, Apache-2.0) run on the computer with ONNX
// Runtime, eSpeak NG for the pronunciation and LAME for the MP3. None of
// them is part of Blup: Install downloads them (pinned, SHA-256 checked)
// when the user picks the local voice, and they run from that folder:
// ONNX Runtime loaded as a library, eSpeak NG and LAME as programs.
package kokoro

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrUnsupported means this system has no local voice package.
var ErrUnsupported = errors.New("the local voice is not available on this system")

// ErrNotInstalled means the local voice was not downloaded yet.
var ErrNotInstalled = errors.New("the local voice is not installed")

// SampleRate is Kokoro's output, in Hz.
const SampleRate = 24000

// maxTokens is the longest input Kokoro takes (plus the two pads).
const maxTokens = 510

// Voice is one of the local voices offered.
type Voice struct {
	ID     string `json:"id"`     // Kokoro's name, "pf_dora"
	Name   string `json:"name"`   // "Dora"
	Lang   string `json:"lang"`   // the site language it speaks: "pt" or "en"
	Female bool   `json:"female"` // for the list
}

// Voices are the local voices, the default of each language first.
var Voices = []Voice{
	{"pf_dora", "Dora", "pt", true},
	{"pm_alex", "Alex", "pt", false},
	{"pm_santa", "Santa", "pt", false},
	{"af_heart", "Heart", "en", true},
	{"af_bella", "Bella", "en", true},
	{"am_michael", "Michael", "en", false},
	{"am_fenrir", "Fenrir", "en", false},
	{"bf_emma", "Emma (UK)", "en", true},
	{"bm_george", "George (UK)", "en", false},
}

// Engine runs the local voice installed in Dir.
type Engine struct{ Dir string }

func exe(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func (e Engine) espeak() string     { return filepath.Join(e.Dir, "tools", exe("espeak-ng")) }
func (e Engine) espeakData() string { return filepath.Join(e.Dir, "tools") }
func (e Engine) lame() string       { return filepath.Join(e.Dir, "tools", exe("lame")) }
func (e Engine) model() string      { return filepath.Join(e.Dir, "model.onnx") }
func (e Engine) voice(id string) string {
	return filepath.Join(e.Dir, "voices", id+".bin")
}

// Speak reads text aloud with voice and returns the MP3 and its length in
// seconds. progress gets the parts done of total.
func (e Engine) Speak(ctx context.Context, text, voice string, progress func(done, total int)) ([]byte, float64, error) {
	if !Installed(e.Dir) {
		return nil, 0, ErrNotInstalled
	}
	style, err := loadVoice(e.voice(voice))
	if err != nil {
		return nil, 0, err
	}
	// Paragraph by paragraph (the text comes with blank lines between
	// them), so a pause goes between them: Kokoro alone runs everything
	// together.
	type part struct {
		ps    []rune
		pause float64 // silence after it, in seconds
	}
	var parts []part
	paras := strings.Split(strings.TrimSpace(text), "\n\n")
	for i, para := range paras {
		if strings.TrimSpace(para) == "" {
			continue
		}
		ps, err := Phonemize(ctx, e.espeak(), e.espeakData(), voice, para)
		if err != nil {
			return nil, 0, err
		}
		chunks := chunk(ps, maxTokens)
		for j, c := range chunks {
			p := part{ps: c}
			if j == len(chunks)-1 {
				p.pause = pauseFor(i, len(paras))
			}
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return nil, 0, errors.New("kokoro: nothing to read")
	}
	lib, err := libraryPath(e.Dir)
	if err != nil {
		return nil, 0, err
	}
	s, err := openSession(lib, e.model())
	if err != nil {
		return nil, 0, err
	}
	defer s.close()
	var pcm []float32
	for i, pt := range parts {
		if progress != nil {
			progress(i, len(parts))
		}
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		p := pt.ps
		ids := make([]int64, 0, len(p)+2)
		ids = append(ids, 0)
		for _, r := range p {
			ids = append(ids, vocab[r])
		}
		ids = append(ids, 0)
		// The style depends on the input's length: one row per length,
		// 0 to 509 (as kokoro.js does).
		n := min(len(p), maxTokens-1)
		row := style[n*256 : n*256+256]
		audio, err := s.run(ids, row, 1)
		if err != nil {
			return nil, 0, err
		}
		pcm = append(pcm, audio...)
		pcm = append(pcm, make([]float32, int(pt.pause*SampleRate))...)
	}
	if progress != nil {
		progress(len(parts), len(parts))
	}
	mp3, err := e.encode(ctx, pcm)
	return mp3, float64(len(pcm)) / SampleRate, err
}

// pauseFor is the silence after paragraph i of n: longer after the first
// (the title), none after the last.
func pauseFor(i, n int) float64 {
	switch {
	case i == n-1:
		return 0
	case i == 0:
		return 0.9
	}
	return 0.6
}

// loadVoice reads a voice: 510 style rows of 256 floats.
func loadVoice(path string) ([]float32, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) != maxTokens*256*4 {
		return nil, fmt.Errorf("kokoro: %s is not a voice", filepath.Base(path))
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out, nil
}

// chunk cuts phonemes into parts of at most max symbols, after a
// punctuation mark when it can, else at a space.
func chunk(ps string, max int) [][]rune {
	rs := []rune(strings.TrimSpace(ps))
	var out [][]rune
	for len(rs) > 0 {
		if len(rs) <= max {
			out = append(out, rs)
			break
		}
		cut := -1
		for i := max - 1; i > max/2 && cut < 0; i-- {
			if strings.ContainsRune(".!?;:,…—", rs[i]) {
				cut = i + 1
			}
		}
		for i := max - 1; i > 0 && cut < 0; i-- {
			if rs[i] == ' ' {
				cut = i
			}
		}
		if cut < 0 {
			cut = max
		}
		out = append(out, []rune(strings.TrimSpace(string(rs[:cut]))))
		rs = []rune(strings.TrimSpace(string(rs[cut:])))
	}
	return out
}

// encode turns 24 kHz samples into a mono MP3 (64 kbit/s: clear speech at
// about half a megabyte a minute) with LAME.
func (e Engine) encode(ctx context.Context, pcm []float32) ([]byte, error) {
	raw := make([]byte, len(pcm)*2)
	for i, v := range pcm {
		if v != v { // NaN
			v = 0
		}
		v = float32(math.Max(-1, math.Min(1, float64(v))))
		binary.LittleEndian.PutUint16(raw[i*2:], uint16(int16(v*32767)))
	}
	cmd := exec.CommandContext(ctx, e.lame(), "--quiet", "-r", "-s", "24", "--bitwidth", "16",
		"--signed", "--little-endian", "-m", "m", "-b", "64", "-q", "2", "-", "-")
	cmd.Stdin = bytes.NewReader(raw)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	hideWindow(cmd)
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(errb.String()); msg != "" {
			return nil, &toolError{"lame", msg}
		}
		return nil, err
	}
	return out.Bytes(), nil
}
