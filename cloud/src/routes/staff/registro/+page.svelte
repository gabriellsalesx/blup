<script lang="ts">
	import { staffApi, staffHref, actionLabel, when } from "#lib/staff.ts";
	import { createPager } from "#lib/pager.svelte.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import Pager from "#lib/ui/Pager.svelte";

	// Every staff action, newest first, a page at a time.
	const pager = createPager((cursor) => staffApi.log(cursor));
	pager.reset().then(() => { if (pager.failed) toast("Não deu para carregar o registro.", { kind: "error" }); });
</script>

<h1 class="text-3xl font-extrabold tracking-tight">Registro</h1>
<p class="mt-1 text-sm text-neutral-500">Tudo o que a equipe fez: quem, o quê, em qual conta e quando.</p>

{#if pager.items.length || !pager.busy}
	<div class="mt-6 overflow-hidden rounded-xl border border-neutral-200 dark:border-neutral-800">
		<table class="w-full text-left text-sm">
			<thead class="bg-neutral-50 text-xs text-neutral-500 uppercase dark:bg-neutral-900">
				<tr><th class="px-4 py-2.5 font-semibold">Quando</th><th class="px-4 py-2.5 font-semibold">Quem</th><th class="px-4 py-2.5 font-semibold">O quê</th><th class="px-4 py-2.5 font-semibold">Conta</th></tr>
			</thead>
			<tbody>
				{#each pager.items as r (r.id)}
					<tr class="border-t border-neutral-200 dark:border-neutral-800">
						<td class="px-4 py-2.5 whitespace-nowrap text-neutral-500">{when(r.createdAt)}</td>
						<td class="px-4 py-2.5">{r.staffEmail}</td>
						<td class="px-4 py-2.5"><b>{actionLabel[r.action] ?? r.action}</b>{#if r.detail}<br /><span class="text-neutral-500">{r.detail}</span>{/if}</td>
						<td class="px-4 py-2.5">{#if r.userId}<a class="text-blup hover:underline" href={staffHref(`/staff/contas/${r.userId}`)}>abrir</a>{/if}</td>
					</tr>
				{:else}
					<tr><td class="px-4 py-6 text-neutral-500" colspan="4">Nada registrado ainda.</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
	<Pager {pager} />
{/if}
