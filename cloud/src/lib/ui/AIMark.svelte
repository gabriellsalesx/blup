<script lang="ts">
	import type { AIMarkKind } from "#lib/staff-ai.ts";

	// The mark of where an AI list runs: Blup Cloud's blup and Deals' coral
	// drop blup, each as its own favicon (static/blup.svg and
	// deals/static/favicon.svg), or OpenRouter's logo (Simple Icons 16.36.0,
	// CC0, unchanged; one color, so it follows the text).
	let { kind, class: extra = "size-6" }: { kind: AIMarkKind; class?: string } = $props();
	const openRouter = "M16.778 1.844v1.919q-.569-.026-1.138-.032-.708-.008-1.415.037c-1.93.126-4.023.728-6.149 2.237-2.911 2.066-2.731 1.95-4.14 2.75-.396.223-1.342.574-2.185.798-.841.225-1.753.333-1.751.333v4.229s.768.108 1.61.333c.842.224 1.789.575 2.185.799 1.41.798 1.228.683 4.14 2.75 2.126 1.509 4.22 2.11 6.148 2.236.88.058 1.716.041 2.555.005v1.918l7.222-4.168-7.222-4.17v2.176c-.86.038-1.611.065-2.278.021-1.364-.09-2.417-.357-3.979-1.465-2.244-1.593-2.866-2.027-3.68-2.508.889-.518 1.449-.906 3.822-2.59 1.56-1.109 2.614-1.377 3.978-1.466.667-.044 1.418-.017 2.278.02v2.176L24 6.014Z";
</script>

{#if kind === "openrouter"}
	<svg class="shrink-0 text-neutral-900 dark:text-white {extra}" viewBox="0 0 24 24" role="img" aria-label="OpenRouter"><path fill="currentColor" d={openRouter} /></svg>
{:else if kind === "deals"}
	<svg class="shrink-0 {extra}" viewBox="0 0 200 200" fill="none" stroke="#0e1b45" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" role="img" aria-label="Blup Deals">
		<ellipse cx="101" cy="183" rx="69" ry="10" fill="#0e1b45" stroke="none" opacity=".12" />
		<path d="M100 26C121 26 177 103 177 137C177 165 155 176 103 176C53 176 25 164 25 139C25 105 79 26 100 26Z" fill="#ff7a59" />
		<path d="M54 111Q59 92 74 77" fill="none" stroke="#fffaf0" opacity=".8" />
		<g transform="translate(103 123) rotate(-9)"><rect x="-5" y="-13" width="10" height="26" rx="3" fill="#0B1220" stroke="none" /></g>
		<g transform="translate(136 120) rotate(-9)"><rect x="-4.5" y="-12" width="9" height="24" rx="2.7" fill="#0B1220" stroke="none" /></g>
	</svg>
{:else}
	<svg class="shrink-0 {extra}" viewBox="0 0 64 64" role="img" aria-label="Blup Cloud">
		<path fill="#a9ddf9" stroke="#0b1220" stroke-width="1.5" d="M32 4c19 0 29 8 29 28S52 60 32 60 3 52 3 32 13 4 32 4Z" />
		<g fill="#0b1220"><rect x="31" y="23" width="7" height="18" rx="2.5" transform="rotate(-12 34.5 32)" /><rect x="47" y="23" width="6" height="17" rx="2" transform="rotate(-12 50 31.5)" /></g>
	</svg>
{/if}
