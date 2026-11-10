<script lang="ts">
	import type { PageProps } from "./$types";
	import { resolve } from "$app/paths";
	import AccountId from "#lib/ui/AccountId.svelte";
	import { accountIdHelp } from "#lib/account-id.ts";
	import { goto, refreshAll, replaceState } from "$app/navigation";
	import ArrowLeft from "@lucide/svelte/icons/arrow-left";
	import External from "@lucide/svelte/icons/external-link";
	import KeyRound from "@lucide/svelte/icons/key-round";
	import Trash from "@lucide/svelte/icons/trash-2";
	import { staffApi, staffHref, actionLabel, billingLabel, money, when, type AppsumoCode } from "#lib/staff.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import { confirm } from "#lib/confirm.svelte.ts";
	import Button from "#lib/ui/Button.svelte";
	import Row from "#lib/ui/Row.svelte";
	import UsageBars from "#lib/ui/UsageBars.svelte";
	import AddonBars from "#lib/ui/AddonBars.svelte";
	import Field from "#lib/ui/Field.svelte";
	import Modal from "#lib/ui/Modal.svelte";
	import Segmented from "#lib/ui/Segmented.svelte";
	import DeviceIcon from "#lib/ui/DeviceIcon.svelte";
	import { untrack } from "svelte";
	import { page } from "$app/state";
	import { readAgent } from "../../../../../shared/user-agent";
	import { accountTabs, type AccountTab } from "./tabs";

	// One account, in tabs: the summary (ID, plan, deletion), its usage
	// (Cloud, Pulse and Deals), payments, access (sessions with where each
	// one is, passkeys), what the staff did to it, and the support actions
	// (blocking and deleting are for admins).
	let { data, params }: PageProps = $props();
	const admin = $derived(data.me?.role === "admin");
	const tabLabels: Record<AccountTab, string> = { summary: "Resumo", usage: "Uso", billing: "Pagamentos", access: "Acesso", history: "Histórico", actions: "Ações" };
	const tabs = $derived(accountTabs.filter((t) => t !== "actions" || admin).map((value) => ({ value, label: tabLabels[value] })));
	let tab = $state<AccountTab>(untrack(() => data.tab));
	// The tab goes in the address, so a reload (or a link) opens it.
	function show(next: AccountTab) {
		const url = new URL(page.url.href);
		url.searchParams.set("tab", next);
		replaceState(url, {});
	}
	// Where a session is: Blup Studio (with the computer's name) or a browser,
	// and on which system.
	function sessionName(agent: string | null) {
		const info = readAgent(agent);
		if (info.studio) return `Blup Studio · ${info.studio.host}${info.system ? ` (${info.system})` : ""}`;
		if (info.browser && info.system) return `${info.browser} no ${info.system}`;
		// Anything else (a script, a tool) by the first word it gives.
		return info.browser ?? info.system ?? (agent?.trim().split(/\s+/)[0]?.slice(0, 60) || "Aparelho desconhecido");
	}
	const id = $derived(params.id);

	const d = $derived(data.detail);
	// The owner's own account (STAFF_OWNER_EMAIL): nobody blocks or deletes
	// it here, and only the owner ends its sessions or removes its passkeys.
	// The API refuses the same.
	const protectedAccess = $derived(d.ownerAccount && !data.me?.owner);
	let busy = $state(false);
	let banOpen = $state(false);
	let reason = $state("");

	// Irreversible actions ask first (confirm.svelte.ts).
	async function revokeSessions() {
		if (!(await confirm({ title: "Encerrar todas as sessões?", message: "A pessoa sai de todos os aparelhos e do Blup Studio e precisa entrar de novo.", confirmLabel: "Encerrar" }))) return;
		run(() => staffApi.revokeSessions(id), "Sessões encerradas");
	}

	async function removePasskey(pid: string, name: string) {
		if (!(await confirm({ title: "Remover a passkey?", message: `"${name}" deixa de entrar nesta conta. A pessoa entra pelo código por e-mail e cadastra outra.`, confirmLabel: "Remover", danger: true }))) return;
		run(() => staffApi.removePasskey(id, pid), "Passkey removida");
	}

	async function removeAccount() {
		const ok = await confirm({
			title: "Excluir a conta?",
			message: "Encerra as sessões, tira os sites do ar e solicita o cancelamento da assinatura. O suporte pode recuperar os dados por 30 dias; depois, a limpeza apaga tudo de vez.",
			confirmLabel: "Excluir",
			danger: true,
			typeToConfirm: d.user.email ?? d.user.id,
		});
		if (!ok) return;
		busy = true;
		try { await staffApi.remove(id); toast("Conta excluída"); goto(staffHref(resolve("/staff"))); }
		catch { toast("Não deu certo. Tente de novo.", { kind: "error" }); }
		finally { busy = false; }
	}

	// An AppSumo code of the account (admins): the account goes down to the
	// codes it still has.
	async function revokeCode(c: AppsumoCode) {
		if (!(await confirm({ title: `Revogar o código …${c.last4}?`, message: "O código deixa de valer para sempre e a conta desce um nível do plano vitalício na hora. Não dá para desfazer.", confirmLabel: "Revogar", danger: true }))) return;
		run(() => staffApi.revokeCode(id, c.id), "Código revogado");
	}
	const codeStatus: Record<AppsumoCode["status"], string> = { available: "Disponível", redeemed: "Resgatado", revoked: "Revogado" };
	const revokeReason: Record<"refund" | "staff", string> = { refund: "reembolso do AppSumo", staff: "pela staff" };

	async function run(fn: () => Promise<unknown>, done: string) {
		busy = true;
		try { await fn(); toast(done); await refreshAll(); }
		catch { toast("Não deu certo. Tente de novo.", { kind: "error" }); }
		finally { busy = false; }
	}
</script>

<a href={staffHref(resolve("/staff"))} class="inline-flex items-center gap-1.5 text-sm font-semibold text-neutral-500 hover:text-neutral-900 dark:hover:text-white"><ArrowLeft class="size-4" />Contas</a>

{#if d}
	<div class="mt-3 flex flex-wrap items-center gap-3">
		<h1 class="text-3xl font-extrabold tracking-tight break-all">{d.user.email ?? "Sem e-mail (só passkey)"}</h1>
		{#if d.user.banned}<span class="rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-semibold text-red-700 dark:bg-red-950 dark:text-red-300">Bloqueada</span>{/if}
	</div>
	<p class="mt-1 text-sm text-neutral-500">Criada em {when(d.user.createdAt)}</p>
	{#if d.user.banned && d.user.banReason}<p class="mt-2 text-sm text-red-600">Motivo do bloqueio: {d.user.banReason}</p>{/if}

	{#if d.deletion.requestedAt}
		<Row title="Conta excluída" help={d.deletion.restorable ? `Os dados podem ser recuperados até ${when(d.deletion.purgeAfter)}. Recuperar não reinicia assinatura nem sessões antigas.` : "O prazo de recuperação terminou. A limpeza definitiva é processada em lotes."}>
			{#snippet action()}
				{#if d.deletion.restorable}<Button variant="secondary" {busy} onclick={() => run(() => staffApi.restore(id), "Conta recuperada")}>Recuperar conta</Button>{/if}
			{/snippet}
		</Row>
	{/if}

	<div class="mt-6"><Segmented label="Seções da conta" options={tabs} bind:value={tab} onchange={show} /></div>

	<div class="mt-2">
		{#if tab === "summary"}
		<Row title="ID da conta" help={accountIdHelp()}><div class="mt-2"><AccountId id={d.user.id} emphasis /></div></Row>
		<Row title={`Plano ${d.plan.label}`} help={d.plan.lifetime ? "Vitalício (AppSumo). Não renova; os códigos estão em Pagamentos." : d.plan.periodEnd ? `${d.plan.cancelAtPeriodEnd ? "Termina" : "Renova"} em ${when(d.plan.periodEnd)}` : "Sem assinatura paga."}>
			{#snippet action()}
				{#if d.stripeUrl}<a class="inline-flex items-center gap-1.5 rounded-lg border border-neutral-300 px-4 py-2.5 font-semibold hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-900" href={d.stripeUrl} target="_blank" rel="noreferrer">Abrir no Stripe <External class="size-4" /></a>{/if}
			{/snippet}
		</Row>

		{:else if tab === "usage"}
		<Row title="Uso" help="Só números: o conteúdo das notas é criptografado e a staff não vê.">
			<UsageBars usage={d.usage} limits={d.plan.limits} ai={d.ai} />
		</Row>

		<Row title="Adicionais" help="Blup Pulse e Blup Deals: o que o plano permite e quanto a conta usa agora.">
			<AddonBars addons={d.addons} />
		</Row>

		{:else if tab === "billing"}
		<Row title="Pagamentos" help="O que o Stripe avisou: pagamentos, falhas, reembolsos, contestações e alertas de fraude.">
			<ul class="mt-3 grid gap-1.5 text-sm">
				{#each d.billing as b (b.id)}
					{@const l = billingLabel[b.type] ?? { text: b.type, alert: false }}
					<li class="flex flex-wrap items-baseline gap-x-2">
						<b class={l.alert ? "text-red-600" : ""}>{l.text}</b>
						<span>{money(b.amount, b.currency)}</span>
						<span class="text-neutral-500">· {when(b.createdAt)}</span>
					</li>
				{:else}
					<li class="text-neutral-500">Nada ainda.</li>
				{/each}
			</ul>
		</Row>

		<Row title="Códigos do AppSumo" help={d.plan.lifetime ? "Vitalício (AppSumo): um nível por código resgatado, até 5. Só os 4 últimos caracteres." : "Só os 4 últimos caracteres de cada código."}>
			<ul class="mt-3 grid gap-2 text-sm">
				{#each d.appsumo as c (c.id)}
					<li class="flex items-center gap-3 rounded-lg border border-neutral-200 px-3 py-2 dark:border-neutral-800">
						<span class="min-w-0 flex-1">
							<b class="font-mono">…{c.last4}</b> <span class="text-neutral-500">· {codeStatus[c.status]}</span>
							<span class="block text-neutral-500">Importado em {when(c.importedAt)}{#if c.redeemedAt !== null} · resgatado em {when(c.redeemedAt)}{/if}{#if c.revokedAt !== null} · revogado em {when(c.revokedAt)}{#if c.revokeReason} ({revokeReason[c.revokeReason]}){/if}{/if}</span>
						</span>
						{#if admin && c.status === "redeemed"}<Button variant="secondary" small {busy} onclick={() => revokeCode(c)}>Revogar</Button>{/if}
					</li>
				{:else}
					<li class="text-neutral-500">Nenhum código.</li>
				{/each}
			</ul>
		</Row>

		{:else if tab === "access"}
		<!-- The button by the title, the list in full width below it. -->
		<section class="border-b border-neutral-200 py-5 dark:border-neutral-800">
			<div class="flex flex-wrap items-center justify-between gap-x-8 gap-y-3">
				<div class="min-w-0">
					<h2 class="font-semibold">Sessões</h2>
					<p class="mt-0.5 text-sm text-neutral-500">{`${d.sessions.length} ${d.sessions.length === 1 ? "aparelho conectado" : "aparelhos conectados"}`}</p>
				</div>
				<Button variant="secondary" {busy} disabled={!d.sessions.length || protectedAccess} title={protectedAccess ? "Só o dono encerra as sessões desta conta." : undefined} onclick={revokeSessions}>Encerrar todas</Button>
			</div>
			{#if d.sessions.length}
				<ul class="mt-4 grid gap-2 text-sm">
					{#each d.sessions as s (s.id)}
						{@const info = readAgent(s.userAgent)}
						<li class="flex items-center gap-3 rounded-lg border border-neutral-200 px-3 py-2 dark:border-neutral-800" title={s.userAgent ?? ""}>
							<span class="grid size-9 shrink-0 place-items-center rounded-full bg-neutral-100 dark:bg-neutral-800"><DeviceIcon platform={info.platform} class="size-5" /></span>
							<span class="min-w-0 flex-1"><b class="block truncate">{sessionName(s.userAgent)}</b><span class="block truncate text-neutral-500">{when(s.createdAt)} · {s.ipAddress || "IP desconhecido"}</span></span>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<Row title="Passkeys" help="Perdeu o aparelho? Remova a passkey: a pessoa entra pelo código por e-mail e cadastra outra.">
			<ul class="mt-3 grid gap-2">
				{#each d.passkeys as p (p.id)}
					<li class="flex items-center gap-3 rounded-lg border border-neutral-200 px-3 py-2 text-sm dark:border-neutral-800">
						<KeyRound class="size-4 text-neutral-500" />
						<span class="flex-1"><b>{p.name || "Passkey"}</b> <span class="text-neutral-500">· {when(p.createdAt)}</span></span>
						<button class="rounded-md p-1.5 text-neutral-500 hover:bg-neutral-100 hover:text-red-600 dark:hover:bg-neutral-800" aria-label="Remover passkey" title="Remover" disabled={busy || protectedAccess} onclick={() => removePasskey(p.id, p.name || "Passkey")}><Trash class="size-4" /></button>
					</li>
				{:else}
					<li class="text-sm text-neutral-500">Nenhuma passkey.</li>
				{/each}
			</ul>
		</Row>

		{:else if tab === "history"}
		<Row title="Histórico da staff" help="O que a equipe fez nesta conta.">
			<ul class="mt-3 grid gap-1.5 text-sm">
				{#each d.history as h (h.id)}
					<li><b>{actionLabel[h.action] ?? h.action}</b> <span class="text-neutral-500">· {h.staffEmail} · {when(h.createdAt)}</span>{#if h.detail}<br /><span class="text-neutral-500">{h.detail}</span>{/if}</li>
				{:else}
					<li class="text-neutral-500">Nada ainda.</li>
				{/each}
			</ul>
		</Row>

		{:else if tab === "actions" && admin}
			{#if d.user.banned}
				<Row title="Desbloquear a conta" help="A pessoa volta a entrar normalmente.">
					{#snippet action()}<Button variant="secondary" {busy} onclick={() => run(() => staffApi.unban(id), "Conta desbloqueada")}>Desbloquear</Button>{/snippet}
				</Row>
			{:else if d.ownerAccount}
				<Row title="Conta do dono" help="Esta é a conta do dono do Blup Cloud. Ninguém da equipe bloqueia nem exclui esta conta pela staff." />
			{:else}
				<Row title="Bloquear a conta" help="Tira a pessoa de todos os aparelhos e impede novos logins até desbloquear. Pedimos o motivo.">
					{#snippet action()}<Button variant="secondary" {busy} onclick={() => { reason = ""; banOpen = true; }}>Bloquear</Button>{/snippet}
				</Row>
			{/if}
			{#if !d.ownerAccount}
				<Row title="Excluir a conta" help="Encerra as sessões, tira os sites do ar e solicita o cancelamento da assinatura. O suporte pode recuperar os dados por 30 dias; depois, a limpeza apaga tudo de vez.">
					{#snippet action()}<Button class="bg-red-700! text-white!" {busy} disabled={!!d.deletion.requestedAt} onclick={removeAccount}>Excluir</Button>{/snippet}
				</Row>
			{/if}
		{/if}
	</div>

	<Modal bind:open={banOpen} title="Bloquear a conta">
		<p class="text-sm text-neutral-500">A pessoa sai de todos os aparelhos e não consegue entrar até ser desbloqueada.</p>
		<form class="mt-4 grid gap-3" onsubmit={(e) => { e.preventDefault(); banOpen = false; run(() => staffApi.ban(id, reason.trim()), "Conta bloqueada"); }}>
			<Field label="Motivo" required minlength={3} maxlength={300} bind:value={reason} />
			<div class="flex justify-end gap-2">
				<Button variant="secondary" type="button" onclick={() => (banOpen = false)}>Cancelar</Button>
				<Button disabled={reason.trim().length < 3}>Bloquear</Button>
			</div>
		</form>
	</Modal>

{/if}
