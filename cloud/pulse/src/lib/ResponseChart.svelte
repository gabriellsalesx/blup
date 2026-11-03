<script lang="ts">
 import {t,intlLocale,formatDate} from "#lib/i18n.svelte.ts";
 import {stateName,type Check} from "./monitor";
 // Response time of the latest checks as one blue area, oldest on the left.
 // Failed checks get a red dot; hovering shows the closest check.
 let {checks}:{checks:Check[]}=$props();
 const H=240,TOP=12,RIGHT=12,BOTTOM=28,LEFT=56;
 let width=$state(0),hover=$state<number|null>(null);
 const points=$derived([...checks].sort((a,b)=>a.checkedAt-b.checkedAt));
 const step=$derived.by(()=>{const max=Math.max(1,...points.map(p=>p.durationMs))*1.1,raw=max/4,mag=10**Math.floor(Math.log10(raw)),n=raw/mag;return (n<=1?1:n<=2?2:n<=5?5:10)*mag;});
 const yMax=$derived(step*4);
 const start=$derived(points[0]?.checkedAt??0),end=$derived(points.at(-1)?.checkedAt??1);
 const x=(ms:number)=>LEFT+(end===start?0.5:(ms-start)/(end-start))*Math.max(1,width-LEFT-RIGHT);
 const y=(v:number)=>TOP+(1-v/yMax)*(H-TOP-BOTTOM);
 const line=$derived(points.map((p,i)=>`${i?"L":"M"}${x(p.checkedAt).toFixed(1)},${y(p.durationMs).toFixed(1)}`).join(""));
 const area=$derived(points.length?`${line}L${x(end).toFixed(1)},${y(0)}L${x(start).toFixed(1)},${y(0)}Z`:"");
 const multiDay=$derived(end-start>86_400_000);
 const tickFormat=$derived(new Intl.DateTimeFormat(intlLocale(),multiDay?{day:"numeric",month:"short"}:{hour:"2-digit",minute:"2-digit"}));
 const xTicks=$derived.by(()=>{const count=Math.max(2,Math.min(6,Math.floor(width/120)));return points.length<2?[]:Array.from({length:count},(_,i)=>start+(end-start)*i/(count-1));});
 const average=$derived(points.length?Math.round(points.reduce((sum,p)=>sum+p.durationMs,0)/points.length):0);
 const ms=(v:number)=>`${v.toLocaleString(intlLocale())} ms`;
 function move(e:PointerEvent){const rect=(e.currentTarget as SVGElement).getBoundingClientRect(),px=e.clientX-rect.left;let best=0;for(let i=1;i<points.length;i++)if(Math.abs(x(points[i].checkedAt)-px)<Math.abs(x(points[best].checkedAt)-px))best=i;hover=points.length?best:null;}
 const current=$derived(hover===null?null:points[hover]??null);
</script>

{#if points.length}
 <p class="mt-1 text-sm text-neutral-500">{t("pulseAverage",{value:average.toLocaleString(intlLocale())})}</p>
 <div class="relative mt-4" bind:clientWidth={width}>
  {#if width}
   <svg class="block w-full touch-none" {width} height={H} role="img" aria-label={t("pulseDuration")} onpointermove={move} onpointerleave={()=>hover=null}>
    {#each [0,1,2,3,4] as i(i)}
     <line x1={LEFT} x2={width-RIGHT} y1={y(step*i)} y2={y(step*i)} class="stroke-neutral-200 dark:stroke-neutral-800" stroke-width="1"/>
     <text x={LEFT-8} y={y(step*i)} dy="0.32em" text-anchor="end" class="fill-neutral-500 text-xs">{(step*i).toLocaleString(intlLocale())}</text>
    {/each}
    {#each xTicks as tick,i(i)}
     <text x={x(tick)} y={H-8} text-anchor={i===0?"start":i===xTicks.length-1?"end":"middle"} class="fill-neutral-500 text-xs">{tickFormat.format(new Date(tick))}</text>
    {/each}
    <path d={area} class="fill-blup/15"/>
    <path d={line} class="stroke-blup" fill="none" stroke-width="2" stroke-linejoin="round" stroke-linecap="round"/>
    {#each points as p(p.id)}{#if p.outcome==="down"}<circle cx={x(p.checkedAt)} cy={y(p.durationMs)} r="4" class="fill-red-500 stroke-white dark:stroke-neutral-900" stroke-width="2"/>{/if}{/each}
    {#if current}
     <line x1={x(current.checkedAt)} x2={x(current.checkedAt)} y1={TOP} y2={y(0)} class="stroke-neutral-400 dark:stroke-neutral-600" stroke-width="1" stroke-dasharray="3 3"/>
     <circle cx={x(current.checkedAt)} cy={y(current.durationMs)} r="5" class="{current.outcome==='down'?'fill-red-500':'fill-blup'} stroke-white dark:stroke-neutral-900" stroke-width="2"/>
    {/if}
   </svg>
   {#if current}
    {@const left=x(current.checkedAt)}
    <div class="pointer-events-none absolute top-0 z-10 rounded-md bg-neutral-800 px-2 py-1 text-[13px] leading-snug whitespace-nowrap text-white dark:bg-neutral-100 dark:text-neutral-900" style:left="{left}px" style:transform={left<width*0.2?"translateX(8px)":left>width*0.8?"translateX(calc(-100% - 8px))":"translateX(-50%)"}>
     <p class="font-semibold">{ms(current.durationMs)}</p><p>{stateName(current.outcome)}</p><p class="opacity-75">{formatDate(new Date(current.checkedAt))}</p>
    </div>
   {/if}
  {/if}
 </div>
{:else}<p class="mt-4 rounded-lg bg-neutral-50 px-4 py-10 text-center text-sm text-neutral-500 dark:bg-neutral-800/40">{t("pulseNoChartData")}</p>{/if}
