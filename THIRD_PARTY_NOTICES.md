# Third-party notices

Blup is licensed under the GNU AGPL-3.0 (see [LICENSE](LICENSE)). It builds on
the work below, each part under its own license, all of them compatible with
the AGPL-3.0. The full license texts are in the folders named, or in each
project.

## Bundled in Blup Studio

| Part | License | Where |
| --- | --- | --- |
| [Hunspell](https://hunspell.github.io), as [hunspell-wasm](https://github.com/rotemdan/hunspell-wasm) (spell check) | MPL-1.1 (also LGPL-2.0 or GPL-2.0) | [studio/licenses/hunspell-MPL-1.1.txt](studio/licenses/hunspell-MPL-1.1.txt) |
| Portuguese (Brazil) dictionary, [VERO](https://pt-br.libreoffice.org/projetos/projeto-vero-verificador-ortografico/), via [dictionary-pt](https://github.com/wooorm/dictionaries) | MPL-2.0 (also LGPL-3.0) | [studio/licenses/dictionary-pt.txt](studio/licenses/dictionary-pt.txt) |
| English (US) dictionary, via [dictionary-en](https://github.com/wooorm/dictionaries) | MIT and BSD | [studio/licenses/dictionary-en.txt](studio/licenses/dictionary-en.txt) |
| Spanish dictionary, via [dictionary-es](https://github.com/wooorm/dictionaries) | MPL-1.1 (also GPL-3.0 or LGPL-3.0) | [studio/licenses/dictionary-es.txt](studio/licenses/dictionary-es.txt) |
| [Wails](https://wails.io) (desktop window) | MIT | |
| [Svelte](https://svelte.dev) | MIT | |
| [CodeMirror](https://codemirror.net) (editor) | MIT | |
| [xterm.js](https://xtermjs.org) (terminal) | MIT | |
| [Lucide](https://lucide.dev) (icons) | ISC | |
| [go-pty](https://github.com/aymanbagabas/go-pty) (terminal) | MIT | |
| [fsnotify](https://github.com/fsnotify/fsnotify) | BSD-3-Clause | |
| [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) (previews) | BSD-3-Clause | |
| [Bergamot](https://github.com/mozilla/translations) JavaScript glue of Firefox's translation engine (local translation), unchanged | MPL-2.0 | [studio/frontend/public/bergamot](studio/frontend/public/bergamot) |
| [klauspost/compress](https://github.com/klauspost/compress) (unpacks the local translation's models) | BSD-3-Clause | |

## Blup (command line) and its themes

| Part | License | Where |
| --- | --- | --- |
| [Lucide](https://lucide.dev) icons in the themes | ISC | [theme/licenses/lucide-ISC.txt](theme/licenses/lucide-ISC.txt) |
| [Bubble Tea, Huh, Lip Gloss, Bubbles](https://charm.land) (terminal screens) | MIT | |
| [go-toml](https://github.com/pelletier/go-toml) | MIT | |
| [go-yaml](https://github.com/yaml/go-yaml) | Apache-2.0 | |
| [BLAKE3](https://lukechampine.com/blake3) | MIT | |
| [golang.org/x/net, x/text, x/crypto](https://pkg.go.dev/golang.org/x) | BSD-3-Clause | |
| [coder/websocket](https://github.com/coder/websocket) (Blup Cloud live sync) | ISC | |

## Downloaded when needed (not bundled)

| Part | License |
| --- | --- |
| [Hugo](https://gohugo.io), builds the site | Apache-2.0 |
| [Pagefind](https://pagefind.app), the site's search | MIT |

### Blup Studio's local translation

Downloaded only when the user turns on "Translation on the computer" in Blup
Studio (Settings, AI): the same files Firefox downloads from Mozilla's
Remote Settings, each pinned and checked by SHA-256
([internal/localtr/localtr.go](internal/localtr/localtr.go)).

| Part | License | Where it comes from |
| --- | --- | --- |
| Firefox Translations' engine, [Bergamot](https://github.com/mozilla/translations) 4.0 (Marian, compiled to WebAssembly) | MPL-2.0 | Mozilla's CDN (firefox-settings-attachments.cdn.mozilla.net) |
| Firefox Translations' models (English with Portuguese and Spanish, version 3.0) | MPL-2.0 | same |

### Blup Studio's local voice

Downloaded only when the user picks "Voice on the computer" in Blup Studio
(Settings, AI), each file pinned and checked by SHA-256
([internal/kokoro/install.go](internal/kokoro/install.go)). None of them is
part of Blup's binary: ONNX Runtime is loaded as a library at run time,
eSpeak NG and LAME run as separate programs.

| Part | License | Where it comes from |
| --- | --- | --- |
| [ONNX Runtime](https://onnxruntime.ai) 1.31.1 (Microsoft), runs the model | MIT | Microsoft's GitHub releases. Its C headers are in [internal/kokoro/ort](internal/kokoro/ort), with [its license](internal/kokoro/ort/LICENSE-onnxruntime-MIT.txt) |
| [Kokoro 82M](https://huggingface.co/hexgrad/Kokoro-82M) (hexgrad), the voice model and voices, in [ONNX form](https://huggingface.co/onnx-community/Kokoro-82M-v1.0-ONNX) | Apache-2.0 | Hugging Face, at a fixed revision |
| [eSpeak NG](https://github.com/espeak-ng/espeak-ng) 1.52.0, the pronunciation | GPL-3.0-or-later | built from the unmodified official source by [studio/build/voice/build-tools.sh](studio/build/voice/build-tools.sh) and published with Blup's releases, with its license and source address |
| [LAME](https://lame.sourceforge.io) 3.100, the MP3 | LGPL-2.0-or-later | same as eSpeak NG |

The rewriting of eSpeak NG's phonemes into Kokoro's alphabet follows
[misaki](https://github.com/hexgrad/misaki) (Apache-2.0), Kokoro's own
pipeline.
