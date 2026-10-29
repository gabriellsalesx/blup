<script lang="ts">
  // The error of a save, next to its button (success is a toast).
  let { state }: { state: { busy: boolean; ok: boolean; error: string } } = $props();
</script>

{#if state.error}<span class="bad">{state.error}</span>{/if}
