<script lang="ts">
	import type { PageProps } from "./$types";
	import FileUp from "@lucide/svelte/icons/file-up";
	import { staffApi, StaffError, type AppsumoStats } from "#lib/staff.ts";
	import { chunks, codesFromList, codesFromRefunds } from "#lib/appsumo-codes.ts";
	import { planLabel } from "../../../../shared/plan-i18n";
	import { toast } from "#lib/toast.svelte.ts";
	import { confirm } from "#lib/confirm.svelte.ts";
	import Button from "#lib/ui/Button.svelte";
	import Row from "#lib/ui/Row.svelte";
	import Stat from "#lib/ui/Stat.svelte";
	import PageHeader from "#lib/ui/PageHeader.svelte";

	// AppSumo's lifetime codes: how many there are and how many accounts are
	// on each level; importing the codes AppSumo gets (the owner) and
	// revoking the ones AppSumo refunded (admins). The files are read here:
	// only the codes go to the server, in parts.
	let { data }: PageProps = $props();
	let stats = $derived<AppsumoStats | null>(data.stats);
	const owner = $derived(!!data.me?.owner);
	const admin = $derived(data.me?.role === "admin");
	// What the server takes per request: up to 2500 codes to import; the
	// refunds lower accounts in the same transaction, so fewer at a time.
	const IMPORT_PART = 2000, REFUND_PART = 100;

	type Picked = { name: string; codes: string[] };
	let toImport = $state<Picked | null>(null);
	let toRefund = $state<Picked | null>(null);
	let busy = $state<"" | "import" | "refund">("");
	let progress = $state("");
	let imported = $state<{ added: number; skipped: number } | null>(null);
	let refunded = $state<{ revoked: number; notFound: number; users: number } | null>(null);

	const n = (v: number) => v.toLocaleString("pt-BR");
	function failure(e: unknown) {
		if (e instanceof StaffError && e.code === "UNAVAILABLE") return "O AppSumo está desligado neste ambiente (falta o APPSUMO_CODE_PEPPER ou ele tem menos de 32 caracteres).";
		if (e instanceof StaffError && e.code === "OWNER_ONLY") return "Só o dono importa códigos.";
		if (e instanceof StaffError && e.code === "ADMIN_ONLY") return "Só admins cuidam dos reembolsos.";
		if (e instanceof StaffError && e.code === "INVALID") return "O servidor recusou esta parte. Confira o arquivo.";
		return "Não deu certo. Tente de novo.";
	}
	const reload = () => staffApi.appsumo().then((s) => (stats = s)).catch(() => {});

	async function pick(e: Event & { currentTarget: HTMLInputElement }, read: (text: string) => string[]) {
		const file = e.currentTarget.files?.[0];
		e.currentTarget.value = "";
		if (!file) return null;
		try { return { name: file.name, codes: read(await file.text()) }; }
		catch { toast("Não deu para ler o arquivo.", { kind: "error" }); return null; }
	}

	// Each part waits for the previous one; a failure stops there and says
	// how far it got (what went before is saved).
	async function send<T>(kind: "import" | "refund", codes: string[], size: number, post: (part: string[]) => Promise<T>, add: (part: T) => void) {
		const parts = chunks(codes, size);
		busy = kind;
		try {
			for (const [i, part] of parts.entries()) {
				progress = `Enviando parte ${i + 1} de ${parts.length} (${n(part.length)} códigos)`;
				try { add(await post(part)); }
				catch (e) { toast(`Parou na parte ${i + 1} de ${parts.length}. ${failure(e)}`, { kind: "error" }); return false; }
			}
			return true;
		} finally {
			busy = "";
			progress = "";
			await reload();
		}
	}

	async function importAll() {
		if (!toImport) return;
		const totals = { added: 0, skipped: 0 };
		imported = totals;
		const ok = await send("import", toImport.codes, IMPORT_PART, staffApi.importCodes, (r) => { totals.added += r.added; totals.skipped += r.skipped; imported = { ...totals }; });
		if (ok) { toast("Códigos importados"); toImport = null; }
	}

	async function refundAll() {
		if (!toRefund) return;
		const count = toRefund.codes.length;
		if (!(await confirm({ title: `Revogar ${n(count)} ${count === 1 ? "código" : "códigos"}?`, message: "Cada código encontrado deixa de valer para sempre e a conta que o resgatou desce de nível na hora. Não dá para desfazer.", confirmLabel: "Revogar", danger: true }))) return;
		const totals = { revoked: 0, notFound: 0, users: 0 };
		refunded = totals;
		const ok = await send("refund", toRefund.codes, REFUND_PART, staffApi.refundCodes, (r) => { totals.revoked += r.revoked; totals.notFound += r.notFound; totals.users += r.users; refunded = { ...totals }; });
		if (ok) { toast("Reembolsos aplicados"); toRefund = null; }
	}
</script>

<PageHeader title="AppSumo" lead="Os códigos do plano vitalício. O banco guarda só o HMAC e os 4 últimos caracteres de cada código; ninguém da equipe vê um código inteiro." />

{#if stats}
	{#if !stats.available}<p class="mt-6 rounded-xl border border-blup p-5">O resgate está desligado neste ambiente: falta o APPSUMO_CODE_PEPPER ou ele tem menos de 32 caracteres. Importar e reembolsar também não funcionam.</p>{/if}
	<div class="mt-6 grid grid-cols-2 gap-3 lg:grid-cols-3">
		<Stat label="Disponíveis" value={stats.codes.available} hint="Importados e ainda não resgatados" />
		<Stat label="Resgatados" value={stats.codes.redeemed} />
		<Stat label="Revogados" value={stats.codes.revoked} hint="Reembolso ou pela staff" />
	</div>
	<h2 class="mt-8 font-bold">Contas por nível</h2>
	<div class="mt-3 grid grid-cols-2 gap-3 lg:grid-cols-5">
		{#each stats.levels as l (l.level)}
			<Stat label={`Nível ${l.level}`} value={l.users} hint={planLabel(l.plan, "pt")} />
		{/each}
	</div>
{/if}

<div class="mt-6">
	<Row title="Importar códigos" help={owner ? "Arquivo .csv ou .txt com um código por linha, sem cabeçalho. Código repetido ou que já existe (mesmo usado ou revogado) fica como está." : "Só o dono importa códigos."}>
		{#if owner}
			<div class="mt-3 flex flex-wrap items-center gap-3">
				<input id="appsumo-import" type="file" accept=".csv,.txt,text/csv,text/plain" class="peer sr-only" disabled={!!busy} onchange={async (e) => { const picked = await pick(e, codesFromList); if (picked) { toImport = picked; imported = null; } }} />
				<label for="appsumo-import" class="inline-flex cursor-pointer items-center gap-2 rounded-lg border border-neutral-300 px-3 py-1.5 text-sm font-semibold transition-colors peer-focus-visible:ring-2 peer-focus-visible:ring-blup hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-800 {busy ? 'pointer-events-none opacity-50' : ''}"><FileUp class="size-4" />Escolher arquivo</label>
				{#if toImport}
					<span class="text-sm text-neutral-500">{toImport.name}: {n(toImport.codes.length)} {toImport.codes.length === 1 ? "código" : "códigos"}</span>
					<Button small busy={busy === "import"} disabled={!toImport.codes.length || !!busy} onclick={importAll}>Importar</Button>
				{/if}
			</div>
			<div class="mt-3 text-sm" aria-live="polite">
				{#if busy === "import"}<p class="text-neutral-500">{progress}</p>{/if}
				{#if imported}<p>Adicionados: <b>{n(imported.added)}</b>. Ignorados (repetidos, já existentes ou inválidos): <b>{n(imported.skipped)}</b>.</p>{/if}
			</div>
		{/if}
	</Row>

	<Row title="Reembolsos do AppSumo" help={admin ? "O CSV de códigos reembolsados que o AppSumo envia. Lemos a coluna de código (ou, sem ela, o que parece um código BLUP) e mostramos quantos achamos antes de enviar." : "Só admins cuidam dos reembolsos."}>
		{#if admin}
			<div class="mt-3 flex flex-wrap items-center gap-3">
				<input id="appsumo-refunds" type="file" accept=".csv,.txt,text/csv,text/plain" class="peer sr-only" disabled={!!busy} onchange={async (e) => { const picked = await pick(e, codesFromRefunds); if (picked) { toRefund = picked; refunded = null; } }} />
				<label for="appsumo-refunds" class="inline-flex cursor-pointer items-center gap-2 rounded-lg border border-neutral-300 px-3 py-1.5 text-sm font-semibold transition-colors peer-focus-visible:ring-2 peer-focus-visible:ring-blup hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-800 {busy ? 'pointer-events-none opacity-50' : ''}"><FileUp class="size-4" />Escolher arquivo</label>
				{#if toRefund}
					<span class="text-sm text-neutral-500">{toRefund.name}: {toRefund.codes.length ? `${n(toRefund.codes.length)} ${toRefund.codes.length === 1 ? "código encontrado" : "códigos encontrados"}` : "nenhum código encontrado"}</span>
					{#if toRefund.codes.length}<Button small busy={busy === "refund"} disabled={!!busy} onclick={refundAll}>Revogar</Button>{/if}
				{/if}
			</div>
			<div class="mt-3 text-sm" aria-live="polite">
				{#if busy === "refund"}<p class="text-neutral-500">{progress}</p>{/if}
				{#if refunded}<p>Revogados: <b>{n(refunded.revoked)}</b>. Não encontrados ou já revogados: <b>{n(refunded.notFound)}</b>. Contas afetadas (soma das partes): <b>{n(refunded.users)}</b>.</p>{/if}
			</div>
		{/if}
	</Row>
</div>
