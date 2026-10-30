package narrate

import (
	"bytes"
	"errors"
)

// MP3 frame headers (MPEG audio Layer III, ISO 11172-3 and 13818-3): enough
// to join the parts the voice model returns into one file, frame by frame,
// without the tags and the Xing/Info frame each part starts with.

// ErrNotMP3 means the data has no MPEG Layer III frames.
var ErrNotMP3 = errors.New("not an MP3 file")

// Bit rates of Layer III in kbit/s, by version (MPEG-1, MPEG-2 and 2.5).
var (
	bitrates1 = [16]int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0}
	bitrates2 = [16]int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0}
)

// Sample rates in Hz by version: MPEG-1, MPEG-2, MPEG-2.5.
var sampleRates = [3][3]int{{44100, 48000, 32000}, {22050, 24000, 16000}, {11025, 12000, 8000}}

type frame struct {
	size       int // bytes, header included
	sampleRate int
	samples    int // samples per channel in the frame
}

// parseFrame reads the 4-byte header at the start of b; ok is false when it
// is not a valid Layer III header.
func parseFrame(b []byte) (f frame, ok bool) {
	if len(b) < 4 || b[0] != 0xFF || b[1]&0xE0 != 0xE0 {
		return f, false
	}
	version := (b[1] >> 3) & 3 // 0: MPEG-2.5, 1: reserved, 2: MPEG-2, 3: MPEG-1
	layer := (b[1] >> 1) & 3   // 1: Layer III
	brIndex := b[2] >> 4
	srIndex := (b[2] >> 2) & 3
	padding := int((b[2] >> 1) & 1)
	if version == 1 || layer != 1 || brIndex == 0 || brIndex == 15 || srIndex == 3 {
		return f, false
	}
	switch version {
	case 3:
		f.sampleRate = sampleRates[0][srIndex]
		f.samples = 1152
		f.size = 144*bitrates1[brIndex]*1000/f.sampleRate + padding
	case 2:
		f.sampleRate = sampleRates[1][srIndex]
		f.samples = 576
		f.size = 72*bitrates2[brIndex]*1000/f.sampleRate + padding
	default:
		f.sampleRate = sampleRates[2][srIndex]
		f.samples = 576
		f.size = 72*bitrates2[brIndex]*1000/f.sampleRate + padding
	}
	return f, f.size > 4
}

// skipID3 returns b without the ID3v2 tag at its start.
func skipID3(b []byte) []byte {
	if len(b) < 10 || string(b[:3]) != "ID3" {
		return b
	}
	size := int(b[6]&0x7F)<<21 | int(b[7]&0x7F)<<14 | int(b[8]&0x7F)<<7 | int(b[9]&0x7F)
	size += 10
	if b[5]&0x10 != 0 { // footer present
		size += 10
	}
	if size > len(b) {
		return nil
	}
	return b[size:]
}

// infoFrame tells whether a frame only carries the Xing/Info/VBRI header
// (an encoder's summary of the whole file, wrong once files are joined).
func infoFrame(fr []byte) bool {
	head := fr
	if len(head) > 64 {
		head = head[:64]
	}
	return bytes.Contains(head, []byte("Xing")) || bytes.Contains(head, []byte("Info")) || bytes.Contains(head, []byte("VBRI"))
}

// frames returns the audio frames of one MP3 file.
func frames(b []byte) ([][]byte, []frame) {
	b = skipID3(b)
	var out [][]byte
	var info []frame
	for i := 0; i+4 <= len(b); {
		if string(b[i:min(i+3, len(b))]) == "TAG" && len(b)-i == 128 {
			break // ID3v1 tag at the end
		}
		f, ok := parseFrame(b[i:])
		if !ok || i+f.size > len(b) {
			i++ // lost sync: look for the next header
			continue
		}
		fr := b[i : i+f.size]
		if !(len(out) == 0 && infoFrame(fr)) {
			out = append(out, fr)
			info = append(info, f)
		}
		i += f.size
	}
	return out, info
}

// JoinMP3 joins MP3 files into one, keeping only their audio frames, and
// returns it with its length in seconds.
func JoinMP3(parts [][]byte) ([]byte, float64, error) {
	var out bytes.Buffer
	seconds := 0.0
	for _, p := range parts {
		fr, info := frames(p)
		for i, f := range fr {
			out.Write(f)
			seconds += float64(info[i].samples) / float64(info[i].sampleRate)
		}
	}
	if out.Len() == 0 {
		return nil, 0, ErrNotMP3
	}
	return out.Bytes(), seconds, nil
}
