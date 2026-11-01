<script lang="ts">
  import type { HTMLInputAttributes } from "svelte/elements";

  // The Studio's text field: the label above, then a help line or the
  // error below. `large` is the bigger one of the creation screens.
  let { label, value = $bindable(""), help = "", error = "", type = "text", large = false, ...rest }: Omit<HTMLInputAttributes, "type" | "value"> & {
    label: string;
    value?: string;
    help?: string;
    error?: string;
    type?: "text" | "email" | "password" | "url";
    large?: boolean;
  } = $props();
</script>

<label class="field">
  <span>{label}</span>
  <input {type} bind:value class={{ large }} aria-invalid={!!error || undefined} {...rest} />
  {#if error}<small class="bad">{error}</small>{:else if help}<small>{help}</small>{/if}
</label>

<style>
  /* The name of a new site or vault. */
  .large { font-size: 1.25rem; font-weight: 600; padding: .7rem .8rem; }
</style>
