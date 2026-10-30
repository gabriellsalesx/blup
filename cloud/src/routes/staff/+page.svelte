<script lang="ts">
	import Search from "@lucide/svelte/icons/search";
	import { staffApi, when, type UserRow } from "#lib/staff.ts";
	import { toast } from "#lib/toast.svelte.ts";

	// Accounts: the newest, or found by the start of the e-mail or the ID.
	let q = $state("");
	let rows = $state<UserRow[] | null>(null);
	let timer: ReturnType<typeof setTimeout>;

	async function search() {
		try { rows = await staffApi.users(q); }
		catch { toast("Não deu para buscar.", { kind: "error" }); }
	}
	search();
	const typed = () => { clearTimeout(timer); timer = setTimeout(search, 250); };
</script>

<h1 class="text-2xl font-extrabold tracking-tight">Contas</h1>
<label class="relative mt-5 block">
	<Search class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-neutral-400" />
	<input
		class="w-full rounded-lg border border-neutral-300 bg-white py-2.5 pr-3 pl-9 outline-none focus:border-blup dark:border-neutral-700 dark:bg-neutral-900"
		placeholder="E-mail ou ID da conta"
		aria-label="Buscar conta"
		bind:value={q}
		oninput={typed}
	/>
</label>

{#if rows}
	{#if rows.length}
		<div class="mt-5 overflow-hidden rounded-xl border border-neutral-200 dark:border-neutral-800">
			<table class="w-full text-left text-sm">
				<thead class="bg-neutral-50 text-xs text-neutral-500 uppercase dark:bg-neutral-900">
					<tr><th class="px-4 py-2.5 font-semibold">E-mail</th><th class="px-4 py-2.5 font-semibold">Plano</th><th class="px-4 py-2.5 font-semibold">Criada em</th></tr>
				</thead>
				<tbody>
					{#each rows as r (r.id)}
						<tr class="border-t border-neutral-200 hover:bg-neutral-50 dark:border-neutral-800 dark:hover:bg-neutral-900">
							<td class="px-4 py-2.5">
								<a class="font-semibold hover:underline" href="/staff/contas/{r.id}">{r.email ?? "Sem e-mail (só passkey)"}</a>
								{#if r.banned}<span class="ml-2 rounded-full bg-red-100 px-2 py-0.5 text-xs font-semibold text-red-700 dark:bg-red-950 dark:text-red-300">Bloqueada</span>{/if}
							</td>
							<td class="px-4 py-2.5">{r.plan}</td>
							<td class="px-4 py-2.5 text-neutral-500">{when(r.createdAt)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<p class="mt-6 text-sm text-neutral-500">Nenhuma conta encontrada.</p>
	{/if}
{/if}
