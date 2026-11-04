// Where the stores sell, and in which money. Shopee has one store (and one
// affiliate program, with its own API) per country: a person's keys work
// only on their country's API, in its currency. AliExpress has one
// program for the world: the search takes the country it ships to and the
// currency and language of the answer.
export const shopeeCountries=["br","my","sg","th","id","vn","ph","tw"] as const;
export type ShopeeCountry=(typeof shopeeCountries)[number];
const shopeeDomain:Record<ShopeeCountry,string>={br:"com.br",my:"com.my",sg:"sg",th:"co.th",id:"co.id",vn:"vn",ph:"ph",tw:"tw"};
export const shopeeApi=(country:ShopeeCountry)=>`https://open-api.affiliate.shopee.${shopeeDomain[country]}/graphql`;
export const shopeeCurrency:Record<ShopeeCountry,string>={br:"BRL",my:"MYR",sg:"SGD",th:"THB",id:"IDR",vn:"VND",ph:"PHP",tw:"TWD"};

// The countries the panel offers for AliExpress. The currency is the
// country's own when AliExpress converts to it (its list: USD, GBP, CAD,
// EUR, UAH, MXN, TRY, RUB, BRL, AUD, INR, JPY, IDR, SEK, KRW), else US
// dollars; the language the closest one it answers in, else English.
export const aliexpressCountries=["us","ca","mx","br","ar","cl","co","pe","gb","ie","fr","de","es","pt","it","nl","be","at","pl","se","ua","tr","il","sa","ae","au","nz","jp","kr","in","id","th","vn","ph","my","sg"] as const;
export type AliexpressCountry=(typeof aliexpressCountries)[number];
const euro=new Set<string>(["ie","fr","de","es","pt","it","nl","be","at"]);
const own:Partial<Record<AliexpressCountry,string>>={us:"USD",ca:"CAD",mx:"MXN",br:"BRL",gb:"GBP",se:"SEK",ua:"UAH",tr:"TRY",au:"AUD",jp:"JPY",kr:"KRW",in:"INR",id:"IDR"};
export const aliexpressCurrency=(country:AliexpressCountry)=>euro.has(country)?"EUR":own[country]??"USD";
const languages:Partial<Record<AliexpressCountry,string>>={br:"PT",pt:"PT",mx:"ES",ar:"ES",cl:"ES",co:"ES",pe:"ES",es:"ES",fr:"FR",be:"FR",de:"DE",at:"DE",it:"IT",nl:"NL",pl:"PL",tr:"TR",sa:"AR",ae:"AR",jp:"JA",kr:"KO",th:"TH",vn:"VI",ua:"RU"};
export const aliexpressLanguage=(country:AliexpressCountry)=>languages[country]??"EN";

// How many decimals a currency's minor unit has (the yen and the won have
// none): money is kept in minor units, the same as the panel shows it.
export function currencyDigits(currency:string){
 try{return new Intl.NumberFormat("en",{style:"currency",currency}).resolvedOptions().maximumFractionDigits??2;}catch{return 2;}
}
