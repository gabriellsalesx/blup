import {translate,type messages,type MessageKey} from "../../src/lib/i18n/catalog";
import type {Language,TranslationArgs} from "../../shared/language";

// The public status page: one HTML file with inline styles and inline
// Lucide icons (the CSP allows nothing else). It only reads the closed
// public DTO, so private names, URLs and DSNs never reach it.

export type PublicDay={date:string;up:number;down:number;unknown:number};
export type PublicComponent={name:string;state:"up"|"down"|"unknown";lastCheckedAt:string|null;days:PublicDay[]};
export type PublicUpdate={body:string;state:string;createdAt:string};
export type PublicIncident={title:string;state:string;impact:string;createdAt:string;updatedAt:string;updates:PublicUpdate[]};
export type PublicStatus={title:string;language:Language;historyDays:number;components:PublicComponent[];incidents:PublicIncident[]};

const escape=(v:string)=>v.replace(/[&<>"']/g,c=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]??""));
const DAY=86_400_000;

// Lucide icons (check, triangle-alert, x, circle-help), copied as paths.
const icon={
 up:'<path d="M20 6 9 17l-5-5"/>',
 partial:'<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"/><path d="M12 9v4"/><path d="M12 17h.01"/>',
 down:'<path d="M18 6 6 18"/><path d="m6 6 12 12"/>',
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
.wrap{max-width:780px;margin:0 auto;padding:40px 20px 32px}
header{display:flex;align-items:center;gap:12px;margin-bottom:56px}header strong{font-size:18px;font-weight:700;overflow-wrap:anywhere}header span{color:var(--muted);padding-left:12px;border-left:1px solid var(--line)}
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
.up,.bars i.up{background:var(--up)}.partial,.bars i.partial{background:var(--partial)}.down,.bars i.down{background:var(--down)}.bars i.none{background:var(--none)}

.t-up{color:var(--up)}.t-partial{color:var(--partial)}.t-down{color:var(--down)}.t-unknown{color:var(--muted)}
.b-unknown{background:var(--muted)}
[data-tip]{position:relative}[data-tip]:hover::after{content:attr(data-tip);position:absolute;bottom:calc(100% + 8px);left:50%;transform:translateX(-50%);z-index:10;background:var(--tip);color:var(--tip-text);padding:4px 8px;border-radius:6px;font:500 13px/1.4 Inter,ui-sans-serif,system-ui,sans-serif;white-space:pre;text-align:center;pointer-events:none}
[data-tip]:hover::before{content:"";position:absolute;bottom:calc(100% + 3px);left:50%;transform:translateX(-50%);z-index:10;border:5px solid transparent;border-top-color:var(--tip);border-bottom:0;pointer-events:none}
.bars i:nth-child(-n+6):hover::after{left:0;transform:none}.bars i:nth-last-child(-n+6):hover::after{left:auto;right:0;transform:none}
.day{padding:18px 20px 4px}.day h3{margin:0 0 10px;padding-bottom:8px;border-bottom:1px solid var(--line);font-size:15px;font-weight:600}.day>p{color:var(--muted);margin:0 0 14px}
.incident{margin:0 0 16px}.incident h4{margin:0 0 6px;font-size:15px}.tag{display:inline-block;font-size:12px;font-weight:600;padding:1px 8px;border-radius:999px;margin-left:6px;vertical-align:1px;background:var(--head);border:1px solid var(--line)}
.update{margin:6px 0 0}.update p{margin:0;white-space:pre-wrap;overflow-wrap:anywhere}.update small{color:var(--muted)}
.active{border-color:var(--partial)}.active .head{background:transparent;color:var(--partial)}
footer{display:flex;justify-content:space-between;gap:16px;flex-wrap:wrap;color:var(--muted);font-size:13px;padding:8px 2px}footer a{color:inherit;font-weight:600;text-decoration:none}footer a:hover{text-decoration:underline}
@media (max-width:640px){.wrap{padding-top:28px}header{margin-bottom:36px}h1{font-size:24px}.bars i:nth-last-child(n+31){display:none}.bars i:nth-last-child(30):hover::after{left:0;transform:none}.from-long{display:none}.from-short{display:inline}}
`;

// Day key (UTC) of a date, as days since the epoch.
const dayOf=(ms:number)=>Math.floor(ms/DAY);
const iso=(day:number)=>new Date(day*DAY).toISOString().slice(0,10);

function dayTone(d:PublicDay|undefined):Tone|"none"{
 const total=d?d.up+d.down:0;
 if(!d||total===0)return "none";
 const ratio=d.up/total;
 return ratio===1?"up":ratio>=0.9?"partial":"down";
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

 // The overall state, from the components (and an open incident).
 const states=page.components.map(c=>c.state),open=page.incidents.filter(i=>i.state!=="resolved");
 const down=states.filter(s=>s==="down").length,known=states.filter(s=>s!=="unknown").length;
 const overall:{tone:Tone;label:string}=
  !page.components.length||known===0?{tone:"unknown",label:t("pulseWaiting")}:
  down&&down===known?{tone:"down",label:t("pulseAllDown")}:
  down||open.length?{tone:"partial",label:t("pulseSomeDown")}:
  {tone:"up",label:t("pulseAllUp")};
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
   const tip=tone==="none"?[date,t("pulseNotMonitored")]:tone==="up"?[date,t("pulseOperational")]:[date,tone==="partial"?t("pulseDegraded"):t("pulseDown"),share];
   return `<i class="${tone}" data-tip="${escape(tip.join("\n")).replace(/\n/g,"&#10;")}"></i>`;
  }).join("");
  const tone:Tone=c.state==="up"?"up":c.state==="down"?"down":"unknown";
  const stateLabel=c.state==="up"?t("pulseOperational"):c.state==="down"?t("pulseDown"):t("pulseUnknown");
  const pct=total?t("pulseUptime",{value:number.format(up/total*100)}):t("pulseNoData");
  return `<div class="row"><div class="line"><span class="name"><span class="dot ${tone==="unknown"?"b-unknown":tone}">${svg(tone,12,3)}</span>${escape(c.name)}</span><span class="state t-${tone}">${escape(stateLabel)}</span></div><div class="bars" role="img" aria-label="${escape(pct)}">${bars}</div><div class="scale"><span><span class="from-long">${escape(t("pulseDaysAgo",{days:range}))}</span><span class="from-short">${escape(t("pulseDaysAgo",{days:Math.min(range,30)}))}</span></span><span class="pct">${escape(pct)}</span><span>${escape(t("pulseToday"))}</span></div></div>`;
 };

 const stateName=(s:string)=>s==="investigating"?t("pulseInvestigating"):s==="identified"?t("pulseIdentified"):s==="monitoring"?t("pulseMonitoring"):t("pulseResolved");
 const incident=(i:PublicIncident)=>`<div class="incident"><h4>${escape(i.title)}<span class="tag">${escape(stateName(i.state))}</span></h4>${i.updates.map(u=>`<div class="update"><p>${escape(u.body)}</p><small>${escape(stateName(u.state))} · ${escape(stamp.format(new Date(u.createdAt)))} UTC</small></div>`).join("")}</div>`;
 const openCard=open.length?`<section class="card active"><div class="head"><h2>${escape(t("pulseIncidents"))}</h2></div><div class="day">${open.map(incident).join("")}</div></section>`:"";
 // The last seven days, newest first, each with its incidents.
 const history=Array.from({length:7},(_,i)=>today-i).map(day=>{
  const list=page.incidents.filter(i=>dayOf(Date.parse(i.createdAt))===day);
  return `<div class="day"><h3>${escape(longDate.format(new Date(day*DAY)))}</h3>${list.length?list.map(incident).join(""):`<p>${escape(t("pulseNoIncidentsDay"))}</p>`}</div>`;
 }).join("");
 const year=new Date(now).getUTCFullYear();

 return `<!doctype html><html lang="${lang}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="refresh" content="60"><title>${escape(page.title)}</title><meta name="description" content="${escape(overall.label)}"><style>${css}</style></head><body><div class="wrap">
<header>${blup}<strong>${escape(page.title)}</strong><span>Status</span></header>
<section class="hero"><span class="badge ${overall.tone==="unknown"?"b-unknown":overall.tone}">${svg(overall.tone,40,2.5)}</span><h1>${escape(overall.label)}</h1>${updated}</section>
${openCard}<section class="card"><div class="head"><h2>${escape(t("pulseServices"))}</h2><div class="legend">${([["up","pulseOperational"],["partial","pulseDegraded"],["down","pulseDown"],["maint","pulseMaintenance"],["none","pulseNotMonitored"]] as const).map(([k,key])=>`<span class="l-${k}" data-tip="${escape(t(key))}" role="img" aria-label="${escape(t(key))}">${legendSvg(k)}</span>`).join("")}</div></div>${page.components.map(component).join("")||`<div class="row"><p class="updated">${escape(t("pulseNoData"))}</p></div>`}</section>
<section class="card"><div class="head"><h2>${escape(t("pulseIncidents"))}</h2></div>${history}</section>
<footer><span>© ${year} ${escape(page.title)}</span><span>${escape(t("pulseMadeWith"))} <a href="https://getblup.io/blup-pulse/" rel="noopener">Blup Pulse</a></span></footer>
</div></body></html>`;
}
