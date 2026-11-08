import {translate,type messages,type MessageKey} from "../../src/lib/i18n/catalog";
import type {Language,TranslationArgs} from "../../shared/language";

// The public status page: one HTML file with inline styles and inline
// Lucide icons (the CSP allows nothing else). It only reads the closed
// public DTO, so private names, URLs and DSNs never reach it.

export type PublicDay={date:string;up:number;down:number;unknown:number;maintenance:number};
export type PublicComponent={name:string;description:string;state:"up"|"down"|"unknown";incident:"downtime"|"degraded"|null;maintenance:boolean;lastCheckedAt:string|null;days:PublicDay[]};
export type PublicMaintenance={title:string;body:string;state:"scheduled"|"active";startsAt:string;endsAt:string;services:string[]};
export type PublicUpdate={body:string;state:string;createdAt:string};
export type PublicService={name:string;status:"downtime"|"degraded"|"resolved"};
export type PublicIncident={id:string;title:string;state:string;impact:string;createdAt:string;updatedAt:string;services:PublicService[];updates:PublicUpdate[]};
export type PublicLook={showTitle:boolean;links:{label:string;url:string}[];logos:{light:string|null;dark:string|null}};
// `home` is where the page lives (its own domain when it has one), set
// when it is served.
export type PublicStatus={home?:string;title:string;language:Language;historyDays:number;components:PublicComponent[];incidents:PublicIncident[];maintenance:PublicMaintenance[]}&PublicLook;

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
@media (prefers-color-scheme:dark){:root:not([data-theme="light"]){--bg:#0a0a0a;--card:#171717;--head:#1c1c1c;--line:#262626;--text:#f5f5f5;--muted:#a3a3a3;--none:#303030;--legend:#525252;--tip:#f5f5f5;--tip-text:#171717}}
:root[data-theme="light"]{color-scheme:light}:root[data-theme="dark"]{color-scheme:dark;--bg:#0a0a0a;--card:#171717;--head:#1c1c1c;--line:#262626;--text:#f5f5f5;--muted:#a3a3a3;--none:#303030;--legend:#525252;--tip:#f5f5f5;--tip-text:#171717}
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
.actions{display:flex;flex-shrink:0;align-items:center;gap:4px;margin-left:auto}.icon-button{display:inline-flex;align-items:center;gap:2px;padding:8px;border:0;border-radius:4px;background:none;color:var(--muted);font:inherit;cursor:pointer}.icon-button:hover{background:color-mix(in srgb,var(--line) 60%,transparent);color:var(--text)}
.lang-code{font-size:12px;font-weight:600;text-transform:uppercase}.lang-picker .icon-button{gap:4px;padding:8px}
.dropdown{margin:0;inset:auto;padding:4px;border:1px solid var(--line);border-radius:6px;background:var(--card);color:var(--text);box-shadow:0 10px 15px -3px rgb(0 0 0/.1),0 4px 6px -4px rgb(0 0 0/.1);min-width:9rem;position-area:bottom span-left;margin-top:6px}
@supports not (position-area:bottom){.dropdown{inset:60px 16px auto auto}}
.dropdown a,.dropdown button{display:flex;width:100%;align-items:center;gap:12px;padding:8px 12px;border:0;border-radius:4px;background:none;color:inherit;font:inherit;font-size:14px;text-align:left;text-decoration:none;cursor:pointer}.dropdown a:hover,.dropdown a[aria-checked="true"],.dropdown button:hover,.dropdown button[aria-checked="true"]{background:color-mix(in srgb,var(--line) 70%,transparent)}
:root:not(.js) .theme-picker{display:none}.when-dark{display:none}
@media (prefers-color-scheme:dark){:root:not([data-theme="light"]) .when-light{display:none}:root:not([data-theme="light"]) .when-dark{display:block}}
:root[data-theme="dark"] .when-light{display:none}:root[data-theme="dark"] .when-dark{display:block}
.menu-toggle{margin-left:0}
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
.bars i{position:relative}
.tip{display:none;position:absolute;bottom:calc(100% + 10px);left:50%;transform:translateX(-50%);z-index:20;width:max-content;min-width:190px;max-width:260px;padding:12px 14px 10px;border:1px solid var(--line);border-radius:12px;background:var(--card);color:var(--text);box-shadow:0 10px 15px -3px rgb(0 0 0/.1),0 4px 6px -4px rgb(0 0 0/.1);font:500 14px/1.4 Inter,ui-sans-serif,system-ui,sans-serif;text-align:left;white-space:normal;pointer-events:none}
.bars i{cursor:pointer;outline:none}.bars i:hover .tip,.bars i:focus .tip{display:block}.bars i:focus-visible{box-shadow:0 0 0 2px var(--bg),0 0 0 4px var(--muted)}.bars i:nth-child(-n+6) .tip{left:0;transform:none}.bars i:nth-last-child(-n+6) .tip{left:auto;right:0;transform:none}
.line{position:relative}.help{position:static;display:inline-grid;place-items:center;flex:none;color:var(--muted);cursor:pointer;outline:none;border-radius:999px}.help:hover,.help:focus{color:var(--text)}.help:focus-visible{box-shadow:0 0 0 2px var(--bg),0 0 0 4px var(--muted)}.help .tip{left:0;transform:none;font-weight:400;color:var(--text)}.help:hover .tip,.help:focus .tip{display:block}
.tip-head{display:flex;align-items:center;gap:8px;font-weight:600;overflow-wrap:anywhere}.tip-head .dot{flex:none}.tip-none{display:inline-grid;color:var(--muted)}.tip-body{display:block;margin-top:6px;color:var(--muted);font-weight:400}.tip-date{display:block;margin-top:10px;padding-top:8px;border-top:1px solid var(--line);color:var(--muted);font-size:13px;font-weight:400;text-align:center}
.day{padding:18px 20px 4px}.day h3{margin:0 0 10px;padding-bottom:8px;border-bottom:1px solid var(--line);font-size:15px;font-weight:600}.day>p{color:var(--muted);margin:0 0 14px}
.incident{margin:0 0 16px}
.inc{display:block;margin:0 -12px 8px;padding:12px;border-radius:10px;color:inherit;text-decoration:none;transition:background-color .2s}.inc:hover{background:var(--head)}
.inc-head{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:12px}.inc-head h4{margin:0;font-size:15px;font-weight:600;overflow-wrap:anywhere}
.pill{flex:none;font-size:13px;font-weight:500;padding:2px 10px;border-radius:999px;background:var(--head);color:var(--muted);border:1px solid var(--line)}.pill.p-up{background:color-mix(in srgb,var(--up) 15%,transparent);color:var(--up);border-color:transparent}.pill.p-down{background:color-mix(in srgb,var(--down) 15%,transparent);color:var(--down);border-color:transparent}.pill.p-partial{background:color-mix(in srgb,var(--partial) 18%,transparent);color:var(--partial);border-color:transparent}
.tl{display:grid;grid-template-columns:20px minmax(0,1fr);gap:12px}.tl-mark{display:grid;place-items:start center;padding-top:14px;color:var(--muted)}.tl-mark.t-up{color:var(--up)}
.stack{position:relative;z-index:0}.stack.more{margin-bottom:14px}.stack.more::before,.stack.more::after{content:"";position:absolute;z-index:-1;inset:auto 8px -7px;height:28px;border:1px solid color-mix(in srgb,var(--line),var(--muted) 30%);border-top:0;border-radius:0 0 10px 10px;background:var(--head);box-shadow:0 1px 2px 0 rgb(0 0 0/.08)}.stack.more::after{inset:auto 16px -14px;z-index:-2;opacity:.7}
.upd{position:relative;padding:12px 16px;border:1px solid var(--line);border-radius:10px;background:var(--card);box-shadow:0 1px 2px 0 rgb(0 0 0/.05)}.upd p{margin:0;white-space:pre-wrap;overflow-wrap:anywhere;color:var(--muted)}.upd .meta{margin-bottom:6px;color:var(--text)}.upd .meta strong{font-weight:600;margin-right:6px}.upd .meta span{color:var(--muted)}
.prev{display:grid;grid-template-columns:20px minmax(0,1fr);gap:12px;align-items:center;margin:6px 0 0;color:var(--muted);font-size:14px}.prev span{display:grid;place-items:center}
.inc-line{position:relative}.inc-line .tl+.tl{margin-top:16px}.inc-line .tl:not(:last-child) .tl-mark{background:linear-gradient(var(--line),var(--line)) center 36px/1px calc(100% - 22px) no-repeat}
.back{display:inline-flex;align-items:center;gap:8px;margin-bottom:20px;color:var(--muted);font-weight:500;text-decoration:none}.back:hover{color:var(--text)}
.inc-top{padding:24px 24px 20px}.inc-top h1{margin:12px 0 6px;font-size:26px;text-align:left}.inc-services{display:flex;flex-wrap:wrap;align-items:center;gap:8px;padding:16px 24px;border-top:1px solid var(--line)}.inc-services>span{color:var(--muted);margin-right:4px}.chip{display:inline-flex;align-items:center;gap:6px;padding:2px 10px;border-radius:999px;background:var(--head);border:1px solid var(--line);font-size:14px}
.history-top{text-align:center;margin-bottom:32px}.history-top h1{margin:0 0 14px}.window{display:inline-flex;align-items:center;gap:14px;color:var(--muted)}.arrow{display:inline-grid;place-items:center;width:32px;height:32px;border-radius:8px;background:var(--head);border:1px solid var(--line);color:var(--text)}.arrow:hover{background:var(--line)}.arrow.off{opacity:.4}
.month-day{display:flex;justify-content:space-between;gap:12px;padding:12px 20px;background:var(--head);border-top:1px solid var(--line);color:var(--muted);font-size:14px}.head+.month-day{border-top:0}.month-list{padding:12px 20px}.month-list .inc{margin:0 -12px}
.more{border-top:1px solid var(--line)}.more>summary{display:block;padding:12px 20px;color:var(--muted);font-size:14px;font-weight:500;text-align:center;cursor:pointer;list-style:none}.more>summary::-webkit-details-marker{display:none}.more>summary:hover{color:var(--text);background:var(--head)}.more[open]>summary{display:none}
.empty-month{display:flex;align-items:center;justify-content:center;gap:8px;margin:0;padding:40px 20px;color:var(--muted)}
.head-link{color:var(--muted);font-size:14px;font-weight:500;text-decoration:none}.head-link:hover{color:var(--text)}
.inc-body{padding:20px 24px 24px;border-top:1px solid var(--line);background:var(--head);border-radius:0 0 12px 12px}.services{display:flex;flex-wrap:wrap;gap:4px 16px;margin:0 0 8px;font-size:13px;color:var(--muted)}.incident h4{margin:0 0 6px;font-size:15px}.tag{display:inline-block;font-size:12px;font-weight:600;padding:1px 8px;border-radius:999px;margin-left:6px;vertical-align:1px;background:var(--head);border:1px solid var(--line)}
.update{margin:6px 0 0}.update p{margin:0;white-space:pre-wrap;overflow-wrap:anywhere}.update small{color:var(--muted)}

footer{display:flex;justify-content:space-between;gap:16px;flex-wrap:wrap;color:var(--muted);font-size:13px;padding:8px 2px}footer a{color:inherit;font-weight:600;text-decoration:none}footer a:hover{text-decoration:underline}
@media (max-width:640px){.wrap{padding-top:36px}.bar{padding:0 16px}h1{font-size:24px}.bars i:nth-last-child(n+31){display:none}.bars i:nth-last-child(-n+30):nth-last-child(n+25) .tip{left:0;transform:none}.from-long{display:none}.from-short{display:inline}}
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
// top with the logos (the dark one on a dark theme, when there is one, else
// the Blup) and the title unless hidden on the left, the page's links in
// the middle, and on the right the language (links: the server writes the
// page in it), the theme (light, dark or the system's, kept by the
// visitor's browser) and, on a small screen, a menu with the links
// (popovers).
const lucide=(paths:string,extra="",size=20)=>`<svg${extra} width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths}</svg>`;
const marks={
 menu:'<path d="M4 5h16"/><path d="M4 12h16"/><path d="M4 19h16"/>',x:'<path d="M18 6 6 18"/><path d="m6 6 12 12"/>',
 globe:'<circle cx="12" cy="12" r="10"/><path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"/><path d="M2 12h20"/>',
 sun:'<circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/>',
 moon:'<path d="M20.985 12.486a9 9 0 1 1-9.473-9.472c.405-.022.617.46.402.803a6 6 0 0 0 8.268 8.268c.344-.215.825-.004.803.401"/>',
 monitor:'<rect width="20" height="14" x="2" y="3" rx="2"/><line x1="8" x2="16" y1="21" y2="21"/><line x1="12" x2="12" y1="17" y2="21"/>',
 chevron:'<path d="m6 9 6 6 6-6"/>',
 done:'<circle cx="12" cy="12" r="10"/><path d="m9 12 2 2 4-4"/>',
 dashed:'<path d="M10.1 2.182a10 10 0 0 1 3.8 0"/><path d="M13.9 21.818a10 10 0 0 1-3.8 0"/><path d="M17.609 3.721a10 10 0 0 1 2.69 2.7"/><path d="M2.182 13.9a10 10 0 0 1 0-3.8"/><path d="M20.279 17.609a10 10 0 0 1-2.7 2.69"/><path d="M21.818 10.1a10 10 0 0 1 0 3.8"/><path d="M3.721 6.391a10 10 0 0 1 2.7-2.69"/><path d="M6.391 20.279a10 10 0 0 1-2.69-2.7"/>',
 back:'<path d="m12 19-7-7 7-7"/><path d="M19 12H5"/>',
 info:'<circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>',
 left:'<path d="m15 18-6-6 6-6"/>',right:'<path d="m9 18 6-6-6-6"/>',
} as const;
const languageNames:Record<Language,string>={pt:"Português",en:"English",es:"Español"};
// The page's one script, allowed by its hash: the stored theme before the
// page paints, and the theme menu.
export const THEME_SCRIPT=`(function(){var r=document.documentElement,k="theme";r.classList.add("js");try{var t=localStorage.getItem(k);if(t==="light"||t==="dark")r.dataset.theme=t}catch(e){}function sync(){var c=r.dataset.theme||"system";document.querySelectorAll("[data-set-theme]").forEach(function(b){b.setAttribute("aria-checked",String(b.getAttribute("data-set-theme")===c))})}document.addEventListener("click",function(e){var b=e.target.closest&&e.target.closest("[data-set-theme]");if(!b)return;var v=b.getAttribute("data-set-theme");try{localStorage.setItem(k,v)}catch(e){}if(v==="system")delete r.dataset.theme;else r.dataset.theme=v;var m=document.getElementById("theme-menu");if(m&&m.hidePopover)m.hidePopover();sync()});document.addEventListener("DOMContentLoaded",sync)})();`;
function headerOf(page:PublicStatus,lang:Language,t:(key:"language"|"theme"|"light"|"dark"|"system")=>string,root=""){
 const {light,dark}=page.logos,alt=page.showTitle?"":escape(page.title);
 const img=(file:string,version:string,extra="")=>`<img class="logo${extra}" src="${root}${file}?v=${version}" alt="${alt}">`;
 const logo=light&&dark?img("logo-light",light," when-light")+img("logo-dark",dark," when-dark"):light?img("logo-light",light):dark?img("logo-dark",dark):blup;
 const links=page.links.map(l=>`<a href="${escape(l.url)}" rel="noopener nofollow">${escape(l.label)}</a>`).join("");
 const nav=page.links.length?`<nav class="nav" aria-label="Menu">${links}</nav>`:"";
 const languages=`<div class="lang-picker"><button class="icon-button" type="button" popovertarget="lang-menu" aria-label="${escape(t("language"))}: ${lang}" style="anchor-name:--lang">${lucide(marks.globe)}<span class="lang-code">${lang}</span></button><div id="lang-menu" class="dropdown" popover style="position-anchor:--lang" role="menu">${(Object.keys(languageNames) as Language[]).map(l=>`<a role="menuitemradio" href="?lang=${l}" hreflang="${l}" lang="${l}" aria-checked="${l===lang}">${languageNames[l]}</a>`).join("")}</div></div>`;
 const themes=`<div class="theme-picker"><button class="icon-button" type="button" popovertarget="theme-menu" aria-label="${escape(t("theme"))}" style="anchor-name:--theme">${lucide(marks.sun,' class="when-light"')}${lucide(marks.moon,' class="when-dark"')}${lucide(marks.chevron,"",16)}</button><div id="theme-menu" class="dropdown" popover style="position-anchor:--theme" role="menu">${([["light",marks.sun],["dark",marks.moon],["system",marks.monitor]] as const).map(([v,mark])=>`<button type="button" role="menuitemradio" data-set-theme="${v}">${lucide(mark)}${escape(t(v))}</button>`).join("")}</div></div>`;
 const menu=page.links.length?`<button class="icon-button menu-toggle" type="button" popovertarget="mobile-menu" aria-label="Menu">${lucide(marks.menu,' class="open"',22)}${lucide(marks.x,' class="close"',22)}</button><nav id="mobile-menu" class="mobile-menu" popover aria-label="Menu">${links}</nav>`:"";
 return `<header class="site-header"><div class="bar"><a class="brand" href="${page.home?escape(page.home):root||"./"}"${page.showTitle?"":` aria-label="${escape(page.title)}"`}>${logo}${page.showTitle?`<span>${escape(page.title)}</span>`:""}</a>${nav}<div class="actions">${languages}${themes}${menu}</div></div></header>`;
}
// The badge for another site (in an iframe): the state's mark and words,
// linking to the page. Light, dark or following the visitor's screen, in
// `language` (the visitor's, chosen when it is served).
export function renderBadge(page:PublicStatus,theme:"light"|"dark"|"auto",language:Language=page.language){
 const state=overallOf(page),label=translate(language,state.key);
 const ink={light:"#171717",dark:"#f5f5f5"};
 const color=theme==="auto"?`color:${ink.light}}@media (prefers-color-scheme:dark){a{color:${ink.dark}}`:`color:${ink[theme]}}`;
 return `<!doctype html><html lang="${language}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="refresh" content="60"><title>${escape(label)}</title><style>html,body{margin:0;background:transparent;overflow:hidden}a{display:inline-flex;align-items:center;gap:8px;height:30px;font:500 14px/1 Inter,ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;text-decoration:none;white-space:nowrap;-webkit-font-smoothing:antialiased;${color}a:hover span:last-child{text-decoration:underline}.dot{display:inline-grid;place-items:center;width:18px;height:18px;border-radius:999px;background:${toneColor[state.tone]};color:#fff}</style></head><body><a href="./" target="_blank" rel="noopener"><span class="dot">${svg(state.tone,12,3)}</span><span>${escape(label)}</span></a></body></html>`;
}
// The pieces an incident is drawn with, in the list and on its own page:
// its pill (resolved in green; open, by its impact) and an update's card.
function incidentParts(lang:Language,stamp:Intl.DateTimeFormat){
 const t=<K extends MessageKey>(key:K,...args:TranslationArgs<(typeof messages)[K][0]>)=>translate(lang,key,...args);
 const stateName=(s:string)=>s==="investigating"?t("pulseInvestigating"):s==="identified"?t("pulseIdentified"):s==="monitoring"?t("pulseMonitoring"):t("pulseResolved");
 const pillOf=(i:PublicIncident):{tone:Tone|"none";label:string}=>i.state==="resolved"?{tone:"up",label:t("pulseResolved")}:i.impact==="major_outage"||i.impact==="partial_outage"?{tone:"down",label:t("pulseDown")}:i.impact==="degraded"?{tone:"partial",label:t("pulseDegraded")}:{tone:"none",label:stateName(i.state)};
 const pill=(i:PublicIncident)=>{const p=pillOf(i);return `<span class="pill p-${p.tone}">${escape(p.label)}</span>`;};
 return {
  stateName,pill,
  updateCard:(u:PublicUpdate)=>`<div class="upd"><p class="meta"><strong>${escape(stateName(u.state))}</strong><span>${escape(stamp.format(new Date(u.createdAt)))} UTC</span></p><p>${escape(u.body)}</p></div>`,
  serviceTone:(s:PublicService["status"])=>s==="downtime"?"down":s==="degraded"?"partial":"up",
  // An incident in a list: its title and pill, then its latest update as
  // a card on a timeline mark; earlier updates show as cards stacked under
  // it and a count. The whole card opens the incident's own page.
  incidentCard:(i:PublicIncident)=>{const [latest,...earlier]=i.updates;return `<a class="inc" href="incidents/${i.id}?lang=${lang}"><div class="inc-head"><h4>${escape(i.title)}</h4>${pill(i)}</div>${latest?`<div class="tl"><span class="tl-mark${latest.state==="resolved"?" t-up":""}">${lucide(latest.state==="resolved"?marks.done:marks.dashed)}</span><div class="stack${earlier.length?" more":""}"><div class="upd"><p class="meta"><strong>${escape(stateName(latest.state))}</strong><span>${escape(stamp.format(new Date(latest.createdAt)))} UTC</span></p><p>${escape(latest.body)}</p></div></div></div>`:""}${earlier.length?`<p class="prev"><span>${lucide(marks.dashed)}</span>${escape(earlier.length===1?t("pulsePreviousUpdate"):t("pulsePreviousUpdates",{count:earlier.length}))}</p>`:""}</a>`;},
 };
}
// What every public page shares: the head (title, the theme script, the
// favicon of the state), the header and the footer. `root` leads back to
// the page's own address (logos and the home link are relative to it).
function shell(page:PublicStatus,lang:Language,title:string,body:string,now:number,root=""){
 const t=<K extends MessageKey>(key:K,...args:TranslationArgs<(typeof messages)[K][0]>)=>translate(lang,key,...args);
 const state=overallOf(page);
 return `<!doctype html><html lang="${lang}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="refresh" content="60"><title>${escape(title)}</title><script>${THEME_SCRIPT}</script><link rel="icon" type="image/svg+xml" href="${faviconOf(state.tone)}"><meta name="description" content="${escape(t(state.key))}"><meta property="og:type" content="website"><meta property="og:title" content="${escape(title)}"><meta property="og:description" content="${escape(t(state.key))}"><meta property="og:site_name" content="${escape(page.title)}"><style>${css}</style></head><body>${headerOf(page,lang,key=>t(key),root)}<div class="wrap">
${body}
<footer><span>© ${new Date(now).getUTCFullYear()} ${escape(page.title)}</span><span>${escape(t("pulseMadeWith"))} <a href="https://getblup.io/blup-pulse/" rel="noopener">Blup Pulse</a></span></footer>
</div></body></html>`;
}
// One incident's own page: its pill, title and start, the services it
// touched and every update, newest first, on a timeline.
export function renderIncidentPage(page:PublicStatus,i:PublicIncident,now=Date.now(),language:Language=page.language):string{
 const lang=language,t=<K extends MessageKey>(key:K,...args:TranslationArgs<(typeof messages)[K][0]>)=>translate(lang,key,...args);
 const stamp=new Intl.DateTimeFormat(lang,{dateStyle:"medium",timeStyle:"short",timeZone:"UTC"});
 const {pill,updateCard,serviceTone}=incidentParts(lang,stamp);
 const services=i.services.length?`<div class="inc-services"><span>${escape(t("pulseAffectedServices"))}</span>${i.services.map(s=>`<span class="chip"><span class="dot ${serviceTone(s.status)}">${svg(serviceTone(s.status),10,3)}</span>${escape(s.name)}</span>`).join("")}</div>`:"";
 const updates=i.updates.map(u=>`<div class="tl"><span class="tl-mark${u.state==="resolved"?" t-up":""}">${lucide(u.state==="resolved"?marks.done:marks.dashed)}</span>${updateCard(u)}</div>`).join("");
 const body=`<a class="back" href="../incidents?lang=${lang}">${lucide(marks.back,"",18)}${escape(t("pulsePreviousIncidents"))}</a>
<section class="card"><div class="inc-top">${pill(i)}<h1>${escape(i.title)}</h1><p class="updated">${escape(stamp.format(new Date(i.createdAt)))} UTC</p></div>${services}${updates?`<div class="inc-body"><div class="inc-line">${updates}</div></div>`:""}</section>`;
 return shell(page,lang,`${i.title} · ${page.title}`,body,now,"../");
}
// The page's previous incidents, three months at a time (newest first): a
// card per month, its incidents by day with how many, or that none was
// reported; arrows go to the months before and after.
const MONTH_FIRST=5;
export type IncidentWindow={months:{key:string;start:number;end:number}[];older:string|null;newer:string|null};
export function renderIncidentHistory(page:PublicStatus,window:IncidentWindow,incidents:PublicIncident[],now=Date.now(),language:Language=page.language):string{
 const lang=language,t=<K extends MessageKey>(key:K,...args:TranslationArgs<(typeof messages)[K][0]>)=>translate(lang,key,...args);
 const stamp=new Intl.DateTimeFormat(lang,{dateStyle:"medium",timeStyle:"short",timeZone:"UTC"});
 const monthName=new Intl.DateTimeFormat(lang,{month:"long",year:"numeric",timeZone:"UTC"}),shortMonth=new Intl.DateTimeFormat(lang,{month:"short",year:"numeric",timeZone:"UTC"}),dayName=new Intl.DateTimeFormat(lang,{dateStyle:"medium",timeZone:"UTC"});
 const {incidentCard}=incidentParts(lang,stamp);
 const months=window.months.map(m=>{
  const list=incidents.filter(i=>{const at=Date.parse(i.createdAt);return at>=m.start&&at<m.end;});
  const days=[...new Set(list.map(i=>dayOf(Date.parse(i.createdAt))))].sort((a,b)=>b-a);
  const dayBlock=(day:number)=>{const items=list.filter(i=>dayOf(Date.parse(i.createdAt))===day);return `<div class="month-day"><span>${escape(dayName.format(new Date(day*DAY)))}</span><span>${escape(items.length===1?t("pulseIncidentOne"):t("pulseIncidentMany",{count:items.length}))}</span></div><div class="month-list">${items.map(incidentCard).join("")}</div>`;};
  // The first days until five incidents show; the rest of the month opens
  // with "Show more" (a details element: no script).
  let shown=0;const first:number[]=[],rest:number[]=[];
  for(const day of days){if(shown<MONTH_FIRST){first.push(day);shown+=list.filter(i=>dayOf(Date.parse(i.createdAt))===day).length;}else rest.push(day);}
  const more=list.length-shown;
  const body=days.length?first.map(dayBlock).join("")+(rest.length?`<details class="more"><summary>${escape(more===1?t("pulseShowMoreOne"):t("pulseShowMore",{count:more}))}</summary>${rest.map(dayBlock).join("")}</details>`:""):`<p class="empty-month">${lucide(marks.done,"",18)}${escape(t("pulseNoIncidentsDay"))}</p>`;
  return `<section class="card"><div class="head"><h2>${escape(monthName.format(new Date(m.start)))}</h2></div>${body}</section>`;
 }).join("");
 const arrow=(to:string|null,mark:string,label:string)=>to?`<a class="arrow" href="incidents?to=${to}&amp;lang=${lang}" aria-label="${escape(label)}">${lucide(mark,"",18)}</a>`:`<span class="arrow off" aria-hidden="true">${lucide(mark,"",18)}</span>`;
 const range=t("pulseRange",{from:shortMonth.format(new Date(window.months[2].start)),to:shortMonth.format(new Date(window.months[0].start))});
 const body=`<a class="back" href="./?lang=${lang}">${lucide(marks.back,"",18)}${escape(t("pulseBackToOverview"))}</a>
<section class="history-top"><h1>${escape(t("pulsePreviousIncidents"))}</h1><div class="window">${arrow(window.older,marks.left,t("pulseOlderMonths"))}<span>${escape(range)}</span>${arrow(window.newer,marks.right,t("pulseNewerMonths"))}</div></section>
${months}`;
 return shell(page,lang,`${t("pulsePreviousIncidents")} · ${page.title}`,body,now);
}
// `language` is the one the visitor chose (?lang=), else the page's own.
export function renderStatusPage(page:PublicStatus,now=Date.now(),language:Language=page.language):string{
 const lang=language;
 const t=<K extends MessageKey>(key:K,...args:TranslationArgs<(typeof messages)[K][0]>)=>translate(lang,key,...args);
 const number=new Intl.NumberFormat(lang,{maximumFractionDigits:2});
 const longDate=new Intl.DateTimeFormat(lang,{dateStyle:"long",timeZone:"UTC"});
 const stamp=new Intl.DateTimeFormat(lang,{dateStyle:"medium",timeStyle:"short",timeZone:"UTC"});
 const today=dayOf(now),range=Math.min(Math.max(page.historyDays,1),90);
 const days=Array.from({length:range},(_,i)=>today-range+1+i);

 const state=overallOf(page),overall={tone:state.tone,label:t(state.key)},open=page.incidents.filter(i=>i.state!=="resolved");
 const lastCheck=Math.max(0,...page.components.map(c=>c.lastCheckedAt?Date.parse(c.lastCheckedAt):0));
 const updated=lastCheck?`<p class="updated">${escape(t("pulseUpdatedAt",{date:stamp.format(new Date(lastCheck))+" UTC"}))}</p>`:"";

 const dayDate=new Intl.DateTimeFormat(lang,{dateStyle:"medium",timeZone:"UTC"});
 // An incident that touched a service (by its public name) on a day:
 // started by then and not resolved before it.
 const incidentOn=(service:string,day:number)=>page.incidents.find(i=>i.services.some(s=>s.name===service)&&dayOf(Date.parse(i.createdAt))<=day&&(i.state!=="resolved"||dayOf(Date.parse(i.updatedAt))>=day));
 const component=(c:PublicComponent)=>{
  const byDay=new Map(c.days.map(d=>[d.date,d]));
  let up=0,total=0;
  const bars=days.map(day=>{
   const d=byDay.get(iso(day)),tone=dayTone(d);
   if(d){up+=d.up;total+=d.up+d.down;}
   // The tip of a day, a small card: the state's mark with its name (or
   // the title of an incident on this service that day), how it went
   // when not all up, and the date.
   const date=dayDate.format(new Date(day*DAY));
   const share=d&&d.up+d.down?t("pulseUptime",{value:number.format(d.up/(d.up+d.down)*100)}):"";
   const incident=tone==="partial"||tone==="down"?incidentOn(c.name,day):undefined;
   const name=tone==="none"?t("pulseNotMonitored"):tone==="up"?t("pulseOperational"):tone==="maint"?t("pulseMaintenance"):incident?.title??(tone==="partial"?t("pulseDegraded"):t("pulseDown"));
   const detail=tone==="partial"||tone==="down"?[incident?(tone==="partial"?t("pulseDegraded"):t("pulseDown")):"",share].filter(Boolean).join(" · "):"";
   const mark=tone==="none"?`<span class="tip-none">${legendSvg("none")}</span>`:`<span class="dot ${tone}">${svg(tone,11,3)}</span>`;
   // Focusable, so a tap opens the tip on a phone (and the keyboard too).
   return `<i class="${tone}" tabindex="0"><span class="tip" role="tooltip"><span class="tip-head">${mark}${escape(name)}</span>${detail?`<span class="tip-body">${escape(detail)}</span>`:""}<span class="tip-date">${escape(date)}</span></span></i>`;
  }).join("");
  const tone=toneOf(c);
  const stateLabel=tone==="up"?t("pulseOperational"):tone==="down"?t("pulseDown"):tone==="partial"?t("pulseDegraded"):tone==="maint"?t("pulseInMaintenance"):t("pulseUnknown");
  const pct=total?t("pulseUptime",{value:number.format(up/total*100)}):t("pulseNoData");
  return `<div class="row"><div class="line"><span class="name"><span class="dot ${tone==="unknown"?"b-unknown":tone}">${svg(tone,12,3)}</span>${escape(c.name)}${c.description?`<span class="help" tabindex="0" role="button" aria-label="${escape(c.description)}">${lucide(marks.info,"",16)}<span class="tip" role="tooltip">${escape(c.description)}</span></span>`:""}</span><span class="state t-${tone}">${escape(stateLabel)}</span></div><div class="bars" role="img" aria-label="${escape(pct)}">${bars}</div><div class="scale"><span><span class="from-long">${escape(t("pulseDaysAgo",{days:range}))}</span><span class="from-short">${escape(t("pulseDaysAgo",{days:Math.min(range,30)}))}</span></span><span class="pct">${escape(pct)}</span><span>${escape(t("pulseToday"))}</span></div></div>`;
 };

 const {incidentCard}=incidentParts(lang,stamp);
 const incident=incidentCard;
 // The last seven days, newest first, each with its incidents (the card
 // shows only once the page has had an incident within what is kept); an
 // incident still open from before them shows on its own day after them.
 const older=[...new Set(open.map(i=>dayOf(Date.parse(i.createdAt))).filter(day=>day<=today-7))].sort((a,b)=>b-a);
 const history=[...Array.from({length:7},(_,i)=>today-i),...older].map(day=>{
  const list=page.incidents.filter(i=>dayOf(Date.parse(i.createdAt))===day&&(day>today-7||i.state!=="resolved"));
  return `<div class="day"><h3>${escape(longDate.format(new Date(day*DAY)))}</h3>${list.length?list.map(incident).join(""):`<p>${escape(t("pulseNoIncidentsDay"))}</p>`}</div>`;
 }).join("");
 // Maintenance now and coming, soonest first: its window, what it is about
 // and the services in it.
 const windowOf=(m:PublicMaintenance)=>`${t("pulseRange",{from:stamp.format(new Date(m.startsAt)),to:stamp.format(new Date(m.endsAt))})} UTC`;
 const maintenance=page.maintenance.length?`<section class="card"><div class="head"><h2>${escape(t("pulseScheduledMaintenance"))}</h2></div>${page.maintenance.map(m=>`<div class="day"><div class="incident"><h4>${escape(m.title)}<span class="tag t-maint">${escape(m.state==="active"?t("pulseMaintenanceActive"):t("pulseMaintenanceScheduled"))}</span></h4><p class="services"><span>${escape(windowOf(m))}</span>${m.services.map(name=>`<span>${escape(name)}</span>`).join("")}</p>${m.body?`<div class="update"><p>${escape(m.body)}</p></div>`:""}</div></div>`).join("")}</section>`:"";

 return shell(page,lang,page.title,`<section class="hero"><span class="badge ${overall.tone==="unknown"?"b-unknown":overall.tone}">${svg(overall.tone,40,2.5)}</span><h1>${escape(overall.label)}</h1>${updated}</section>
<section class="card"><div class="head"><h2>${escape(t("pulseServices"))}</h2><div class="legend">${([["up","pulseOperational"],["partial","pulseDegraded"],["down","pulseDown"],["maint","pulseMaintenance"],["none","pulseNotMonitored"]] as const).map(([k,key])=>`<span class="l-${k}" data-tip="${escape(t(key))}" role="img" aria-label="${escape(t(key))}">${legendSvg(k)}</span>`).join("")}</div></div>${page.components.map(component).join("")||`<div class="row"><p class="updated">${escape(t("pulseNoData"))}</p></div>`}</section>
${maintenance}
${page.incidents.length?`<section class="card"><div class="head"><h2>${escape(t("pulseIncidents"))}</h2><a class="head-link" href="incidents?lang=${lang}">${escape(t("pulsePreviousIncidents"))}</a></div>${history}</section>`:""}`,now);
}
