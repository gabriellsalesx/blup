<script lang="ts">
 import {planLabel} from "../../../../shared/plan-i18n";
 import {t,language} from "#lib/i18n.svelte.ts";
	import MonitorSmartphone from "@lucide/svelte/icons/monitor-smartphone";
	import CircleCheck from "@lucide/svelte/icons/circle-check";
	import { authClient } from "#lib/auth.ts";
	import Button from "#lib/ui/Button.svelte";
	import PageHeader from "#lib/ui/PageHeader.svelte";
	import FloatField from "#lib/ui/FloatField.svelte";

	// Blup Studio signing in through the browser: the Studio shows a code
	// and opens this page; the person checks it is the same code and
	// approves. Checking the code claims it for this session, so only this
	// person can approve it.
	let { data } = $props();

	type Step = "type" | "confirm" | "approved" | "denied";
	let step = $derived<Step>(data.checked ? "confirm" : "type");
	let code = $derived(data.code);
	let error = $derived<(()=>string)|null>(data.checked === false ? () => t("deviceInvalid") : null);
	let busy = $state(false);

	const clean = (c: string) => c.replace(/[^0-9a-z]/gi, "").toUpperCase();
	const shown = $derived(clean(code).replace(/^(.{4})(.+)$/, "$1-$2"));

	async function check(e?: SubmitEvent) {
		e?.preventDefault();
		error = null;
		busy = true;
		const { error: err } = await authClient.device({ query: { user_code: clean(code) } });
		busy = false;
		if (err) return (error = ()=>t("deviceInvalid"));
		step = "confirm";
	}

	async function decide(approve: boolean) {
		error = null;
		busy = true;
		const { error: err } = approve ? await authClient.device.approve({ userCode: clean(code) }) : await authClient.device.deny({ userCode: clean(code) });
		busy = false;
		if (err && "code" in err && err.code === "DEVICES_LIMIT") return (error = ()=>data.me.plan.limits.devices===1?t("deviceLimitOne",{plan:planLabel(data.me.plan.name,language())}):t("deviceLimit",{plan:planLabel(data.me.plan.name,language()),count:data.me.plan.limits.devices}));
		if (err) return (error = ()=>t("deviceInvalid"));
		step = approve ? "approved" : "denied";
	}
</script>

<PageHeader title={t("deviceTitle")}/>

<div class="mt-6 max-w-lg">
	{#if step === "type"}
		<p class="text-neutral-500">{t("deviceEnter")}</p>
		<form class="mt-6 grid gap-4" onsubmit={check}>
			<FloatField id="code" label={t("code")}  class="font-mono tracking-[0.2em] uppercase" autocomplete="off" required bind:value={code} />
			{#if error}<p role="alert" class="text-sm text-red-600">{error()}</p>{/if}
			<Button {busy} disabled={clean(code).length < 8}>{t("continue")}</Button>
		</form>
	{:else if step === "confirm"}
		<div class="flex items-start gap-4">
			<MonitorSmartphone class="mt-1 size-6 shrink-0 text-blup" />
			<p>{t("deviceWants",{account:data.me.email??"Blup Cloud"})}</p>
		</div>
		<p class="mt-6 text-sm text-neutral-500">{t("deviceCheck")}</p>
		<p class="mt-2 font-mono text-4xl font-bold tracking-[0.2em]">{shown}</p>
		<p class="mt-6 text-sm text-neutral-500">{t("deviceSafety")}</p>
		{#if error}<p role="alert" class="mt-4 text-sm text-red-600">{error()}</p>{/if}
		<div class="mt-6 flex gap-3">
			<Button {busy} onclick={() => decide(true)}>{t("approve")}</Button>
			<Button variant="secondary" disabled={busy} onclick={() => decide(false)}>{t("deny")}</Button>
		</div>
	{:else if step === "approved"}
		<div class="flex items-start gap-4">
			<CircleCheck class="mt-1 size-6 shrink-0 text-green-600" />
			<p>{t("deviceApprovedHelp")}</p>
		</div>
	{:else}
		<p>{t("deviceDeniedHelp")}</p>
	{/if}
</div>
