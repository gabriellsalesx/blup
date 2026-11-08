import type {Language} from "./language";

// The currency the plans are shown in, per language (as on getblup.io):
// approximate amounts, converted from the Stripe prices in BRL (worker/fx.ts).
export const displayCurrency:Readonly<Record<Language,"usd"|"eur"|null>>={pt:null,en:"usd",es:"eur"};

export function shownPrice(price:{unitAmount:number;currency:string;approx?:{usd:number;eur:number}|null},language:Language){
	const c=displayCurrency[language];
	return c&&price.approx?{unitAmount:price.approx[c],currency:c,approx:true}:{unitAmount:price.unitAmount,currency:price.currency,approx:false};
}
