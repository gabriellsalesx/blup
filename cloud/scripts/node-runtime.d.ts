// Minimal contracts for the Node 24 standard-library APIs used by this
// CLI. No Node type package is installed; avoid adding a dependency just
// for two synchronous bootstrap calls. CLI tests exercise the real APIs.
declare module "node:fs" {
	export function readFileSync(path:URL,encoding:"utf8"):string;
}
declare module "node:url" {
	export function pathToFileURL(path:string):URL;
}
declare const process:{
	env:Readonly<Record<string,string|undefined>>;
	argv:string[];
	exitCode:number|undefined;
};
interface ImportMeta { readonly url:string }
