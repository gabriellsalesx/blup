<script lang="ts">
	import Webhook from "@lucide/svelte/icons/webhook";
	import Globe from "@lucide/svelte/icons/globe";
	import BrandIcon from "./BrandIcon.svelte";
	// The mark of a kind of destination: Discord, Telegram, Slack and
	// WhatsApp with their own (Simple Icons), a generic webhook with
	// Lucide's "webhook" and the Deals page with its "globe".
	let { type, class: extra = "size-5" }: { type: "discord" | "telegram" | "slack" | "whatsapp" | "webhook" | "site"; class?: string } = $props();
</script>

{#if type === "webhook"}<Webhook class="shrink-0 {extra} text-neutral-700 dark:text-neutral-300" aria-label="Webhook" />
{:else if type === "site"}<Globe class="shrink-0 {extra} text-neutral-700 dark:text-neutral-300" aria-label="Site" />
{:else}<BrandIcon name={type} class={extra} />{/if}
