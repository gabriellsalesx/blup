import {afterEach,describe,expect,it,vi} from "vitest";
import {chooseLanguage,i18n,setLanguage,t,intlLocale} from "./i18n.svelte";
import {errorMessage} from "./i18n/errors";
import {authClient} from "./auth";
import {money} from "./money";
afterEach(()=>{vi.unstubAllGlobals();i18n.locale="en";i18n.staff=false;});
describe("panel language and localized errors",()=>{
 it("sends the selected language through the actual Better Auth client",async()=>{
  i18n.locale="es";let selected:string|null=null;
  const fetchMock=vi.fn(async(_input:RequestInfo|URL,init?:RequestInit)=>{selected=new Headers(init?.headers).get("x-blup-language");return Response.json(null);});
  await authClient.getSession({fetchOptions:{customFetchImpl:fetchMock}});expect(selected).toBe("es");
 });

 it("prefers manual storage, then cookie, then browser, with English fallback",()=>{
  expect(chooseLanguage("es","pt",["en-US"])).toBe("es");expect(chooseLanguage("invalid","pt",["en"])).toBe("pt");
  expect(chooseLanguage(null,null,["de","pt-BR"])).toBe("pt");expect(chooseLanguage(null,null,["ja"])).toBe("en");
 });
 it("stores a manual choice and updates the document language",()=>{
  const setItem=vi.fn();const document={cookie:"",documentElement:{lang:"en"}};
  vi.stubGlobal("localStorage",{setItem});vi.stubGlobal("document",document);vi.stubGlobal("location",new URL("https://cloud.audit.invalid"));
  setLanguage("es");expect(setItem).toHaveBeenCalledWith("blup-lang","es");expect(document.cookie).toContain("SameSite=Lax; Secure");expect(document.documentElement.lang).toBe("es");expect(t("save")).toBe("Guardar");
 });
 it("keeps the document and UI language usable when both storage mechanisms are blocked",()=>{
  const document={documentElement:{lang:"en"},set cookie(_value:string){throw new Error("denied");}};
  vi.stubGlobal("localStorage",{setItem(){throw new Error("denied");}});vi.stubGlobal("document",document);vi.stubGlobal("location",new URL("https://cloud.audit.invalid"));
  expect(()=>setLanguage("es")).not.toThrow();expect(document.documentElement.lang).toBe("es");expect(t("save")).toBe("Guardar");
 });
 it("keeps selection usable when persistent storage is blocked",()=>{
  vi.stubGlobal("localStorage",{setItem(){throw new Error("denied");}});setLanguage("pt");expect(t("save")).toBe("Salvar");
 });
 it("maps API codes without showing untranslated or private provider messages",()=>{
  i18n.locale="es";expect(errorMessage({code:"INVALID_OTP",message:"private provider payload"})).toContain("código");
  i18n.locale="en";expect(errorMessage({detail:{data:{code:"QUOTA"}},statusCode:413})).toContain("storage");
  expect(errorMessage({code:"UNKNOWN",message:"private provider payload"})).toBe("Something went wrong. Please try again.");
 });
 it("changes existing translated expressions and leaves staff in Portuguese",()=>{
  const label=()=>t("plan");i18n.locale="en";expect(label()).toBe("Plan");i18n.locale="pt";expect(label()).toBe("Plano");
  i18n.locale="es";i18n.staff=true;expect(t("save")).toBe("Salvar");expect(intlLocale()).toBe("pt-BR");
 });
 it("formats money using the selected locale and Stripe units",()=>{
  expect(money(1750,"brl","en")).toContain("17.50");expect(money(1750,"brl","pt")).toContain("17,50");expect(money(500,"jpy","es")).toContain("500");
 });
});
