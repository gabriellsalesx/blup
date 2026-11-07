/** @jsxRuntime automatic */
/** @jsxImportSource hono/jsx */
import {raw} from "hono/html";
import type {Child} from "hono/jsx";
import {translate,type messages,type MessageKey} from "../../src/lib/i18n/catalog";
import {languageTag,type Language,type TranslationArgs} from "../../shared/language";
import {dealCategories,storeTitles,platformNames,networkOf,type SiteNetwork} from "../../shared/deals-post";
import {socialMarks} from "../../shared/social-marks";
import type {SiteSocial} from "./site-rules";
import {THEME_SCRIPT} from "../pulse/public-html";
import {money} from "./message";

// The public deals page, a minimal theme of its own that follows Nori's and
// Mochi's logic: the same sticky header (brand, menu, search, language and
// light/dark/system), then the deals as cards. Rendered on the server with
// Hono's JSX (typed props, escaped text); the only script is the theme's,
// allowed by its hash. Product photos come from the stores (https).

export type DealCategory=(typeof dealCategories)[number];
export type SiteLink={label:string;url:string};
export type SiteInfo={slug:string;title:string;description:string;language:Language;logos:{light:string|null;dark:string|null};showTitle:boolean;links:SiteLink[];socials:SiteSocial[]};
export type SiteDeal={code:string;title:string;image:string|null;price:number|null;originalPrice:number|null;currency:string|null;discount:number|null;store:string;coupon:string|null;description:string|null;category:DealCategory|null;sentAt:number;kind:string;platforms:string[];endsAt:number|null};
export type SitePageProps={
 site:SiteInfo;
 lang:Language;
 // Where the page lives: "/<slug>" on deals.getblup.site, "" on its own domain.
 base:string;
 origin:string;
 // Where the short links are (https://deals.getblup.site): they count the
 // clicks, on the page's own domain too.
 links:string;
 deals:SiteDeal[];
 query:string|null;
 category:DealCategory|null;
 categories:DealCategory[];
 // The cursor of the next page of deals, when there is one.
 next:number|null;
 // The deal open on its own page (/o/<code>/), when it is one.
 detail?:string;
 // When the page is made: the footer's year follows it.
 now:number;
};

const languageNames:Record<Language,string>={pt:"Português",en:"English",es:"Español"};
export const categoryKey={electronics:"dealsCatElectronics",computers:"dealsCatComputers",phones:"dealsCatPhones",games:"dealsCatGames",home:"dealsCatHome",kitchen:"dealsCatKitchen",fashion:"dealsCatFashion",beauty:"dealsCatBeauty",sports:"dealsCatSports",kids:"dealsCatKids",pets:"dealsCatPets",books:"dealsCatBooks",tools:"dealsCatTools",auto:"dealsCatAuto",food:"dealsCatFood",other:"dealsCatOther"} as const satisfies Record<DealCategory,MessageKey>;

// Lucide's icons (ISC), inline: search, globe, sun, moon, monitor,
// chevron-down, tag, ticket, share-2, arrow-left, external-link,
// chart-line, x, menu, check.
const paths={
 search:<><path d="m21 21-4.34-4.34"/><circle cx="11" cy="11" r="8"/></>,
 globe:<><circle cx="12" cy="12" r="10"/><path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"/><path d="M2 12h20"/></>,
 sun:<><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></>,
 moon:<path d="M20.985 12.486a9 9 0 1 1-9.473-9.472c.405-.022.617.46.402.803a6 6 0 0 0 8.268 8.268c.344-.215.825-.004.803.401"/>,
 monitor:<><rect width="20" height="14" x="2" y="3" rx="2"/><path d="M8 21h8"/><path d="M12 17v4"/></>,
 chevron:<path d="m6 9 6 6 6-6"/>,
 tag:<><path d="M12.586 2.586A2 2 0 0 0 11.172 2H4a2 2 0 0 0-2 2v7.172a2 2 0 0 0 .586 1.414l8.704 8.704a2.426 2.426 0 0 0 3.42 0l6.58-6.58a2.426 2.426 0 0 0 0-3.42z"/><circle cx="7.5" cy="7.5" r=".5" fill="currentColor"/></>,
 ticket:<><path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/><path d="M13 5v2"/><path d="M13 17v2"/><path d="M13 11v2"/></>,
 share:<><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" x2="15.42" y1="13.51" y2="17.49"/><line x1="15.41" x2="8.59" y1="6.51" y2="10.49"/></>,
 back:<><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></>,
 external:<><path d="M15 3h6v6"/><path d="M10 14 21 3"/><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/></>,
 chart:<><path d="M3 3v16a2 2 0 0 0 2 2h16"/><path d="m19 9-5 5-4-4-3 3"/></>,
 x:<><path d="M18 6 6 18"/><path d="m6 6 12 12"/></>,
 check:<path d="M20 6 9 17l-5-5"/>,
 menu:<><path d="M4 5h16"/><path d="M4 12h16"/><path d="M4 19h16"/></>,
} as const;
export function Icon({name,small=false}:{name:keyof typeof paths;small?:boolean}){
 return <svg class={small?"icon icon-sm":"icon"} viewBox="0 0 24 24" aria-hidden="true">{paths[name]}</svg>;
}

// The page's address with a search, a category, a cursor and a language
// (each left out when it is the default).
export function pageHref(props:SitePageProps,change:{q?:string|null;category?:DealCategory|null;before?:number|null;lang?:Language}){
 const params=new URLSearchParams();
 const q=change.q===undefined?props.query:change.q,category=change.category===undefined?props.category:change.category,lang=change.lang??props.lang;
 if(q)params.set("q",q);
 if(category)params.set("category",category);
 if(change.before)params.set("before",String(change.before));
 if(lang!==props.site.language)params.set("lang",lang);
 const search=params.toString();
 return `${props.base}/${search?`?${search}`:""}`;
}

// A deal's own page, in the page's language.
export function dealHref(props:Pick<SitePageProps,"base"|"lang"|"site">,code:string,lang=props.lang){
 return `${props.base}/o/${code}/${lang!==props.site.language?`?lang=${lang}`:""}`;
}

// The logo, and the name beside it when the page shows it (a page
// without a logo always shows its name).
function Brand({site,base}:{site:SiteInfo;base:string}){
 const {light,dark}=site.logos,named=site.showTitle||!(light||dark);
 return <a class="brand" href={`${base}/`} aria-label={named?undefined:site.title}>
  {light?<img class={dark?"when-light":undefined} src={`${base}/logo-light?v=${light}`} alt="" width="160" height="32"/>:null}
  {dark?<img class={light?"when-dark":undefined} src={`${base}/logo-dark?v=${dark}`} alt="" width="160" height="32"/>:null}
  {named?<span>{site.title}</span>:null}
 </a>;
}

// Where a deal is: the store, or for a free game the platforms it is on
// (the source of the giveaways is credited in the posts, not here).
export function placeOf(deal:Pick<SiteDeal,"store"|"kind"|"platforms">){
 if(deal.kind==="game_free"){
  const named=deal.platforms.filter(p=>p!=="pc"&&Object.hasOwn(platformNames,p)).map(p=>platformNames[p]!);
  return named.length?named.join(", "):deal.platforms.includes("pc")?"PC":"";
 }
 return storeTitles[deal.store]??deal.store;
}
// Until when a deal holds, when the store says: "Grátis até 12 de novembro".
export function untilOf(deal:Pick<SiteDeal,"endsAt"|"kind"|"price">,lang:Language,t:Translate,now=Date.now()){
 if(deal.endsAt===null||deal.endsAt<=now)return null;
 const date=new Intl.DateTimeFormat(languageTag[lang],{day:"numeric",month:"long",timeZone:"America/Sao_Paulo"}).format(deal.endsAt);
 return deal.kind==="game_free"||deal.price===0?t("dealsSiteFreeUntil",{date}):t("dealsSiteUntil",{date});
}
// Every photo fills its square box (a wide game cover is cut at the
// sides), with no frame around it.

function DealCard({deal,lang,t,links,detail}:{deal:SiteDeal;lang:Language;t:Translate;links:string;detail:string}){
 const store=placeOf(deal),free=deal.kind==="game_free"||deal.price===0,href=`${links}/r/${deal.code}`,until=untilOf(deal,lang,t);
 return <article class="deal">
  <a class="deal-image" href={detail} tabindex={-1} aria-hidden="true">
   {deal.image?<img src={deal.image} alt="" loading="lazy" decoding="async" referrerpolicy="no-referrer" width="320" height="320"/>:<span class="deal-blank"><Icon name="tag"/></span>}
   {deal.discount&&deal.discount>0?<span class="deal-off">{t("dealsSiteOff",{percent:deal.discount})}</span>:null}
  </a>
  <div class="deal-body">
   {store?<p class="deal-store">{store}</p>:null}
   <h2 class="deal-title"><a href={detail}>{deal.title}</a></h2>
   {deal.description?<p class="deal-text">{deal.description}</p>:null}
   <p class="deal-price">
    {free?<strong>{t("dealsSiteFree")}</strong>:deal.price!==null&&deal.currency?<strong>{money(deal.price,deal.currency,lang)}</strong>:null}
    {!free&&deal.originalPrice&&deal.currency&&deal.price!==null&&deal.originalPrice>deal.price?<s>{money(deal.originalPrice,deal.currency,lang)}</s>:null}
   </p>
   {free&&deal.originalPrice&&deal.currency?<p class="deal-until">{t("dealsSiteWorth",{price:money(deal.originalPrice,deal.currency,lang)})}</p>:null}
   {until?<p class="deal-until deal-ends">{until}</p>:null}
   {deal.coupon?<p class="deal-coupon"><Icon name="ticket" small/>{t("dealsSiteCoupon",{code:deal.coupon})}</p>:null}
   <a class="button button-good deal-go" href={href} rel="nofollow sponsored noopener">{t("dealsSiteSee")}</a>
  </div>
 </article>;
}

// The page's social networks, under its description: each a small pill
// with the network's mark in its own color and its name, so a visitor
// sees at once where else to follow the deals.
const networkColor:Record<SiteNetwork,string>={whatsapp:"#25D366",telegram:"#26A5E4",discord:"#5865F2",instagram:"#FF0069",tiktok:"currentColor"};
function Socials({socials,t}:{socials:SiteSocial[];t:Translate}){
 if(!socials.length)return null;
 return <nav class="socials" aria-label={t("dealsSiteFollow")}>
  <span class="socials-label">{t("dealsSiteFollow")}</span>
  {socials.map(s=><a href={s.url} rel="noopener nofollow me" target="_blank">
   <svg class="icon icon-sm" viewBox="0 0 24 24" aria-hidden="true" style={`fill:${networkColor[s.network]};stroke:none`}><path d={socialMarks[s.network]}/></svg>{networkOf(s.network)?.name??s.network}
  </a>)}
 </nav>;
}

export type Translate=<K extends MessageKey>(key:K,...args:TranslationArgs<(typeof messages)[K][0]>)=>string;
export const translator=(lang:Language):Translate=>(key,...args)=>translate(lang,key,...args);

// What every page of the site shares: the head (with its own title,
// description, preview image and structured data), the header with the
// search, language and theme, and the footer.
export type ShellProps={
 page:SitePageProps;
 title:string;
 description:string;
 canonical:string;
 // The same page in each language.
 alternates:Record<Language,string>;
 image:string|null;
 data:object;
 scripts?:string[];
 children:Child;
};
export function Shell({page:props,title,description,canonical,alternates,image,data,scripts=[],children}:ShellProps){
 const {site,lang}=props,t=translator(lang);
 return <html lang={languageTag[lang]}>
  <head>
   <meta charset="utf-8"/>
   <meta name="viewport" content="width=device-width,initial-scale=1"/>
   <title>{title}</title>
   <meta name="description" content={description}/>
   <link rel="canonical" href={canonical}/>
   {(["pt","en","es"] as const).map(l=><link rel="alternate" hreflang={languageTag[l]} href={alternates[l]}/>)}
   <meta property="og:type" content="website"/>
   <meta property="og:title" content={title}/>
   <meta property="og:description" content={description}/>
   <meta property="og:url" content={canonical}/>
   {image?<meta property="og:image" content={image}/>:null}
   <meta name="twitter:card" content="summary_large_image"/>
   <meta name="theme-color" content="#fafafa" media="(prefers-color-scheme: light)"/>
   <meta name="theme-color" content="#09090b" media="(prefers-color-scheme: dark)"/>
   <link rel="icon" href={site.logos.light?`${props.base}/logo-light?v=${site.logos.light}`:FAVICON}/>
   <script>{raw(THEME_SCRIPT)}</script>
   <script>{raw(SEARCH_SCRIPT)}</script>
   {scripts.map(code=><script>{raw(code)}</script>)}
   <style>{raw(SITE_CSS)}</style>
   <script type="application/ld+json">{raw(JSON.stringify(data).replace(/</g,"\\u003c"))}</script>
  </head>
  <body>
   <header class="site-header">
    <div class="container">
     <Brand site={site} base={props.base}/>
     {site.links.length?<nav class="nav" aria-label={t("dealsSiteMenu")}>{site.links.map(link=><a href={link.url} rel="noopener nofollow">{link.label}</a>)}</nav>:null}
     <div class="header-actions">
      <button class="icon-button" type="button" data-search-open aria-label={t("dealsSiteSearch")}><Icon name="search"/></button>
      <div class="picker">
       <button class="icon-button" type="button" popovertarget="lang-menu" aria-label={`${t("language")}: ${lang}`} style="anchor-name:--lang"><Icon name="globe"/><span class="lang-code">{lang}</span></button>
       <div id="lang-menu" class="dropdown" popover style="position-anchor:--lang" role="menu">
        {(["pt","en","es"] as const).map(l=><a role="menuitemradio" href={alternates[l]} hreflang={languageTag[l]} lang={languageTag[l]} aria-checked={String(l===lang)}>{languageNames[l]}</a>)}
       </div>
      </div>
      <div class="picker theme-picker">
       <button class="icon-button" type="button" popovertarget="theme-menu" aria-label={t("theme")} style="anchor-name:--theme">
        <span class="when-light"><Icon name="sun"/></span><span class="when-dark"><Icon name="moon"/></span><Icon name="chevron" small/>
       </button>
       <div id="theme-menu" class="dropdown" popover style="position-anchor:--theme" role="menu">
        <button type="button" role="menuitemradio" data-set-theme="light"><Icon name="sun"/>{t("light")}</button>
        <button type="button" role="menuitemradio" data-set-theme="dark"><Icon name="moon"/>{t("dark")}</button>
        <button type="button" role="menuitemradio" data-set-theme="system"><Icon name="monitor"/>{t("system")}</button>
       </div>
      </div>
      {site.links.length?<>
       <button class="icon-button menu-toggle" type="button" popovertarget="mobile-menu" aria-label={t("dealsSiteMenu")}><span class="open"><Icon name="menu"/></span><span class="close"><Icon name="x"/></span></button>
       <nav id="mobile-menu" class="mobile-menu" popover aria-label={t("dealsSiteMenu")}>
        {site.links.map(link=><a href={link.url} rel="noopener nofollow">{link.label}</a>)}
       </nav>
      </>:null}
     </div>
    </div>
   </header>
   <dialog id="search" class="search" aria-label={t("dealsSiteSearch")} data-endpoint={`${props.base}/search`} data-lang={lang!==site.language?lang:""} data-empty={t("dealsSiteSearchEmpty")} data-unavailable={t("dealsSiteSearchUnavailable")}>
    <form class="search-field" method="get" action={`${props.base}/`} role="search">
     <Icon name="search"/>
     <input type="search" name="q" placeholder={t("dealsSiteSearchPlaceholder")} autocomplete="off" aria-label={t("dealsSiteSearch")} maxlength={80}/>
     {lang!==site.language?<input type="hidden" name="lang" value={lang}/>:null}
     <button class="icon-button" type="button" data-search-close aria-label={t("dealsSiteClose")}><Icon name="x"/></button>
    </form>
    <div class="search-results" aria-live="polite"></div>
   </dialog>
   <main class="container">{children}</main>
   <footer class="site-footer">
    <div class="container">
     <span>© {new Date(props.now).getUTCFullYear()} {site.title}</span>
     <span>{t("dealsSiteMadeWith")} <a href="https://getblup.io/blup-deals/" rel="noopener">Blup Deals</a></span>
    </div>
   </footer>
  </body>
 </html>;
}

export function SitePage(props:SitePageProps){
 const {site,lang,deals}=props,t=translator(lang);
 const canonical=`${props.origin}${props.base}/`;
 const title=props.query?`${props.query} · ${site.title}`:props.category?`${t(categoryKey[props.category])} · ${site.title}`:site.title;
 // Search engines and link previews: the list of deals on this page.
 const listing={"@context":"https://schema.org","@type":"ItemList",name:site.title,itemListElement:deals.slice(0,12).map((d,i)=>({"@type":"ListItem",position:i+1,url:`${props.origin}${dealHref(props,d.code)}`,name:d.title}))};
 const alternates=Object.fromEntries((["pt","en","es"] as const).map(l=>[l,`${props.origin}${pageHref({...props,query:null,category:null},{lang:l})}`])) as Record<Language,string>;
 const tabs:{href:string;label:string;on:boolean}[]=[{href:pageHref(props,{category:null,q:null}),label:t("dealsSiteAll"),on:!props.category},...props.categories.map(c=>({href:pageHref(props,{category:c,q:null}),label:t(categoryKey[c]),on:props.category===c}))];
 return <Shell page={props} title={title} description={site.description||site.title} canonical={canonical} alternates={alternates} image={deals[0]?.image??null} data={listing}>
  <section class="intro">
   <h1>{site.title}</h1>
   {site.description?<p>{site.description}</p>:null}
   <Socials socials={site.socials} t={t}/>
  </section>
  {tabs.length>1?<nav class="chips" aria-label={t("dealsSiteCategories")}>{tabs.map(tab=><a href={tab.href} aria-current={tab.on?"page":undefined}>{tab.label}</a>)}</nav>:null}
  {props.query?<p class="results">{deals.length?null:t("dealsSiteNothing",{query:props.query})} <a href={pageHref(props,{q:null})}>{t("dealsSiteClear")}</a></p>:null}
  {deals.length?<div class="deals">{deals.map(deal=><DealCard deal={deal} lang={lang} t={t} links={props.links} detail={dealHref(props,deal.code)}/>)}</div>:props.query?null:<p class="empty">{t("dealsSiteEmpty")}</p>}
  {props.next?<p class="more"><a class="button button-ghost" href={pageHref(props,{before:props.next})} rel="next">{t("dealsSiteMore")}</a></p>:null}
 </Shell>;
}

// Deals' coral blup, for a page without its own logo.
const FAVICON="data:image/svg+xml,"+encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 200"><path d="M100 26C121 26 177 103 177 137C177 165 155 176 103 176C53 176 25 164 25 139C25 105 79 26 100 26Z" fill="#ff7a59" stroke="#0e1b45" stroke-width="3"/><rect x="98" y="110" width="10" height="26" rx="3" fill="#0B1220"/><rect x="131" y="108" width="9" height="24" rx="2.7" fill="#0B1220"/></svg>`);

// Mochi's tokens (zinc, light-dark()) and header, made smaller; the deals
// speak in the colors of a good buy: the price now and the button in
// green, the old price struck in red, the end of a sale in amber: one file,
// inline, so the page needs no other request to paint.
export const SITE_CSS=`@layer reset,tokens,base,layout,components;
@layer reset{*,*::before,*::after{box-sizing:border-box;margin:0}img,svg{display:block;max-width:100%}img{height:auto}a{color:inherit;text-decoration:none}button{font:inherit;color:inherit;background:none;border:0;cursor:pointer}input{font:inherit;color:inherit}}
@layer tokens{:root{color-scheme:light dark;--main-25:light-dark(#fafafa,#09090b);--main-100:light-dark(#f4f4f5,#18181b);--main-200:light-dark(#e4e4e7,#27272a);--main-300:light-dark(#d4d4d8,#3f3f46);--main-400:light-dark(#a1a1aa,#71717a);--main-500:light-dark(#6e6e76,#8f8f96);--main-600:light-dark(#52525b,#a1a1aa);--main-700:light-dark(#3f3f46,#d4d4d8);--main-900:light-dark(#18181b,#f4f4f5);--card:light-dark(#ffffff,#18181b);--accent:light-dark(#c2410c,#fb923c);--on-accent:#ffffff;--good:light-dark(#15803d,#4ade80);--good-fill:light-dark(#16a34a,#22c55e);--on-good:light-dark(#ffffff,#052e16);--bad:light-dark(#b91c1c,#f87171);--soon:light-dark(#b45309,#fbbf24);--primary:light-dark(#18181b,#f4f4f5);--on-primary:light-dark(#ffffff,#18181b);--font:"Inter",ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;--container:80rem;--gutter:1rem;--header-h:4rem;--ease:cubic-bezier(.4,0,.2,1);--shadow-sm:0 1px 2px 0 rgb(0 0 0/.05);--shadow-lg:0 10px 15px -3px rgb(0 0 0/.1),0 4px 6px -4px rgb(0 0 0/.1)}:root[data-theme="light"]{color-scheme:light}:root[data-theme="dark"]{color-scheme:dark}@media (width>=40rem){:root{--gutter:1.5rem}}}
@layer base{html{background:var(--main-25)}body{font-family:var(--font);line-height:1.5;background:var(--main-25);color:var(--main-900);-webkit-font-smoothing:antialiased;min-height:100dvh;display:flex;flex-direction:column}main{flex:1}:focus-visible{outline:2px solid var(--accent);outline-offset:2px;border-radius:.25rem}.icon{width:1.25rem;height:1.25rem;flex-shrink:0;stroke:currentColor;fill:none;stroke-width:1.75;stroke-linecap:round;stroke-linejoin:round}.icon-sm{width:1rem;height:1rem}}
@layer layout{.container{width:100%;max-width:var(--container);margin-inline:auto;padding-inline:var(--gutter)}
.site-header{position:sticky;top:0;z-index:50;background:color-mix(in srgb,var(--main-100) 92%,transparent);backdrop-filter:saturate(1.4) blur(8px);box-shadow:var(--shadow-sm)}.site-header .container{display:flex;align-items:center;height:var(--header-h);gap:.5rem}
.brand{display:flex;align-items:center;gap:.5rem;min-width:0;font-size:1.0625rem;font-weight:600;letter-spacing:-.01em}.brand span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.brand img{height:2rem;width:auto}
.nav{display:none;flex:1;justify-content:center;gap:1.25rem}.nav a{font-weight:500;color:var(--main-600);padding:.5rem .75rem;border-radius:.5rem}.nav a:hover,.nav a[aria-current]{background:color-mix(in srgb,var(--main-200) 60%,transparent);color:var(--main-900)}
.header-actions{display:flex;align-items:center;gap:.25rem;margin-inline-start:auto}.icon-button{display:inline-flex;align-items:center;gap:.125rem;padding:.5rem;border-radius:.25rem;color:var(--main-600)}.icon-button:hover{background:color-mix(in srgb,var(--main-200) 60%,transparent);color:var(--main-900)}.lang-code{font-size:.75rem;font-weight:600;text-transform:uppercase}
.menu-toggle .close{display:none}body:has(.mobile-menu:popover-open) .menu-toggle .open{display:none}body:has(.mobile-menu:popover-open) .menu-toggle .close{display:block}
.mobile-menu{position:fixed;inset:var(--header-h) 0 auto 0;margin:0;width:100%;max-width:none;border:0;border-block-end:1px solid var(--main-200);padding:.5rem var(--gutter);background:var(--main-100);color:var(--main-900);box-shadow:var(--shadow-lg)}.mobile-menu a{display:block;padding:.75rem 0;font-weight:500}
@media (width>=48rem){.nav{display:flex}.menu-toggle{display:none}}
.site-footer .container{display:flex;justify-content:space-between;flex-wrap:wrap;gap:1rem;padding-block:2.5rem;font-size:.8125rem;color:var(--main-500)}.site-footer a{font-weight:600}.site-footer a:hover{text-decoration:underline}}
@layer components{.dropdown{margin:0;inset:auto;padding:.25rem;border:1px solid var(--main-200);border-radius:.375rem;background:var(--main-100);color:var(--main-700);box-shadow:var(--shadow-lg);min-width:9rem;position-area:bottom span-left;margin-block-start:.375rem}@supports not (position-area:bottom){.dropdown{inset:calc(var(--header-h) - .25rem) 1rem auto auto}}
.dropdown a,.dropdown button{display:flex;width:100%;align-items:center;gap:.75rem;padding:.5rem .75rem;border-radius:.25rem;font-size:.875rem}.dropdown a:hover,.dropdown button:hover,.dropdown [aria-checked="true"]{background:var(--main-200);color:var(--main-900)}
:root:not(.js) .theme-picker{display:none}.when-dark{display:none}@media (prefers-color-scheme:dark){:root:not([data-theme="light"]) .when-light{display:none}:root:not([data-theme="light"]) .when-dark{display:block}}:root[data-theme="dark"] .when-light{display:none}:root[data-theme="dark"] .when-dark{display:block}
.search{margin:5rem auto auto;width:min(48rem,calc(100% - 2rem));padding:0;border:0;border-radius:.75rem;background:var(--main-25);color:var(--main-900);box-shadow:0 25px 50px -12px rgb(0 0 0/.25);max-height:calc(100dvh - 8rem);overflow:hidden}.search::backdrop{background:rgb(9 9 11/.4)}
.search-field{display:flex;align-items:center;gap:.75rem;padding:0 1rem;border-block-end:1px solid var(--main-200)}.search-field>.icon{color:var(--main-400)}.search-field input{flex:1;min-width:0;height:3rem;border:0;outline:0;background:transparent}.search-field input::-webkit-search-cancel-button{appearance:none}
.search-results{overflow-y:auto;max-height:calc(100dvh - 12rem);padding:.5rem}.search-results:empty{display:none}.search-results a{display:block;padding:.75rem;border-radius:.5rem}.search-results a:hover,.search-results a:focus-visible{background:var(--main-100)}.search-results strong{display:block;font-weight:600}.search-results p{font-size:.875rem;color:var(--main-600);margin-block-start:.25rem}.search-results mark{background:none;color:var(--accent);font-weight:600}.search-status{padding:1.25rem;text-align:center;font-size:.875rem;color:var(--main-500)}:root:not(.js) [data-search-open]{display:none}
.button{display:inline-flex;align-items:center;justify-content:center;gap:.5rem;padding:.5rem 1rem;border-radius:.375rem;font-weight:600;background:var(--primary);color:var(--on-primary);transition:opacity .2s}.button:hover{opacity:.85}.button-ghost{background:transparent;color:var(--main-900);box-shadow:inset 0 0 0 1px var(--main-300)}
.intro{padding-block:2.5rem 1.5rem}.intro h1{font-size:clamp(1.75rem,1.4rem + 1.5vw,2.5rem);font-weight:700;line-height:1.1;letter-spacing:-.025em}.intro p{margin-block-start:.5rem;max-width:42rem;color:var(--main-600)}.socials{display:flex;flex-wrap:wrap;align-items:center;gap:.5rem;margin-block-start:1rem}.socials-label{font-size:.8125rem;font-weight:600;color:var(--main-500);margin-inline-end:.25rem}.socials a{display:inline-flex;align-items:center;gap:.375rem;padding:.3125rem .75rem;border:1px solid var(--main-200);border-radius:999px;background:var(--card);font-size:.8125rem;font-weight:600;color:var(--main-700);transition:border-color .2s,color .2s}.socials a:hover{border-color:var(--main-400);color:var(--main-900)}
.chips{display:flex;gap:.5rem;overflow-x:auto;scrollbar-width:none;padding-block-end:1.25rem}.chips a{flex:none;padding:.375rem .875rem;border:1px solid var(--main-200);border-radius:999px;font-size:.875rem;font-weight:500;color:var(--main-600)}.chips a[aria-current]{background:var(--main-900);border-color:var(--main-900);color:var(--main-25)}
.results{margin-block-end:1rem;color:var(--main-600)}.results a{text-decoration:underline}.empty{padding-block:4rem;text-align:center;color:var(--main-500)}
.deals{display:grid;gap:1rem;grid-template-columns:repeat(auto-fill,minmax(15rem,1fr))}
.deal{display:flex;flex-direction:column;overflow:hidden;border:1px solid var(--main-200);border-radius:.75rem;background:var(--card)}.deal-image{position:relative;display:block;aspect-ratio:1;overflow:hidden;background:var(--main-100)}.deal-image img{width:100%;height:100%;object-fit:cover}.deal-blank{display:grid;place-items:center;height:100%;color:var(--main-500)}
.deal-off{position:absolute;inset:.5rem auto auto .5rem;padding:.125rem .5rem;border-radius:999px;background:var(--good-fill);color:var(--on-good);box-shadow:var(--shadow-sm);font-size:.75rem;font-weight:700}
.deal-body{display:flex;flex:1;flex-direction:column;gap:.375rem;padding:1rem}.deal-store{font-size:.75rem;font-weight:600;text-transform:uppercase;letter-spacing:.04em;color:var(--main-500)}.deal-title{font-size:1rem;font-weight:600;line-height:1.35}.deal-title a:hover{text-decoration:underline}.deal-text{font-size:.875rem;color:var(--main-600)}
.deal-price{display:flex;align-items:baseline;gap:.5rem;margin-block-start:auto;padding-block-start:.25rem}.deal-price strong{font-size:1.25rem;color:var(--good)}.deal-price s{font-size:.875rem;color:var(--bad);opacity:.85;text-decoration-thickness:1.5px}
.deal-coupon{display:inline-flex;align-items:center;gap:.375rem;width:fit-content;padding:.125rem .5rem;border:1px dashed var(--main-300);border-radius:.375rem;font-size:.8125rem;font-weight:600}.deal-go{margin-block-start:.5rem}.deal-until{font-size:.8125rem;font-weight:500;color:var(--main-600)}.deal-ends{color:var(--soon);font-weight:600}
.more{display:flex;justify-content:center;padding-block:2rem}
.button-accent,.button-good{background:var(--good-fill);color:var(--on-good)}:root:not(.js) .share{display:none}.share .copied-icon{display:none}.share.copied{color:light-dark(#15803d,#4ade80);box-shadow:inset 0 0 0 1px currentColor;background:color-mix(in srgb,currentColor 10%,transparent)}.share.copied .share-icon{display:none}.share.copied .copied-icon{display:inline-flex}
.back{padding-block:1.5rem 1rem;font-size:.875rem}.back a{display:inline-flex;align-items:center;gap:.375rem;color:var(--main-600)}.back a:hover{color:var(--main-900)}
.detail{display:grid;gap:1.5rem}@media (width>=48rem){.detail{grid-template-columns:minmax(0,26rem) minmax(0,1fr);gap:2.5rem;align-items:start}}
.detail-image{position:relative;aspect-ratio:1;border:1px solid var(--main-200);border-radius:.75rem;overflow:hidden;background:var(--main-100)}.detail-image img{width:100%;height:100%;object-fit:cover}
.detail-body{display:flex;flex-direction:column;align-items:flex-start;gap:.75rem}.detail-meta{display:flex;flex-wrap:wrap;align-items:center;gap:.5rem;font-size:.8125rem;color:var(--main-500)}.detail-store{padding:.125rem .5rem;border-radius:.375rem;background:var(--main-200);color:var(--main-900);font-weight:600}
.detail-title{font-size:clamp(1.375rem,1.2rem + .8vw,1.875rem);font-weight:700;line-height:1.2;letter-spacing:-.02em}.detail-price{display:flex;align-items:baseline;flex-wrap:wrap;gap:.625rem}.detail-price strong{font-size:2rem;line-height:1.1;color:var(--good)}.detail-price s{color:var(--bad);opacity:.85;text-decoration-thickness:1.5px}.detail-text{color:var(--main-600);max-width:40rem}
.detail-go,.share{width:100%;max-width:24rem;padding-block:.75rem}.detail-ended{padding:.75rem 1rem;border-radius:.5rem;background:var(--main-100);color:var(--main-600)}
.history{padding-block:2.5rem 1rem}.history h2{font-size:1.25rem;font-weight:700}.history-help{color:var(--main-600);margin-block-end:1rem}.history-card{border:1px solid var(--main-200);border-radius:.75rem;background:var(--card);padding:1.25rem;margin-block-end:1rem}
.history-mood{display:flex;align-items:center;gap:.5rem;font-weight:600}.mood-below{color:light-dark(#15803d,#4ade80)}.mood-above{color:light-dark(#b91c1c,#f87171)}.mood-average{color:var(--main-700)}
.bar{margin-block-start:1rem;padding-block-start:3rem}.bar-track{position:relative;height:.5rem;border-radius:999px;background:linear-gradient(90deg,#22c55e,#eab308,#f97316,#ef4444)}.bar-mark{position:absolute;top:50%;width:1rem;height:1rem;border:3px solid var(--card);border-radius:999px;background:var(--main-900);transform:translate(-50%,-50%);box-shadow:var(--shadow-sm)}
.bar-arrow{position:absolute;width:0;height:0;border-inline:.5rem solid transparent;transform:translateX(-50%)}.bar-arrow-down{bottom:calc(100% + .45rem);border-block-start:.7rem solid var(--accent)}.bar-arrow-up{top:calc(100% + .45rem);border-block-end:.7rem solid var(--accent)}
.bar-now{position:absolute;bottom:calc(100% + 1.4rem);font-size:.8125rem;font-weight:700;color:var(--main-900);white-space:nowrap}.bar-now-center{transform:translateX(-50%)}.bar-now-start{transform:translateX(-.5rem)}.bar-now-end{transform:translateX(calc(-100% + .5rem))}
.bar-ends{display:flex;justify-content:space-between;flex-wrap:wrap;gap:.5rem;margin-block-start:1.75rem;font-size:.8125rem;color:var(--main-500)}
.chart{margin:0}.chart-plot{display:grid;grid-template-columns:auto minmax(0,1fr);gap:.75rem;height:14rem}.chart-y{position:relative;width:5.5rem;font-size:.75rem;color:var(--main-500);text-align:right}.chart-y span{position:absolute;right:0;transform:translateY(-50%);white-space:nowrap}
.chart svg{display:block;width:100%;height:14rem;min-width:0;overflow:visible}.chart-grid{stroke:var(--main-200);stroke-width:1;vector-effect:non-scaling-stroke}.chart-line{fill:none;stroke:var(--accent);stroke-width:2;vector-effect:non-scaling-stroke;stroke-linejoin:round}.chart-area{fill:color-mix(in srgb,var(--accent) 16%,transparent)}
.chart-x{display:flex;justify-content:space-between;margin-inline-start:6.25rem;margin-block-start:.5rem;font-size:.75rem;color:var(--main-500)}}`;

// One deal as a search result: its page, its name and where and for how
// much (the dialog shows them as the blog themes show their posts).
export function searchResult(props:Pick<SitePageProps,"base"|"lang"|"site">,deal:SiteDeal){
 const t=translator(props.lang),free=deal.kind==="game_free"||deal.price===0;
 const price=free?t("dealsSiteFree"):deal.price!==null&&deal.currency?money(deal.price,deal.currency,props.lang):null;
 return {url:dealHref(props,deal.code),title:deal.title,detail:[placeOf(deal),price].filter(Boolean).join(" · ")};
}

// The search dialog, as in the Nori and Mochi themes: the magnifier opens
// it, results come as the person types (from /search, the server's own
// search: these pages have no Pagefind index), Enter opens the first one
// (or, with none, the list of results); a click outside it (on the
// backdrop: the dialog itself) closes it. Allowed by its hash.
export const SEARCH_SCRIPT=`(function(){var seq=0,timer;function dialog(){return document.getElementById("search")}function esc(s){return s.replace(/[&<>"]/g,function(c){return{"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;"}[c]})}function mark(s,q){var k=s.toLowerCase().indexOf(q.toLowerCase());return k<0?esc(s):esc(s.slice(0,k))+"<mark>"+esc(s.slice(k,k+q.length))+"</mark>"+esc(s.slice(k+q.length))}function run(){var d=dialog(),i=d.querySelector("input[name=q]"),o=d.querySelector(".search-results"),q=i.value.trim(),n=++seq;if(!q){o.replaceChildren();return}fetch(d.dataset.endpoint+"?q="+encodeURIComponent(q)+(d.dataset.lang?"&lang="+d.dataset.lang:"")).then(function(r){return r.ok?r.json():Promise.reject()}).then(function(j){if(n!==seq)return;o.innerHTML=j.results.length?j.results.map(function(r){return'<a href="'+esc(r.url)+'"><strong>'+mark(r.title,q)+"</strong><p>"+esc(r.detail)+"</p></a>"}).join(""):'<p class="search-status">'+esc(d.dataset.empty)+"</p>"}).catch(function(){if(n===seq)o.innerHTML='<p class="search-status">'+esc(d.dataset.unavailable)+"</p>"})}document.addEventListener("click",function(e){var d=dialog();if(!d||!e.target.closest)return;if(e.target.closest("[data-search-open]")){d.showModal();d.querySelector("input[name=q]").focus();return}if(e.target===d||e.target.closest("[data-search-close]"))d.close()});document.addEventListener("input",function(e){if(e.target.name==="q"&&e.target.closest("#search")){clearTimeout(timer);timer=setTimeout(run,150)}});document.addEventListener("submit",function(e){if(!e.target.closest("#search"))return;var a=dialog().querySelector(".search-results a");if(a){e.preventDefault();a.click()}})})();`;
