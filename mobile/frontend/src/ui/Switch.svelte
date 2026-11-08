<script lang="ts">
  import { NotesService } from "../../bindings/github.com/gabriellsalesx/blup/mobile";

  let { label, hint = "", checked, onchange }: { label: string; hint?: string; checked: boolean; onchange: (on: boolean) => void } = $props();
</script>

<button type="button" role="switch" aria-checked={checked} onclick={() => { NotesService.Haptic(); onchange(!checked); }}>
  <span>{label}{#if hint}<small>{hint}</small>{/if}</span><i></i>
</button>

<style>
  button { display: flex; align-items: center; justify-content: space-between; gap: 1rem; width: 100%; min-height: 2.75rem; font-size: .9rem; font-weight: 600; text-align: start; }
  small { display: block; margin-block-start: .1rem; font-size: .85rem; font-weight: 400; line-height: 1.3; color: var(--muted); }
  i { position: relative; flex: none; width: 2.75rem; height: 1.6rem; border-radius: 999px; background: var(--line); transition: background .15s; }
  i::after { content: ""; position: absolute; inset-block-start: .2rem; inset-inline-start: .2rem; width: 1.2rem; height: 1.2rem; border-radius: 50%; background: #fff; transition: transform .15s; }
  [aria-checked="true"] i { background: var(--blue); }
  [aria-checked="true"] i::after { transform: translateX(1.15rem); }
</style>
