<script lang="ts">
	import { goto } from "$app/navigation";
	import { api } from "#lib/api.ts";
	import { afterSignIn, authClient } from "#lib/auth.ts";
	import SignInForm from "#lib/ui/SignInForm.svelte";
</script>

<SignInForm client={authClient} siteKey={async () => (await api.config()).turnstileSiteKey} ondone={() => void goto(afterSignIn(), { replace: true })} />
