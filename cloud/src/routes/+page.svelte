<script lang="ts">
	import { goto } from "$app/navigation";
	import { authClient } from "#lib/auth.ts";

	// The start: the account when signed in, else the sign-in page.
	authClient.getSession().then(({ data }) => goto(data ? "/conta" : "/entrar", { replaceState: true }));
</script>
