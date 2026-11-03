<script lang="ts">
	import type { Snippet } from "svelte";

	// A dark tip above something, shown on hover or keyboard focus (the
	// status page's). `text` may have line breaks; `align` keeps it inside
	// the screen at the ends of a row.
	let { text, align = "center", class: extra = "", children }: { text: string; align?: "center" | "start" | "end"; class?: string; children?: Snippet } = $props();
</script>

<span class="tip {extra}" data-tip={text} data-align={align}>{@render children?.()}</span>

<style>
	.tip { position: relative; }
	.tip:hover::after, .tip:focus-visible::after {
		content: attr(data-tip);
		position: absolute;
		bottom: calc(100% + 8px);
		left: 50%;
		transform: translateX(-50%);
		z-index: 30;
		padding: 4px 8px;
		border-radius: 6px;
		background: #262626;
		color: #fff;
		font: 500 13px/1.4 Inter, ui-sans-serif, system-ui, sans-serif;
		white-space: pre;
		text-align: center;
		pointer-events: none;
	}
	.tip:hover::before, .tip:focus-visible::before {
		content: "";
		position: absolute;
		bottom: calc(100% + 3px);
		left: 50%;
		transform: translateX(-50%);
		z-index: 30;
		border: 5px solid transparent;
		border-top-color: #262626;
		border-bottom: 0;
		pointer-events: none;
	}
	.tip[data-align="start"]:hover::after { left: 0; transform: none; }
	.tip[data-align="end"]:hover::after { left: auto; right: 0; transform: none; }
	:global([data-theme="dark"]) .tip:hover::after, :global([data-theme="dark"]) .tip:focus-visible::after { background: #f5f5f5; color: #171717; }
	:global([data-theme="dark"]) .tip:hover::before, :global([data-theme="dark"]) .tip:focus-visible::before { border-top-color: #f5f5f5; }
</style>
