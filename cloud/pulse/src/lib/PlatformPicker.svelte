<script lang="ts">
	import {t} from "#lib/i18n.svelte.ts";
	import FloatField from "#lib/ui/FloatField.svelte";
	import Segmented from "#lib/ui/Segmented.svelte";
	import {platforms,type PlatformGroup,type PlatformId} from "../../../shared/pulse-platforms";

	// The platform of a new project: a group of tabs (or a filter across all
	// of them) and a grid of choices. Untested SDKs say so.
	let {value=$bindable()}:{value:PlatformId}=$props();
	let group=$state<PlatformGroup>("popular"),filter=$state("");
	const tabs=$derived([{value:"popular",label:t("pulseGroupPopular")},{value:"browser",label:t("pulseGroupBrowser")},{value:"server",label:t("pulseGroupServer")},{value:"mobile",label:t("pulseGroupMobile")},{value:"desktop",label:t("pulseGroupDesktop")}] as const);
	const label=(p:(typeof platforms)[number])=>p.id==="other"?t("pulseOther"):p.label;
	const shown=$derived.by(()=>{const q=filter.trim().toLowerCase();const list=q?platforms.filter(p=>label(p).toLowerCase().includes(q)):platforms.filter(p=>p.groups.some(g=>g===group));return q?list:[...list,...platforms.filter(p=>p.id==="other")];});
</script>

<div class="flex flex-wrap items-start justify-between gap-4">
	<div class="min-w-0 flex-1 basis-64"><h2 class="font-bold">{t("pulsePlatform")}</h2><p class="mt-0.5 text-sm text-neutral-500">{t("pulsePlatformHelp")}</p></div>
	<div class="w-full sm:w-64"><FloatField id="platform-filter" surface="bg-white dark:bg-neutral-900" label={t("pulseFilterPlatforms")} type="search" bind:value={filter}/></div>
</div>
{#if !filter.trim()}<div class="mt-5"><Segmented label={t("pulsePlatform")} options={tabs} bind:value={group}/></div>{/if}
<div class="mt-4 grid grid-cols-2 gap-2 sm:grid-cols-3" role="radiogroup" aria-label={t("pulsePlatform")}>
	{#each shown as platform(platform.id)}
		<button type="button" role="radio" aria-checked={value===platform.id} class="flex min-h-11 items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm font-semibold transition-colors {value===platform.id?'border-blup bg-emerald-50 text-blup dark:bg-emerald-950/40':'border-neutral-200 hover:border-neutral-400 dark:border-neutral-800 dark:hover:border-neutral-600'}" onclick={()=>value=platform.id}>
			<span>{label(platform)}</span>{#if !platform.tested&&platform.id!=="other"}<span class="rounded-full bg-neutral-100 px-1.5 py-0.5 text-[10px] font-semibold text-neutral-500 dark:bg-neutral-800">{t("pulseInTest")}</span>{/if}
		</button>
	{:else}<p class="col-span-full py-6 text-center text-sm text-neutral-500">{t("pulseNoPlatform")}</p>{/each}
</div>
<p class="mt-3 text-xs text-neutral-500">{t("pulseTestedNote")}</p>
