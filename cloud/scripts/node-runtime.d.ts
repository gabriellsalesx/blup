// Minimal contracts for the Node 24 standard-library APIs used by this
// CLI. No Node type package is installed; avoid adding a dependency just
// for two synchronous bootstrap calls. CLI tests exercise the real APIs.
declare module "node:fs" {
	export function readFileSync(path:URL|string,encoding:"utf8"):string;
	export function existsSync(path:string):boolean;
	export function writeFileSync(path:string,data:string,options:{mode:number}):void;
	export function chmodSync(path:string,mode:number):void;
	export function rmSync(path:string,options:{force:boolean}):void;
}
declare module "node:os" {
	export function homedir():string;
}
declare module "node:path" {
	export function join(...parts:string[]):string;
}
declare module "node:child_process" {
	export function spawnSync(cmd:string,args:string[],options:{stdio:"inherit"}):{status:number|null};
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
