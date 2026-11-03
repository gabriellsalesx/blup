<script lang="ts">
	import { resolve } from "$app/paths";
	import { goto } from "$app/navigation";
	import { authClient } from "#lib/auth.ts";

	// The start: the account when signed in, else the sign-in page.
	authClient.getSession().then(({ data }) => goto(data ? resolve("/(panel)/account") : resolve("/(auth)/sign-in"), { replace: true }));
</script>
