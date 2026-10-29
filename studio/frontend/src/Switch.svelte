<script lang="ts">
  // An on/off switch with its label.
  let { checked = $bindable(false), label, help = "", disabled = false, onchange }: { checked?: boolean; label: string; help?: string; disabled?: boolean; onchange?: (v: boolean) => void } = $props();
</script>

<label class="switch" class:disabled>
  <span class="text"><b>{label}</b>{#if help}<small>{help}</small>{/if}</span>
  <input type="checkbox" role="switch" bind:checked {disabled} onchange={() => onchange?.(checked)} />
  <span class="track" aria-hidden="true"><span class="knob"></span></span>
</label>

<style>
  .switch { display: flex; align-items: center; justify-content: space-between; gap: 1.5rem; cursor: pointer; }
  .switch.disabled { opacity: .5; cursor: not-allowed; }
  .text { display: grid; gap: .15rem; }
  .text small { color: var(--muted); font-size: .8rem; }
  input { position: absolute; opacity: 0; pointer-events: none; }
  .track { flex: none; width: 2.6rem; height: 1.5rem; border-radius: 999px; background: var(--line); position: relative; transition: background .15s; }
  .knob { position: absolute; inset-block-start: .2rem; inset-inline-start: .2rem; width: 1.1rem; height: 1.1rem; border-radius: 50%; background: #fff; box-shadow: 0 1px 2px rgb(0 0 0 / .25); transition: transform .15s; }
  input:checked + .track { background: var(--blue); }
  input:checked + .track .knob { transform: translateX(1.1rem); }
  input:focus-visible + .track { outline: 2px solid var(--blue); outline-offset: 2px; }
</style>
