// The one node:zlib call the Pulse ingestion makes (Brotli request bodies,
// which the Workers DecompressionStream does not take). Workers ship
// node:zlib with nodejs_compat; no Node type package is installed, so this
// is the minimal contract, as in scripts/node-runtime.d.ts. Tests run the
// real API.
declare module "node:zlib" {
	export function brotliDecompressSync(buffer:Uint8Array,options:{maxOutputLength:number}):Uint8Array;
	export function brotliCompressSync(buffer:Uint8Array|string):Uint8Array;
}
