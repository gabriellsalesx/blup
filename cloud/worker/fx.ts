import {z} from "zod";

// Approximate prices for the plans page, as on getblup.io: English in US
// dollars, Spanish in euros (shared/display-currency.ts picks one by the
// panel's language, so switching it needs no reload), converted from the
// Stripe prices in BRL with the
// ECB rate (Frankfurter). Display only: the checkout charges the Stripe price
// and Stripe's Adaptive Pricing shows the final amount in the customer's
// currency. No rate (fetch failed, other currency) means no approximation.
const RATES_URL="https://api.frankfurter.dev/v1/latest?base=BRL&symbols=USD,EUR";
const TTL=6*60*60_000;
const answer=z.object({base:z.literal("BRL"),rates:z.object({USD:z.number().positive().max(10),EUR:z.number().positive().max(10)})});
type Rates={usd:number;eur:number};

let cached:{rates:Rates;until:number}|null=null;
let pending:Promise<Rates|null>|null=null;

async function load(send:typeof fetch):Promise<Rates|null>{
	try{
		const res=await send(RATES_URL,{signal:AbortSignal.timeout(5_000),redirect:"manual",cf:{cacheTtl:TTL/1000,cacheEverything:true}});
		if(!res.ok)return null;
		const parsed=answer.safeParse(await res.json());
		if(!parsed.success)return null;
		const rates={usd:parsed.data.rates.USD,eur:parsed.data.rates.EUR};
		cached={rates,until:Date.now()+TTL};
		return rates;
	}catch{
		return null;
	}
}

// The rates, at most TTL old; on a failure, the last ones this isolate had.
export async function brlRates(send:typeof fetch=fetch):Promise<Rates|null>{
	if(cached&&Date.now()<cached.until)return cached.rates;
	pending??=load(send).finally(()=>{pending=null;});
	return (await pending)??cached?.rates??null;
}

export function approximate(price:{unitAmount:number;currency:string}|null,rates:Rates|null){
	if(!price||!rates||price.currency.toLowerCase()!=="brl")return null;
	return {usd:Math.round(price.unitAmount*rates.usd),eur:Math.round(price.unitAmount*rates.eur)};
}

export function resetRatesForTest(){cached=null;pending=null;}
