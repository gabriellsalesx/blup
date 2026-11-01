# Bergamot (local translation)

`bergamot-translator.js` is the JavaScript glue of Firefox's translation
engine, copied unchanged from Firefox (`toolkit/components/translations/
bergamot-translator/bergamot-translator.js`, release branch, Bergamot
v0.6.0, engine major version 4). It is MPL-2.0 (see its header).

The WebAssembly engine and the language models are not in the repository:
Blup Studio downloads them from Mozilla (the same files Firefox uses,
checked by SHA-256, see `internal/localtr`) when the person turns the local
translation on, and serves them at `/local-translate/`.

`worker.js` is Blup's: it runs the engine, as Firefox's
`translations-engine.worker.js` does.
