import { z } from "zod";
import type Stripe from "stripe";
export type Price={unitAmount:number;currency:string};
const priceSchema=z.object({
	id:z.string().regex(/^price_[A-Za-z0-9_]+$/).max(128),object:z.literal("price"),active:z.boolean(),
	lookup_key:z.string().max(128),currency:z.string().regex(/^[a-z]{3}$/),
	unit_amount:z.number().int().min(0).max(Number.MAX_SAFE_INTEGER),
	type:z.literal("recurring"),billing_scheme:z.literal("per_unit"),
	recurring:z.object({interval:z.enum(["month","year"]),interval_count:z.literal(1),usage_type:z.literal("licensed")}),
	product:z.object({id:z.string().regex(/^prod_[A-Za-z0-9_]+$/).max(128),active:z.boolean()}),
	transform_quantity:z.null().optional(),custom_unit_amount:z.null().optional(),
});
const listSchema=z.object({object:z.literal("list"),has_more:z.literal(false),data:z.array(z.unknown()).max(100)});
const lookupSchema=z.object({lookup_key:z.string().max(128)});

export function validateStripePrices(value:unknown,intervals:ReadonlyMap<string,string>):ReadonlyMap<string,Price|null>{
	const list=listSchema.parse(value),result=new Map<string,Price|null>([...intervals.keys()].map(k=>[k,null])),seen=new Set<string>();
	for(const raw of list.data){
		const lookup=lookupSchema.safeParse(raw);
		if(!lookup.success||!result.has(lookup.data.lookup_key))continue;
		const key=lookup.data.lookup_key;
		if(seen.has(key)){result.set(key,null);continue;}
		seen.add(key);
		const parsed=priceSchema.safeParse(raw);
		if(!parsed.success)continue;
		const p=parsed.data;
		if(!p.active||!p.product.active||p.recurring.interval!==intervals.get(key))continue;
		result.set(key,{unitAmount:p.unit_amount,currency:p.currency});
	}
	return result;
}


// Stripe's GET /v1/prices takes at most 10 lookup_keys per call (API
// reference, "List all prices", 2026-10-28.endive). The catalog has more,
// so it is read in chunks and merged into one list for validateStripePrices.
export const LOOKUP_KEYS_PER_CALL=10;
export async function listPricesByLookup(stripe:Stripe,keys:readonly string[],options:Stripe.RequestOptions={timeout:5_000,maxNetworkRetries:0}){
	const chunks:string[][]=[];
	for(let i=0;i<keys.length;i+=LOOKUP_KEYS_PER_CALL)chunks.push(keys.slice(i,i+LOOKUP_KEYS_PER_CALL));
	const pages=await Promise.all(chunks.map(chunk=>stripe.prices.list({active:true,type:"recurring",lookup_keys:chunk,limit:100,expand:["data.product"]},options)));
	const lists=pages.map(page=>listSchema.parse(page));
	return {object:"list" as const,has_more:false as const,data:lists.flatMap(list=>list.data)};
}
