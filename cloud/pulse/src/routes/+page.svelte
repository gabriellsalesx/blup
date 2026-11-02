<script lang="ts">
 import {onMount} from "svelte";
 import Copy from "@lucide/svelte/icons/copy";
 import Plus from "@lucide/svelte/icons/plus";
 import {t,intlLocale,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import {pulseApi,parseResponse,type Project,type Issue} from "../lib/api";
 import {panel} from "../lib/panel.svelte";
 let projects=$state<Project[]>([]),selected=$state<Project|null>(null),issues=$state<Issue[]>([]),name=$state(""),platform=$state<"javascript"|"node"|"go"|"other">("javascript"),dsn=$state(""),busy=$state(false),error=$state<unknown>(null),filter=$state<"open"|"resolved"|"ignored">("open");
 let events=$state<Awaited<ReturnType<typeof getEvents>>>([]);
 const writable=$derived(panel.usage?.readOnly===false);
 async function refresh(){projects=(await parseResponse(pulseApi.projects.$get())).projects;}
 async function select(p:Project){selected=p;events=[];issues=(await parseResponse(pulseApi.projects[":project"].issues.$get({param:{project:p.id},query:{state:filter}}))).issues;}
 async function run(fn:()=>Promise<void>){busy=true;error=null;try{await fn();}catch(e){error=e;}finally{busy=false;}}
 async function create(e:SubmitEvent){e.preventDefault();await run(async()=>{const result=await parseResponse(pulseApi.projects.$post({json:{name:name.trim(),platform}}));dsn=result.key.dsn;name="";await refresh();});}
 async function copy(){try{await navigator.clipboard.writeText(dsn);toast(()=>t("copied"));}catch{toast(()=>t("copyFailed"));}}
 async function rotate(p:Project){if(!await confirm({title:t("pulseRotate"),message:t("pulseRotateAsk")}))return;await run(async()=>{dsn=(await parseResponse(pulseApi.projects[":project"].keys.$post({param:{project:p.id}}))).dsn;});}
 async function remove(p:Project){if(!await confirm({title:t("delete"),message:t("pulseDeleteProject")}))return;await run(async()=>{await parseResponse(pulseApi.projects[":project"].$delete({param:{project:p.id}}));selected=null;dsn="";await refresh();});}
 async function change(issue:Issue,state:Issue["state"]){if(!selected)return;const project=selected;await run(async()=>{await parseResponse(pulseApi.projects[":project"].issues[":issue"].$patch({param:{project:project.id,issue:issue.id},json:{state}}));await select(project);});}
 async function showEvents(issue:Issue){const project=selected;if(project)await run(async()=>{events=await getEvents(project.id,issue.id);});}
 async function getEvents(project:string,issue:string){return (await parseResponse(pulseApi.projects[":project"].issues[":issue"].events.$get({param:{project,issue}}))).events;}
 onMount(()=>{void run(refresh);});
</script>
<h1 class="text-3xl font-bold">{t("pulseProjects")}</h1>
{#if panel.usage?.limits}<p class="mt-3 text-sm text-neutral-500">{t("pulseUsage",{accepted:panel.usage.accepted.toLocaleString(intlLocale()),reserved:panel.usage.reserved.toLocaleString(intlLocale()),limit:panel.usage.limits.events.toLocaleString(intlLocale())})}</p>{/if}
{#if writable}<form class="my-6 flex flex-wrap items-end gap-3" onsubmit={create}><FloatField id="project-name" label={t("pulseProjectName")} required maxlength={80} bind:value={name}/><label class="grid gap-2 text-sm">{t("pulsePlatform")}<select class="rounded-lg border p-3 dark:bg-neutral-900" bind:value={platform}><option value="javascript">JavaScript</option><option value="node">Node.js</option><option value="go">Go</option><option value="other">{t("pulseOther")}</option></select></label><Button {busy}><Plus class="size-4"/>{t("pulseCreateProject")}</Button></form>{/if}
{#if error}<p role="alert" class="my-4 text-red-600">{errorMessage(error)}</p>{/if}
{#if dsn}<Card><p>{t("pulseSDK")}</p><div class="my-3 flex items-center gap-2"><code class="break-all text-sm">{dsn}</code><Button variant="secondary" onclick={copy} aria-label={t("copy")}><Copy class="size-4"/></Button></div><p class="text-sm text-neutral-500">{t("pulseDSNOnce")}</p></Card>{/if}
<div class="grid gap-3">{#each projects as project(project.id)}<Card><div class="flex flex-wrap items-center justify-between gap-3"><button class="text-left" onclick={()=>run(()=>select(project))}><h2 class="font-semibold">{project.name}</h2><p class="text-xs text-neutral-500">{project.platform} · {project.active?t("pulseActive"):t("pulsePaused")}</p></button><div class="flex gap-2">{#if writable}<Button variant="secondary" onclick={()=>rotate(project)}>{t("pulseRotate")}</Button>{/if}<Button variant="secondary" onclick={()=>remove(project)}>{t("delete")}</Button></div></div></Card>{:else}<p class="my-8 text-neutral-500">{t("pulseEmpty")}</p>{/each}</div>
{#if selected}<section class="mt-8"><div class="mb-5 flex flex-wrap items-center justify-between gap-3"><h2 class="text-xl font-bold">{selected.name}: {t("pulseIssues")}</h2><select class="rounded-lg border p-2 dark:bg-neutral-900" aria-label={t("pulseIssues")} bind:value={filter} onchange={()=>{const project=selected;if(project)void run(()=>select(project));}}><option value="open">{t("pulseOpen")}</option><option value="resolved">{t("pulseResolved")}</option><option value="ignored">{t("pulseIgnored")}</option></select></div><div class="grid gap-3">{#each issues as issue(issue.id)}<Card><button class="text-left" onclick={()=>showEvents(issue)}><h3 class="font-semibold break-words">{issue.title}</h3><p class="my-2 text-xs text-neutral-500">{t("pulseCount",{count:issue.count.toLocaleString(intlLocale())})} · {formatDate(new Date(issue.lastAt))}</p></button>{#if writable}<div class="flex flex-wrap gap-2"><Button variant="secondary" onclick={()=>change(issue,"resolved")}>{t("pulseResolve")}</Button><Button variant="secondary" onclick={()=>change(issue,"ignored")}>{t("pulseIgnore")}</Button><Button variant="secondary" onclick={()=>change(issue,"open")}>{t("pulseReopen")}</Button></div>{/if}</Card>{:else}<p class="text-neutral-500">{t("pulseNoIssues")}</p>{/each}</div></section>{/if}
{#if events.length}<section class="mt-8"><h2 class="mb-3 text-xl font-bold">{t("pulseEvents")}</h2>{#each events as event(event.id)}<details class="my-2 rounded-lg border border-neutral-200 p-3 dark:border-neutral-800"><summary>{formatDate(new Date(event.receivedAt))}</summary><pre class="mt-3 overflow-auto whitespace-pre-wrap break-words text-xs">{JSON.stringify(event.event,null,2)}</pre></details>{/each}</section>{/if}
<p class="mt-8 text-xs text-neutral-500">{t("pulsePrivacy")}</p>
