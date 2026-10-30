<script lang="ts">
	import { goto, invalidateAll } from "$app/navigation";
	import { page } from "$app/state";
	import ArrowLeft from "@lucide/svelte/icons/arrow-left";
	import External from "@lucide/svelte/icons/external-link";
	import KeyRound from "@lucide/svelte/icons/key-round";
	import Trash from "@lucide/svelte/icons/trash-2";
	import { staffApi, actionLabel, when } from "#lib/staff.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import Button from "#lib/ui/Button.svelte";
	import Card from "#lib/ui/Card.svelte";
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
	let delOpen = $state(false);
	let confirmEmail = $state("");

	async function run(fn: () => Promise<unknown>, done: string) {
		busy = true;
		try { await fn(); toast(done); await invalidateAll(); }
		catch { toast("Não deu certo. Tente de novo.", { kind: "error" }); }
		finally { busy = false; }
	}
</script>

<a href="/staff" class="inline-flex items-center gap-1.5 text-sm font-semibold text-neutral-500 hover:text-neutral-900 dark:hover:text-white"><ArrowLeft class="size-4" />Contas</a>

{#if d}
	<div class="mt-3 flex flex-wrap items-center gap-3">
		<h1 class="text-2xl font-extrabold tracking-tight">{d.user.email}</h1>
		{#if d.user.banned}<span class="rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-semibold text-red-700 dark:bg-red-950 dark:text-red-300">Bloqueada</span>{/if}
	</div>
	<p class="mt-1 text-sm text-neutral-500">Criada em {when(d.user.createdAt)} · <code class="text-xs">{d.user.id}</code></p>
	{#if d.user.banned && d.user.banReason}<p class="mt-2 text-sm text-red-600">Motivo do bloqueio: {d.user.banReason}</p>{/if}

	<div class="mt-6 grid gap-5 md:grid-cols-2">
		<Card title={`Plano ${d.plan.label}`} help={d.plan.periodEnd ? `${d.plan.cancelAtPeriodEnd ? "Termina" : "Renova"} em ${when(d.plan.periodEnd)}` : "Sem assinatura paga."}>
			{#if d.stripeUrl}
				<a class="mt-3 inline-flex items-center gap-1.5 text-sm font-semibold text-blup hover:underline" href={d.stripeUrl} target="_blank" rel="noreferrer">Abrir no Stripe <External class="size-3.5" /></a>
			{/if}
		</Card>

		<Card title="Sessões" help={`${d.sessions.length} ${d.sessions.length === 1 ? "aparelho conectado" : "aparelhos conectados"}`}>
			{#snippet action()}
				<Button variant="secondary" class="shrink-0 px-3 py-2 text-sm" {busy} disabled={!d.sessions.length} onclick={() => run(() => staffApi.revokeSessions(id), "Sessões encerradas")}>Encerrar todas</Button>
			{/snippet}
			<ul class="mt-3 grid gap-1.5 text-sm">
				{#each d.sessions as s (s.id)}
					<li class="truncate text-neutral-600 dark:text-neutral-400" title={s.userAgent ?? ""}>{when(s.createdAt)} · {s.ipAddress || "IP desconhecido"}</li>
				{/each}
			</ul>
		</Card>

		<Card title="Passkeys" help="Perdeu o aparelho? Remova a passkey: a pessoa entra pelo código por e-mail e cadastra outra.">
			<ul class="mt-3 grid gap-2">
				{#each d.passkeys as p (p.id)}
					<li class="flex items-center gap-3 rounded-lg border border-neutral-200 px-3 py-2 text-sm dark:border-neutral-800">
						<KeyRound class="size-4 text-neutral-500" />
						<span class="flex-1">{p.name || "Passkey"} <span class="text-neutral-500">· {when(p.createdAt)}</span></span>
						<button class="rounded-md p-1.5 text-neutral-500 hover:bg-neutral-100 hover:text-red-600 dark:hover:bg-neutral-800" aria-label="Remover passkey" title="Remover" disabled={busy} onclick={() => run(() => staffApi.removePasskey(id, p.id), "Passkey removida")}><Trash class="size-4" /></button>
					</li>
				{:else}
					<li class="text-sm text-neutral-500">Nenhuma passkey.</li>
				{/each}
			</ul>
		</Card>

		<Card title="Histórico da staff" help="O que a equipe fez nesta conta.">
			<ul class="mt-3 grid gap-1.5 text-sm">
				{#each d.history as h (h.id)}
					<li><b>{actionLabel[h.action] ?? h.action}</b> <span class="text-neutral-500">· {h.staffEmail} · {when(h.createdAt)}</span>{#if h.detail}<br /><span class="text-neutral-500">{h.detail}</span>{/if}</li>
				{:else}
					<li class="text-neutral-500">Nada ainda.</li>
				{/each}
			</ul>
		</Card>
	</div>

	{#if admin}
		<div class="mt-8 rounded-xl border border-red-200 p-5 dark:border-red-900">
			<h2 class="font-bold text-red-700 dark:text-red-400">Administração</h2>
			<div class="mt-3 flex flex-wrap gap-3">
				{#if d.user.banned}
					<Button variant="secondary" {busy} onclick={() => run(() => staffApi.unban(id), "Conta desbloqueada")}>Desbloquear</Button>
				{:else}
					<Button variant="secondary" {busy} onclick={() => { reason = ""; banOpen = true; }}>Bloquear a conta</Button>
				{/if}
				<Button variant="secondary" class="text-red-600" {busy} onclick={() => { confirmEmail = ""; delOpen = true; }}>Excluir a conta</Button>
			</div>
		</div>
	{/if}

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

	<Modal bind:open={delOpen} title="Excluir a conta">
		<p class="text-sm text-neutral-500">Cancela a assinatura no Stripe e apaga a conta, as sessões e as passkeys. Não dá para desfazer.</p>
		<form class="mt-4 grid gap-3" onsubmit={async (e) => { e.preventDefault(); delOpen = false; busy = true; try { await staffApi.remove(id); toast("Conta excluída"); goto("/staff"); } catch { toast("Não deu certo. Tente de novo.", { kind: "error" }); } finally { busy = false; } }}>
			<Field label={`Digite ${d.user.email} para confirmar`} bind:value={confirmEmail} autocomplete="off" />
			<div class="flex justify-end gap-2">
				<Button variant="secondary" type="button" onclick={() => (delOpen = false)}>Cancelar</Button>
				<Button class="bg-red-600! text-white!" disabled={confirmEmail.trim().toLowerCase() !== d.user.email.toLowerCase()}>Excluir</Button>
			</div>
		</form>
	</Modal>
{/if}
