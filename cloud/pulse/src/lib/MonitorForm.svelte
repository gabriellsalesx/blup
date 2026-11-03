<script lang="ts">
 import {untrack} from "svelte";
 import Plus from "@lucide/svelte/icons/plus";
 import {t} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import FieldHelp from "#lib/ui/FieldHelp.svelte";
 import Select from "#lib/ui/Select.svelte";
 import type {Project} from "./api";
 import {panel} from "./panel.svelte";
 // A monitor's name, the address to check, the project whose destinations
 // get its alerts and how often it is checked. Once the monitor exists its
 // project is fixed.
 type Values={name:string;url:string;projectId:string;intervalMinutes:number};
 let {projects,initial,editing=false,busy=false,onsave,oncancel}:{projects:Project[];initial?:Values;editing?:boolean;busy?:boolean;onsave:(values:Values)=>void;oncancel:()=>void}=$props();
 const minInterval=$derived(panel.usage?.limits?.intervalMinutes??5);
 // The fields start from `initial` (or the first project and the plan's
 // shortest common interval) and are the form's own from then on.
 const start=untrack(()=>initial??{name:"",url:"https://",projectId:projects[0]?.id??"",intervalMinutes:Math.max(5,minInterval)});
 let name=$state(start.name),url=$state(start.url),projectId=$state(start.projectId),intervalMinutes=$state(start.intervalMinutes);
 const intervals=$derived([...new Set([1,2,5,10,15,30,60,start.intervalMinutes])].filter(m=>m>=minInterval).sort((a,b)=>a-b).map(m=>({value:m,label:t("pulseEvery",{minutes:m})})));
 const projectOptions=$derived(projects.map(p=>({value:p.id,label:p.name})));
 function submit(e:SubmitEvent){e.preventDefault();onsave({name:name.trim(),url:url.trim(),projectId,intervalMinutes});}
</script>

<form onsubmit={submit}><Card><div class="grid grid-cols-[minmax(0,1fr)] gap-6">
 <FloatField id="monitor-name" surface="bg-white dark:bg-neutral-900" label={t("pulseMonitorName")} maxlength={80} required bind:value={name}/>
 <div><FloatField id="monitor-url" surface="bg-white dark:bg-neutral-900" label={t("pulseURL")} type="url" maxlength={2048} required bind:value={url}/><p class="mt-2 px-3 text-sm text-neutral-500">{t("pulseURLHelp")}</p></div>
 <FieldHelp help={t("pulseProjectHelp")}>{#if projectOptions.length}<Select surface="bg-white dark:bg-neutral-900" label={t("pulseAllProjects")} options={projectOptions} disabled={editing} bind:value={projectId}/>{:else}<p class="text-sm text-neutral-500">{t("pulseEmpty")}</p>{/if}</FieldHelp>
 <FieldHelp help={t("pulseIntervalHelp")}><Select surface="bg-white dark:bg-neutral-900" label={t("pulseInterval")} options={intervals} bind:value={intervalMinutes}/></FieldHelp>
 <div class="flex justify-end gap-2"><Button type="button" variant="secondary" onclick={oncancel}>{t("cancel")}</Button><Button {busy} disabled={!projectId||!name.trim()}>{#if !editing}<Plus class="size-4"/>{/if}{editing?t("save"):t("pulseCreateMonitor")}</Button></div>
</div></Card></form>
