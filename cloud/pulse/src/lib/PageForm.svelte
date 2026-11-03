<script lang="ts">
 import {untrack} from "svelte";
 import Plus from "@lucide/svelte/icons/plus";
 import {t} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import type {Monitor} from "./status-pages";
 // A status page's title, address and the monitors it shows, each with the
 // name visitors see. The address is fixed once the page exists.
 type Values={title:string;slug:string;components:{monitorId:string;name:string}[]};
 let {monitors,initial,editing=false,busy=false,onsave,oncancel}:{monitors:Monitor[];initial?:Values;editing?:boolean;busy?:boolean;onsave:(values:Values)=>void;oncancel:()=>void}=$props();
 // The fields start from `initial` and are the form's own from then on;
 // a different page remounts the form ({#key} in the page).
 const start=untrack(()=>initial);
 let title=$state(start?.title??""),slug=$state(start?.slug??""),chosen=$state<Record<string,string>>(Object.fromEntries((start?.components??[]).map(c=>[c.monitorId,c.name])));
 const valid=$derived(Object.values(chosen).every(v=>v.trim().length>0));
 function toggle(id:string,on:boolean){if(on)chosen[id]="";else delete chosen[id];}
 function submit(e:SubmitEvent){e.preventDefault();onsave({title,slug,components:Object.entries(chosen).map(([monitorId,name])=>({monitorId,name:name.trim()}))});}
</script>

<form onsubmit={submit}><Card><div class="grid gap-6">
 <div class="grid gap-4 sm:grid-cols-2"><FloatField id="page-title" surface="bg-white dark:bg-neutral-900" label={t("pulsePageTitle")} required maxlength={100} bind:value={title}/><FloatField id="page-slug" surface="bg-white dark:bg-neutral-900" label={t("pulseSlug")} required disabled={editing} maxlength={40} pattern="[a-z0-9]+(-[a-z0-9]+)*" bind:value={slug}/></div>
 <fieldset><legend class="mb-3 text-sm font-semibold">{t("pulseComponents")}</legend><div class="overflow-hidden rounded-lg border border-neutral-200 dark:border-neutral-800">{#each monitors as monitor(monitor.id)}<div class="grid gap-3 border-b border-neutral-200 px-4 py-3 last:border-b-0 sm:grid-cols-[1fr_minmax(0,18rem)] sm:items-center dark:border-neutral-800"><Switch tone="mint" label={monitor.name} checked={Object.hasOwn(chosen,monitor.id)} onchange={on=>toggle(monitor.id,on)}/>{#if Object.hasOwn(chosen,monitor.id)}<FloatField id={`public-${monitor.id}`} surface="bg-white dark:bg-neutral-900" label={t("pulsePublicName")} maxlength={80} required bind:value={chosen[monitor.id]}/>{/if}</div>{:else}<p class="px-4 py-6 text-center text-sm text-neutral-500">{t("pulseNoMonitors")}</p>{/each}</div></fieldset>
 <div class="flex flex-wrap justify-end gap-2"><Button type="button" variant="secondary" onclick={oncancel}>{t("cancel")}</Button><Button {busy} disabled={!valid}>{#if !editing}<Plus class="size-4"/>{/if}{editing?t("save"):t("pulseCreatePage")}</Button></div>
</div></Card></form>
