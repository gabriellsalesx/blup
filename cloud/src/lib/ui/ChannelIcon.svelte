<script lang="ts">
	import Webhook from "@lucide/svelte/icons/webhook";
	import BrandIcon from "./BrandIcon.svelte";
	// The mark of a kind of destination: Discord, Telegram, Slack and
	// WhatsApp with their own (Simple Icons) and a generic webhook with
	// Lucide's "webhook".
	let { type, class: extra = "size-5" }: { type: "discord" | "telegram" | "slack" | "whatsapp" | "webhook"; class?: string } = $props();
</script>

{#if type !== "webhook"}<BrandIcon name={type} class={extra} />
{:else}<Webhook class="shrink-0 {extra} text-neutral-700 dark:text-neutral-300" aria-label="Webhook" />{/if}
