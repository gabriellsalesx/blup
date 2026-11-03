<script lang="ts">
	import Webhook from "@lucide/svelte/icons/webhook";
	import BrandIcon from "./BrandIcon.svelte";
	// The mark of a kind of alert destination: Discord, Telegram and Slack
	// with their own (Simple Icons) and a generic webhook with Lucide's
	// "webhook".
	let { type, class: extra = "size-5" }: { type: "discord" | "telegram" | "slack" | "webhook"; class?: string } = $props();
</script>

{#if type !== "webhook"}<BrandIcon name={type} class={extra} />
{:else}<Webhook class="{extra} text-neutral-700 dark:text-neutral-300" aria-label="Webhook" />{/if}
