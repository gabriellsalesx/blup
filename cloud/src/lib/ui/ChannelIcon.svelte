<script lang="ts">
	import Hash from "@lucide/svelte/icons/hash";
	import Webhook from "@lucide/svelte/icons/webhook";
	import BrandIcon from "./BrandIcon.svelte";
	// The mark of a kind of alert destination: Discord and Telegram with
	// their own (Simple Icons), Slack with Lucide's "hash" in its color
	// (Simple Icons no longer carries Slack's mark) and a generic webhook
	// with Lucide's "webhook".
	let { type, class: extra = "size-5" }: { type: "discord" | "telegram" | "slack" | "webhook"; class?: string } = $props();
</script>

{#if type === "discord" || type === "telegram"}<BrandIcon name={type} class={extra} />
{:else if type === "slack"}<Hash class="{extra} text-[#4A154B] dark:text-[#E01E5A]" aria-label="Slack" />
{:else}<Webhook class="{extra} text-neutral-700 dark:text-neutral-300" aria-label="Webhook" />{/if}
