<script lang="ts">
	import { goto, invalidateAll } from "$app/navigation";
	import { page } from "$app/state";
	import ArrowLeft from "@lucide/svelte/icons/arrow-left";
	import External from "@lucide/svelte/icons/external-link";
	import KeyRound from "@lucide/svelte/icons/key-round";
	import Trash from "@lucide/svelte/icons/trash-2";
	import { staffApi, staffHref, actionLabel, when } from "#lib/staff.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import { confirm } from "#lib/confirm.svelte.ts";
	import Button from "#lib/ui/Button.svelte";
	import Row from "#lib/ui/Row.svelte";
	import Field from "#lib/ui/Field.svelte";
	import Modal from "#lib/ui/Modal.svelte";

	// One account: plan, sessions, passkeys, what the staff did to it, and the
	// support actions (blocking and deleting are for admins).
	let { data } = $props();
	const admin = $derived(data.me?.role === "admin");
	const id = $derived(page.params.id ?? "");

	const d = $derived(data.detail);
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
			message: "Cancela a assinatura no Stripe e apaga a conta, as sessões e as passkeys. Não dá para desfazer.",
			confirmLabel: "Excluir",
			danger: true,
			typeToConfirm: d.user.email ?? d.user.id,
		});
		if (!ok) return;
		busy = true;
		try { await staffApi.remove(id); toast("Conta excluída"); goto(staffHref("/staff")); }
		catch { toast("Não deu certo. Tente de novo.", { kind: "error" }); }
		finally { busy = false; }
	}

	async function run(fn: () => Promise<unknown>, done: string) {
		busy = true;
		try { await fn(); toast(done); await invalidateAll(); }
		catch { toast("Não deu certo. Tente de novo.", { kind: "error" }); }
		finally { busy = false; }
	}
</script>

<a href={staffHref("/staff")} class="inline-flex items-center gap-1.5 text-sm font-semibold text-neutral-500 hover:text-neutral-900 dark:hover:text-white"><ArrowLeft class="size-4" />Contas</a>

{#if d}
	<div class="mt-3 flex flex-wrap items-center gap-3">
		<h1 class="text-3xl font-extrabold tracking-tight break-all">{d.user.email ?? "Sem e-mail (só passkey)"}</h1>
		{#if d.user.banned}<span class="rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-semibold text-red-700 dark:bg-red-950 dark:text-red-300">Bloqueada</span>{/if}
	</div>
	<p class="mt-1 text-sm text-neutral-500">Criada em {when(d.user.createdAt)} · <code class="text-xs">{d.user.id}</code></p>
	{#if d.user.banned && d.user.banReason}<p class="mt-2 text-sm text-red-600">Motivo do bloqueio: {d.user.banReason}</p>{/if}

	<div class="mt-6">
		<Row title={`Plano ${d.plan.label}`} help={d.plan.periodEnd ? `${d.plan.cancelAtPeriodEnd ? "Termina" : "Renova"} em ${when(d.plan.periodEnd)}` : "Sem assinatura paga."}>
			{#snippet action()}
				{#if d.stripeUrl}<a class="inline-flex items-center gap-1.5 rounded-lg border border-neutral-300 px-4 py-2.5 font-semibold hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-900" href={d.stripeUrl} target="_blank" rel="noreferrer">Abrir no Stripe <External class="size-4" /></a>{/if}
			{/snippet}
		</Row>

		<Row title="Sessões" help={`${d.sessions.length} ${d.sessions.length === 1 ? "aparelho conectado" : "aparelhos conectados"}`}>
			{#snippet action()}
				<Button variant="secondary" {busy} disabled={!d.sessions.length} onclick={revokeSessions}>Encerrar todas</Button>
			{/snippet}
			{#if d.sessions.length}
				<ul class="mt-3 grid gap-1.5 text-sm">
					{#each d.sessions as s (s.id)}
						<li class="truncate text-neutral-600 dark:text-neutral-400" title={s.userAgent ?? ""}>{when(s.createdAt)} · {s.ipAddress || "IP desconhecido"}</li>
					{/each}
				</ul>
			{/if}
		</Row>

		<Row title="Passkeys" help="Perdeu o aparelho? Remova a passkey: a pessoa entra pelo código por e-mail e cadastra outra.">
			<ul class="mt-3 grid gap-2">
				{#each d.passkeys as p (p.id)}
					<li class="flex items-center gap-3 rounded-lg border border-neutral-200 px-3 py-2 text-sm dark:border-neutral-800">
						<KeyRound class="size-4 text-neutral-500" />
						<span class="flex-1"><b>{p.name || "Passkey"}</b> <span class="text-neutral-500">· {when(p.createdAt)}</span></span>
						<button class="rounded-md p-1.5 text-neutral-500 hover:bg-neutral-100 hover:text-red-600 dark:hover:bg-neutral-800" aria-label="Remover passkey" title="Remover" disabled={busy} onclick={() => removePasskey(p.id, p.name || "Passkey")}><Trash class="size-4" /></button>
					</li>
				{:else}
					<li class="text-sm text-neutral-500">Nenhuma passkey.</li>
				{/each}
			</ul>
		</Row>

		<Row title="Histórico da staff" help="O que a equipe fez nesta conta.">
			<ul class="mt-3 grid gap-1.5 text-sm">
				{#each d.history as h (h.id)}
					<li><b>{actionLabel[h.action] ?? h.action}</b> <span class="text-neutral-500">· {h.staffEmail} · {when(h.createdAt)}</span>{#if h.detail}<br /><span class="text-neutral-500">{h.detail}</span>{/if}</li>
				{:else}
					<li class="text-neutral-500">Nada ainda.</li>
				{/each}
			</ul>
		</Row>

		{#if admin}
			{#if d.user.banned}
				<Row title="Desbloquear a conta" help="A pessoa volta a entrar normalmente.">
					{#snippet action()}<Button variant="secondary" {busy} onclick={() => run(() => staffApi.unban(id), "Conta desbloqueada")}>Desbloquear</Button>{/snippet}
				</Row>
			{:else}
				<Row title="Bloquear a conta" help="Tira a pessoa de todos os aparelhos e impede novos logins até desbloquear. Pedimos o motivo.">
					{#snippet action()}<Button variant="secondary" {busy} onclick={() => { reason = ""; banOpen = true; }}>Bloquear</Button>{/snippet}
				</Row>
			{/if}
			<Row title="Excluir a conta" help="Cancela a assinatura no Stripe e apaga a conta, as sessões e as passkeys. Não dá para desfazer.">
				{#snippet action()}<Button class="bg-red-700! text-white!" {busy} onclick={removeAccount}>Excluir</Button>{/snippet}
			</Row>
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
