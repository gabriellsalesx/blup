import {z} from "zod";
import {languageSchema} from "../../shared/language";
import {reservedSlugs,slugShape} from "../pulse/page-rules";
import {navLinks} from "../pulse/page-look";
import {dealCategories,siteNetworks,socialURL} from "../../shared/deals-post";

// A deals page (a "site" channel): its address deals.getblup.site/<slug>/,
// the words on top and the language of its own texts. The slug keeps clear
// of the host's own paths (r/ is the short links).
export const siteReserved=[...reservedSlugs,"r","deals","site","sites","logo","robots.txt","sitemap.xml","favicon.ico"] as const;
export const siteSlug=slugShape.refine(v=>!(siteReserved as readonly string[]).includes(v),{message:"RESERVED"});
export const siteTitle=z.string().trim().min(1).max(80);
export const siteDescription=z.string().trim().max(300);
// The page's social networks: one of each, each on its own site.
export const siteSocials=z.array(z.object({network:z.enum(siteNetworks),url:z.string().max(300)}).strict()).max(siteNetworks.length)
 .refine(l=>new Set(l.map(x=>x.network)).size===l.length,"ONE_EACH")
 .transform((l,ctx)=>l.map(x=>{const url=socialURL(x.network,x.url);if(!url)ctx.addIssue({code:"custom",message:"SOCIAL_URL",path:[x.network]});return {network:x.network,url:url??""};}));
// (Written out from the schema, so the panel's typed API spells the networks.)
export type SiteSocial=z.output<typeof siteSocials>[number];
// The stored list, or none when it is not one.
export function socialsOf(stored:string):SiteSocial[]{try{const p=siteSocials.safeParse(JSON.parse(stored));return p.success?p.data:[];}catch{return [];}}
export const siteCreate=z.object({title:siteTitle,slug:siteSlug,description:siteDescription.default(""),language:languageSchema,showTitle:z.boolean().default(true),links:navLinks.default([]),socials:siteSocials.default([])}).strict();
// The header (the name beside the logo, up to five links) as a Pulse page's.
export const siteUpdate=z.object({title:siteTitle,description:siteDescription,language:languageSchema,showTitle:z.boolean(),links:navLinks,socials:siteSocials}).strict();

// What a visit asks of the page: a search, a category, the page of results
// (a cursor: the last deal's time) and the language of its texts.
export const SITE_PAGE=24;
// How long a deal stays on the page after it was sent: older ones leave on
// their own, so the page does not pile up.
export const SITE_DAYS=30;
export const siteQuery=z.object({
 q:z.string().trim().max(80).optional().catch(undefined),
 category:z.enum(dealCategories).optional().catch(undefined),
 before:z.coerce.number().int().positive().max(Number.MAX_SAFE_INTEGER).optional().catch(undefined),
 lang:z.optional(languageSchema).catch(undefined),
});

// The R2 key of a page's logo.
export const siteLogoKey=(site:string,variant:"light"|"dark")=>`sites/${site}/logo-${variant}`;
