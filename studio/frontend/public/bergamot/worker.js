// The local translation (Blup Studio): Firefox's engine (Bergamot) in a
// worker, so the window never waits. Messages: { id, from, to, items:
// [{ text, html }] } → { id, out: [string] } or { id, error }. The engine
// and the models come from /local-translate/ (downloaded from Mozilla); a
// language pair without English goes through it (pt → en → es).
/* global loadBergamot */
importScripts("bergamot-translator.js");

const ALIGN = { model: 256, lex: 64, vocab: 64 };
const FILES = (pair) => ({ model: `model.${pair}.intgemm.alphas.bin`, lex: `lex.50.50.${pair}.s2t.bin`, vocab: `vocab.${pair}.spm` });

let engine = null; // the wasm module
let service = null;
const models = new Map(); // "enpt" → TranslationModel

const config = (o) => "\n" + Object.entries(o).map(([k, v]) => `            ${k}: ${v}\n`).join("") + "            ";

async function load(file) {
  const res = await fetch(`/local-translate/${file}`);
  if (!res.ok) throw new Error(`missing ${file}`);
  return res.arrayBuffer();
}

async function start() {
  if (engine) return;
  const wasmBinary = await load("bergamot-translator.wasm");
  engine = await new Promise((resolve, reject) => {
    const m = loadBergamot({
      // Small at first; it grows as needed (less memory than a big start).
      INITIAL_MEMORY: 41943040,
      wasmBinary,
      print: () => {},
      printErr: () => {},
      onAbort: () => reject(new Error("the translation engine stopped")),
      onRuntimeInitialized: async () => { await Promise.resolve(); resolve(m); },
    });
  });
  service = new engine.BlockingService({ cacheSize: 0 });
}

async function model(pair) {
  if (models.has(pair)) return models.get(pair);
  const files = FILES(pair);
  const mem = {};
  for (const [type, name] of Object.entries(files)) {
    const buf = await load(name);
    const m = new engine.AlignedMemory(buf.byteLength, ALIGN[type]);
    m.getByteArrayView().set(new Uint8Array(buf));
    mem[type] = m;
  }
  const vocabs = new engine.AlignedMemoryList();
  vocabs.push_back(mem.vocab);
  const m = new engine.TranslationModel(pair.slice(0, 2), pair.slice(2), config({
    "beam-size": "1", normalize: "1.0", "word-penalty": "0", "max-length-break": "128", "mini-batch-words": "1024",
    workspace: "128", "max-length-factor": "2.0", "skip-cost": "true", "cpu-threads": "0", quiet: "true",
    "quiet-translation": "true", "gemm-precision": "int8shiftAlphaAll", alignment: "soft",
  }), mem.model, mem.lex, vocabs, null);
  models.set(pair, m);
  return m;
}

onmessage = async ({ data }) => {
  const { id, from, to, items } = data;
  try {
    await start();
    const chain = from === "en" || to === "en" ? [await model(from + to)] : [await model(from + "en"), await model("en" + to)];
    const messages = new engine.VectorString();
    const options = new engine.VectorResponseOptions();
    for (const it of items) {
      messages.push_back(it.text);
      options.push_back({ qualityScores: false, alignment: true, html: !!it.html });
    }
    const res = chain.length === 1 ? service.translate(chain[0], messages, options) : service.translateViaPivoting(chain[0], chain[1], messages, options);
    const out = [];
    for (let i = 0; i < res.size(); i++) out.push(res.get(i).getTranslatedText());
    messages.delete();
    options.delete();
    postMessage({ id, out });
  } catch (e) {
    postMessage({ id, error: String(e && e.message ? e.message : e) });
  }
};
