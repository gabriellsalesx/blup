<script lang="ts">
	import type { PageProps } from "./$types";
	import Trash from "@lucide/svelte/icons/trash-2";
	import { staffApi, roleLabel, when, type Role, type Team } from "#lib/staff.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import { confirm } from "#lib/confirm.svelte.ts";
	import Button from "#lib/ui/Button.svelte";
	import Row from "#lib/ui/Row.svelte";
	import Field from "#lib/ui/Field.svelte";
	import Select from "#lib/ui/Select.svelte";
	import PageHeader from "#lib/ui/PageHeader.svelte";

	// Who is on the staff. Each one also needs to be allowed in Cloudflare
	// Access (the door); this list is the role inside.
	let { data }: PageProps = $props();
	let team = $derived<Team | null>(data.team);
	let email = $state("");
	let role = $state<Role>("support");
	let busy = $state(false);

	const load = () => staffApi.team().then((t) => (team = t)).catch(() => toast("Não deu para carregar a equipe.", { kind: "error" }));

	async function add(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try { await staffApi.setMember(email.trim(), role); toast("Equipe atualizada"); email = ""; await load(); }
		catch { toast("Confira o e-mail.", { kind: "error" }); }
		finally { busy = false; }
	}
	async function remove(m: string) {
		if (!(await confirm({ title: "Tirar da equipe?", message: `${m} perde o acesso à staff na hora. Lembre de tirar também do Cloudflare Access.`, confirmLabel: "Tirar", danger: true }))) return;
		try { await staffApi.removeMember(m); toast("Saiu da equipe"); await load(); }
		catch { toast("Não deu certo.", { kind: "error" }); }
	}
	async function change(m: string, r: Role) {
		try { await staffApi.setMember(m, r); toast("Papel alterado"); await load(); }
		catch { toast("Não deu certo.", { kind: "error" }); }
	}
	const allRoles = [{ value: "support", label: roleLabel("support") }, { value: "admin", label: roleLabel("admin") }] as const satisfies readonly { value: Role; label: string }[];
	// Making, unmaking or removing an admin: the owner (STAFF_OWNER_EMAIL)
	// only. The API refuses the same; here the controls are just off.
	const owner = $derived(!!data.me?.owner);
	const roles = $derived(owner ? allRoles : allRoles.filter((r) => r.value === "support"));
	const locked = (m: { email: string; role: Role }) => m.email === data.me?.email || (!owner && m.role === "admin");
</script>

<PageHeader title="Equipe" lead="Suporte vê as contas e ajuda (sessões, passkeys). Admin também bloqueia, exclui contas e cuida do suporte. Só o dono promove, rebaixa ou tira um admin. Libere o e-mail também no Cloudflare Access." />

<div class="mt-6">
	<Row title="Adicionar" help="Quem entra pela primeira vez já fica com o papel escolhido.">
		<form class="mt-3 flex flex-wrap items-end gap-3" onsubmit={add}>
			<div class="min-w-64 flex-1"><Field label="E-mail" type="email" required bind:value={email} /></div>
			<Select compact surface="bg-white dark:bg-neutral-900" label="Papel" options={roles} bind:value={role} />
			<Button {busy}>Adicionar</Button>
		</form>
	</Row>

	{#if team}
		<Row title="Quem está na equipe">
			<ul class="mt-3 grid gap-2 text-sm">
				{#if team.owner}<li class="flex items-center gap-3 rounded-lg border border-neutral-200 px-3 py-2.5 dark:border-neutral-800"><span class="flex-1 font-semibold">{team.owner}</span><span class="text-neutral-500">Admin (dono)</span></li>{/if}
				{#each team.staff as m (m.email)}
					<li class="flex items-center gap-3 rounded-lg border border-neutral-200 px-3 py-2.5 dark:border-neutral-800">
						<span class="flex-1"><b>{m.email}</b><br /><span class="text-xs text-neutral-500">por {m.addedBy} · {when(m.createdAt)}</span></span>
						<Select compact surface="bg-white dark:bg-neutral-900" label="Papel de {m.email}" options={allRoles} value={m.role} disabled={!owner || m.email === data.me?.email} onchange={(next) => change(m.email, next)} />
						<button class="rounded-md p-1.5 text-neutral-500 hover:bg-neutral-100 hover:text-red-600 disabled:opacity-40 dark:hover:bg-neutral-800" aria-label="Tirar da equipe" title={!owner && m.role === "admin" ? "Só o dono tira um admin da equipe" : "Tirar da equipe"} disabled={locked(m)} onclick={() => remove(m.email)}><Trash class="size-4" /></button>
					</li>
				{/each}
			</ul>
		</Row>
	{/if}
</div>
