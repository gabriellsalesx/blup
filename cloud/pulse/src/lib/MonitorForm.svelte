<script module lang="ts">
 import type {Monitor} from "./monitor";
 export type MonitorOptions=Monitor["options"];
 // A header to keep: `value` null keeps the one stored for its name.
 export type HeaderEdit={name:string;value:string|null};
 export type MonitorValues={name:string;url:string;projectId:string;intervalMinutes:number;options:MonitorOptions;headers:HeaderEdit[]};
</script>

<script lang="ts">
 import {untrack} from "svelte";
 import Plus from "@lucide/svelte/icons/plus";
 import X from "@lucide/svelte/icons/x";
 import ChevronDown from "@lucide/svelte/icons/chevron-down";
 import {t} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import FieldHelp from "#lib/ui/FieldHelp.svelte";
 import Select from "#lib/ui/Select.svelte";
 import type {Project} from "./api";
 import {panel} from "./panel.svelte";
 // A monitor's name, the address to check, the project whose destinations
 // get its alerts and how often it is checked; and, folded until asked
 // for, how the check asks and what counts as up: method, accepted
 // statuses, a keyword, the time to wait and request headers. Header values
 // are never shown back: a stored one stays unless typed again. Once the
 // monitor exists its project is fixed.
 const DEFAULTS:MonitorOptions={method:"GET",expectedStatus:[],keyword:null,timeoutSeconds:10};
 let {projects,initial,editing=false,busy=false,onsave,oncancel}:{projects:Project[];initial?:Pick<Monitor,"name"|"url"|"projectId"|"intervalMinutes"|"options"|"headerNames">;editing?:boolean;busy?:boolean;onsave:(values:MonitorValues)=>void;oncancel:()=>void}=$props();
 const minInterval=$derived(panel.usage?.limits?.intervalMinutes??5);
 // The fields start from `initial` (or the first project and the plan's
 // shortest common interval) and are the form's own from then on.
 const start=untrack(()=>initial??{name:"",url:"https://",projectId:projects[0]?.id??"",intervalMinutes:Math.max(5,minInterval),options:DEFAULTS,headerNames:[] as string[]});
 let name=$state(start.name),url=$state(start.url),projectId=$state(start.projectId),intervalMinutes=$state(start.intervalMinutes);
 let method=$state(start.options.method),statuses=$state(start.options.expectedStatus.join(", ")),keyword=$state(start.options.keyword?.text??""),keywordMode=$state(start.options.keyword?.mode??"contains"),timeoutSeconds=$state(start.options.timeoutSeconds);
 // Each header row: its name, a new value (empty keeps a stored one) and
 // whether it was stored before.
 let headers=$state(start.headerNames.map(n=>({name:n,value:"",stored:true})));
 const custom=start.options.method!=="GET"||start.options.expectedStatus.length>0||!!start.options.keyword||start.options.timeoutSeconds!==10||start.headerNames.length>0;
 let advanced=$state(custom);
 const intervals=$derived([...new Set([1,2,5,10,15,30,60,start.intervalMinutes])].filter(m=>m>=minInterval).sort((a,b)=>a-b).map(m=>({value:m,label:t("pulseEvery",{minutes:m})})));
 const projectOptions=$derived(projects.map(p=>({value:p.id,label:p.name})));
 const methods=[{value:"GET",label:"GET"},{value:"HEAD",label:"HEAD"}] as const;
 const keywordModes=$derived([{value:"contains",label:t("pulseKeywordContains")},{value:"absent",label:t("pulseKeywordAbsent")}] as const);
 const timeouts=$derived([...new Set([5,10,15,20,30,start.options.timeoutSeconds])].sort((a,b)=>a-b).map(s=>({value:s,label:t("pulseSeconds",{seconds:String(s)})})));
 // "200, 204" as a list; null when something is not a status code.
 const parsedStatuses=$derived.by(()=>{const parts=statuses.split(/[\s,;]+/).filter(Boolean);const codes=parts.map(Number);return codes.every(c=>Number.isInteger(c)&&c>=100&&c<=599)&&codes.length<=10?[...new Set(codes)]:null;});
 const headerNameOk=(n:string)=>/^[A-Za-z0-9!#$%&'*+.^_`|~-]{1,64}$/.test(n);
 const headersOk=$derived(headers.every(h=>headerNameOk(h.name.trim())&&(h.stored||h.value.length>0))&&new Set(headers.map(h=>h.name.trim().toLowerCase())).size===headers.length);
 const valid=$derived(!!projectId&&!!name.trim()&&parsedStatuses!==null&&headersOk&&!(method==="HEAD"&&keyword.trim()));
 function submit(e:SubmitEvent){e.preventDefault();if(!valid||!parsedStatuses)return;onsave({name:name.trim(),url:url.trim(),projectId,intervalMinutes,options:{method,expectedStatus:parsedStatuses,keyword:keyword.trim()?{text:keyword.trim(),mode:keywordMode}:null,timeoutSeconds},headers:headers.map(h=>({name:h.name.trim(),value:h.value?h.value:h.stored?null:""}))});}
</script>

<form class="grid grid-cols-[minmax(0,1fr)] gap-6" onsubmit={submit}>
 <Card><div class="grid grid-cols-[minmax(0,1fr)] gap-6">
  <FloatField id="monitor-name" surface="bg-white dark:bg-neutral-900" label={t("pulseMonitorName")} maxlength={80} required bind:value={name}/>
  <div><FloatField id="monitor-url" surface="bg-white dark:bg-neutral-900" label={t("pulseURL")} type="url" maxlength={2048} required bind:value={url}/><p class="mt-2 px-3 text-sm text-neutral-500">{t("pulseURLHelp")}</p></div>
  <FieldHelp help={t("pulseProjectHelp")}>{#if projectOptions.length}<Select surface="bg-white dark:bg-neutral-900" label={t("pulseAllProjects")} options={projectOptions} disabled={editing} bind:value={projectId}/>{:else}<p class="text-sm text-neutral-500">{t("pulseEmpty")}</p>{/if}</FieldHelp>
  <FieldHelp help={t("pulseIntervalHelp")}><Select surface="bg-white dark:bg-neutral-900" label={t("pulseInterval")} options={intervals} bind:value={intervalMinutes}/></FieldHelp>
 </div></Card>

 <Card title={t("pulseAdvancedCheck")} help={t("pulseAdvancedCheckHelp")}>
  {#snippet action()}<Button small variant="secondary" type="button" aria-expanded={advanced} onclick={()=>advanced=!advanced}><ChevronDown class="size-4 transition-transform {advanced?"rotate-180":""}"/>{advanced?t("pulseHide"):t("pulseShow")}</Button>{/snippet}
  {#if advanced}
   <div class="mt-6 grid grid-cols-[minmax(0,1fr)] gap-6">
    <div class="grid grid-cols-[minmax(0,1fr)] gap-6 sm:grid-cols-3">
     <Select surface="bg-white dark:bg-neutral-900" label={t("pulseMethod")} options={methods} bind:value={method}/>
     <div class="sm:col-span-2"><FloatField id="monitor-statuses" surface="bg-white dark:bg-neutral-900" label={t("pulseExpectedStatus")} maxlength={80} bind:value={statuses}/><p class="mt-2 px-3 text-sm {parsedStatuses===null?"text-red-600 dark:text-red-400":"text-neutral-500"}">{parsedStatuses===null?t("pulseExpectedStatusInvalid"):t("pulseExpectedStatusHelp")}</p></div>
    </div>
    <div class="grid grid-cols-[minmax(0,1fr)] gap-6 sm:grid-cols-3">
     <div class="sm:col-span-2"><FloatField id="monitor-keyword" surface="bg-white dark:bg-neutral-900" label={t("pulseKeyword")} maxlength={200} disabled={method==="HEAD"} bind:value={keyword}/><p class="mt-2 px-3 text-sm text-neutral-500">{method==="HEAD"?t("pulseKeywordNeedsGet"):t("pulseKeywordHelp")}</p></div>
     <Select surface="bg-white dark:bg-neutral-900" label={t("pulseKeywordMode")} options={keywordModes} disabled={method==="HEAD"||!keyword.trim()} bind:value={keywordMode}/>
    </div>
    <FieldHelp help={t("pulseTimeoutHelp")}><Select surface="bg-white dark:bg-neutral-900" label={t("pulseTimeout")} options={timeouts} bind:value={timeoutSeconds}/></FieldHelp>
    <div>
     <p class="font-semibold">{t("pulseRequestHeaders")}</p><p class="mt-0.5 text-sm text-neutral-500">{t("pulseRequestHeadersHelp")}</p>
     <div class="mt-4 grid grid-cols-[minmax(0,1fr)] gap-3">
      {#each headers as header,i(i)}
       <div class="grid grid-cols-[minmax(0,1fr)_auto] gap-3 sm:grid-cols-[minmax(0,14rem)_minmax(0,1fr)_auto]">
        <FloatField id={`header-name-${i}`} surface="bg-white dark:bg-neutral-900" label={t("pulseHeaderName")} maxlength={64} disabled={header.stored} bind:value={header.name}/>
        <div class="col-span-2 row-start-2 sm:col-span-1 sm:row-start-auto"><FloatField id={`header-value-${i}`} surface="bg-white dark:bg-neutral-900" label={header.stored?t("pulseHeaderValueKept"):t("pulseHeaderValue")} type="password" autocomplete="off" maxlength={1024} required={!header.stored} bind:value={header.value}/></div>
        <button type="button" class="grid size-14 place-items-center rounded-md border border-neutral-300 text-neutral-500 hover:bg-neutral-100 hover:text-red-600 dark:border-neutral-700 dark:hover:bg-neutral-800" aria-label={t("pulseRemoveHeader")} onclick={()=>headers.splice(i,1)}><X class="size-4"/></button>
       </div>
      {/each}
      {#if headers.length<5}<Button small variant="secondary" type="button" class="justify-self-start" onclick={()=>headers.push({name:"",value:"",stored:false})}><Plus class="size-4"/>{t("pulseAddHeader")}</Button>{/if}
     </div>
    </div>
   </div>
  {/if}
 </Card>

 <div class="flex justify-end gap-2"><Button type="button" variant="secondary" onclick={oncancel}>{t("cancel")}</Button><Button {busy} disabled={!valid}>{#if !editing}<Plus class="size-4"/>{/if}{editing?t("save"):t("pulseCreateMonitor")}</Button></div>
</form>
