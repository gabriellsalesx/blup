<script lang="ts">
	import { goto, invalidateAll } from "$app/navigation";
	import KeyRound from "@lucide/svelte/icons/key-round";
	import Trash from "@lucide/svelte/icons/trash-2";
	import { authClient, cancelled, message } from "#lib/auth.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import Button from "#lib/ui/Button.svelte";
	import Field from "#lib/ui/Field.svelte";
	import Modal from "#lib/ui/Modal.svelte";
	import Row from "#lib/ui/Row.svelte";

	// The profile: name, e-mail, passkeys, leaving every device, deleting.
	let { data } = $props();
	const passkeys = authClient.useListPasskeys();
	let busy = $state(false);
	let nameOpen = $state(false);
	let name = $state("");
	let delOpen = $state(false);
	let confirmEmail = $state("");

	const date = (d: Date | string) => new Date(d).toLocaleDateString("pt-BR", { day: "numeric", month: "short", year: "numeric" });

	async function saveName(e: SubmitEvent) {
		e.preventDefault();
		const { error } = await authClient.updateUser({ name: name.trim() });
		if (error) return toast(message(error), { kind: "error" });
		nameOpen = false;
		toast("Nome salvo");
		await invalidateAll();
	}

	async function addPasskey() {
		busy = true;
		const { error } = await authClient.passkey.addPasskey();
		busy = false;
		if (error && !cancelled(error)) toast(message(error), { kind: "error" });
		else if (!error) toast("Passkey adicionada");
	}

	async function removePasskey(id: string) {
		const { error } = await authClient.passkey.deletePasskey({ id });
		if (error) toast(message(error), { kind: "error" });
		else toast("Passkey removida");
	}

	async function leaveEverywhere() {
		busy = true;
		await authClient.revokeSessions();
		await authClient.signOut();
		goto("/entrar", { replaceState: true });
	}

	async function deleteAccount(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const { error } = await authClient.deleteUser();
		busy = false;
		if (error) {
			delOpen = false;
			// Better Auth asks for a recent sign in before deleting.
			return toast(error.code === "SESSION_EXPIRED" ? "Por segurança, saia e entre de novo antes de excluir a conta." : message(error), { kind: "error" });
		}
		goto("/entrar", { replaceState: true });
		toast("Conta excluída. Até logo!");
	}
</script>

<h1 class="text-3xl font-extrabold tracking-tight">Seu perfil</h1>

<div class="mt-6">
	<Row title="Nome" value={data.me.name || "Sem nome"}>
		{#snippet action()}<Button variant="secondary" onclick={() => { name = data.me.name; nameOpen = true; }}>Alterar</Button>{/snippet}
	</Row>

	<Row title="E-mail" value={data.me.email} help="É para ele que mandamos o código de entrar." />

	<Row title="Passkeys" help="Entre com a digital, o rosto ou o PIN do aparelho, sem esperar o código por e-mail.">
		{#snippet action()}<Button variant="secondary" {busy} onclick={addPasskey}>Adicionar</Button>{/snippet}
		{#if $passkeys.data?.length}
			<ul class="mt-3 grid gap-2">
				{#each $passkeys.data as p (p.id)}
					<li class="flex items-center gap-3 rounded-lg border border-neutral-200 px-3 py-2 dark:border-neutral-800">
						<KeyRound class="size-4 text-neutral-500" />
						<span class="min-w-0 flex-1 text-sm"><b>{p.name || "Passkey"}</b> <span class="text-neutral-500">· adicionada em {date(p.createdAt)}</span></span>
						<button class="rounded-md p-1.5 text-neutral-500 hover:bg-neutral-100 hover:text-red-600 dark:hover:bg-neutral-800" title="Remover" aria-label="Remover passkey" onclick={() => removePasskey(p.id)}><Trash class="size-4" /></button>
					</li>
				{/each}
			</ul>
		{/if}
	</Row>

	<Row title="Sair de todos os lugares" help="Desconecta todos os aparelhos e o Blup Studio. Você entra de novo em cada um.">
		{#snippet action()}<Button variant="secondary" {busy} onclick={leaveEverywhere}>Sair</Button>{/snippet}
	</Row>

	<Row title="Excluir conta" help="Apaga a conta, cancela a assinatura e remove as notas guardadas na nuvem. Pedimos uma confirmação antes.">
		{#snippet action()}<Button class="bg-red-700! text-white!" onclick={() => { confirmEmail = ""; delOpen = true; }}>Excluir</Button>{/snippet}
	</Row>
</div>

<Modal bind:open={nameOpen} title="Seu nome">
	<form class="grid gap-3" onsubmit={saveName}>
		<Field label="Nome" maxlength={80} bind:value={name} />
		<div class="flex justify-end gap-2">
			<Button variant="secondary" type="button" onclick={() => (nameOpen = false)}>Cancelar</Button>
			<Button>Salvar</Button>
		</div>
	</form>
</Modal>

<Modal bind:open={delOpen} title="Excluir a conta">
	<p class="text-sm text-neutral-500">Não dá para desfazer. A assinatura é cancelada na hora e as notas da nuvem são apagadas (as do seu computador continuam lá).</p>
	<form class="mt-4 grid gap-3" onsubmit={deleteAccount}>
		<Field label={`Digite ${data.me.email} para confirmar`} autocomplete="off" bind:value={confirmEmail} />
		<div class="flex justify-end gap-2">
			<Button variant="secondary" type="button" onclick={() => (delOpen = false)}>Cancelar</Button>
			<Button class="bg-red-700! text-white!" {busy} disabled={confirmEmail.trim().toLowerCase() !== data.me.email.toLowerCase()}>Excluir</Button>
		</div>
	</form>
</Modal>
