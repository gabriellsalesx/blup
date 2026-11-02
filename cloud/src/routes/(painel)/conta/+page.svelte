<script lang="ts">
	import { goto, invalidateAll } from "$app/navigation";
	import KeyRound from "@lucide/svelte/icons/key-round";
	import Trash from "@lucide/svelte/icons/trash-2";
	import { authClient, cancelled, message } from "#lib/auth.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import { confirm } from "#lib/confirm.svelte.ts";
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
	// Linking an e-mail (accounts made with a passkey): the address, then the
	// code sent to it.
	let linkOpen = $state(false);
	let linkStep = $state<"email" | "code">("email");
	let newEmail = $state("");
	let linkCode = $state("");
	// What to type to confirm deleting: the e-mail, or a word when there is none.
	const confirmWord = $derived(data.me.email ?? "EXCLUIR");

	const date = (d: Date | string) => new Date(d).toLocaleDateString("pt-BR", { day: "numeric", month: "short", year: "numeric" });

	async function saveName(e: SubmitEvent) {
		e.preventDefault();
		const { error } = await authClient.updateUser({ name: name.trim() });
		if (error) return toast(message(error), { kind: "error" });
		nameOpen = false;
		toast("Nome salvo");
		await invalidateAll();
	}

	async function sendLinkCode(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const { error } = await authClient.emailOtp.requestEmailChange({ newEmail: newEmail.trim() });
		busy = false;
		if (error) return toast(message(error), { kind: "error" });
		linkStep = "code";
	}

	async function linkEmail(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const { error } = await authClient.emailOtp.changeEmail({ newEmail: newEmail.trim(), otp: linkCode.trim() });
		busy = false;
		if (error) return toast(message(error), { kind: "error" });
		linkOpen = false;
		toast("E-mail vinculado");
		await invalidateAll();
	}

	async function addPasskey() {
		busy = true;
		const { error } = await authClient.passkey.addPasskey();
		busy = false;
		if (error && !cancelled(error)) toast(message(error), { kind: "error" });
		else if (!error) toast("Passkey adicionada");
	}

	async function removePasskey(id: string, name: string) {
		if (!(await confirm({ title: "Remover a passkey?", message: `"${name}" deixa de entrar na sua conta. Você pode adicionar outra depois.`, confirmLabel: "Remover", danger: true }))) return;
		const { error } = await authClient.passkey.deletePasskey({ id });
		if (error) toast(message(error), { kind: "error" });
		else toast("Passkey removida");
	}

	async function leaveEverywhere() {
		if (!(await confirm({ title: "Sair de todos os lugares?", message: "Todos os aparelhos e o Blup Studio saem da sua conta, inclusive este. Você entra de novo em cada um.", confirmLabel: "Sair de todos" }))) return;
		busy = true;
		await authClient.revokeSessions();
		await authClient.signOut();
		goto("/entrar", { replaceState: true });
	}

	async function deleteAccount() {
		const ok = await confirm({
			title: "Excluir a conta?",
			message: "Suas sessões são encerradas, os sites saem do ar e o cancelamento da assinatura é solicitado. O suporte pode recuperar os dados por 30 dias. Depois, a limpeza apaga os dados da nuvem de vez. As notas do computador continuam lá.",
			confirmLabel: "Excluir",
			danger: true,
			typeToConfirm: confirmWord,
		});
		if (!ok) return;
		busy = true;
		const { error } = await authClient.deleteUser();
		busy = false;
		if (error) {
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

	{#if data.me.email}
		<Row title="E-mail" value={data.me.email} help="É para ele que mandamos o código de entrar." />
	{:else}
		<Row title="E-mail" value="Nenhum e-mail" help="Vincule um e-mail para entrar por código, assinar um plano e recuperar a conta se perder o aparelho da passkey.">
			{#snippet action()}<Button onclick={() => { newEmail = ""; linkCode = ""; linkStep = "email"; linkOpen = true; }}>Vincular</Button>{/snippet}
		</Row>
	{/if}

	<Row title="Passkeys" help="Entre com a digital, o rosto ou o PIN do aparelho, sem esperar o código por e-mail.">
		{#snippet action()}<Button variant="secondary" {busy} onclick={addPasskey}>Adicionar</Button>{/snippet}
		{#if $passkeys.data?.length}
			<ul class="mt-3 grid gap-2">
				{#each $passkeys.data as p (p.id)}
					<li class="flex items-center gap-3 rounded-lg border border-neutral-200 px-3 py-2 dark:border-neutral-800">
						<KeyRound class="size-4 text-neutral-500" />
						<span class="min-w-0 flex-1 text-sm"><b>{p.name || "Passkey"}</b> <span class="text-neutral-500">· adicionada em {date(p.createdAt)}</span></span>
						<button class="rounded-md p-1.5 text-neutral-500 hover:bg-neutral-100 hover:text-red-600 dark:hover:bg-neutral-800" title="Remover" aria-label="Remover passkey" onclick={() => removePasskey(p.id, p.name || "Passkey")}><Trash class="size-4" /></button>
					</li>
				{/each}
			</ul>
		{/if}
	</Row>

	<Row title="Sair de todos os lugares" help="Desconecta todos os aparelhos e o Blup Studio. Você entra de novo em cada um.">
		{#snippet action()}<Button variant="secondary" {busy} onclick={leaveEverywhere}>Sair</Button>{/snippet}
	</Row>

	<Row title="Excluir conta" help="Encerra o acesso agora e solicita o cancelamento da assinatura. O suporte pode recuperar os dados por 30 dias; depois ocorre a exclusão definitiva.">
		{#snippet action()}<Button class="bg-red-700! text-white!" onclick={deleteAccount}>Excluir</Button>{/snippet}
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

<Modal bind:open={linkOpen} title="Vincular e-mail">
	{#if linkStep === "email"}
		<form class="grid gap-3" onsubmit={sendLinkCode}>
			<p class="text-sm text-neutral-500">Mandamos um código para confirmar que o e-mail é seu.</p>
			<Field label="E-mail" type="email" autocomplete="email" required bind:value={newEmail} />
			<div class="flex justify-end gap-2">
				<Button variant="secondary" type="button" onclick={() => (linkOpen = false)}>Cancelar</Button>
				<Button {busy} disabled={!newEmail.trim()}>Receber código</Button>
			</div>
		</form>
	{:else}
		<form class="grid gap-3" onsubmit={linkEmail}>
			<p class="text-sm text-neutral-500">Digite o código que mandamos para <b>{newEmail}</b>.</p>
			<Field class="text-center text-2xl font-bold tracking-[0.5em]" inputmode="numeric" autocomplete="one-time-code" maxlength={6} aria-label="Código" required bind:value={linkCode} />
			<div class="flex justify-end gap-2">
				<Button variant="secondary" type="button" onclick={() => (linkStep = "email")}>Voltar</Button>
				<Button {busy} disabled={linkCode.trim().length < 6}>Vincular</Button>
			</div>
		</form>
	{/if}
</Modal>

