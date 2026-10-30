<script lang="ts">
	// The whole Blup gang on the sign-in panel (Aquarela's drawing), inline so
	// each Blup can hop and blink on its own rhythm.
	import scene from "#lib/assets/login-blups.svg?raw";
</script>

<div class="scene" aria-hidden="true">{@html scene}</div>

<style>
	.scene :global(svg) { width: 100%; height: auto; overflow: visible; }
	.scene :global(.blup),
	.scene :global(.eyes) { transform-box: fill-box; transform-origin: 50% 100%; }
	.scene :global(.blup) { animation: hop 3.2s ease-in-out infinite; }
	.scene :global(.b2) { animation-delay: 0.4s; animation-duration: 3.6s; }
	.scene :global(.b3) { animation-delay: 0.9s; animation-duration: 2.9s; }
	.scene :global(.b4) { animation-delay: 1.3s; animation-duration: 3.4s; }
	.scene :global(.b5) { animation-delay: 0.2s; animation-duration: 3.8s; }
	.scene :global(.b6) { animation-delay: 1.7s; animation-duration: 3.1s; }
	.scene :global(.b7) { animation-delay: 0.6s; animation-duration: 3.5s; }
	.scene :global(.b8) { animation-delay: 1.1s; animation-duration: 3s; }
	.scene :global(.b9) { animation-delay: 1.5s; animation-duration: 3.3s; }
	.scene :global(.eyes) { transform-origin: 50% 50%; animation: blink 5s infinite; }
	.scene :global(.b2 .eyes) { animation-delay: 1.2s; }
	.scene :global(.b3 .eyes) { animation-delay: 2.6s; }
	.scene :global(.b4 .eyes) { animation-delay: 0.7s; }
	.scene :global(.b5 .eyes) { animation-delay: 3.4s; }
	.scene :global(.b6 .eyes) { animation-delay: 1.9s; }
	@keyframes hop {
		0%, 70%, 100% { transform: translateY(0) scale(1, 1); }
		76% { transform: translateY(0) scale(1.08, 0.9); }
		84% { transform: translateY(-10px) scale(0.95, 1.06); }
		92% { transform: translateY(0) scale(1.04, 0.96); }
	}
	@keyframes blink {
		0%, 93%, 100% { transform: scaleY(1); }
		95% { transform: scaleY(0.1); }
	}
	@media (prefers-reduced-motion: reduce) {
		.scene :global(.blup), .scene :global(.eyes) { animation: none; }
	}
</style>
