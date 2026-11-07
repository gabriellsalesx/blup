<script lang="ts">
	import type { PageProps } from "./$types";
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import Plus from "@lucide/svelte/icons/plus";
	import { staffApi, staffHref, StaffError, type AIFeature } from "#lib/staff.ts";
	import { featureInfo, tabLabels, tabOf } from "#lib/staff-ai.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import BackLink from "#lib/ui/BackLink.svelte";
	import Button from "#lib/ui/Button.svelte";
	import Card from "#lib/ui/Card.svelte";
	import FloatField from "#lib/ui/FloatField.svelte";
	import PageHeader from "#lib/ui/PageHeader.svelte";
	import AIMark from "#lib/ui/AIMark.svelte";

	// Adding a model to an AI list, on a page of its own (as Pulse's new
	// alert destination): its OpenRouter ID, checked against the catalog
	// when saved. Marking it as the recommended (or main) one is done on the
	// list, from its menu. Then back to the list.
	let { data, params }: PageProps = $props();
	const name = $derived(params.feature as AIFeature);
	const info = $derived(featureInfo[name]);
	const list = $derived(data.ai?.features.find((f) => f.feature === name) ?? null);
	const voice = $derived(list?.kind === "speech");
	const back = $derived(staffHref(resolve("/staff/ai/[feature]", { feature: name })));
	let id = $state("");
	let busy = $state(false);
	const valid = /^[a-z0-9][a-z0-9._-]{0,63}\/[a-z0-9][a-z0-9._:-]{0,99}$/;

	async function add(e: SubmitEvent) {
		e.preventDefault();
		if (!list) return;
		const model = id.trim().toLowerCase();
		if (!valid.test(model)) return toast("Use o ID do modelo no OpenRouter, como deepseek/deepseek-v4.1-flash.", { kind: "error" });
		const now = list.models.map((m) => m.id);
		if (now.includes(model)) return toast("Esse modelo já está na lista.", { kind: "error" });
		if (now.length >= list.max) return toast(`No máximo ${list.max} modelos.`, { kind: "error" });
		busy = true;
		try {
			await staffApi.setModels(name, [...now, model], list.recommended);
			toast("Modelo adicionado");
			await goto(back);
		} catch (err) {
			toast(err instanceof StaffError && err.code === "UNKNOWN_MODEL" ? `Esse modelo não existe no OpenRouter ou não é ${voice ? "de voz" : "de texto"}. Confira o ID.` : err instanceof StaffError && err.code === "REASONING_MODEL" ? "Esse modelo sempre raciocina antes de responder e não serve aqui. Escolha outro." : "Não deu certo.", { kind: "error" });
		} finally {
			busy = false;
		}
	}
</script>

<div class="mb-4"><BackLink href={back} label={`${tabLabels[tabOf(name)]} · ${info.title}`} /></div>
<PageHeader title="Adicionar modelo" lead={voice ? "Um modelo de voz do OpenRouter para o Blup Studio mostrar nesta lista." : "Um modelo de texto do OpenRouter para esta lista."} />

<div class="mt-8">
	<Card title={info.title} help={info.lead}>
		<form class="mt-5 grid grid-cols-[minmax(0,1fr)] gap-4" onsubmit={add}>
			<div class="flex items-center gap-3 text-sm text-neutral-500"><AIMark kind={info.mark} class="size-6" /><span>O ID aparece na página do modelo no OpenRouter, como <b>{voice ? "microsoft/mai-voice-2.1" : "deepseek/deepseek-v4.1-flash"}</b>.</span></div>
			<FloatField id="ai-model" surface="bg-white dark:bg-neutral-900" label="ID do modelo no OpenRouter" autocomplete="off" spellcheck={false} maxlength={164} required bind:value={id} />
			<div class="flex justify-end"><Button {busy} disabled={!id.trim() || busy}><Plus class="size-4" />Adicionar</Button></div>
		</form>
	</Card>
</div>
