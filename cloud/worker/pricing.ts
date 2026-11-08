import {planLabel,planPerks} from "../shared/plan-i18n";
import type {Language} from "../shared/language";
import { listPricesByLookup,validateStripePrices,type Price } from "./price-schema";
import { PLANS,SUGGESTED,type Plan } from "./plans";
import { createStripe } from "./stripe-client";

type Prices={monthly:Price|null;yearly:Price|null};
const keys=PLANS.flatMap(p=>[p.lookupKey,p.yearlyLookupKey].filter((key):key is string=>key!==undefined));
const intervals=new Map(PLANS.flatMap(p=>[...(p.lookupKey?[[p.lookupKey,"month"] as const]:[]),...(p.yearlyLookupKey?[[p.yearlyLookupKey,"year"] as const]:[])]));

// A successful but incomplete catalog makes missing/unsupported prices
// unavailable. Transport/malformed-list failures retain the last good
// snapshot. Never substitute the old hardcoded display amounts.
export const validatePrices=(value:unknown)=>validateStripePrices(value,intervals);

export class PriceCatalog{
	private snapshot:ReadonlyMap<string,Price|null>=new Map();
	private nextRefresh=0;
	private pending:Promise<void>|null=null;
	constructor(private readonly load:()=>Promise<unknown>){}
	async prices():Promise<ReadonlyMap<string,Price|null>>{
		if(Date.now()>=this.nextRefresh){
			this.pending??=this.refresh().finally(()=>{this.pending=null;});
			await this.pending;
		}
		return this.snapshot;
	}
	private async refresh(){
		try{
			this.snapshot=validatePrices(await this.load());
			this.nextRefresh=Date.now()+10*60_000;
		}catch{
			// No hot retry loop on the public endpoint during an outage.
			this.nextRefresh=Date.now()+30_000;
			console.error("[blup] pricing_refresh_failed");
		}
	}
}

let current:{secret:string;catalog:PriceCatalog}|null=null;
function catalog(env:Pick<Cloudflare.Env,"STRIPE_SECRET_KEY">){
	if(!current||current.secret!==env.STRIPE_SECRET_KEY){
		const secret=env.STRIPE_SECRET_KEY;
		current={secret,catalog:new PriceCatalog(async()=>{
			if(!secret)throw new Error("Pricing unavailable");
			return listPricesByLookup(createStripe(secret),keys);
		})};
	}
	return current.catalog;
}
function pricesOf(plan:Plan,values:ReadonlyMap<string,Price|null>):Prices{
	return {monthly:plan.lookupKey?values.get(plan.lookupKey)??null:null,yearly:plan.yearlyLookupKey?values.get(plan.yearlyLookupKey)??null:null};
}
export async function publicPlans(env:Pick<Cloudflare.Env,"STRIPE_SECRET_KEY">,language:Language="pt"){
	const values=await catalog(env).prices();
	return PLANS.map(plan=>({name:plan.name,basePlan:plan.basePlan,pulse:plan.pulse,deals:plan.deals,label:planLabel(plan.name,language),limits:plan.limits,perks:planPerks(language,plan.name,plan.limits),suggested:plan.basePlan===SUGGESTED,prices:pricesOf(plan,values)}));
}
export async function purchaseAvailable(env:Pick<Cloudflare.Env,"STRIPE_SECRET_KEY">,name:string,annual:boolean){
	const plan=PLANS.find(p=>p.name===name);
	if(!plan||plan.name==="free")return false;
	return pricesOf(plan,await catalog(env).prices())[annual?"yearly":"monthly"]!==null;
}
