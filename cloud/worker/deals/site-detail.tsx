/** @jsxRuntime automatic */
/** @jsxImportSource hono/jsx */
import {languageTag,type Language} from "../../shared/language";
import {money} from "./message";
import {Shell,Icon,dealHref,translator,placeOf,untilOf,type SiteDeal,type SitePageProps,type Translate} from "./site-page";

// A deal's own page (/o/<code>/): the photo, the price, the coupon and the
// post's words, a button to the store (through the short link, which
// counts the click), sharing, and how the price moved in the last 30 days,
// drawn on the server as SVG: no chart library reaches the browser.

export const HISTORY_DAYS=30;
const DAY=86_400_000;
export type PricePoint={at:number;price:number};
export type SiteDetail=SiteDeal&{expired:boolean;prices:PricePoint[]};
export type DetailPageProps=SitePageProps&{deal:SiteDetail;now:number};

// The prices are kept only when they change, so each one holds until the
// next: the average weighs each by how long it lasted. Under two prices
// there is nothing to compare.
export type PriceStats={points:PricePoint[];start:number;end:number;current:number;average:number;low:number;high:number;percent:number;mood:"below"|"above"|"average"};
export function priceStats(prices:readonly PricePoint[],current:number|null,now:number,days=HISTORY_DAYS):PriceStats|null{
 const start=now-days*DAY,sorted=[...prices].sort((a,b)=>a.at-b.at);
 // The price at the window's start is the last one before it.
 const before=sorted.filter(p=>p.at<=start).at(-1),inside=sorted.filter(p=>p.at>start&&p.at<=now);
 const points=[...(before?[{at:start,price:before.price}]:[]),...inside];
 if(points.length<2||new Set(points.map(p=>p.price)).size<2)return null;
 const from=points[0]!.at;
 let weighted=0;
 points.forEach((p,i)=>{weighted+=p.price*((points[i+1]?.at??now)-p.at);});
 const average=now>from?weighted/(now-from):points[0]!.price,price=current??points.at(-1)!.price;
 const percent=Math.round((price-average)/average*100);
 return {points,start:from,end:now,current:price,average,low:Math.min(...points.map(p=>p.price)),high:Math.max(...points.map(p=>p.price)),percent:Math.abs(percent),mood:percent<=-2?"below":percent>=2?"above":"average"};
}

// A round top for the scale: four steps of a round size (1, 1.5, 2, 2.5,
// 3, 4, 5, 6 or 8 times a power of ten), just above the highest price.
function niceTop(value:number){
 const step=value/4,power=10**Math.floor(Math.log10(step)),unit=[1,1.5,2,2.5,3,4,5,6,8,10].find(u=>u*power>=step)??10;
 return unit*power*4;
}

function PriceChart({stats,currency,lang,t}:{stats:PriceStats;currency:string;lang:Language;t:Translate}){
 const top=niceTop(stats.high*1.1),span=stats.end-stats.start||1;
 const x=(at:number)=>((at-stats.start)/span*100).toFixed(2),y=(price:number)=>((1-price/top)*100).toFixed(2);
 // A step line: each price holds until the next.
 let line=`M0 ${y(stats.points[0]!.price)}`;
 for(const p of stats.points.slice(1))line+=` H${x(p.at)} V${y(p.price)}`;
 line+=" H100";
 const ticks=[0,1,2,3,4].map(i=>top*i/4),days=[0,1,2,3,4].map(i=>stats.start+span*i/4);
 const date=new Intl.DateTimeFormat(languageTag[lang],{day:"2-digit",month:"short",timeZone:"America/Sao_Paulo"});
 return <figure class="chart">
  <div class="chart-plot">
   <div class="chart-y" aria-hidden="true">{ticks.map(v=><span style={`top:${y(v)}%`}>{money(Math.round(v),currency,lang)}</span>)}</div>
   <svg viewBox="0 0 100 100" preserveAspectRatio="none" role="img" aria-label={t("dealsSiteChartLabel",{days:HISTORY_DAYS})}>
    {ticks.map(v=><line class="chart-grid" x1="0" x2="100" y1={y(v)} y2={y(v)}/>)}
    <path class="chart-area" d={`${line} V100 H0 Z`}/>
    <path class="chart-line" d={line}/>
   </svg>
  </div>
  <div class="chart-x" aria-hidden="true">{days.map(d=><span>{date.format(d)}</span>)}</div>
 </figure>;
}

// Where the price sits between the period's lowest and highest: a mark
// on the bar with an arrow above and below pointing at it, and the price
// now over the top arrow (kept inside the bar's ends).
function PriceBar({stats,currency,lang,t}:{stats:PriceStats;currency:string;lang:Language;t:Translate}){
 const at=Math.min(100,Math.max(0,stats.high>stats.low?(stats.current-stats.low)/(stats.high-stats.low)*100:50));
 const align=at<15?"start":at>85?"end":"center";
 return <div class="bar">
  <div class="bar-track">
   <span class={`bar-now bar-now-${align}`} style={`left:${at.toFixed(1)}%`}>{t("dealsSiteNow",{price:money(stats.current,currency,lang)})}</span>
   <span class="bar-arrow bar-arrow-down" style={`left:${at.toFixed(1)}%`} aria-hidden="true"/>
   <span class="bar-mark" style={`left:${at.toFixed(1)}%`}/>
   <span class="bar-arrow bar-arrow-up" style={`left:${at.toFixed(1)}%`} aria-hidden="true"/>
  </div>
  <div class="bar-ends"><span>{t("dealsSiteLowest")}: {money(stats.low,currency,lang)}</span><span>{t("dealsSiteAverageLabel")}: {money(Math.round(stats.average),currency,lang)}</span><span>{t("dealsSiteHighest")}: {money(stats.high,currency,lang)}</span></div>
 </div>;
}

export function DetailPage(props:DetailPageProps){
 const {site,lang,deal}=props,t=translator(lang);
 const store=placeOf(deal),until=untilOf(deal,lang,t,props.now),free=deal.kind==="game_free"||deal.price===0,go=`${props.links}/r/${deal.code}`;
 const canonical=`${props.origin}${dealHref(props,deal.code,site.language)}`;
 const alternates=Object.fromEntries((["pt","en","es"] as const).map(l=>[l,`${props.origin}${dealHref(props,deal.code,l)}`])) as Record<Language,string>;
 const stats=deal.currency&&!free?priceStats(deal.prices,deal.price,props.now):null;
 const posted=new Intl.DateTimeFormat(languageTag[lang],{dateStyle:"long",timeZone:"America/Sao_Paulo"}).format(deal.sentAt);
 // Search engines: a product with its offer.
 const data={"@context":"https://schema.org","@type":"Product",name:deal.title,...(deal.image?{image:deal.image}:{}),...(deal.description?{description:deal.description}:{}),...(deal.price!==null&&deal.currency?{offers:{"@type":"Offer",price:(deal.price/100).toFixed(2),priceCurrency:deal.currency,url:go,availability:deal.expired?"https://schema.org/Discontinued":"https://schema.org/InStock"}}:{})};
 return <Shell page={props} title={`${deal.title} · ${site.title}`} description={deal.description||(store?`${deal.title} · ${store}`:deal.title)} canonical={canonical} alternates={alternates} image={deal.image} data={data} scripts={[SHARE_SCRIPT]}>
  <p class="back"><a href={`${props.base}/${lang!==site.language?`?lang=${lang}`:""}`}><Icon name="back" small/>{t("dealsSiteBack")}</a></p>
  <article class="detail">
   <div class="detail-image">
    {deal.image?<img src={deal.image} alt={deal.title} decoding="async" referrerpolicy="no-referrer" width="560" height="560"/>:<span class="deal-blank"><Icon name="tag"/></span>}
    {deal.discount&&deal.discount>0?<span class="deal-off">{t("dealsSiteOff",{percent:deal.discount})}</span>:null}
   </div>
   <div class="detail-body">
    <p class="detail-meta">{store?<span class="detail-store">{store}</span>:null}<time datetime={new Date(deal.sentAt).toISOString()}>{t("dealsSitePosted",{date:posted})}</time></p>
    <h1 class="detail-title">{deal.title}</h1>
    <p class="detail-price">
     {free?<strong>{t("dealsSiteFree")}</strong>:deal.price!==null&&deal.currency?<strong>{money(deal.price,deal.currency,lang)}</strong>:null}
     {!free&&deal.originalPrice&&deal.currency&&deal.price!==null&&deal.originalPrice>deal.price?<s>{money(deal.originalPrice,deal.currency,lang)}</s>:null}
    </p>
    {free&&deal.originalPrice&&deal.currency?<p class="deal-until">{t("dealsSiteWorth",{price:money(deal.originalPrice,deal.currency,lang)})}</p>:null}
    {until&&!deal.expired?<p class="deal-until">{until}</p>:null}
    {deal.coupon?<p class="deal-coupon"><Icon name="ticket" small/>{t("dealsSiteCoupon",{code:deal.coupon})}</p>:null}
    {deal.description?<p class="detail-text">{deal.description}</p>:null}
    {deal.expired?<p class="detail-ended">{t("dealsSiteEnded")}</p>:<a class="button button-accent detail-go" href={go} rel="nofollow sponsored noopener">{t("dealsSiteGo")}<Icon name="external" small/></a>}
    <button class="button button-ghost share" type="button" data-share={canonical} data-copied={t("dealsSiteCopied")}><Icon name="share" small/><span data-label>{t("dealsSiteShare")}</span></button>
   </div>
  </article>
  {stats&&deal.currency?<section class="history" aria-labelledby="history-title">
   <h2 id="history-title">{t("dealsSiteHistory")}</h2>
   <p class="history-help">{t("dealsSiteHistoryHelp",{days:HISTORY_DAYS})}</p>
   <div class="history-card">
    <p class={`history-mood mood-${stats.mood}`}><Icon name="chart" small/>{stats.mood==="below"?t("dealsSiteBelow",{percent:stats.percent,days:HISTORY_DAYS}):stats.mood==="above"?t("dealsSiteAbove",{percent:stats.percent,days:HISTORY_DAYS}):t("dealsSiteAverage",{days:HISTORY_DAYS})}</p>
    <PriceBar stats={stats} currency={deal.currency} lang={lang} t={t}/>
   </div>
   <div class="history-card"><PriceChart stats={stats} currency={deal.currency} lang={lang} t={t}/></div>
  </section>:null}
 </Shell>;
}

// Sharing: the phone's own share sheet, or the link copied. Allowed by its
// hash, as the theme's.
export const SHARE_SCRIPT=`document.addEventListener("click",function(e){var b=e.target.closest&&e.target.closest("[data-share]");if(!b)return;var u=b.getAttribute("data-share");if(navigator.share){navigator.share({title:document.title,url:u}).catch(function(){});return}if(navigator.clipboard){navigator.clipboard.writeText(u).then(function(){var l=b.querySelector("[data-label]");if(l)l.textContent=b.getAttribute("data-copied")})}});`;
