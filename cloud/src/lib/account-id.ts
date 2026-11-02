const help={pt:"Em caso de ajuda, informe seu ID.",en:"If you need help, share your ID.",es:"Si necesitas ayuda, comparte tu ID."};
export const accountIdHelp=(locale:"pt"|"en"|"es"="pt")=>help[locale];
export const shortAccountId=(id:string)=>id.length>12?`${id.slice(0,8)}…${id.slice(-4)}`:id;
export async function copyAccountId(id:string,clipboard:Pick<Clipboard,"writeText">|undefined=globalThis.navigator?.clipboard):Promise<boolean>{
	if(!clipboard||!id)return false;
	try{await clipboard.writeText(id);return true;}catch{return false;}
}
