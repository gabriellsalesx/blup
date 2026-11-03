<script lang="ts">
	import type { PageProps } from "./$types";
	import { resolve } from "$app/paths";
 import {t,language,intlLocale} from "#lib/i18n.svelte.ts";
	import AccountId from "#lib/ui/AccountId.svelte";
	import PageHeader from "#lib/ui/PageHeader.svelte";
	import { accountIdHelp } from "#lib/account-id.ts";
	import { goto, invalidate, refreshAll } from "$app/navigation";
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
	let { data }: PageProps = $props();
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
	const confirmWord = $derived(data.me.email ?? t("deleteWord"));

	const date = (d: Date | string) => new Date(d).toLocaleDateString(intlLocale(), { day: "numeric", month: "short", year: "numeric" });

	async function saveName(e: SubmitEvent) {
		e.preventDefault();
		const { error } = await authClient.updateUser({ name: name.trim() });
		if (error) return toast(()=>message(error), { kind: "error" });
		nameOpen = false;
		toast(()=>t("nameSaved"));
		await refreshAll();
	}

	async function sendLinkCode(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const { error } = await authClient.emailOtp.requestEmailChange({ newEmail: newEmail.trim() });
		busy = false;
		if (error) return toast(()=>message(error), { kind: "error" });
		linkStep = "code";
	}

	async function linkEmail(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const { error } = await authClient.emailOtp.changeEmail({ newEmail: newEmail.trim(), otp: linkCode.trim() });
		busy = false;
		if (error) return toast(()=>message(error), { kind: "error" });
		linkOpen = false;
		toast(()=>t("emailLinked"));
		await refreshAll();
	}

	async function addPasskey() {
		busy = true;
		const { error } = await authClient.passkey.addPasskey();
		busy = false;
		if (error && !cancelled(error)) toast(()=>message(error), { kind: "error" });
		else if (!error) { toast(()=>t("passkeyAdded")); await invalidate("app:passkeys"); }
	}

	async function removePasskey(id: string, name: string) {
		if (!(await confirm({ title: t("removePasskeyTitle"), message: t("removePasskeyBody",{name}), confirmLabel: t("remove"), danger: true }))) return;
		const { error } = await authClient.passkey.deletePasskey({ id });
		if (error) toast(()=>message(error), { kind: "error" });
		else { toast(()=>t("passkeyRemoved")); await invalidate("app:passkeys"); }
	}

	async function leaveEverywhere() {
		if (!(await confirm({ title: t("leaveAllTitle"), message: t("leaveAllBody"), confirmLabel: t("leaveAllConfirm") }))) return;
		busy = true;
		await authClient.revokeSessions();
		await authClient.signOut();
		goto(resolve("/(auth)/sign-in"), { replace: true });
	}

	async function deleteAccount() {
		const ok = await confirm({
			title: t("deleteAccountTitle"),
			message: t("deleteAccountBody"),
			confirmLabel: t("delete"),
			danger: true,
			typeToConfirm: confirmWord,
		});
		if (!ok) return;
		busy = true;
		const { error } = await authClient.deleteUser();
		busy = false;
		if (error) {
			// Better Auth asks for a recent sign in before deleting.
			return toast(()=>error?.code === "SESSION_EXPIRED" ? t("freshDelete") : message(error), { kind: "error" });
		}
		goto(resolve("/(auth)/sign-in"), { replace: true });
		toast(()=>t("accountDeleted"));
	}
</script>

<PageHeader title={t("profileTitle")}/>

<div class="mt-6">
 <Row title={t("accountId")}  help={accountIdHelp(language())}><div class="mt-2"><AccountId id={data.me.id} /></div></Row>
	<Row title={t("name")}  value={data.me.name || t("noName")}>
		{#snippet action()}<Button variant="secondary" onclick={() => { name = data.me.name; nameOpen = true; }}>{t("change")}</Button>{/snippet}
	</Row>

	{#if data.me.email}
		<Row title={t("email")}  value={data.me.email} help={t("emailHelp")}  />
	{:else}
		<p class="py-3 text-sm text-neutral-500">{t("emailNudge")} <button type="button" class="font-semibold text-blup hover:underline" onclick={()=>{newEmail="";linkCode="";linkStep="email";linkOpen=true;}}>{t("linkEmail")}</button></p>
		<Row title={t("email")}  value={t("noEmail")}  help={t("emailLinkHelp")} >
			{#snippet action()}<Button onclick={() => { newEmail = ""; linkCode = ""; linkStep = "email"; linkOpen = true; }}>{t("link")}</Button>{/snippet}
		</Row>
	{/if}

	<Row title={t("passkeys")}  help={t("passkeyHelp")} >
		{#snippet action()}<Button variant="secondary" {busy} onclick={addPasskey}>{t("add")}</Button>{/snippet}
		{#if data.passkeys.length}
			<ul class="mt-3 grid gap-2">
				{#each data.passkeys as p (p.id)}
					<li class="flex items-center gap-3 rounded-lg border border-neutral-200 px-3 py-2 dark:border-neutral-800">
						<KeyRound class="size-4 text-neutral-500" />
						<span class="min-w-0 flex-1 text-sm"><b>{p.name || t("passkey")}</b> <span class="text-neutral-500">· {t("passkeyAddedAt",{date:date(p.createdAt)})}</span></span>
						<button class="rounded-md p-1.5 text-neutral-500 hover:bg-neutral-100 hover:text-red-600 dark:hover:bg-neutral-800" title={t("remove")}  aria-label={t("removePasskey")}  onclick={() => removePasskey(p.id, p.name || t("passkey"))}><Trash class="size-4" /></button>
					</li>
				{/each}
			</ul>
		{/if}
	</Row>

	<Row title={t("leaveAll")}  help={t("leaveAllHelp")} >
		{#snippet action()}<Button variant="secondary" {busy} onclick={leaveEverywhere}>{t("signOut")}</Button>{/snippet}
	</Row>

	<Row title={t("deleteAccount")}  help={t("deleteAccountHelp")} >
		{#snippet action()}<Button class="bg-red-700! text-white!" onclick={deleteAccount}>{t("delete")}</Button>{/snippet}
	</Row>
</div>

<Modal bind:open={nameOpen} title={t("yourName")} >
	<form class="grid gap-3" onsubmit={saveName}>
		<Field label={t("name")}  maxlength={80} bind:value={name} />
		<div class="flex justify-end gap-2">
			<Button variant="secondary" type="button" onclick={() => (nameOpen = false)}>{t("cancel")}</Button>
			<Button>{t("save")}</Button>
		</div>
	</form>
</Modal>

<Modal bind:open={linkOpen} title={t("linkBeforeSubscribeAction")} >
	{#if linkStep === "email"}
		<form class="grid gap-3" onsubmit={sendLinkCode}>
			<p class="text-sm text-neutral-500">{t("linkEmailCode")}</p>
			<Field label={t("email")}  type="email" autocomplete="email" required bind:value={newEmail} />
			<div class="flex justify-end gap-2">
				<Button variant="secondary" type="button" onclick={() => (linkOpen = false)}>{t("cancel")}</Button>
				<Button {busy} disabled={!newEmail.trim()}>{t("getCode")}</Button>
			</div>
		</form>
	{:else}
		<form class="grid gap-3" onsubmit={linkEmail}>
			<p class="text-sm text-neutral-500">{t("enterSentCode",{email:newEmail})}</p>
			<Field class="text-center text-2xl font-bold tracking-[0.5em]" inputmode="numeric" autocomplete="one-time-code" maxlength={6} aria-label={t("code")}  required bind:value={linkCode} />
			<div class="flex justify-end gap-2">
				<Button variant="secondary" type="button" onclick={() => (linkStep = "email")}>{t("back")}</Button>
				<Button {busy} disabled={linkCode.trim().length < 6}>{t("link")}</Button>
			</div>
		</form>
	{/if}
</Modal>

