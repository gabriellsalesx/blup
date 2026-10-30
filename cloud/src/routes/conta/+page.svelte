<script lang="ts">
	import { goto } from "$app/navigation";
	import KeyRound from "@lucide/svelte/icons/key-round";
	import Plus from "@lucide/svelte/icons/plus";
	import Trash from "@lucide/svelte/icons/trash-2";
	import LogOut from "@lucide/svelte/icons/log-out";
	import { authClient, cancelled, message } from "#lib/auth.ts";

	// The account: who is signed in, the passkeys, signing out.
	const session = authClient.useSession();
	const passkeys = authClient.useListPasskeys();
	let error = $state("");
	let busy = $state(false);

	async function addPasskey() {
		error = "";
		busy = true;
		const { error: err } = await authClient.passkey.addPasskey();
		busy = false;
		if (err && !cancelled(err)) error = message(err);
	}

	async function removePasskey(id: string) {
		error = "";
		const { error: err } = await authClient.passkey.deletePasskey({ id });
		if (err) error = message(err);
	}

	async function signOut() {
		await authClient.signOut();
		goto("/entrar", { replaceState: true });
	}

	const date = (d: Date | string) => new Date(d).toLocaleDateString("pt-BR", { day: "numeric", month: "short", year: "numeric" });
</script>

{#if $session.data}
	<section class="w-full max-w-lg grid gap-6">
		<div>
			<h1 class="text-2xl font-extrabold tracking-tight">Sua conta</h1>
			<p class="mt-1 text-sm text-neutral-500">{$session.data.user.email}</p>
		</div>

		<div class="rounded-xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
			<div class="flex items-start justify-between gap-4">
				<div>
					<h2 class="font-bold">Passkeys</h2>
					<p class="mt-0.5 text-sm text-neutral-500">Entre com a digital, o rosto ou o PIN do aparelho, sem esperar o código por e-mail.</p>
				</div>
				<button class="flex shrink-0 items-center gap-1.5 rounded-lg bg-neutral-900 px-3 py-2 text-sm font-semibold text-white disabled:opacity-50 dark:bg-white dark:text-neutral-900" disabled={busy} onclick={addPasskey}>
					<Plus class="size-4" />Adicionar
				</button>
			</div>
			{#if $passkeys.data?.length}
				<ul class="mt-4 grid gap-2">
					{#each $passkeys.data as p (p.id)}
						<li class="flex items-center gap-3 rounded-lg border border-neutral-200 px-3 py-2.5 dark:border-neutral-800">
							<KeyRound class="size-4 text-neutral-500" />
							<span class="flex-1 min-w-0">
								<span class="block truncate text-sm font-semibold">{p.name || "Passkey"}</span>
								<span class="block text-xs text-neutral-500">Adicionada em {date(p.createdAt)}</span>
							</span>
							<button class="rounded-md p-1.5 text-neutral-500 hover:bg-neutral-100 hover:text-red-600 dark:hover:bg-neutral-800" title="Remover" aria-label="Remover passkey" onclick={() => removePasskey(p.id)}>
								<Trash class="size-4" />
							</button>
						</li>
					{/each}
				</ul>
			{:else if !$passkeys.isPending}
				<p class="mt-4 text-sm text-neutral-500">Nenhuma passkey ainda.</p>
			{/if}
		</div>

		{#if error}<p class="text-sm text-red-600" role="alert">{error}</p>{/if}

		<button class="flex items-center gap-2 justify-self-start text-sm font-semibold text-neutral-500 hover:text-neutral-900 dark:hover:text-white" onclick={signOut}>
			<LogOut class="size-4" />Sair
		</button>
	</section>
{/if}
