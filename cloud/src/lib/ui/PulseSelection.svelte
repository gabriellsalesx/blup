<script lang="ts">
 import {t} from "#lib/i18n.svelte.ts";
 type Keep={projects:string[];monitors:string[];pages:string[]};
 let {resources,limits,keep=$bindable()}: {resources:{projects:{id:string;name:string}[];monitors:{id:string;name:string;projectId:string}[];pages:{id:string;name:string}[]};limits:{projects:number;monitors:number;pages:number};keep:Keep}=$props();
 const fields=[{id:"projects",label:"pulseProjects"},{id:"monitors",label:"pulseMonitors"},{id:"pages",label:"pulsePages"}] as const;
 function toggle(field:keyof Keep,id:string,checked:boolean){keep={...keep,[field]:checked?[...keep[field],id]:keep[field].filter(v=>v!==id)};if(field==="projects")keep={...keep,monitors:keep.monitors.filter(id=>resources.monitors.some(m=>m.id===id&&keep.projects.includes(m.projectId)))};}
</script>
<section class="my-5 grid gap-4 rounded-xl border border-amber-300 p-5">
 <h3 class="font-semibold">{t("billingKeep")}</h3><p class="text-sm">{t("billingKeepHelp")}</p>
 {#each fields as field(field.id)}<fieldset class="grid gap-2"><legend class="mb-2 font-semibold">{t(field.label)} <span class="text-sm font-normal">{t("billingPickLimit",{limit:limits[field.id]})}</span></legend>{#each resources[field.id] as item(item.id)}<label class="flex gap-2 text-sm"><input type="checkbox" checked={keep[field.id].includes(item.id)} disabled={field.id==="monitors"&&!resources.monitors.some(m=>m.id===item.id&&keep.projects.includes(m.projectId))} onchange={e=>toggle(field.id,item.id,e.currentTarget.checked)}/>{item.name}</label>{/each}</fieldset>{/each}
</section>
