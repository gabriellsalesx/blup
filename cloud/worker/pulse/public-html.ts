import {translate,type messages,type MessageKey} from "../../src/lib/i18n/catalog";
import type {Language,TranslationArgs} from "../../shared/language";

// The public status page: one HTML file with inline styles and inline
// Lucide icons (the CSP allows nothing else). It only reads the closed
// public DTO, so private names, URLs and DSNs never reach it.

export type PublicDay={date:string;up:number;down:number;unknown:number;maintenance:number};
export type PublicComponent={name:string;state:"up"|"down"|"unknown";incident:"downtime"|"degraded"|null;maintenance:boolean;lastCheckedAt:string|null;days:PublicDay[]};
export type PublicMaintenance={title:string;body:string;state:"scheduled"|"active";startsAt:string;endsAt:string;services:string[]};
export type PublicUpdate={body:string;state:string;createdAt:string};
export type PublicService={name:string;status:"downtime"|"degraded"|"resolved"};
export type PublicIncident={title:string;state:string;impact:string;createdAt:string;updatedAt:string;services:PublicService[];updates:PublicUpdate[]};
export type PublicLook={showTitle:boolean;links:{label:string;url:string}[];logos:{light:string|null;dark:string|null}};
export type PublicStatus={title:string;language:Language;historyDays:number;components:PublicComponent[];incidents:PublicIncident[];maintenance:PublicMaintenance[]}&PublicLook;

const escape=(v:string)=>v.replace(/[&<>"']/g,c=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]??""));
const DAY=86_400_000;

// Lucide icons (check, triangle-alert, x, wrench, circle-help), copied as paths.
const icon={
 up:'<path d="M20 6 9 17l-5-5"/>',
 partial:'<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"/><path d="M12 9v4"/><path d="M12 17h.01"/>',
 down:'<path d="M18 6 6 18"/><path d="m6 6 12 12"/>',
 maint:'<path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.106-3.105c.32-.322.863-.22.983.218a6 6 0 0 1-8.259 7.057l-7.91 7.91a1 1 0 0 1-2.999-3l7.91-7.91a6 6 0 0 1 7.057-8.259c.438.12.54.662.219.984z"/>',
 unknown:'<circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><path d="M12 17h.01"/>',
} as const;
type Tone=keyof typeof icon;
// The legend of the services card (Lucide circle-check, triangle-alert,
// circle-x, wrench and circle), gray until hovered: the same marks as the
// services' own state.
const legendIcon={
 up:'<circle cx="12" cy="12" r="10"/><path d="m16 9-5.5 5.5L8 12"/>',
 partial:'<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"/><path d="M12 9v4"/><path d="M12 17h.01"/>',
 down:'<circle cx="12" cy="12" r="10"/><path d="m15 9-6 6"/><path d="m9 9 6 6"/>',
 maint:'<path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.106-3.105c.32-.322.863-.22.983.218a6 6 0 0 1-8.259 7.057l-7.91 7.91a1 1 0 0 1-2.999-3l7.91-7.91a6 6 0 0 1 7.057-8.259c.438.12.54.662.219.984z"/>',
 none:'<circle cx="12" cy="12" r="10"/>',
} as const;
const legendSvg=(k:keyof typeof legendIcon)=>`<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${legendIcon[k]}</svg>`;
const svg=(tone:Tone,size:number,width=2)=>`<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="${width}" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${icon[tone]}</svg>`;
// The mint Blup of the Pulse, the same drawing as its favicon.
const blup='<svg width="34" height="34" viewBox="0 0 64 64" fill="none" stroke="#0e1b45" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M33 4C38 10 58 29 58 41C58 53 47 59 32 59C17 59 6 53 6 41C6 29 24 13 33 4Z" fill="#9ee6c8"/><rect x="29" y="33.5" width="4" height="11" rx="1.4" fill="#0B1220" stroke="none" transform="rotate(8 31 39)"/><rect x="41" y="32.5" width="4" height="11" rx="1.4" fill="#0B1220" stroke="none" transform="rotate(8 43 38)"/></svg>';

const css=`:root{color-scheme:light dark;--bg:#fafafa;--card:#fff;--head:#fafafa;--line:#e5e5e5;--text:#171717;--muted:#737373;--up:#10b981;--partial:#f59e0b;--down:#ef4444;--maint:#6366f1;--none:#e5e5e5;--legend:#d4d4d4;--tip:#262626;--tip-text:#fff}
@media (prefers-color-scheme:dark){:root{--bg:#0a0a0a;--card:#171717;--head:#1c1c1c;--line:#262626;--text:#f5f5f5;--muted:#a3a3a3;--none:#303030;--legend:#525252;--tip:#f5f5f5;--tip-text:#171717}}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font:15px/1.5 Inter,ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;-webkit-font-smoothing:antialiased}
.wrap{max-width:780px;margin:0 auto;padding:56px 20px 32px}
.site-header{position:sticky;top:0;z-index:50;background:color-mix(in srgb,var(--bg) 92%,transparent);backdrop-filter:saturate(1.4) blur(8px);box-shadow:0 1px 2px 0 rgb(0 0 0/.05)}
.bar{display:flex;align-items:center;gap:8px;max-width:80rem;height:64px;margin:0 auto;padding:0 24px}
.brand{display:flex;align-items:center;gap:8px;flex-shrink:1;min-width:0;font-size:17px;font-weight:600;letter-spacing:-.01em;color:var(--text);text-decoration:none}.brand span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.logo{display:block;height:32px;width:auto;max-width:220px;object-fit:contain;margin-bottom:3px}
.nav{display:none;flex:1;justify-content:center;gap:20px;padding:0 20px}.nav a{font-weight:500;color:var(--muted);padding:8px 12px;border-radius:8px;text-decoration:none;transition:background-color .3s}.nav a:hover{background:color-mix(in srgb,var(--line) 60%,transparent);color:var(--text)}
.menu-toggle{display:inline-flex;margin-left:auto;padding:8px;border:0;border-radius:4px;background:none;color:var(--muted);cursor:pointer}.menu-toggle:hover{background:color-mix(in srgb,var(--line) 60%,transparent);color:var(--text)}.menu-toggle .close{display:none}
body:has(.mobile-menu:popover-open) .menu-toggle .open{display:none}body:has(.mobile-menu:popover-open) .menu-toggle .close{display:block}
.mobile-menu{position:fixed;inset:64px 0 auto 0;margin:0;width:100%;max-width:none;border:0;border-bottom:1px solid var(--line);padding:8px 20px;background:var(--card);color:var(--text);box-shadow:0 10px 15px -3px rgb(0 0 0/.1),0 4px 6px -4px rgb(0 0 0/.1)}.mobile-menu a{display:block;padding:12px 0;font-weight:500;color:var(--text);text-decoration:none}
@media (min-width:768px){.nav{display:flex}.menu-toggle{display:none}}
.hero{text-align:center;margin-bottom:48px}.badge{display:inline-grid;place-items:center;width:80px;height:80px;border-radius:999px;color:#fff}.badge svg{color:#fff}
h1{font-size:30px;line-height:1.2;margin:22px 0 8px;font-weight:700;letter-spacing:-.02em}.updated{color:var(--muted);font-size:14px;margin:0}
.card{background:var(--card);border:1px solid var(--line);border-radius:12px;margin-bottom:28px}
.head{display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap;padding:16px 20px;background:var(--head);border-bottom:1px solid var(--line);border-radius:12px 12px 0 0}.head h2{margin:0;font-size:16px;font-weight:600}
.legend{display:flex;gap:14px}.legend span{display:inline-grid;color:var(--legend);transition:color .15s}.legend .l-up:hover{color:var(--up)}.legend .l-partial:hover{color:var(--partial)}.legend .l-down:hover{color:var(--down)}.legend .l-maint:hover{color:var(--maint)}.legend .l-none:hover{color:var(--muted)}
.row{padding:16px 20px;border-top:1px solid var(--line)}.head+.row{border-top:0}
.line{display:flex;align-items:center;justify-content:space-between;gap:16px}.name{display:flex;align-items:center;gap:8px;font-weight:600;min-width:0;overflow-wrap:anywhere}
.dot{flex:none;display:inline-grid;place-items:center;width:18px;height:18px;border-radius:999px;color:#fff}.dot svg{color:#fff}
.state{font-size:14px;white-space:nowrap}.pct{color:var(--muted);font-size:14px;white-space:nowrap}
.bars{display:flex;gap:2px;height:36px;margin-top:12px}.bars i{flex:1;min-width:2px;border-radius:999px;background:var(--none)}
.from-short{display:none}.scale{display:flex;justify-content:space-between;color:var(--muted);font-size:12px;margin-top:6px}
.up,.bars i.up{background:var(--up)}.partial,.bars i.partial{background:var(--partial)}.down,.bars i.down{background:var(--down)}.maint,.bars i.maint{background:var(--maint)}.bars i.none{background:var(--none)}

.t-up{color:var(--up)}.t-partial{color:var(--partial)}.t-down{color:var(--down)}.t-maint{color:var(--maint)}.t-unknown{color:var(--muted)}
.b-unknown{background:var(--muted)}
[data-tip]{position:relative}[data-tip]:hover::after{content:attr(data-tip);position:absolute;bottom:calc(100% + 8px);left:50%;transform:translateX(-50%);z-index:10;background:var(--tip);color:var(--tip-text);padding:4px 8px;border-radius:6px;font:500 13px/1.4 Inter,ui-sans-serif,system-ui,sans-serif;white-space:pre;text-align:center;pointer-events:none}
[data-tip]:hover::before{content:"";position:absolute;bottom:calc(100% + 3px);left:50%;transform:translateX(-50%);z-index:10;border:5px solid transparent;border-top-color:var(--tip);border-bottom:0;pointer-events:none}
.bars i:nth-child(-n+6):hover::after{left:0;transform:none}.bars i:nth-last-child(-n+6):hover::after{left:auto;right:0;transform:none}
.day{padding:18px 20px 4px}.day h3{margin:0 0 10px;padding-bottom:8px;border-bottom:1px solid var(--line);font-size:15px;font-weight:600}.day>p{color:var(--muted);margin:0 0 14px}
.incident{margin:0 0 16px}.services{display:flex;flex-wrap:wrap;gap:4px 16px;margin:0 0 8px;font-size:13px;color:var(--muted)}.incident h4{margin:0 0 6px;font-size:15px}.tag{display:inline-block;font-size:12px;font-weight:600;padding:1px 8px;border-radius:999px;margin-left:6px;vertical-align:1px;background:var(--head);border:1px solid var(--line)}
.update{margin:6px 0 0}.update p{margin:0;white-space:pre-wrap;overflow-wrap:anywhere}.update small{color:var(--muted)}

footer{display:flex;justify-content:space-between;gap:16px;flex-wrap:wrap;color:var(--muted);font-size:13px;padding:8px 2px}footer a{color:inherit;font-weight:600;text-decoration:none}footer a:hover{text-decoration:underline}
@media (max-width:640px){.wrap{padding-top:36px}.bar{padding:0 16px}h1{font-size:24px}.bars i:nth-last-child(n+31){display:none}.bars i:nth-last-child(30):hover::after{left:0;transform:none}.from-long{display:none}.from-short{display:inline}}
`;

// Day key (UTC) of a date, as days since the epoch.
const dayOf=(ms:number)=>Math.floor(ms/DAY);
const iso=(day:number)=>new Date(day*DAY).toISOString().slice(0,10);

function dayTone(d:PublicDay|undefined):Tone|"none"{
 // Checks during a maintenance count apart: a day with one shows it,
 // unless the other checks of the day found the service down.
 const total=d?d.up+d.down:0;
 if(!d||total===0)return d?.maintenance?"maint":"none";
 const ratio=d.up/total,tone=ratio===1?"up":ratio>=0.9?"partial":"down";
 return tone==="up"&&d.maintenance?"maint":tone;
}

// A service's state: maintenance while one lasts, else its monitor's,
// unless an open incident says it is down or slow.
const toneOf=(c:PublicComponent):Tone=>c.maintenance?"maint":c.state==="down"||c.incident==="downtime"?"down":c.incident==="degraded"?"partial":c.state==="up"?"up":"unknown";
type OverallKey="pulseWaiting"|"pulseAllDown"|"pulseSomeDown"|"pulseMaintenanceNow"|"pulseAllUp";
// The page's overall state, from the services (and an open incident): the
// top of the page, its favicon and the badge all show it.
export function overallOf(page:Pick<PublicStatus,"components"|"incidents">):{tone:Tone;key:OverallKey}{
 const states=page.components.map(c=>{const tone=toneOf(c);return tone==="partial"||tone==="maint"?"up":tone;}),open=page.incidents.filter(i=>i.state!=="resolved");
 const down=states.filter(s=>s==="down").length,known=states.filter(s=>s!=="unknown").length;
 return !page.components.length||known===0?{tone:"unknown",key:"pulseWaiting"}:
  down&&down===known?{tone:"down",key:"pulseAllDown"}:
  down||open.length?{tone:"partial",key:"pulseSomeDown"}:
  page.components.some(c=>c.maintenance)?{tone:"maint",key:"pulseMaintenanceNow"}:
  {tone:"up",key:"pulseAllUp"};
}
// The state as a favicon: the same colored circle and Lucide mark.
const toneColor:Record<Tone,string>={up:"#10b981",partial:"#f59e0b",down:"#ef4444",maint:"#6366f1",unknown:"#737373"};
export function faviconOf(tone:Tone){
 const mark=`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><circle cx="16" cy="16" r="16" fill="${toneColor[tone]}"/><g transform="translate(6 6) scale(.8333)" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">${icon[tone]}</g></svg>`;
 return `data:image/svg+xml,${encodeURIComponent(mark)}`;
}
// The header, as in the Blup themes (Nori, Bento, Mochi): a bar across the
// top with the logos (the dark one on a dark screen, when there is one, else
// the Blup) and the title unless hidden on the left, the page's links in
// the middle and, on a small screen, a menu with them (a popover: the page
// runs no script).
const menuIcon=(name:"open"|"close",paths:string)=>`<svg class="${name}" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths}</svg>`;
function headerOf(page:PublicStatus){
 const {light,dark}=page.logos,alt=page.showTitle?"":escape(page.title);
 const img=(file:string,version:string)=>`<img class="logo" src="${file}?v=${version}" alt="${alt}">`;
 const logo=light&&dark?`<picture><source srcset="logo-dark?v=${dark}" media="(prefers-color-scheme: dark)">${img("logo-light",light)}</picture>`:light?img("logo-light",light):dark?img("logo-dark",dark):blup;
 const links=page.links.map(l=>`<a href="${escape(l.url)}" rel="noopener nofollow">${escape(l.label)}</a>`).join("");
 const menu=page.links.length?`<nav class="nav" aria-label="Menu">${links}</nav><button class="menu-toggle" type="button" popovertarget="mobile-menu" aria-label="Menu">${menuIcon("open",'<path d="M4 5h16"/><path d="M4 12h16"/><path d="M4 19h16"/>')}${menuIcon("close",'<path d="M18 6 6 18"/><path d="m6 6 12 12"/>')}</button><nav id="mobile-menu" class="mobile-menu" popover aria-label="Menu">${links}</nav>`:"";
 return `<header class="site-header"><div class="bar"><a class="brand" href="./"${page.showTitle?"":` aria-label="${escape(page.title)}"`}>${logo}${page.showTitle?`<span>${escape(page.title)}</span>`:""}</a>${menu}</div></header>`;
}
// The badge for another site (in an iframe): the state's mark and words,
// linking to the page. Light, dark or following the visitor's screen.
export function renderBadge(page:PublicStatus,theme:"light"|"dark"|"auto"){
 const state=overallOf(page),label=translate(page.language,state.key);
 const ink={light:"#171717",dark:"#f5f5f5"};
 const color=theme==="auto"?`color:${ink.light}}@media (prefers-color-scheme:dark){a{color:${ink.dark}}`:`color:${ink[theme]}}`;
 return `<!doctype html><html lang="${page.language}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="refresh" content="60"><title>${escape(label)}</title><style>html,body{margin:0;background:transparent;overflow:hidden}a{display:inline-flex;align-items:center;gap:8px;height:30px;font:500 14px/1 Inter,ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;text-decoration:none;white-space:nowrap;-webkit-font-smoothing:antialiased;${color}a:hover span:last-child{text-decoration:underline}.dot{display:inline-grid;place-items:center;width:18px;height:18px;border-radius:999px;background:${toneColor[state.tone]};color:#fff}</style></head><body><a href="./" target="_blank" rel="noopener"><span class="dot">${svg(state.tone,12,3)}</span><span>${escape(label)}</span></a></body></html>`;
}
export function renderStatusPage(page:PublicStatus,now=Date.now()):string{
 const lang=page.language;
 const t=<K extends MessageKey>(key:K,...args:TranslationArgs<(typeof messages)[K][0]>)=>translate(lang,key,...args);
 const number=new Intl.NumberFormat(lang,{maximumFractionDigits:2});
 const longDate=new Intl.DateTimeFormat(lang,{dateStyle:"long",timeZone:"UTC"});
 const shortDate=new Intl.DateTimeFormat(lang,{day:"numeric",month:"short",timeZone:"UTC"});
 const stamp=new Intl.DateTimeFormat(lang,{dateStyle:"medium",timeStyle:"short",timeZone:"UTC"});
 const today=dayOf(now),range=Math.min(Math.max(page.historyDays,1),90);
 const days=Array.from({length:range},(_,i)=>today-range+1+i);

 const state=overallOf(page),overall={tone:state.tone,label:t(state.key)},open=page.incidents.filter(i=>i.state!=="resolved");
 const lastCheck=Math.max(0,...page.components.map(c=>c.lastCheckedAt?Date.parse(c.lastCheckedAt):0));
 const updated=lastCheck?`<p class="updated">${escape(t("pulseUpdatedAt",{date:stamp.format(new Date(lastCheck))+" UTC"}))}</p>`:"";

 const component=(c:PublicComponent)=>{
  const byDay=new Map(c.days.map(d=>[d.date,d]));
  let up=0,total=0;
  const bars=days.map(day=>{
   const d=byDay.get(iso(day)),tone=dayTone(d);
   if(d){up+=d.up;total+=d.up+d.down;}
   // The tip of a day: its date, its state and, when not all up, how much.
   const date=shortDate.format(new Date(day*DAY));
   const share=d&&d.up+d.down?t("pulseUptime",{value:number.format(d.up/(d.up+d.down)*100)}):"";
   const tip=tone==="none"?[date,t("pulseNotMonitored")]:tone==="up"?[date,t("pulseOperational")]:tone==="maint"?[date,t("pulseMaintenance")]:[date,tone==="partial"?t("pulseDegraded"):t("pulseDown"),share];
   return `<i class="${tone}" data-tip="${escape(tip.join("\n")).replace(/\n/g,"&#10;")}"></i>`;
  }).join("");
  const tone=toneOf(c);
  const stateLabel=tone==="up"?t("pulseOperational"):tone==="down"?t("pulseDown"):tone==="partial"?t("pulseDegraded"):tone==="maint"?t("pulseInMaintenance"):t("pulseUnknown");
  const pct=total?t("pulseUptime",{value:number.format(up/total*100)}):t("pulseNoData");
  return `<div class="row"><div class="line"><span class="name"><span class="dot ${tone==="unknown"?"b-unknown":tone}">${svg(tone,12,3)}</span>${escape(c.name)}</span><span class="state t-${tone}">${escape(stateLabel)}</span></div><div class="bars" role="img" aria-label="${escape(pct)}">${bars}</div><div class="scale"><span><span class="from-long">${escape(t("pulseDaysAgo",{days:range}))}</span><span class="from-short">${escape(t("pulseDaysAgo",{days:Math.min(range,30)}))}</span></span><span class="pct">${escape(pct)}</span><span>${escape(t("pulseToday"))}</span></div></div>`;
 };

 const stateName=(s:string)=>s==="investigating"?t("pulseInvestigating"):s==="identified"?t("pulseIdentified"):s==="monitoring"?t("pulseMonitoring"):t("pulseResolved");
 const serviceName=(s:PublicService["status"])=>s==="downtime"?t("pulseDown"):s==="degraded"?t("pulseDegraded"):t("pulseResolved");
 const serviceTone=(s:PublicService["status"])=>s==="downtime"?"down":s==="degraded"?"partial":"up";
 const incident=(i:PublicIncident)=>`<div class="incident"><h4>${escape(i.title)}<span class="tag">${escape(stateName(i.state))}</span></h4>${i.services.length?`<p class="services">${i.services.map(s=>`<span>${escape(s.name)} · <span class="t-${serviceTone(s.status)}">${escape(serviceName(s.status))}</span></span>`).join("")}</p>`:""}${i.updates.map(u=>`<div class="update"><p>${escape(u.body)}</p><small>${escape(stateName(u.state))} · ${escape(stamp.format(new Date(u.createdAt)))} UTC</small></div>`).join("")}</div>`;
 // The last seven days, newest first, each with its incidents; an
 // incident still open from before them shows on its own day after them.
 const older=[...new Set(open.map(i=>dayOf(Date.parse(i.createdAt))).filter(day=>day<=today-7))].sort((a,b)=>b-a);
 const history=[...Array.from({length:7},(_,i)=>today-i),...older].map(day=>{
  const list=page.incidents.filter(i=>dayOf(Date.parse(i.createdAt))===day&&(day>today-7||i.state!=="resolved"));
  return `<div class="day"><h3>${escape(longDate.format(new Date(day*DAY)))}</h3>${list.length?list.map(incident).join(""):`<p>${escape(t("pulseNoIncidentsDay"))}</p>`}</div>`;
 }).join("");
 // Maintenance now and coming, soonest first: its window, what it is about
 // and the services in it.
 const windowOf=(m:PublicMaintenance)=>`${stamp.format(new Date(m.startsAt))} – ${stamp.format(new Date(m.endsAt))} UTC`;
 const maintenance=page.maintenance.length?`<section class="card"><div class="head"><h2>${escape(t("pulseScheduledMaintenance"))}</h2></div>${page.maintenance.map(m=>`<div class="day"><div class="incident"><h4>${escape(m.title)}<span class="tag t-maint">${escape(m.state==="active"?t("pulseMaintenanceActive"):t("pulseMaintenanceScheduled"))}</span></h4><p class="services"><span>${escape(windowOf(m))}</span>${m.services.map(name=>`<span>${escape(name)}</span>`).join("")}</p>${m.body?`<div class="update"><p>${escape(m.body)}</p></div>`:""}</div></div>`).join("")}</section>`:"";
 const year=new Date(now).getUTCFullYear();

 return `<!doctype html><html lang="${lang}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="refresh" content="60"><title>${escape(page.title)}</title><link rel="icon" type="image/svg+xml" href="${faviconOf(overall.tone)}"><meta name="description" content="${escape(overall.label)}"><style>${css}</style></head><body>${headerOf(page)}<div class="wrap">
<section class="hero"><span class="badge ${overall.tone==="unknown"?"b-unknown":overall.tone}">${svg(overall.tone,40,2.5)}</span><h1>${escape(overall.label)}</h1>${updated}</section>
<section class="card"><div class="head"><h2>${escape(t("pulseServices"))}</h2><div class="legend">${([["up","pulseOperational"],["partial","pulseDegraded"],["down","pulseDown"],["maint","pulseMaintenance"],["none","pulseNotMonitored"]] as const).map(([k,key])=>`<span class="l-${k}" data-tip="${escape(t(key))}" role="img" aria-label="${escape(t(key))}">${legendSvg(k)}</span>`).join("")}</div></div>${page.components.map(component).join("")||`<div class="row"><p class="updated">${escape(t("pulseNoData"))}</p></div>`}</section>
${maintenance}
<section class="card"><div class="head"><h2>${escape(t("pulseIncidents"))}</h2></div>${history}</section>
<footer><span>© ${year} ${escape(page.title)}</span><span>${escape(t("pulseMadeWith"))} <a href="https://getblup.io/blup-pulse/" rel="noopener">Blup Pulse</a></span></footer>
</div></body></html>`;
}
