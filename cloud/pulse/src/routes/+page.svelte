<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import Copy from "@lucide/svelte/icons/copy";
 import Plus from "@lucide/svelte/icons/plus";
 import RefreshCw from "@lucide/svelte/icons/refresh-cw";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import Bug from "@lucide/svelte/icons/bug";
 import {t,intlLocale} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import Button from "#lib/ui/Button.svelte";
 import ActionMenu from "#lib/ui/ActionMenu.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import StatusDot from "#lib/ui/StatusDot.svelte";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import {platforms} from "../../../shared/pulse-platforms";
 import {pulseApi,parseResponse,type Project} from "../lib/api";
 import {panel} from "../lib/panel.svelte";
 let {data}:PageProps=$props();
 let projects=$derived<Project[]>(data.projects),dsn=$state(""),busy=$state(false);
 const writable=$derived(panel.usage?.readOnly===false);
 const usage=$derived(panel.usage?.limits?{used:panel.usage.accepted,limit:panel.usage.limits.events}:null);
 // The issues of one project: /issues?project=<id>.
 const issuesOf=(id:string)=>resolve(`/issues?${new URLSearchParams({project:id})}`);
 const platformName=(id:string)=>id==="other"?t("pulseOther"):platforms.find(p=>p.id===id)?.label??id;
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function refresh(){projects=(await parseResponse(pulseApi.projects.$get())).projects;}
 async function copy(){try{await navigator.clipboard.writeText(dsn);toast(()=>t("copied"));}catch{toast(()=>t("copyFailed"),{kind:"error"});}}
 async function rotate(p:Project){if(!await confirm({title:t("pulseRotate"),message:t("pulseRotateAsk")}))return;await run(async()=>{dsn=(await parseResponse(pulseApi.projects[":project"].keys.$post({param:{project:p.id}}))).dsn;toast(()=>t("pulseToastRotated"));});}
 async function remove(p:Project){if(!await confirm({title:t("delete"),message:t("pulseDeleteProject"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.projects[":project"].$delete({param:{project:p.id}}));dsn="";await refresh();toast(()=>t("pulseToastDeleted"));});}
</script>

<PageHeader title={t("pulseProjects")} lead={t("pulseScope")}>
 {#snippet aside()}{#if writable}<Button onclick={()=>goto(resolve("/projects/new"))}><Plus class="size-4"/>{t("pulseCreateNewProject")}</Button>{/if}{/snippet}
</PageHeader>

{#if usage}<div class="mt-6 max-w-sm"><p class="text-sm text-neutral-500">{t("pulseMonthUsage",{accepted:usage.used.toLocaleString(intlLocale()),limit:usage.limit.toLocaleString(intlLocale())})}</p><div class="mt-2 h-2 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800"><div class="h-full rounded-full bg-blup" style:width="{Math.min(100,usage.used/Math.max(1,usage.limit)*100)}%"></div></div></div>{/if}

{#if dsn}<div class="mt-6 rounded-xl border border-blup bg-white p-5 dark:bg-neutral-900"><p class="font-semibold">{t("pulseSDK")}</p><div class="mt-3 flex items-center gap-2 rounded-lg bg-neutral-100 p-3 dark:bg-neutral-800"><code class="min-w-0 flex-1 break-all text-sm">{dsn}</code><Button small variant="secondary" onclick={copy}><Copy class="size-4"/>{t("copy")}</Button></div><p class="mt-3 text-sm text-neutral-500">{t("pulseDSNOnce")}</p></div>{/if}

<div class="mt-6"><List empty={t("pulseEmpty")}>
 {#each projects as project(project.id)}
  <ListItem href={issuesOf(project.id)} label={project.name}>
   <div class="flex items-center gap-3"><StatusDot state={project.active?"up":"unknown"}/><div class="min-w-0"><p class="truncate font-semibold">{project.name}</p><p class="text-sm text-neutral-500">{platformName(project.platform)} · {project.active?t("pulseActive"):t("pulsePaused")}</p></div></div>
   {#snippet menu()}<ActionMenu label={t("pulseActions")} items={[{label:t("pulseSeeIssues"),Icon:Bug,onselect:()=>void goto(issuesOf(project.id))},...(project.writable?[{label:t("pulseRotate"),Icon:RefreshCw,onselect:()=>void rotate(project)}]:[]),{label:t("delete"),Icon:Trash2,danger:true,onselect:()=>void remove(project)}]}/>{/snippet}
  </ListItem>
 {/each}
</List></div>

<p class="mt-10 text-xs text-neutral-500">{t("pulsePrivacy")}</p>
