// Read-only check of the catalog configured in the Stripe Dashboard.
// Amounts are never created or copied from the application's source.
import { readFileSync } from "node:fs";
import { pathToFileURL } from "node:url";
import { PLANS } from "../worker/plans.ts";
import { createStripe } from "../worker/stripe-client.ts";
import { listPricesByLookup,validateStripePrices } from "../worker/price-schema.ts";

export async function checkCatalog(secret:string,log:(message:string)=>void=console.log):Promise<boolean>{
 if(!/^(?:sk|rk)_(?:test|live)_[A-Za-z0-9_]+$/.test(secret)||secret.includes("coloque")){
  log("Falta uma chave Stripe válida no ambiente ou em .dev.vars.");return false;
 }
 log(/^(?:sk|rk)_live_/.test(secret)?"Conferindo Stripe em produção (somente leitura).":"Conferindo Stripe no sandbox (somente leitura).");
 const keys=PLANS.flatMap(p=>[p.lookupKey,p.yearlyLookupKey].filter((k):k is string=>k!==undefined));
 const intervals=new Map(PLANS.flatMap(p=>[...(p.lookupKey?[[p.lookupKey,"month"] as const]:[]),...(p.yearlyLookupKey?[[p.yearlyLookupKey,"year"] as const]:[])]));
 const values=validateStripePrices(await listPricesByLookup(createStripe(secret),keys),intervals);
 let valid=true;
 for(const key of keys){
  const price=values.get(key);
  if(!price){log(`${key}: indisponível. Crie/ative o preço recorrente correto no Stripe e atribua o lookup key.`);valid=false;}
  else log(`${key}: ${price.unitAmount} unidades menores, ${price.currency.toUpperCase()}.`);
 }
 return valid;
}
function secretFromEnvironment(){
 if(process.env.STRIPE_SECRET_KEY)return process.env.STRIPE_SECRET_KEY;
 try{
  const vars=readFileSync(new URL("../.dev.vars",import.meta.url),"utf8");
  const match=vars.match(/^(?:export\s+)?STRIPE_SECRET_KEY\s*=\s*(?:"([^"]*)"|'([^']*)'|([^#\r\n]*))/m);
  return (match?.[1]??match?.[2]??match?.[3]??"").trim();
 }catch{return "";}
}
if(process.argv[1]&&import.meta.url===pathToFileURL(process.argv[1]).href){
 try{if(!await checkCatalog(secretFromEnvironment()))process.exitCode=1;}
 catch{console.error("Não foi possível conferir o catálogo Stripe. Confira permissões e conexão.");process.exitCode=1;}
}
