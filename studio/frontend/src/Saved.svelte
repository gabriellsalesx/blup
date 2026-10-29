<script lang="ts">
  // "Saved ✓" or the error after a save; busy while saving.
  import { t } from "./i18n";
  let { state }: { state: { busy: boolean; ok: boolean; error: string } } = $props();
</script>

{#if state.error}<span class="bad">{state.error}</span>{:else if state.ok}<span class="ok">{t.savedOk}</span>{/if}
