import {languageTag,type Language} from "../../shared/language";
// Stripe amounts use minor units. ISK/UGX retain two API decimals even
// though Intl formats those currencies without decimal fractions.
export function money(amount:number,currency:string,language:Language="pt"){
	const format=new Intl.NumberFormat(languageTag[language],{style:"currency",currency:currency.toUpperCase()});
	const decimals=["isk","ugx"].includes(currency.toLowerCase())?2:format.resolvedOptions().maximumFractionDigits??2;
	return format.format(amount/10**decimals);
}
