<script lang="ts">
 import {planLabel} from "../../../../../shared/plan-i18n";
 import {t,formatDate,language} from "#lib/i18n.svelte.ts";
	import Lock from "@lucide/svelte/icons/lock";
	import Monitor from "@lucide/svelte/icons/monitor";
	import Row from "#lib/ui/Row.svelte";
	import Button from "#lib/ui/Button.svelte";
	import {errorMessage} from "#lib/i18n/errors.ts";
	import { api, type Device } from "#lib/api.ts";
	import { confirm } from "#lib/confirm.svelte.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import { bytes, megabytes } from "#lib/usage.ts";

	// Sync: the space the notes use in the cloud and the computers with Blup
	// Studio signed in (each can be disconnected).
	let { data } = $props();

	const limit = $derived(data.me.plan.limits.storageMB);
	const used = $derived(data.me.usage.storageBytes);
	const maxDevices = $derived(data.me.plan.limits.devices);
	let devices = $state<Device[] | null>(null);
	let busy = $state("");

	const when = (d: string) => formatDate(d);
	async function load() {
		try {
			devices = (await api.devices()).devices;
		} catch(error) {
			toast(()=>errorMessage(error),{kind:"error"});
			devices = [];
		}
	}
	load();

	async function disconnect(d: Device) {
		const ok = await confirm({ title: t("disconnectTitle"), message: t("disconnectBody",{name:d.name}), confirmLabel: t("disconnect"), danger: true });
		if (!ok) return;
		busy = d.id;
		try {
			await api.disconnect(d.id);
			toast(()=>t("disconnected"));
			await load();
		} catch(error) {
			toast(()=>errorMessage(error), { kind: "error" });
		} finally {
			busy = "";
		}
	}
</script>

<h1 class="text-3xl font-extrabold tracking-tight">{t("sync")}</h1>
<p class="mt-2 text-neutral-500">{t("syncLead")}</p>

<div class="mt-6">
	<Row title={t("spaceUsed")}  help={t("spacePlan",{used:bytes(used),limit:megabytes(limit),plan:planLabel(data.me.plan.name,language())})}>
		<div class="mt-3 h-2 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800">
			<div class="h-full rounded-full bg-blup" style:width="{Math.min(100, (used / (limit * 1024 * 1024)) * 100)}%"></div>
		</div>
	</Row>
	<Row
		title={t("connectedDevices")}
		help={devices === null ? t("loading") : t("deviceCount",{used:maxDevices>=0?t("ofLimit",{used:devices.length,limit:maxDevices}):devices.length,plan:planLabel(data.me.plan.name,language())})+(devices.length?"":". "+t("signInStudio"))}
	/>
	{#each devices ?? [] as d (d.id)}
		<Row title={d.name} help={t("deviceDates",{created:when(d.createdAt),seen:when(d.lastSeen)})}>
			{#snippet action()}
				<div class="flex items-center gap-3">
					<Monitor class="size-5 text-neutral-400" />
					<Button variant="secondary" busy={busy === d.id} onclick={() => disconnect(d)}>{t("disconnect")}</Button>
				</div>
			{/snippet}
		</Row>
	{/each}
	<Row title={t("encryption")}  help={t("encryptionHelp")} >
		{#snippet action()}<Lock class="size-5 text-green-600" />{/snippet}
	</Row>
</div>
