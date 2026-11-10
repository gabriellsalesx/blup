<script lang="ts">
	import type { PageProps } from "./$types";
	import { resolve } from "$app/paths";
	import { goto } from "$app/navigation";
	import { planLabel } from "../../../../shared/plan-i18n";
	import type { PlanName } from "../../../../shared/pulse-plans";
	import { appsumoLevels } from "../../../../shared/appsumo-levels";
	import type { Language } from "../../../../shared/language";
	import { t, language, formatDate } from "#lib/i18n.svelte.ts";
	import { errorMessage } from "#lib/i18n/errors.ts";
	import { api, type RedeemState } from "#lib/api.ts";
	import Button from "#lib/ui/Button.svelte";
	import Badge from "#lib/ui/Badge.svelte";
	import Card from "#lib/ui/Card.svelte";
	import Row from "#lib/ui/Row.svelte";
	import PageHeader from "#lib/ui/PageHeader.svelte";
	import FloatField from "#lib/ui/FloatField.svelte";
	import Check from "@lucide/svelte/icons/check";
	import CircleCheck from "@lucide/svelte/icons/circle-check";
	import Download from "@lucide/svelte/icons/download";
	import Smartphone from "@lucide/svelte/icons/smartphone";

	// AppSumo's lifetime deal: each code moves the account up one level (the
	// ladder in shared/appsumo-levels.ts), up to the fifth. The code comes in
	// the address (?code=) or is typed; the server normalizes it.
	let { data }: PageProps = $props();
	// The download page of the site in each language (getblup.io).
	const downloadURL: Record<Language, string> = { en: "https://getblup.io/download/", pt: "https://getblup.io/pt/download/", es: "https://getblup.io/es/descargar/" };

	let info = $derived<RedeemState | null>(data.state);
	let code = $derived(data.code);
	let busy = $state(false);
	let error = $state<unknown>(null);
	let done = $state<{ last4: string; plan: PlanName } | null>(null);

	async function redeem(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		done = null;
		let result: Awaited<ReturnType<typeof api.redeem>>;
		try {
			result = await api.redeem(code.trim());
		} catch (err) {
			error = err;
			return;
		} finally {
			busy = false;
		}
		// The answer is the whole new state: shown at once, the field empties
		// and the focus goes to the message (focused when it appears: at the
		// top level the form is gone).
		const { redeemed, ...state } = result;
		info = state;
		done = { last4: redeemed.last4, plan: state.plan };
		code = "";
		// Loaded again without ?code= and with the new plan in the panel.
		goto(resolve("/(panel)/redeem"), { replace: true, reset: false, refreshAll: true }).catch(() => {});
	}
</script>

<PageHeader title={t("redeemTitle")} lead={t("redeemLead")} />

<div class="mt-6 grid max-w-2xl gap-6">
	{#if !info}
		<p class="text-neutral-500">{t("errorGeneric")}</p>
	{:else}
		<Card title={info.level ? t("redeemLevel", { level: info.level, max: info.max }) : t("redeemNoLevel")} help={info.level ? planLabel(info.plan, language()) : undefined}>
			{#snippet action()}
				{#if info?.level}<Badge tone="blue">{t("lifetime")}</Badge>{/if}
			{/snippet}
			<ol class="mt-4 grid gap-2 sm:grid-cols-5" aria-label={t("redeemLevels")}>
				{#each appsumoLevels as plan, i (plan)}
					{@const reached = i < info.level}
					{@const current = i + 1 === info.level}
					<li class="flex items-center gap-2 rounded-lg border px-3 py-2 text-sm sm:flex-col sm:items-start sm:gap-1 {current ? 'border-blup ring-1 ring-blup' : 'border-neutral-200 dark:border-neutral-800'}" aria-current={current ? "step" : undefined}>
						<span class="flex items-center gap-1 text-xs text-neutral-500">
							{#if reached}<Check class="size-3.5 text-green-600" />{/if}
							{t("redeemLevelShort", { level: i + 1 })}
						</span>
						<b class={reached ? "" : "text-neutral-500"}>{planLabel(plan, language())}</b>
					</li>
				{/each}
			</ol>
		</Card>

		{#if info.stripeActive && info.level}
			<div class="rounded-xl border border-blup p-5">
				<p>{t("redeemStripe")}</p>
				<a class="mt-3 inline-block text-blup underline" href={resolve("/(panel)/account/plan")}>{t("redeemGoPlan")}</a>
			</div>
		{/if}

		<!-- The one live region of the page: the result or the error. -->
		<div aria-live="polite">
			{#if done}
				<p class="flex items-start gap-3 outline-none" tabindex="-1" {@attach (el) => el.focus()}><CircleCheck class="mt-0.5 size-5 shrink-0 text-green-600" />{t("redeemDone", { last4: done.last4, plan: planLabel(done.plan, language()) })}</p>
			{/if}
			{#if error}<p class="text-sm text-red-600">{errorMessage(error, { INVALID: "redeemInvalid", TOO_MANY_REQUESTS: "redeemRate" })}</p>{/if}
		</div>

		{#if !info.available}
			<p class="text-neutral-500">{t("redeemUnavailable")}</p>
		{:else if info.level >= info.max}
			<p class="text-neutral-500">{t("redeemTop")}</p>
		{:else}
			<form class="grid gap-4 sm:grid-cols-[1fr_auto] sm:items-center" onsubmit={redeem}>
				<FloatField id="redeem-code" label={t("redeemCode")} class="font-mono tracking-wider" autocomplete="off" autocapitalize="characters" spellcheck={false} maxlength={200} required bind:value={code} />
				<Button {busy} disabled={!code.trim()}>{t("redeemAction")}</Button>
			</form>
		{/if}

		{#if done}
			<Card title={t("redeemNext")}>
				<div class="mt-2">
					<Row title={t("redeemStudio")} help={t("redeemStudioHelp")}>
						{#snippet action()}<Download class="size-5 text-blup" />{/snippet}
						<a class="mt-2 inline-block text-sm text-blup underline" href={downloadURL[language()]}>{t("redeemDownloadStudio")}</a>
					</Row>
					<Row title={t("redeemNotes")} help={t("redeemNotesHelp")}>
						{#snippet action()}<Smartphone class="size-5 text-blup" />{/snippet}
						<a class="mt-2 inline-block text-sm text-blup underline" href={downloadURL[language()]}>{t("redeemNotesLink")}</a>
					</Row>
				</div>
			</Card>
		{/if}

		{#if info.codes.length}
			<Row title={t("redeemCodes")}>
				<ul class="mt-3 grid gap-2 text-sm">
					{#each info.codes as c (c.id)}
						<li class="flex flex-wrap items-baseline justify-between gap-x-4 rounded-lg border border-neutral-200 px-3 py-2 dark:border-neutral-800">
							<b class="font-mono">…{c.last4}</b>
							<span class="text-neutral-500">{t("redeemOn", { date: formatDate(new Date(c.redeemedAt)) })}</span>
						</li>
					{/each}
				</ul>
			</Row>
		{/if}
	{/if}
</div>
