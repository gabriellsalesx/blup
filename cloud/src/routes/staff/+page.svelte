<script lang="ts">
	import { resolve } from "$app/paths";
	import AccountId from "#lib/ui/AccountId.svelte";
	import PageHeader from "#lib/ui/PageHeader.svelte";
	import Search from "@lucide/svelte/icons/search";
	import { staffApi, staffHref, when, type Stats } from "#lib/staff.ts";
	import { createPager } from "#lib/pager.svelte.ts";
	import Pager from "#lib/ui/Pager.svelte";
	import Stat from "#lib/ui/Stat.svelte";

	// Accounts: how many there are (per plan), then the list, newest first,
	// found by the start of the e-mail or the ID, a page at a time.
	let { data } = $props();
	let q = $state("");
	const stats = $derived<Stats | null>(data.stats);
	// A fresh list whenever the route loads its first page again.
	const pager = $derived(createPager((cursor) => staffApi.users(q, cursor), data.first));
	let timer: ReturnType<typeof setTimeout>;

	const typed = () => { clearTimeout(timer); timer = setTimeout(() => pager.reset(), 250); };
</script>

<PageHeader title="Contas"/>

{#if stats}
	<div class="mt-6 grid grid-cols-2 gap-3 lg:grid-cols-4">
		<Stat label="Contas" value={stats.total} hint={`${stats.recent.toLocaleString("pt-BR")} nos últimos 7 dias`} />
		{#each stats.plans as p (p.name)}
			<Stat label={`Plano ${p.label}`} value={p.count} hint={stats.total ? `${Math.round((p.count / stats.total) * 100)}% das contas` : undefined} />
		{/each}
		{#if stats.banned}<Stat label="Bloqueadas" value={stats.banned} />{/if}
	</div>
{/if}

<label class="relative mt-6 block">
	<Search class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-neutral-400" />
	<input
		class="w-full rounded-lg border border-neutral-300 bg-white py-2.5 pr-3 pl-9 outline-none focus:border-blup dark:border-neutral-700 dark:bg-neutral-900"
		placeholder="E-mail ou ID da conta"
		aria-label="Buscar conta"
		bind:value={q}
		oninput={typed}
	/>
</label>

{#if pager.items.length}
	<div class="mt-5 overflow-hidden rounded-xl border border-neutral-200 dark:border-neutral-800">
		<table class="w-full text-left text-sm">
			<thead class="bg-neutral-50 text-xs text-neutral-500 uppercase dark:bg-neutral-900">
				<tr><th class="px-4 py-2.5 font-semibold">E-mail</th><th class="px-4 py-2.5 font-semibold">Plano</th><th class="px-4 py-2.5 font-semibold">Criada em</th></tr>
			</thead>
			<tbody>
				{#each pager.items as r (r.id)}
					<tr class="border-t border-neutral-200 hover:bg-neutral-50 dark:border-neutral-800 dark:hover:bg-neutral-900">
						<td class="px-4 py-2.5">
							<a class="font-semibold hover:underline" href={staffHref(resolve("/staff/accounts/[id]", { id: r.id }))}>{r.email ?? "Sem e-mail (só passkey)"}</a>
							{#if !r.email}<div class="mt-1"><AccountId id={r.id} short /></div>{/if}
							{#if r.banned}<span class="ml-2 rounded-full bg-red-100 px-2 py-0.5 text-xs font-semibold text-red-700 dark:bg-red-950 dark:text-red-300">Bloqueada</span>{/if}
						</td>
						<td class="px-4 py-2.5">{r.plan}</td>
						<td class="px-4 py-2.5 text-neutral-500">{when(r.createdAt)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	<Pager {pager} />
{:else if !pager.busy}
	<p class="mt-6 text-sm text-neutral-500">Nenhuma conta encontrada.</p>
{/if}
