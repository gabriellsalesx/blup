import {describe,expect,it} from "vitest";
import {acceptedLanguage,browserLanguage,requestLanguage} from "../shared/language";
import {messages,translate} from "../src/lib/i18n/catalog";
import {emailMessages} from "../shared/email-i18n";
import {planPerks} from "../shared/plan-i18n";
import {planNamed} from "./plans";
describe("language negotiation and complete typed catalogs",()=>{
 it.each([["pt-BR","pt"],["es-AR","es"],["en-GB","en"],["fr-FR","en"]])("negotiates browser language %s",(value,expected)=>expect(browserLanguage([value])).toBe(expected));
 it("uses the first supported browser preference",()=>expect(browserLanguage(["ja-JP","es-MX","pt"])).toBe("es"));
 it("honors Accept-Language quality, exclusions and unsupported fallback",()=>{
  expect(acceptedLanguage("fr;q=1, es-MX;q=0.8, pt-BR;q=0.9")).toBe("pt");
  expect(acceptedLanguage("pt;q=0,es;q=0.7,en;q=0.8")).toBe("en");
  expect(acceptedLanguage("pt;q=NaN,es;q=2,ja-JP")).toBe("en");
  expect(acceptedLanguage(null)).toBe("en");
  expect(acceptedLanguage("pt;Q=0,en;Q=1")).toBe("en");
 });
 it("prefers a validated explicit selection to the browser header",()=>expect(requestLanguage(new Headers({"x-blup-language":"es","accept-language":"pt-BR"}))).toBe("es"));
 it("keeps every locale and interpolation complete without markup or forbidden copy",()=>{
  const slots=(value:string)=>[...value.matchAll(/\{(\w+)\}/g)].map(m=>m[1]).sort();
  for(const rows of [messages,emailMessages])for(const values of Object.values(rows)){
   expect(values).toHaveLength(3);
   for(const value of values){expect(value.length).toBeGreaterThan(0);expect(value).not.toMatch(/[—–<>]/);expect(value.toLowerCase()).not.toContain(["ob","sidian"].join(""));expect(slots(value)).toEqual(slots(values[0]));}
  }
 });
 it("renders native UI and revised plan benefits in three languages",()=>{
  expect(translate("en","welcome")).toBe("Welcome to Blup");expect(translate("es","welcome")).toContain("bienvenida");
  expect(translate("es","subscribe",{plan:"Max"})).toBe("Contratar Max");
  for(const language of ["pt","en","es"] as const){
   const max=planPerks(language,"max",planNamed("max").limits),ultra=planPerks(language,"ultra",planNamed("ultra").limits);
   expect(max[0]).toContain("5 GB");expect(max[2]).toContain("50 MB");expect(max[4]).toContain("3 ");expect(ultra[0]).toContain("10 GB");
  }
 });
});
// Deliberately unreachable calls prove that keys and required slots are
// checked by the regular tsc gate, without an expected-failure test.
if(false){
 // @ts-expect-error unknown catalog key
 translate("en","missing-key");
 // @ts-expect-error required interpolation value
 translate("en","subscribe");
 // @ts-expect-error incorrect interpolation name
 translate("en","subscribe",{name:"Max"});
}
