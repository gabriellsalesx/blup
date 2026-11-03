// Copies text; false when the browser has no clipboard or refuses.
export async function copyText(text:string,clipboard:Pick<Clipboard,"writeText">|undefined=globalThis.navigator?.clipboard):Promise<boolean>{
	if(!clipboard||!text)return false;
	try{await clipboard.writeText(text);return true;}catch{return false;}
}
